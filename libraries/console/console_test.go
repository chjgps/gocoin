// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

package console

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"time"

	"gocoin/libraries/console/jsre"
	"gocoin/libraries/rpc"
)

const (
	testInstance = "console-tester"
	testAddress  = "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
)

type hookedPrompter struct {
	scheduler chan string
}

func (p *hookedPrompter) PromptInput(prompt string) (string, error) {
	select {
	case p.scheduler <- prompt:
	case <-time.After(time.Second):
		return "", errors.New("prompt timeout")
	}
	select {
	case input := <-p.scheduler:
		return input, nil
	case <-time.After(time.Second):
		return "", errors.New("input timeout")
	}
}

func (p *hookedPrompter) PromptPassword(prompt string) (string, error) {
	return "", errors.New("not implemented")
}
func (p *hookedPrompter) PromptConfirm(prompt string) (bool, error) {
	return false, errors.New("not implemented")
}
func (p *hookedPrompter) SetHistory(history []string)              {}
func (p *hookedPrompter) AppendHistory(command string)             {}
func (p *hookedPrompter) SetWordCompleter(completer WordCompleter) {}

type tester struct {
	workspace string
	console   *Console
	input     *hookedPrompter
	output    *bytes.Buffer
	server    *rpc.Server
	client    *rpc.Client
}

type Service struct{}

type Args struct {
	S string
}

func (s *Service) NoArgsRets() {
}

type Result struct {
	String string
	Int    int
	Args   *Args
}

func (s *Service) Name() string {
	return testInstance
}
func (s *Service) Echo(str string, i int, args *Args) Result {
	return Result{str, i, args}
}

func (s *Service) EchoWithCtx(ctx context.Context, str string, i int, args *Args) Result {
	return Result{str, i, args}
}

func (s *Service) Sleep(ctx context.Context, duration time.Duration) {
	select {
	case <-time.After(duration):
	case <-ctx.Done():
	}
}

func (s *Service) Rets() (string, error) {
	return "", nil
}

func (s *Service) InvalidRets1() (error, string) {
	return nil, ""
}

func (s *Service) InvalidRets2() (string, string) {
	return "", ""
}

func (s *Service) InvalidRets3() (string, string, error) {
	return "", "", nil
}

func (s *Service) Subscription(ctx context.Context) (*rpc.Subscription, error) {
	return nil, nil
}

func newTestServer(serviceName string, service interface{}) *rpc.Server {
	server := rpc.NewServer()
	if err := server.RegisterName(serviceName, service); err != nil {
		panic(err)
	}
	return server
}

func newTester(t *testing.T) *tester {
	workspace, err := ioutil.TempDir("", "console-tester-")
	if err != nil {
		t.Fatalf("failed to create temporary keystore: %v", err)
	}

	rpcServer := newTestServer("service", new(Service))
	//rpcClient := rpc.DialInProc(rpcServer)

	prompter := &hookedPrompter{scheduler: make(chan string)}
	printer := new(bytes.Buffer)

	console, err := New(Config{
		Name:     testInstance,
		DataDir:  workspace,
		DocRoot:  "testdata",
		Client:   rpcClient,
		Prompter: prompter,
		Printer:  printer,
		Preload:  []string{"preload.js"},
	})
	if err != nil {
		t.Fatalf("failed to create JavaScript console: %v", err)
	}

	return &tester{
		workspace: workspace,
		console:   console,
		input:     prompter,
		output:    printer,
		server:    rpcServer,
		client:    rpcClient,
	}

}

func (env *tester) Close(t *testing.T) {
	if err := env.console.Stop(false); err != nil {
		t.Errorf("failed to stop embedded console: %v", err)
	}
	env.client.Close()
	env.server.Stop()

	os.RemoveAll(env.workspace)
}

func TestWelcome(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	tester.console.Welcome()

	output := string(tester.output.Bytes())
	if want := "Welcome"; !strings.Contains(output, want) {
		t.Fatalf("console output missing welcome message: have\n%s\nwant also %s", output, want)
	}
	if want := fmt.Sprintf("instance: %s", testInstance); !strings.Contains(output, want) {
		t.Fatalf("console output missing instance: have\n%s\nwant also %s", output, want)
	}
	if want := fmt.Sprintf("datadir: %s", tester.workspace); !strings.Contains(output, want) {
		t.Fatalf("console output missing coinbase: have\n%s\nwant also %s", output, want)
	}
}

func TestEvaluate(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	tester.console.Evaluate("2 + 2")
	if output := string(tester.output.Bytes()); !strings.Contains(output, "4") {
		t.Fatalf("statement evaluation failed: have %s, want %s", output, "4")
	}
}

func TestInteractive(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	go tester.console.Interactive()

	select {
	case <-tester.input.scheduler:
	case <-time.After(time.Second):
		t.Fatalf("initial prompt timeout")
	}
	select {
	case tester.input.scheduler <- "2+2":
	case <-time.After(time.Second):
		t.Fatalf("input feedback timeout")
	}

	select {
	case <-tester.input.scheduler:
	case <-time.After(time.Second):
		t.Fatalf("secondary prompt timeout")
	}
	if output := string(tester.output.Bytes()); !strings.Contains(output, "4") {
		t.Fatalf("statement evaluation failed: have %s, want %s", output, "4")
	}
}

func TestPreload(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	tester.console.Evaluate("preloaded")
	if output := string(tester.output.Bytes()); !strings.Contains(output, "some-preloaded-string") {
		t.Fatalf("preloaded variable missing: have %s, want %s", output, "some-preloaded-string")
	}
}

func TestExecute(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	tester.console.Execute("exec.js")

	tester.console.Evaluate("execed")
	if output := string(tester.output.Bytes()); !strings.Contains(output, "some-executed-string") {
		t.Fatalf("execed variable missing: have %s, want %s", output, "some-executed-string")
	}
}

func TestPrettyPrint(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	tester.console.Evaluate("obj = {int: 1, string: 'two', list: [3, 3, 3], obj: {null: null, func: function(){}}}")

	var (
		one   = jsre.NumberColor("1")
		two   = jsre.StringColor("\"two\"")
		three = jsre.NumberColor("3")
		null  = jsre.SpecialColor("null")
		fun   = jsre.FunctionColor("function()")
	)

	want := `{
  int: ` + one + `,
  list: [` + three + `, ` + three + `, ` + three + `],
  obj: {
    null: ` + null + `,
    func: ` + fun + `
  },
  string: ` + two + `
}
`
	if output := string(tester.output.Bytes()); output != want {
		t.Fatalf("pretty print mismatch: have %s, want %s", output, want)
	}
}

func TestPrettyError(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)
	tester.console.Evaluate("throw 'hello'")

	want := jsre.ErrorColor("hello") + "\n"
	if output := string(tester.output.Bytes()); output != want {
		t.Fatalf("pretty error mismatch: have %s, want %s", output, want)
	}
}

func TestIndenting(t *testing.T) {
	testCases := []struct {
		input               string
		expectedIndentCount int
	}{
		{`var a = 1;`, 0},
		{`"some string"`, 0},
		{`"some string with (parentesis`, 0},
		{`"some string with newline
		("`, 0},
		{`function v(a,b) {}`, 0},
		{`function f(a,b) { var str = "asd("; };`, 0},
		{`function f(a) {`, 1},
		{`function f(a, function(b) {`, 2},
		{`function f(a, function(b) {
		     var str = "a)}";
		  });`, 0},
		{`function f(a,b) {
		   var str = "a{b(" + a, ", " + b;
		   }`, 0},
		{`var str = "\"{"`, 0},
		{`var str = "'("`, 0},
		{`var str = "\\{"`, 0},
		{`var str = "\\\\{"`, 0},
		{`var str = 'a"{`, 0},
		{`var obj = {`, 1},
		{`var obj = { {a:1`, 2},
		{`var obj = { {a:1}`, 1},
		{`var obj = { {a:1}, b:2}`, 0},
		{`var obj = {}`, 0},
		{`var obj = {
			a: 1, b: 2
		}`, 0},
		{`var test = }`, -1},
		{`var str = "a\""; var obj = {`, 1},
	}

	for i, tt := range testCases {
		counted := countIndents(tt.input)
		if counted != tt.expectedIndentCount {
			t.Errorf("test %d: invalid indenting: have %d, want %d", i, counted, tt.expectedIndentCount)
		}
	}
}

func TestRPC(t *testing.T) {
	tester := newTester(t)
	defer tester.Close(t)

	tester.console.Evaluate("juc.send({method: 'service_name'}).result")
	if output := string(tester.output.Bytes()); !strings.Contains(output, testInstance) {
		t.Fatalf("RPC return variable missing: have %s, want %s", output, testInstance)
	}
}

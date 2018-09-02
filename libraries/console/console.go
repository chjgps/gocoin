// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

package console

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"

	"gocoin/libraries/chain/account"
	"gocoin/libraries/chain/account/keystore"
	"gocoin/libraries/chain/blsgroup"
	"gocoin/libraries/chain/space/protocol"
	"gocoin/libraries/chain/transaction/types"
	"gocoin/libraries/cmd/util"
	"gocoin/libraries/common"
	"gocoin/libraries/ucoin"
	"math/big"
	"math/rand"
	"time"

	"github.com/mattn/go-colorable"
	"github.com/peterh/liner"
	"strconv"
)

var (
	passwordRegexp = regexp.MustCompile(`personal.[nus]`)
	onlyWhitespace = regexp.MustCompile(`^\s*$`)
	exit           = regexp.MustCompile(`^\s*exit\s*;*\s*$`)

	cmdnames = []string{"account", "createspace", "recharge", "setattr", "ttx", "test", "setvrf"}
)

const HistoryFile = "history"

const DefaultPrompt = "> "

type Config struct {
	Name     string
	DataDir  string
	DocRoot  string
	Prompt   string
	Prompter UserPrompter
	Printer  io.Writer
	Preload  []string
	Accman   *accounts.Manager
	Ucoin    *ucoin.Ucoin
}

type Console struct {
	name     string
	prompt   string
	prompter UserPrompter
	histPath string
	history  []string
	printer  io.Writer
	dataDir  string
	accman   *accounts.Manager
	ucoin    *ucoin.Ucoin
}

func New(config Config) (*Console, error) {
	if config.Prompter == nil {
		config.Prompter = Stdin
	}
	if config.Prompt == "" {
		config.Prompt = DefaultPrompt
	}
	if config.Printer == nil {
		config.Printer = colorable.NewColorableStdout()
	}
	console := &Console{
		name:     config.Name,
		prompt:   config.Prompt,
		prompter: config.Prompter,
		printer:  config.Printer,
		histPath: filepath.Join(config.DataDir, HistoryFile),
		dataDir:  config.DataDir,
		accman:   config.Accman,
		ucoin:    config.Ucoin,
	}

	console.prompter.SetWordCompleter(console.AutoCompleteInput)

	return console, nil
}

func (c *Console) AutoCompleteInput(line string) (cmd []string) {

	if len(line) == 0 {
		return nil
	}

	for _, n := range cmdnames {
		if strings.HasPrefix(n, strings.ToLower(line)) {
			cmd = append(cmd, n)
		}
	}

	return
}

func (c *Console) Welcome() {
	fmt.Fprintf(c.printer, "Welcome to the gocoin console!\n\n")
}

func getPassPhrase(prompt string, confirmation bool, i int, passwords []string) string {
	// If a list of passwords was supplied, retrieve from them
	if len(passwords) > 0 {
		if i < len(passwords) {
			return passwords[i]
		}
		return passwords[len(passwords)-1]
	}
	// Otherwise prompt the user for the password
	if prompt != "" {
		fmt.Println(prompt)
	}
	password, err := Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		util.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			util.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			util.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func account(c *Console, args []string) {

	if len(args) == 0 {
		fmt.Println("usage:account [new [passwd]|unlock address [passwd]|list]")
		return
	}
	switch args[0] {
	case "new":
		passwd := ""
		if len(args[1:]) == 0 {
			passwd = getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, []string{})
		} else if len(args[1:]) == 1 {
			passwd = args[1]
		} else {
			fmt.Println("usage:account new [password]")
			break
		}

		ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		ac, err := ks.NewAccount(passwd)
		if err != nil {
			fmt.Println("Failed to create account: %v", err)
		}
		fmt.Println("Address:", ac.Address.Hex())

	case "list":
		if len(args[1:]) == 0 {
			ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
			acc := ks.Accounts()
			if len(acc) > 0 {
				for _, addr := range acc {
					w, _ := c.ucoin.GetGw().Get(addr.Address)
					if w == nil {
						fmt.Println("Address:", addr.Address.Hex(), "W : nil")
					} else {
						fmt.Println("Address:", addr.Address.Hex(), "ID:", w.Id)
					}

				}
			} else {
				fmt.Println("ks.Accounts  size  0 ")
			}

		} else {
			fmt.Println("usage:account list")
		}

	case "unlock":
		passwd := ""
		address := ""
		cmdLen := len(args[1:])
		if cmdLen == 1 {
			address = args[1]
			passwd = getPassPhrase("", false, 0, []string{})
		} else if cmdLen == 2 {
			address = args[1]
			passwd = args[2]
		} else {
			fmt.Println("usage:account unlock address [password]")
			break
		}

		ac := accounts.Account{Address: common.HexToAddress(address)}
		ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		err := ks.Unlock(ac, passwd)
		if err != nil {
			fmt.Println("false")
		} else {
			fmt.Println("true")
		}

	case "delete":
	case "del":
		passwd := ""
		address := ""
		cmdLen := len(args[1:])
		if cmdLen == 1 {
			address = args[1]
			passwd = getPassPhrase("", false, 0, []string{})
		} else if cmdLen == 2 {
			address = args[1]
			passwd = args[2]
		} else {
			fmt.Println("usage:account delete address [password]")
			break
		}
		ac := accounts.Account{Address: common.HexToAddress(address)}
		ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		err := ks.Delete(ac, passwd)
		if err != nil {
			fmt.Println("delete fail")
		}
	/*
		case "find":
			address := args[1]
			ac := accounts.Account{Address: common.HexToAddress(address),}
			ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
			fac, err := ks.Find(ac)

			if err != nil {
				fmt.Println("not found")
			}else{
				fmt.Printf("Address:%x URL:%v\n", fac.Address, fac.URL)
			}

			key, erre := ks.DescryptedKey(fac, args[2])
			if erre != nil {
				fmt.Println("des faile")
			}else{
				fmt.Println("des true", key.PrivateKey)
			}
	*/
	default:
		fmt.Println("usage:account [new [passwd]|unlock address [passwd]|list]")
	}
}

//SendCreatSpace(cost *big.Int, bindAddr common.Address, privateKey *ecdsa.PrivateKey)
func CreatSpace(c *Console, args []string) {
	/*
		if len(args) != 2 {
			fmt.Println("usage:createspace address cost")
		}else{
			address := args[0]
			ac := accounts.Account{Address: common.HexToAddress(address),}
			ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
			fac, err := ks.Find(ac)

			if err != nil {
				fmt.Printf("Not found %x\n", address)
				return
			}

			passwd := getPassPhrase("", false, 0, []string{})
			key, erre := ks.DescryptedKey(fac, passwd)
			if erre != nil {
				fmt.Println("des faile")
				return
			}

			cost := args[1]
			i := new(big.Int)
			if _, ok := i.SetString(cost, 10); ok {
				c.ucoin.SendCreatSpace(i, fac.Address, key.PrivateKey)
			}
		}
	*/
}

//func (s *Ucoin) SendRecharge(to common.Address, amount, cost *big.Int, privateKey *ecdsa.PrivateKey)
func Recharge(c *Console, args []string) {
	/*
		if len(args) != 3 {
			fmt.Println("usage:recharge address amount cost")
			return
		}

		address := args[0]
		ac := accounts.Account{Address: common.HexToAddress(address),}
		ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		fac, err := ks.Find(ac)
		if err != nil {
			fmt.Printf("Not found %x\n", address)
			return
		}

		passwd := getPassPhrase("", false, 0, []string{})
		key, erre := ks.DescryptedKey(fac, passwd)
		if erre != nil {
			fmt.Println("des faile")
			return
		}

		amount := args[1]
		cost := args[2]
		am := new(big.Int)
		co := new(big.Int)
		_, aok := am.SetString(amount, 10)
		_, cok := co.SetString(cost, 10)
		if aok && cok {
			c.ucoin.SendRecharge(fac.Address, am, co, key.PrivateKey)
		}
	*/
}

//SendAttrSet(to common.Address, cost *big.Int, attr protocol.Attr, privateKey *ecdsa.PrivateKey)
func Setattr(c *Console, args []string) {
	/*
		if len(args) != 4 {
			fmt.Println("usage:setattr address cost key value")
			return
		}

		address := args[0]
		ac := accounts.Account{Address: common.HexToAddress(address),}
		ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
		fac, err := ks.Find(ac)
		if err != nil {
			fmt.Printf("Not found %x\n", address)
			return
		}

		passwd := getPassPhrase("", false, 0, []string{})
		key, erre := ks.DescryptedKey(fac, passwd)
		if erre != nil {
			fmt.Println("des faile")
			return
		}

		k, v := args[2], args[3]
		attr := protocol.Attr{
			K: []byte(k),
			V: []byte(v),
		}

		cost := args[1]
		co := new(big.Int)
		if _, ok := co.SetString(cost, 10); ok {
			c.ucoin.SendAttrSet(fac.Address, co, attr, key.PrivateKey)
		}
	*/
}

func TestTx(c *Console, args []string) {
	if len(args) != 3 {
		fmt.Println("usage:testTransactionTx from to amount")
		return
	}
	if args[0] == args[1] {
		fmt.Println("Can not TransactionTx to self")
		return
	}
	address := args[0]
	ac := accounts.Account{Address: common.HexToAddress(address)}
	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	fac, err := ks.Find(ac)
	if err != nil {
		fmt.Printf("Not found %x\n", address)
		return
	}

	//passwd := getPassPhrase("", false, 0, []string{})
	passwd := "1"
	key, erre := ks.DescryptedKey(fac, passwd)
	if erre != nil {
		fmt.Println("des faile")
		return
	}

	amount, _ := new(big.Int).SetString(args[2], 10)
	tx := types.NewTransaction(common.HexToAddress(args[1]), amount)
	signer := types.MakeSigner()
	tx, _ = types.SignTransactionTx(tx, signer, key.PrivateKey)
	c.ucoin.AddTransaction_tx(tx)
}

func testdb(c *Console, args []string) {
	gw := c.ucoin.GetGw()
	gw.TestFunc(20, 10000, c.accman)
}

func setvrf(c *Console, args []string) {
	gw := c.ucoin.GetGw()
	gw.AddVRF(args)
}

func cttx(c *Console, args []string) {
	if len(args) != 1 {
		fmt.Println("usage:cttx num")
		return
	}

	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acc := ks.Accounts()
	count := uint32(len(acc))
	b, _ := new(big.Int).SetString(args[0], 10)
	cn := b.Uint64()

	go func() {
		for {
			src := rand.NewSource(time.Now().Unix())
			rd := rand.New(src)
			from := rd.Uint32() % count
			to := rd.Uint32() % count
			for from == to {
				to = rd.Uint32() % count
			}

			key, erre := ks.DescryptedKey(acc[from], "1")
			if erre != nil {
				fmt.Println("des faile")
				return
			}

			amount := new(big.Int).SetUint64(rd.Uint64()%5 + 1)
			tx := types.NewTransaction(acc[to].Address, amount)
			signer := types.MakeSigner()
			tx, _ = types.SignTransactionTx(tx, signer, key.PrivateKey)
			fmt.Println("CTTX:form:", acc[from].Address.Hex(), "TO:", acc[to].Address.Hex(), "TxHash:", tx.Hash().Hex(), "cons:", amount.Uint64())
			c.ucoin.AddTransaction_tx(tx)
			cn--
			if cn == 0 {
				cn = b.Uint64()
				time.Sleep(10 * time.Second)
			}
		}
	}()
}

func sptx(c *Console, args []string) {
	if len(args) != 1 {
		fmt.Println("usage:cttx num")
		return
	}

	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acc := ks.Accounts()
	count := uint32(len(acc))
	b, _ := new(big.Int).SetString(args[0], 10)
	cn := b.Uint64()
	counts := 1

	go func() {
		for counts > 0 {
			src := rand.NewSource(time.Now().Unix())
			rd := rand.New(src)
			from := uint32(0)
			to := rd.Uint32() % count
			for from == to {
				to = rd.Uint32() % count
			}

			key, erre := ks.DescryptedKey(acc[from], "1")
			if erre != nil {
				fmt.Println("des faile")
				return
			}

			amount := new(big.Int).SetUint64(rd.Uint64()%5 + 1)
			tx := protocol.NewRecharge(acc[to].Address, amount, amount)

			signer := protocol.MakeSigner()
			tx, _ = protocol.SignTx(tx, signer, key.PrivateKey)
			fmt.Println("SPTX:form:", acc[from].Address.Hex(), "TO:", acc[to].Address.Hex(), "TxHash:", tx.Hash().Hex(), "cons:", amount.Uint64())
			c.ucoin.AddTx(tx)
			cn--
			if cn == 0 {
				cn = b.Uint64()
				time.Sleep(10 * time.Second)
			}
		}
	}()
}

func setgroup(c *Console, args []string) {
	ids := []int{}
	if len(args) == 0 {
		ids = append(ids, protocol.IdApplyGroup)
		ids = append(ids, protocol.IdTransactionGroup)
		ids = append(ids, protocol.IdCoinGroup)
	} else {
		for _, v := range args {
			id, err := strconv.Atoi(v)
			if err != nil {
				fmt.Println("usage:setg gid1 gid2 gid3")
				return
			}
			ids = append(ids, id)
		}
	}

	for _, id := range ids {

		switch id {
		case protocol.IdApplyGroup:
			fmt.Println("Austin create group: protocol.IdApplyGroup")
			gw := c.ucoin.GetGw()
			b := new(big.Int).SetUint64(0)
			tx := protocol.ApplyValidateGroup(gw.Gaddr, b, b)
			signer := protocol.MakeSigner()
			tx, _ = protocol.SignTx(tx, signer, gw.Gprv)
			c.ucoin.AddTx(tx)
		case protocol.IdTransactionGroup:
			fmt.Println("Austin create group: protocol.IdTransactionGroup")
			gw := c.ucoin.GetGw()
			b := new(big.Int).SetUint64(0)
			tx := protocol.ApplyTransactionGroup(gw.Gaddr, b, b)
			signer := protocol.MakeSigner()
			tx, _ = protocol.SignTx(tx, signer, gw.Gprv)
			c.ucoin.AddTx(tx)
		case protocol.IdCoinGroup:
			fmt.Println("Austin create group: protocol.IdCoinGroup")
			gw := c.ucoin.GetGw()
			b := new(big.Int).SetUint64(0)
			tx := protocol.ApplyCoinGroup(gw.Gaddr, b, b)
			signer := protocol.MakeSigner()
			tx, _ = protocol.SignTx(tx, signer, gw.Gprv)
			c.ucoin.AddTx(tx)
		}
	}
	/*
		if len(args) != 0 {
			fmt.Println("usage:setgroup groupaddr groupsize")
			return
		}

		gw := c.ucoin.GetGw()
		blsgroup.GetGroup(0).Set(gw.Gaddr, []common.Address{}, 0)
		b := new(big.Int).SetUint64(0)
		tx := protocol.ApplyValidateGroup(gw.Gaddr, b, b)
		signer := protocol.MakeSigner()
		tx, _ = protocol.SignTx(tx, signer, gw.Gprv)
		c.ucoin.AddTx(tx)


			h := common.StringToHash("Hello")
			sig, _ := gw.SignMsg(h)
			addr, _ := gw.SignSender(h, sig)
			fmt.Println(addr.Hex())
	*/
}

func self(c *Console, args []string) {
	if len(args) != 1 {
		fmt.Println("usage:setgroup groupaddr groupsize")
		return
	}

	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	acc := ks.Accounts()

	gidx, _ := new(big.Int).SetString(args[0], 10)
	gw := c.ucoin.GetGw()
	gw.Gaddr = acc[gidx.Uint64()].Address
	fmt.Println("Austin:gw.ROOT:", gw.ROOT.Hex())

	key, erre := ks.DescryptedKey(acc[gidx.Uint64()], "1")
	if erre != nil {
		fmt.Println("des faile")
		return
	}
	gw.Gprv = key.PrivateKey
	blsgroup.GetGroup(0).Set(gw.Gaddr, []common.Address{}, 0)
	if err := blsgroup.GetGroup(0).LoadBlsInfo(); err != nil {
		fmt.Println(err)
	}
	c.ucoin.ResetValidateGroup(gw.Gaddr, 0)
}

func coinpub(c *Console, args []string) {
	if len(args) != 2 {
		fmt.Println("usage:coinpub addr coinnum")
		return
	}

	address := args[0]
	ac := accounts.Account{Address: common.HexToAddress(address)}
	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	fac, err := ks.Find(ac)
	if err != nil {
		fmt.Printf("Not found %x\n", address)
		return
	}

	passwd := getPassPhrase("", false, 0, []string{})
	key, erre := ks.DescryptedKey(fac, passwd)
	if erre != nil {
		fmt.Println("des faile")
		return
	}
	/*
			notary := publish.GetNotary()
			req := notary.ReqNotary(ac.Address, []byte("Hello World"))
			res := notary.WaitNotary(req.CttHash)
			_, _, _ = req, res, key
			for _, v := range res.Notary {
				fmt.Println("Notary:", v.Addr.Hex())
			}

		_ = key
		publish.GetPublish().PublishSession([]byte("Hello World"), ac.Address)
	*/

	amount, _ := new(big.Int).SetString(args[1], 10)
	tx := protocol.NewPublishCon(amount, new(big.Int).SetUint64(0), []byte{})
	signer := protocol.MakeSigner()
	tx, _ = protocol.SignTx(tx, signer, key.PrivateKey)
	c.ucoin.AddTx(tx)
}

func recharge(c *Console, args []string) {
	if len(args) != 3 {
		fmt.Println("usage:rechargs from to amount")
		return
	}
	from := args[0]
	to := args[1]
	amount, _ := new(big.Int).SetString(args[2], 10)

	ac := accounts.Account{Address: common.HexToAddress(from)}
	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	fac, err := ks.Find(ac)
	if err != nil {
		fmt.Printf("Not found %x\n", from)
		return
	}
	passwd := getPassPhrase("", false, 0, []string{})
	key, erre := ks.DescryptedKey(fac, passwd)
	if erre != nil {
		fmt.Println("des faile")
		return
	}

	tx := protocol.NewRecharge(common.HexToAddress(to), amount, amount)

	signer := protocol.MakeSigner()
	tx, _ = protocol.SignTx(tx, signer, key.PrivateKey)
	fmt.Println("SPTX:form:", from, "TO:", to, "TxHash:", tx.Hash().Hex(), "cons:", amount.Uint64())
	c.ucoin.AddTx(tx)
}

func creatSpace(c *Console, args []string) {
	if len(args) != 3 {
		fmt.Println("usage:csp from to cost")
		return
	}
	from := args[0]
	to := args[1]
	cost, _ := new(big.Int).SetString(args[2], 10)

	ac := accounts.Account{Address: common.HexToAddress(from)}
	ks := c.accman.Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)
	fac, err := ks.Find(ac)
	if err != nil {
		fmt.Printf("Not found %x\n", from)
		return
	}
	passwd := getPassPhrase("", false, 0, []string{})
	key, erre := ks.DescryptedKey(fac, passwd)
	if erre != nil {
		fmt.Println("des faile")
		return
	}

	tx := protocol.NewSpaceCreation(cost, common.HexToAddress(to))
	signer := protocol.MakeSigner()
	tx, _ = protocol.SignTx(tx, signer, key.PrivateKey)
	fmt.Println("SPTX:form:", from, "TO:", to, "TxHash:", tx.Hash().Hex(), "cost:", cost.Uint64())
	c.ucoin.AddTx(tx)
}

func settle(c *Console, args []string) {
	if len(args) == 0 {
		fmt.Println("usage:settle [create type network address|start|stop]")
		return
	}

	switch args[0] {
	case "create":
		if len(args[1:]) != 3 {
			fmt.Println("usage:settle [create type network address")
		}
		msgtype, err := strconv.ParseUint(args[1], 10, 32)
		if err != nil {
			fmt.Println("Type shoude be uint32")
			return
		}
		c.ucoin.CreateTcpServer(uint32(msgtype), args[2], args[3])
	case "start":
		c.ucoin.StartTcpServer()
	case "stop":
		c.ucoin.StopTcpServer()
	case "rpc":
		c.ucoin.StartRpcHTTP(args[1])
	}
	return
}

type cmdfunc func(c *Console, args []string)

var exec map[string]cmdfunc = map[string]cmdfunc{
	"account":  account,
	"csp":      creatSpace,
	"recharge": recharge,
	"setattr":  Setattr,
	"ttx":      TestTx,
	"test":     testdb,
	"setvrf":   setvrf,
	"cttx":     cttx,
	"setg":     setgroup,
	"coinpub":  coinpub,
	"sptx":     sptx,
	"self":     self,
	"settle":   settle,
}

func (c *Console) execCmd(input string) {
	list := strings.Split(input, " ")
	args := []string{}
	for _, s := range list {
		if s != "" {
			args = append(args, s)
		}
	}

	cmd := args[0]
	args = args[1:]

	if v, ok := exec[cmd]; ok {
		v(c, args)
	} else {
		fmt.Println("command not found")
	}
}

func (c *Console) Interactive() {
	var (
		prompt    = c.prompt
		indents   = 0
		input     = ""
		scheduler = make(chan string)
	)

	go func() {
		for {
			line, err := c.prompter.PromptInput(<-scheduler)
			if err != nil {
				if err == liner.ErrPromptAborted { // ctrl-C
					prompt, indents, input = c.prompt, 0, ""
					scheduler <- ""
					continue
				}
				close(scheduler)
				return
			}
			scheduler <- line
		}
	}()
	abort := make(chan os.Signal, 1)
	signal.Notify(abort, os.Interrupt)

	for {
		scheduler <- prompt
		select {
		case <-abort:
			fmt.Fprintln(c.printer, "caught interrupt, exiting")
			return

		case line, ok := <-scheduler:
			if !ok || (indents <= 0 && exit.MatchString(line)) {
				return
			}
			if onlyWhitespace.MatchString(line) {
				continue
			}
			input += line
			c.execCmd(input)

			indents = countIndents(input)
			if indents <= 0 {
				prompt = c.prompt
			} else {
				prompt = strings.Repeat(".", indents*3) + " "
			}
			input = ""
		}
	}
}

func countIndents(input string) int {
	var (
		indents     = 0
		inString    = false
		strOpenChar = ' '
		charEscaped = false
	)

	for _, c := range input {
		switch c {
		case '\\':
			if !charEscaped && inString {
				charEscaped = true
			}
		case '\'', '"':
			if inString && !charEscaped && strOpenChar == c {
				inString = false
			} else if !inString && !charEscaped {
				inString = true
				strOpenChar = c
			}
			charEscaped = false
		case '{', '(':
			if !inString {
				indents++
			}
			charEscaped = false
		case '}', ')':
			if !inString {
				indents--
			}
			charEscaped = false
		default:
			charEscaped = false
		}
	}

	return indents
}

func (c *Console) Stop(graceful bool) error {
	if err := ioutil.WriteFile(c.histPath, []byte(strings.Join(c.history, "\n")), 0600); err != nil {
		return err
	}
	if err := os.Chmod(c.histPath, 0600); err != nil {
		return err
	}
	return nil
}

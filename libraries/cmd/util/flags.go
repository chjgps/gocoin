// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

package util

import (
	"strings"

	"github.com/urfave/cli"
	"gocoin/libraries/common"
	"gocoin/libraries/node"
	"io/ioutil"
)

func NewApp() *cli.App {
	app := cli.NewApp()
	app.Name = "ucoin"
	app.HelpName = "ucoin"
	app.Usage = "say the ucoin"

	return app
}

var (
	DataDirFlag = cli.StringFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: node.DefaultDataDir(),
	}
	KeyStoreDirFlag = cli.StringFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	LogLevelFlag = cli.IntFlag{
		Name:  "loglevel",
		Usage: "set log level",
		Value: 3,
	}
	ListenIpFlag = cli.StringFlag{
		Name:  "listen-ip",
		Usage: "listen ip address.",
		Value: "127.0.0.1",
	}
	ListenPortFlag = cli.IntFlag{
		Name:  "listen-port",
		Usage: "listen port",
		Value: 20050,
	}
	BootnodesFlag = cli.StringFlag{
		Name:  "bootnodes",
		Usage: "Comma separated enode URLs for P2P discovery bootstrap (set v4+v5 instead for light servers)",
		Value: "",
	}
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "Cache directory path ",
		Value: "/etc/ucoin/conf.ini",
	}
	CachePathFlag = cli.StringFlag{
		Name:  "cachepath",
		Usage: "Cache directory path ",
		Value: "/var/ucoin/cache",
	}
	ChainPathFlag = cli.StringFlag{
		Name:  "chainpath",
		Usage: "Chain directory path ",
		Value: "/var/ucoin/chain",
	}
	RPCAddrsFlag = cli.StringFlag{
		Name:  "rpcaddrs",
		Usage: "rpc address list",
		Value: "",
	}
	RPCCommandFlag = cli.StringFlag{
		Name:  "rpccmds",
		Usage: "rpc command list",
		Value: "",
	}
	UsbKeySupportFlag = cli.BoolFlag{
		Name:  "usbkey",
		Usage: "usb key support",
	}
	LogPathFlag = cli.StringFlag{
		Name:  "logpath",
		Usage: "log directory path",
		Value: "/var/ucoin/log",
	}
	LogFileSizeFlag = cli.IntFlag{
		Name:  "logfilesize",
		Usage: "Log file size(MB)",
		Value: 5,
	}
	PluginOptionsFlag = cli.StringFlag{
		Name:  "plugins",
		Usage: "plugin option list",
		Value: "",
	}
	StorePathFlag = cli.StringFlag{
		Name:  "storepath",
		Usage: "store directory path",
		Value: "/var/ucoin/store",
	}
	JrePathFlag = cli.StringFlag{
		Name:  "jrepath",
		Usage: "jre directory path",
		Value: "/var/ucoin/jre",
	}
	ExecFlag = cli.StringFlag{
		Name:  "exec",
		Usage: "Execute JavaScript statement",
	}
	PreloadJSFlag = cli.StringFlag{
		Name:  "preload",
		Usage: "Comma separated list of JavaScript files to preload into the console",
	}
	JSpathFlag = cli.StringFlag{
		Name:  "jspath",
		Usage: "JavaScript root path for `loadScript`",
		Value: ".",
	}

	// Account settings
	PasswordFileFlag = cli.StringFlag{
		Name:  "password",
		Usage: "Password file to use for non-interactive password input",
		Value: "",
	}
	DefaultAccountFlag = cli.StringFlag{
		Name:  "account",
		Usage: "Set the default account address",
		Value: "",
	}

	SeedServer = cli.StringFlag{
		Name:  "seedserver",
		Usage: "server or not",
		Value: "",
	}

	SeedNodeFlag = cli.StringFlag{
		Name:  "seednode",
		Usage: "set seed node",
		Value: "",
	}
)

func MakeDataDir(ctx *cli.Context) string {

	if path := ctx.GlobalString(DataDirFlag.Name); path != "" {
		return path
	}
	Fatalf("Cannot determine default data directory, please set manually (--datadir)")
	return ""
}

// MakePasswordList reads password lines from the file specified by the global --password flag.
func MakePasswordList(ctx *cli.Context) []string {
	path := ctx.GlobalString(PasswordFileFlag.Name)
	if path == "" {
		return nil
	}
	text, err := ioutil.ReadFile(path)
	if err != nil {
		Fatalf("Failed to read password file: %v", err)
	}
	lines := strings.Split(string(text), "\n")
	// Sanitise DOS line endings.
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], "\r")
	}
	return lines
}
func MakeConsolePreloads(ctx *cli.Context) []string {
	// Skip preloading if there's nothing to preload
	if ctx.GlobalString(PreloadJSFlag.Name) == "" {
		return nil
	}
	// Otherwise resolve absolute paths and return them
	preloads := []string{}

	assets := ctx.GlobalString(JSpathFlag.Name)
	for _, file := range strings.Split(ctx.GlobalString(PreloadJSFlag.Name), ",") {
		preloads = append(preloads, common.AbsolutePath(assets, strings.TrimSpace(file)))
	}
	return preloads
}

// MigrateFlags sets the global flag from a local flag when it's set.
// This is a temporary function used for migrating old command/flags to the
// new format.
//
// e.g. ucoin account new --keystore /tmp/mykeystore --lightkdf
//
// is equivalent after calling this method with:
//
// ucoin --keystore /tmp/mykeystore --lightkdf account new
//
// This allows the use of the existing configuration functionality.
// When all flags are migrated this function can be removed and the existing
// configuration functionality must be changed that is uses local flags
func MigrateFlags(action func(ctx *cli.Context) error) func(*cli.Context) error {
	return func(ctx *cli.Context) error {
		for _, name := range ctx.FlagNames() {
			if ctx.IsSet(name) {
				ctx.GlobalSet(name, ctx.String(name))
			}
		}
		return action(ctx)
	}
}

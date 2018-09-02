// Offical comman-line client
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/urfave/cli"
	"gocoin/libraries/cmd/util"
	"gocoin/libraries/log"
	"gocoin/libraries/net/p2p"
	"gocoin/libraries/net/p2p/netdiscovery"
	"gocoin/libraries/node"
	"gocoin/libraries/ucoin"
	"path/filepath"
)

const (
	appName          = "utc"
	clientIdentifier = "ucoin"
)

var (
	app       = util.NewApp()
	baseFlags = []cli.Flag{
		util.ConfigFileFlag,
		util.CachePathFlag,
		util.ChainPathFlag,
		util.RPCAddrsFlag,
		util.RPCCommandFlag,
		util.UsbKeySupportFlag,
		util.LogPathFlag,
		util.LogFileSizeFlag,
		util.PluginOptionsFlag,
		util.StorePathFlag,
		util.JrePathFlag,
		util.DataDirFlag,
		util.LogLevelFlag,
		util.ListenIpFlag,
		util.ListenPortFlag,
		util.BootnodesFlag,
		util.SeedNodeFlag,
	}
)

/* Initialize the CLI API and start APP */
func init() {
	app.Action = gocoin
	app.HideVersion = true // we have a command to print the version
	app.Copyright = " Copyright(c)2017-2020 gocoin,Co,.Ltd."

	app.Commands = []cli.Command{
		accountCommand,
		consoleCommand,
		attachCommand,
	}

	app.Flags = append(app.Flags, baseFlags...)
	app.Flags = append(app.Flags, consoleFlags...)

	sort.Sort(cli.CommandsByName(app.Commands))

	app.Before = func(ctx *cli.Context) error {
		util.Setup(ctx)
		return nil
	}
}

func makeNodeConfig(ctx *cli.Context) node.Config {
	conf := defaultNodeConfig(ctx)
	SetP2PConfig(ctx, &conf.P2P)
	setBootstrapNodes(ctx, &conf.P2P)
	return conf
}

// setBootstrapNodes creates a list of bootstrap nodes from the command line
// flags, reverting to pre-configured ones if none have been specified.
func setBootstrapNodes(ctx *cli.Context, cfg *p2p.Config) {
	urls := []string{}

	if ctx.GlobalIsSet(util.BootnodesFlag.Name) {
		urls = strings.Split(ctx.GlobalString(util.BootnodesFlag.Name), ",")
	}

	cfg.BootstrapNodes = make([]*netdiscovery.Node, 0, len(urls))
	for _, url := range urls {
		node, err := netdiscovery.ParseNode(url)
		if err != nil {
			log.Error("Bootstrap URL invalid", "enode", url, "err", err)
			continue
		}
		cfg.BootstrapNodes = append(cfg.BootstrapNodes, node)
	}
}

func SetP2PConfig(ctx *cli.Context, cfg *p2p.Config) {
	setListenAddress(ctx, cfg)
	setSeedNode(ctx, cfg)
}

// setListenAddress creates a TCP listening address string from set command
// line flags.
func setListenAddress(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.GlobalIsSet(util.ListenPortFlag.Name) {
		cfg.ListenAddr = fmt.Sprintf(":%d", ctx.GlobalInt(util.ListenPortFlag.Name))
	}
}

func setSeedNode(ctx *cli.Context, cfg *p2p.Config) {
	if ctx.GlobalIsSet(util.SeedNodeFlag.Name) {
		cfg.SeedNode = fmt.Sprintf("%s", ctx.GlobalString(util.SeedNodeFlag.Name))
		fmt.Println("Austin debuf info seednode:", cfg.SeedNode)
	}
}

func defaultNodeConfig(ctx *cli.Context) node.Config {
	cfg := node.DefaultConfig
	cfg.Name = "gocoin"
	//cfg.DataDir = node.DefaultDataDir(ctx.GlobalString(util.DataDirFlag.Name))
	cfg.DataDir = ctx.GlobalString(util.DataDirFlag.Name)
	if !filepath.IsAbs(cfg.DataDir) {
		cfg.DataDir, _ = filepath.Abs(cfg.DataDir)
	}
	return cfg
}
func gocoin(ctx *cli.Context) {
	log.Info(fmt.Sprintf("gocoin entry ... "))
	config, err := util.LoadConfig(ctx.GlobalString(util.ConfigFileFlag.Name))
	if err == nil {
		util.UpdateConfig(ctx, config)
	}
	conf := makeNodeConfig(ctx)
	n, _ := node.New(&conf)

	n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		fullNode, _ := ucoin.New(ctx, &conf, uint64(5))
		return fullNode, nil
	})

	n.Start()
	n.Wait()
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

package main

import (
	"fmt"
	"github.com/urfave/cli"
	"gocoin/libraries/cmd/util"
	"gocoin/libraries/console"
	"gocoin/libraries/node"
	"gocoin/libraries/ucoin"
)

var (
	consoleFlags = []cli.Flag{util.JSpathFlag, util.ExecFlag, util.PreloadJSFlag}

	consoleCommand = cli.Command{
		Name:      "console",
		Usage:     "Start an interactive JavaScript environment",
		Action:    localConsole,
		Flags:     append(consoleFlags, util.DataDirFlag),
		ArgsUsage: " ",
	}

	attachCommand = cli.Command{
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		Action:    remoteConsole,
		Flags:     append(consoleFlags, util.DataDirFlag),
		ArgsUsage: "[endpoint]",
	}
)

func consoleStart(ctx *cli.Context) error {
	if ctx.NArg() != 0 {
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	fmt.Println("todo: consoleStart")
	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}

	return nil
}

func localConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	iniName := ctx.GlobalString(util.ConfigFileFlag.Name)

	config, err := util.LoadConfig(iniName)
	if err == nil {
		util.UpdateConfig(ctx, config)
	}

	conf := makeNodeConfig(ctx)
	n, _ := node.New(&conf)

	var coin *ucoin.Ucoin
	n.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		fullNode, _ := ucoin.New(ctx, &conf, uint64(5))
		coin = fullNode
		return fullNode, nil
	})

	//if coin != nil {

	n.Start()
	defer n.Stop()

	consoleConfig := console.Config{
		Name:    "ucoin console",
		DataDir: util.MakeDataDir(ctx),
		DocRoot: ctx.GlobalString(util.JSpathFlag.Name),
		Preload: util.MakeConsolePreloads(ctx),
		Accman:  n.Accman(),
		Ucoin:   coin,
	}

	console, err := console.New(consoleConfig)
	if err != nil {
		util.Fatalf("Failed to start console: %v", err)
	}
	defer console.Stop(false)

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
	//}else{
	//	return nil
	//}

}

func remoteConsole(ctx *cli.Context) error {
	// Attach to a remotely running geth instance and start the JavaScript console
	endpoint := ctx.Args().First()
	if endpoint == "" && ctx.GlobalIsSet(util.DataDirFlag.Name) {
		endpoint = fmt.Sprintf("%s/ucoin.ipc", ctx.GlobalString(util.DataDirFlag.Name))
	}

	config := console.Config{
		Name:    "ucoin attach",
		DataDir: util.MakeDataDir(ctx),
		DocRoot: ctx.GlobalString(util.JSpathFlag.Name),
		Preload: util.MakeConsolePreloads(ctx),
	}

	console, err := console.New(config)
	if err != nil {
		util.Fatalf("Failed to start the JavaScript console: %v", err)
	}
	defer console.Stop(false)

	// Otherwise print the welcome screen and enter interactive mode
	console.Welcome()
	console.Interactive()

	return nil
}

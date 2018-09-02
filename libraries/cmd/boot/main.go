// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

// bootnode runs a bootstrap node for the gocoin Discovery Protocol.
package main

import (
	"flag"
	"os"

	"gocoin/libraries/cmd/util"
	"gocoin/libraries/log"
	"gocoin/libraries/net/p2p/netdiscovery"
)

func main() {
	var (
		listenAddr = flag.String("addr", ":20000", "listen address")
		nodeIDHex  = flag.String("nodekeyhex", "86bac922a0cd22a02f5c213fc5c3b909ec0ca794aea0a085f09103d474bee7127ae19dcef39cc5559af0a39b5dead179480c19a2a42d6b42698d11cb1c98334c", "private key as hex (for testing)")
		verbosity  = flag.Int("verbosity", int(log.LvlInfo), "log verbosity (0-9)")
		vmodule    = flag.String("vmodule", "", "log verbosity pattern")
	)
	flag.Parse()

	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(false)))
	glogger.Verbosity(log.Lvl(*verbosity))
	glogger.Vmodule(*vmodule)
	glogger.BacktraceAt("")
	log.Root().SetHandler(glogger)

	nodeID, err := netdiscovery.HexID(*nodeIDHex)
	if err != nil {
		util.Fatalf("%v", err)
	}
	if _, err := netdiscovery.ListenUDP(nodeID, *listenAddr, ""); err != nil {
		util.Fatalf("%v", err)
	}

	select {}
}

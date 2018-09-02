// Copyright(C)2017-2020 gocoin,Co,.Ltd.
// All Right Reserved.
//
// This file is part of the go-storage library.

package node

import (
	//"encoding/binary"
	"os"
	"os/user"
	"path/filepath"
	"runtime"

	"gocoin/libraries/net/p2p"
)

const (
	DefaultHTTPHost = "localhost" // Default host interface for the HTTP RPC server
	DefaultHTTPPort = 8545        // Default TCP port for the HTTP RPC server
	DefaultWSHost   = "localhost" // Default host interface for the websocket RPC server
	DefaultWSPort   = 8546        // Default TCP port for the websocket RPC server
)

// DefaultConfig contains reasonable default settings.
var DefaultConfig = Config{
	DataDir:          DefaultDataDir(),
	Name:             "gocoin",
	PrivateKey:       nil,
	DatabaseCache:    128,
	DatabaseHandles:  1024,
	HTTPHost:         DefaultHTTPHost,
	HTTPPort:         DefaultHTTPPort,
	HTTPModules:      []string{"unit", "web"},
	HTTPVirtualHosts: []string{"localhost"},
	WSHost:           DefaultWSHost,
	WSPort:           DefaultWSPort,
	WSModules:        []string{"unit", "web"},
	P2P: p2p.Config{
		ListenAddr: ":20050",
		MaxPeers:   25,
	},
	IPCPath: "ucoin.ipc",
}

// DefaultDataDir is the default data directory to use for the databases and other
// persistence requirements.
func DefaultDataDir() string {
	// Try to place the data folder in the user's home dir
	home := homeDir()
	if home != "" {
		if runtime.GOOS == "darwin" {
			return filepath.Join(home, "Library", "ucoin")
		} else if runtime.GOOS == "windows" {
			return filepath.Join(home, "AppData", "Roaming", "ucoin")
		} else {
			return filepath.Join(home, ".ucoin")
		}
	}
	// As we cannot guess a stable location, return empty and handle later
	return ""
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}

// Copyright(c)2017-2020 gocoin,Co,.Ltd.
// All Copyright Reserved.
//
// This file is part of the gocoin software.

package main

import (
	"fmt"

	"github.com/urfave/cli"
	"gocoin/libraries/chain/account/keystore"
	"gocoin/libraries/cmd/util"
	"gocoin/libraries/common"
	"gocoin/libraries/common/hexutil"
	"gocoin/libraries/console"
)

var (
	accountCommand = cli.Command{
		Name:  "account",
		Usage: "Manage accounts",
		Subcommands: []cli.Command{
			{
				Name:   "stat",
				Usage:  "Display the current transaction account",
				Action: util.MigrateFlags(accountStat),
				Flags: []cli.Flag{
					util.DataDirFlag,
					util.KeyStoreDirFlag,
				},
				ArgsUsage: " ",
			},
			{
				Name:   "set",
				Usage:  "Set the current transaction account",
				Action: util.MigrateFlags(accountSet),
				Flags: []cli.Flag{
					util.DataDirFlag,
					util.KeyStoreDirFlag,
				},
				ArgsUsage: "<id>",
			},
			{
				Name:   "new",
				Usage:  "Create an account",
				Action: util.MigrateFlags(accountNew),
				Flags: []cli.Flag{
					util.DataDirFlag,
					util.KeyStoreDirFlag,
					util.PasswordFileFlag,
				},
				Subcommands: []cli.Command{
					{
						Name:      "id",
						Usage:     "Create new space id and bind to the account",
						Action:    accountNewId,
						Flags:     []cli.Flag{},
						ArgsUsage: "<account> <cost>",
					},
				},
			},
			{
				Name:      "recharge",
				Usage:     "Recharge the account",
				Action:    util.MigrateFlags(accountRecharge),
				Flags:     []cli.Flag{},
				ArgsUsage: "<account> <value> <cost>",
			},
			{
				Name:  "attr",
				Usage: "List or set attributes",
				Subcommands: []cli.Command{
					{
						Name:      "set",
						Usage:     "Set attributes",
						Action:    util.MigrateFlags(accountAttrSet),
						Flags:     []cli.Flag{},
						ArgsUsage: "<account> <key> <value> <cost>",
					},
					{
						Name:      "list",
						Usage:     "List attributes",
						Action:    util.MigrateFlags(accountAttrList),
						Flags:     []cli.Flag{},
						ArgsUsage: "[account]",
					},
				},
			},
		},
	}
)

func accountStat(ctx *cli.Context) error {
	fmt.Println("todo: accountStat")

	// constructor envirenment and get keystore
	conf := makeNodeConfig(ctx)
	scryptN, scryptP, keydir, err := conf.AccountConfig()
	if err != nil {
		util.Fatalf("Failed to read configuration: %v", err)
	}
	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)
	// todo optimize it in future
	addr := ks.DefAddr
	if addr.Bytes() == nil {
		ac := ks.Wallets()[0].Accounts()[0]
		fmt.Printf("Default account: {%x} %s\n", ac.Address, &ac.URL)

	} else {
		fmt.Println(addr.Hex())
	}

	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}

	return nil
}

func accountSet(ctx *cli.Context) error {
	fmt.Println("todo: accountSet")

	// constructor envirenment and get keystore
	conf := makeNodeConfig(ctx)
	scryptN, scryptP, keydir, err := conf.AccountConfig()
	if err != nil {
		util.Fatalf("Failed to read configuration: %v", err)
	}
	ks := keystore.NewKeyStore(keydir, scryptN, scryptP)

	// get the account address
	args := ctx.Args()
	if len(args) != 1 {
		util.Fatalf("parameter number is error")
	}
	// todo optimize it in future
	addr, err := decodeAddress(args.First())
	if err != nil {
		return fmt.Errorf("invalid address at index : %v", err)
	}

	ks.DefAddr = addr
	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}
	return nil
}

func accountNewId(ctx *cli.Context) error {
	if ctx.NArg() != 2 {
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	fmt.Println("todo: accountNewId")
	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}

	return nil
}

func accountRecharge(ctx *cli.Context) error {
	if ctx.NArg() != 3 {
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	fmt.Println("todo: accountRechange")
	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}

	return nil
}

func accountAttrSet(ctx *cli.Context) error {
	if ctx.NArg() != 4 {
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	fmt.Println("todo: accountAttrSet")
	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}

	return nil
}

func accountAttrList(ctx *cli.Context) error {
	if ctx.NArg() > 1 {
		cli.ShowSubcommandHelp(ctx)
		return nil
	}

	fmt.Println("todo: accountAttrList")
	for _, arg := range ctx.Args() {
		fmt.Println(arg)
	}

	return nil
}

func accountNew(ctx *cli.Context) error {
	fmt.Println("todo: accountNew")

	conf := makeNodeConfig(ctx)

	scryptN, scryptP, keydir, err := conf.AccountConfig()

	if err != nil {
		util.Fatalf("Failed to read configuration: %v", err)
	}

	password := getPassPhrase("Your new account is locked with a password. Please give a password. Do not forget this password.", true, 0, util.MakePasswordList(ctx))

	address, err := keystore.StoreKey(keydir, password, scryptN, scryptP)

	if err != nil {
		util.Fatalf("Failed to create account: %v", err)
	}
	fmt.Printf("Address: {%x}\n", address)
	return nil
}

// getPassPhrase retrieves the password associated with an account, either fetched
// from a list of preloaded passphrases, or requested interactively from the user.
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
	password, err := console.Stdin.PromptPassword("Passphrase: ")
	if err != nil {
		util.Fatalf("Failed to read passphrase: %v", err)
	}
	if confirmation {
		confirm, err := console.Stdin.PromptPassword("Repeat passphrase: ")
		if err != nil {
			util.Fatalf("Failed to read passphrase confirmation: %v", err)
		}
		if password != confirm {
			util.Fatalf("Passphrases do not match")
		}
	}
	return password
}

func decodeAddress(s string) (common.Address, error) {
	b, err := hexutil.Decode(s)
	if err == nil && len(b) != common.AddressLength {
		err = fmt.Errorf("hex has invalid length %d after decoding", len(b))
	}

	return common.BytesToAddress(b), err
}

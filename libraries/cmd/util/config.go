package util

import (
	"io"
	"os"

	colorable "github.com/mattn/go-colorable"
	"github.com/urfave/cli"
	"github.com/widuu/goini"
	"gocoin/libraries/log"
	"gocoin/libraries/log/term"
)

var (
	glogger *log.GlogHandler
)

var (
	paramap = map[string][2]string{
		"cachepath":   {"base", "cache_path"},
		"chainpath":   {"base", "chain_path"},
		"rpcaddrs":    {"rpc", "addr_list"},
		"rpccmds":     {"rpc", "cmd_list"},
		"usbkey":      {"usbkey", "support"},
		"logpath":     {"log", "log_path"},
		"logfilesize": {"log", "log_file_size"},
		"plugins":     {"plugin", "plugin_option_list"},
		"storepath":   {"store", "store_path"},
		"jrepath":     {"jre", "jre_path"},
	}
)

func init() {
	//logging
	usecolor := term.IsTty(os.Stderr.Fd()) && os.Getenv("TERM") != "dumb"
	output := io.Writer(os.Stderr)
	if usecolor {
		output = colorable.NewColorableStderr()
	}
	glogger = log.NewGlogHandler(log.StreamHandler(output, log.TerminalFormat(usecolor)))
}

func LoadConfig(path string) (*goini.Config, error) {
	cfg, err := goini.SetConfig("/etc/ucoin/conf.ini")
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func UpdateConfig(ctx *cli.Context, cfg *goini.Config) {
	for flag, section := range paramap {
		if !ctx.GlobalIsSet(flag) {
			val, err := cfg.GetValue(section[0], section[1])
			if err == nil {
				ctx.GlobalSet(flag, val)
			}
		}
	}
}

func Setup(ctx *cli.Context) {
	log.PrintOrigins(true)
	// 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail
	glogger.Verbosity(log.Lvl(ctx.GlobalInt(LogLevelFlag.Name)))
	glogger.Vmodule("")
	glogger.BacktraceAt("")
	log.Root().SetHandler(glogger)
}

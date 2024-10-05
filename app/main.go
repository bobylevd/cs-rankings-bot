package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"

	"github.com/bobylevd/cs-rankings-bot/app/cmd"
	_ "github.com/glebarez/go-sqlite"
	"github.com/hashicorp/logutils"
	"github.com/jessevdk/go-flags"
)

var options struct {
	Bot   cmd.Bot `command:"bot" description:"run discord bot"`
	Debug bool    `long:"debug" env:"DEBUG" description:"turn on debug mode"`
}

var version = "unknown"

func getVersion() string {
	if bi, ok := debug.ReadBuildInfo(); ok {
		return bi.Main.Version
	}
	return version
}

func main() {
	fmt.Printf("csrankbot, version: %s\n", getVersion())

	p := flags.NewParser(&options, flags.Default)
	p.CommandHandler = func(c flags.Commander, args []string) error {
		setupLog(options.Debug)

		if options.Debug {
			log.Printf("[DEBUG] debug mode on")
		}

		commonOpts := cmd.CommonOpts{Version: getVersion()}
		if cs, ok := c.(interface{ Set(cmd.CommonOpts) }); ok {
			cs.Set(commonOpts)
		}

		if err := c.Execute(args); err != nil {
			log.Printf("[ERROR] failed to execute command: %v", err)
		}

		return nil
	}

	if _, err := p.Parse(); err != nil {
		if errors.Is(err, flags.ErrHelp) {
			os.Exit(0)
		}
		os.Exit(1)
	}
}

func setupLog(dbg bool) {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: "INFO",
		Writer:   os.Stderr,
	}

	logFlags := log.Ldate | log.Ltime

	if dbg {
		logFlags = log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile
		filter.MinLevel = "DEBUG"
	}

	log.SetFlags(logFlags)
	log.SetOutput(filter)
}

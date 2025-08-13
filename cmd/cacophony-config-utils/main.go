package main

import (
	"fmt"
	"os"

	cacophonyconfig "github.com/TheCacophonyProject/go-config/internal/cacophony-config"
	cacophonyconfigsync "github.com/TheCacophonyProject/go-config/internal/cacophony-config-sync"
	"github.com/TheCacophonyProject/go-utils/logging"
)

var log = logging.NewLogger("info")
var version = "<not set>"

func main() {
	err := runMain()
	if err != nil {
		log.Fatal(err)
	}
}

func runMain() error {
	if len(os.Args) < 2 {
		log.Info("Usage: tool <subcommand> [args]")
		return fmt.Errorf("no subcommand given")
	}

	sub := os.Args[1]
	args := os.Args[2:]

	var err error
	switch sub {
	case "sync":
		err = cacophonyconfigsync.Run(args, version)
	case "config":
		err = cacophonyconfig.Run(args, version)
	default:
		err = fmt.Errorf("unknown subcommand: %s", sub)
	}

	return err
}

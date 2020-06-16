package main

import (
	"fmt"
	"os"
	"tinifier/cmd"

	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

func main() {
	var (
		root   = &cmd.Root{}
		parser = flags.NewParser(root, flags.HelpFlag|flags.PassDoubleDash)
		logger = log.New()
	)

	// set basic logger properties
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05.000",
	})

	// set custom commands handler
	parser.CommandHandler = cmd.CommandsHandler(root, logger)

	// parse the arguments
	if _, err := parser.Parse(); err != nil {
		// make error type checking
		if e, ok := err.(*flags.Error); (ok && e.Type != flags.ErrHelp) || !ok {
			// handle execution error using logger
			logger.Error(err)

			// and exit
			os.Exit(1)
		} else if _, outErr := fmt.Fprintln(os.Stdout, err); outErr != nil { // print help message into stdout
			panic(outErr)
		}
	}
}

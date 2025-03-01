package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gh.tarampamp.am/tinifier/v5/internal/cli"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error: "+err.Error())

		os.Exit(1)
	}
}

func run() error {
	// create a context that is canceled when the user interrupts the program
	var ctx, cancel = signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if len(os.Args) < 1 {
		return errors.New("missing application name")
	}

	// run the CLI application
	return cli.NewApp(filepath.Base(os.Args[0])).Run(ctx, os.Args[1:])
}

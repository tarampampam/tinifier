package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"

	"gh.tarampamp.am/tinifier/v5/internal/cli"
)

func main() {
	if err := run(); err != nil {
		if !errors.Is(err, context.Canceled) {
			_, _ = fmt.Fprintln(os.Stderr, "error: "+err.Error())
		}

		os.Exit(1)
	}
}

func run() error {
	// create a context that will be canceled on SIGINT or SIGTERM
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create a channel to receive signals and a channel to stop the signal handler
	sigChan, stop := make(chan os.Signal, 1), make(chan struct{})
	defer close(stop)

	// subscribe to SIGINT and SIGTERM
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		var (
			count uint
			once  sync.Once
		)

		for {
			select {
			case <-stop:
				return
			case <-sigChan:
				count++  // increase the signal counter
				cancel() //nolint:wsl // cancel context on first signal, allowing graceful shutdown

				if count >= 2 { //nolint:mnd // in case of repeated signals
					once.Do(func() {
						_, _ = fmt.Fprintln(os.Stderr, "forced shutdown")

						runtime.Gosched()                    // increase the chance of graceful shutdown a bit
						<-time.After(500 * time.Millisecond) // give the last chance to finish the work
						os.Exit(1)                           // kill the process
					})
				}
			}
		}
	}()

	if len(os.Args) < 1 {
		return errors.New("missing application name")
	}

	// run the CLI application
	return cli.NewApp(filepath.Base(os.Args[0])).Run(ctx, os.Args[1:])
}

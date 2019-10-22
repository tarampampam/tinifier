package main

import (
	"go.uber.org/dig"
	"log"
	"os"
)

var container = dig.New()

func init() {
	var bindError error

	// Bind targets container
	if err := container.Provide(func() ITargets {
		return NewTargets()
	}); err != nil {
		bindError = err
	}

	// Bind logger container
	if err := container.Provide(func() ILogger {
		return NewLogger(
			log.New(os.Stdout, "", 0),
			log.New(os.Stderr, "", 0),
			true,
			true,
		)
	}); err != nil {
		bindError = err
	}

	// In any bind error happens - stop execution
	if bindError != nil {
		panic(bindError)
	}
}

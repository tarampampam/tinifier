package main

import (
	"go.uber.org/dig"
	"log"
	"os"
)

var container = dig.New()

func init() {
	var bindError error

	// Bind colors container
	if err := container.Provide(func() IAnsiColors {
		return NewAnsiColors()
	}); err != nil {
		bindError = err
	}

	// Bind targets container
	if err := container.Provide(func() ITargets {
		return NewTargets()
	}); err != nil {
		bindError = err
	}

	// Bind logger container
	if err := container.Provide(func(c IAnsiColors) ILogger {
		return NewLogger(
			c,
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

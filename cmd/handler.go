package cmd

import (
	"github.com/jessevdk/go-flags"
	log "github.com/sirupsen/logrus"
)

type (
	// CommandHandler is a function that gets called to handle execution of a command.
	CommandHandler func(command flags.Commander, args []string) error

	// Executable interface says that (sub)command can be executed and accepts some prepared objects. Used for passing
	// custom arguments set (instead `flags.Commander` interface).
	Executable interface {
		flags.Commander
		Handle(logger *log.Logger, args []string) error
	}
)

// CommandsHandler return commands handler for `flags` package, that allows to override commands handling methods. By
// default, if command implements `flags.Commander` interface - `Execute` function will be called, but using this
// handler we can checks command for `cmd.Executable` interface implementation and call method `Handle` instead.
func CommandsHandler(root *Root, logger *log.Logger) CommandHandler {
	return func(command flags.Commander, args []string) error {
		// enable verbose/debug mode soon as possible
		if root.IsDebug {
			logger.SetLevel(log.TraceLevel)
		} else if root.IsVerbose {
			logger.SetLevel(log.DebugLevel)
		}

		// The command passed into CommandHandler may be nil in case there is no command to be executed when parsing
		// has finished.
		if command == nil {
			return nil
		}

		if executable, ok := command.(Executable); ok {
			return executable.Handle(logger, args)
		}

		// fallback
		return command.Execute(args)
	}
}

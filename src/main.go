package main

import (
	"github.com/jessevdk/go-flags"
	"github.com/logrusorgru/aurora"
	"os"
)

const VERSION = "0.1.0" // Do not forget update this value before new version releasing

func main() {
	cError := container.Invoke(func(logger ILogger, targets ITargets) {
		// Parse passed options
		if parser, _, err := options.Parse(); parser != nil && err != nil {
			if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
				os.Exit(0)
			} else {
				parser.WriteHelp(os.Stdout)
				os.Exit(1)
			}
		}

		// Proxy verbosity state to the logger
		logger.SetVerbose(options.Verbose)
		// Set colorizing state
		logger.SetColors(!options.DisableColors)

		// Show application version and exit, if flag `-V` passed
		if options.ShowVersion == true {
			logger.Info("Version:", aurora.BrightYellow(VERSION))
			os.Exit(0)
		}

		// Make options check
		if _, err := options.Check(); err != nil {
			logger.Error(err)
			os.Exit(1)
		} else {
			// Set tinypng.com api key
			compressor.SetKey(options.ApiKey)
		}

		// Request for currently used quota
		if options.CheckQuota {
			if current, err := compressor.GetQuotaUsage(); err == nil {
				logger.Info("Current quota usage:", aurora.BrightYellow(current))
				os.Exit(0)
			} else {
				logger.Fatal("Cannot get current quota usage (double check your API key and network settings)")
			}
		}

		// Convert targets into file paths
		files := *targets.Load(options.Targets.Path, &options.FileExtensions)

		// Check for found files
		if filesLen := len(files); filesLen >= 1 {
			logger.Verbose("Found files:", aurora.BrightYellow(filesLen))

			// Set lower threads count if files count less then passed threads count
			if filesLen < options.Threads {
				options.Threads = filesLen
			}
		} else {
			logger.Fatal("Files for processing was not found")
		}

		logger.Verbose("Files list:", files)

		tasks := NewTasks(&files, options.Threads, options.MaxErrors)
		go tasks.FillUpTasks()

		// Enable spinner color if this action is allowed
		if !options.DisableColors {
			if err := tasks.Spin.Color("fgYellow"); err != nil {
				logger.Error(err)
			}
		}

		logger.Verbose("Start", aurora.BrightYellow(options.Threads), "threads")

		tasks.StartWorkers()
		errCount := tasks.Wait(func() {
			logger.Error("Working stopped")
			tasks.PrintResults(os.Stdout)
			tasks.PrintErrors(os.Stderr)
		})

		tasks.PrintResults(os.Stdout)
		tasks.PrintErrors(os.Stderr)

		// Make check for errors count
		if options.MaxErrors > 0 && errCount >= options.MaxErrors {
			logger.Fatal("Too many errors occurred, working stopped")
		}
	})

	if cError != nil {
		panic(cError)
	}
}

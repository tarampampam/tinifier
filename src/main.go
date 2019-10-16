package main

import (
	"github.com/jessevdk/go-flags"
	"os"
)

const VERSION = "0.1.0" // Do not forget update this value before new version releasing

func main() {
	// Parse passed options
	if parser, _, err := options.Parse(); parser != nil && err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			parser.WriteHelp(logger.std.Writer())
			os.Exit(1)
		}
	}

	// Proxy verbosity state to the logger
	logger.isVerbose = options.Verbose
	// Set colorizing state
	colors.enableColors(!options.DisableColors)

	// Show application version and exit, if flag `-V` passed
	if options.ShowVersion == true {
		logger.Info("Version:", colors.au.BrightYellow(VERSION))
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
			logger.Info("Current quota usage:", colors.au.BrightYellow(current))
			os.Exit(0)
		} else {
			logger.Fatal("Cannot get current quota usage (double check your API key and network settings)")
		}
	}

	// Convert targets into file paths
	targets.Load(options.Targets.Path, &options.FileExtensions)

	// Check for found files
	if filesLen := len(targets.Files); filesLen >= 1 {
		logger.Verbose("Found files:", colors.au.BrightYellow(filesLen))

		// Set lower threads count if files count less then passed threads count
		if filesLen < options.Threads {
			options.Threads = filesLen
		}
	} else {
		logger.Fatal("Files for processing was not found")
	}

	logger.Verbose("Files list:", targets.Files)

	tasks := NewTasks(&targets, options.Threads, options.MaxErrors)
	go tasks.FillUpTasks()

	// Enable spinner color if this action is allowed
	if !options.DisableColors {
		if err := tasks.Spin.Color("fgYellow"); err != nil {
			logger.Error(err)
		}
	}

	logger.Verbose("Start", colors.au.BrightYellow(options.Threads), "threads")

	tasks.StartWorkers()
	errCount := tasks.Wait()

	tasks.PrintResults(logger.std.Writer(), logger.err.Writer())

	// Make check for errors count
	if options.MaxErrors > 0 && errCount >= options.MaxErrors {
		logger.Fatal("Too many errors occurred, working stopped")
	}
}

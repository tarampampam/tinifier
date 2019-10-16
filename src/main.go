package main

import (
	"github.com/jessevdk/go-flags"
	color "github.com/logrusorgru/aurora"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
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

	targets.Load(options.Targets.Path, &options.FileExtensions)

	//fmt.Println(targets)
	//return

	// Check for files found
	if filesLen := len(targets.Files); filesLen >= 1 {
		logger.Info("Found files:", color.BrightYellow(filesLen))

		// Set lower threads count if files count less then passed threads count
		if filesLen < options.Threads {
			options.Threads = filesLen
		}
	} else {
		logger.Fatal("Files for processing was not found")
	}

	// Print files list (for verbose mode)
	if options.Verbose {
		logger.Verbose("\nFiles list:")
		for _, filePath := range targets.Files {
			logger.Verbose("  >", color.Blue(filePath))
		}
		logger.Verbose()
	}

	logger.Info("Start", color.BrightYellow(options.Threads), "threads")

	// Create channel with file paths
	channel := make(chan string, len(targets.Files))

	// Create a wait group <https://nathanleclaire.com/blog/2014/02/15/how-to-wait-for-all-goroutines-to-finish-executing-before-continuing/>
	var wg sync.WaitGroup

	wg.Add(options.Threads)

	// Fill up the channel with file paths (async)
	for _, filePath := range targets.Files {
		channel <- filePath
	}

	for i := 0; i < options.Threads; i++ {
		go func() {
			defer wg.Done()

			for {
				if len(channel) > 0 {
					if err := processFile(<-channel); err != nil {
						logger.Error(color.BrightRed(err))
					}
				} else {
					break
				}
			}
		}()
	}

	// Time to time request for the current quota usage
	go func() {
		for {
			if len(channel) > 0 {
				if quotaUsage, err := compressor.GetQuotaUsage(); err == nil {
					logger.Info("Current quota usage:", color.BrightYellow(quotaUsage))
				} else {
					logger.Error(color.BrightRed(err))
				}

				time.Sleep(10 * time.Second)
			} else {
				break
			}
		}
	}()

	wg.Wait()

	// Show current quota usage before exit
	if quotaUsage, err := compressor.GetQuotaUsage(); err == nil {
		logger.Info("Current quota usage:", color.BrightYellow(quotaUsage))
	}
}

// Main function - compress file
func processFile(filePath string) error {
	var (
		logColors = [...]color.Color{
			color.BrightFg, color.GreenFg, color.YellowFg, color.BlueFg, color.MagentaFg, color.CyanFg, color.WhiteFg,
		}
		randLogColor = logColors[(rand.New(rand.NewSource(time.Now().UnixNano()))).Intn(len(logColors))]
		logger       = log.New(logger.std.Writer(), color.Sprintf(color.Colorize("[%s] ", randLogColor|color.BoldFm), filePath), logger.std.Flags()|log.Ltime)
	)

	if options.Verbose {
		logger.Println("Read file info buffer")
	}

	// Read file info buffer
	if imageData, err := ioutil.ReadFile(filePath); err == nil {
		var originalFileLen = int64(len(imageData))
		if options.Verbose {
			logger.Printf("Original file size: %d bytes\n", len(imageData))
		}

		logger.Println("Compressing file (upload and download back)..")

		// Compress file copy
		if err := compressor.CompressBuffer(&imageData, filePath); err == nil {
			imageData = nil // Make clean

			logger.Println("File compressed and overwritten successful")

			if info, err := os.Stat(filePath); err == nil {
				logger.Printf("Compression ratio: %0.1f%%\n", math.Abs(float64(info.Size()-originalFileLen)/float64(originalFileLen)*100))
			}
		} else {
			logger.Println(color.BrightRed("Error while compressing file"), err)

			return err
		}
	} else {
		logger.Println(color.BrightRed("Cannot read file into buffer"), err)

		return err
	}

	return nil
}

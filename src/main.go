package main

import (
	tinypngClient "github.com/gwpp/tinify-go/tinify"
	flags "github.com/jessevdk/go-flags"
	color "github.com/logrusorgru/aurora"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const AppVersion = "0.0.3" // Do not forget update this value on release

func main() {
	var parser = flags.NewParser(&options, flags.Default)

	// Check passed parameters (flags)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			parser.WriteHelp(logger.std.Writer())
			os.Exit(1)
		}
	}

	// Show application version and exit, if flag `-V` passed
	if options.ShowVersion == true {
		logger.Verbose("Version: %s\n", AppVersion)
		os.Exit(0)
	}

	// Check API key
	if key := strings.TrimSpace(options.ApiKey); len(key) >= 1 {
		if options.Verbose {
			logger.Verbose("API key:", color.BrightYellow(key))
		}
		tinypngClient.SetKey(key)
	} else {
		logger.Fatal(color.BrightRed("tinypng.com API key is not provided"))
	}

	// Check threads count
	if options.Threads <= 0 {
		logger.Fatal(color.BrightRed("Threads count cannot be less then 1"))
	}

	var files []string

	// Try to get files list
	files, _ = targetsToFilePath(options.Targets.Path)
	files = filterFilesUsingExtensions(files, &options.FileExtensions)

	// Check for files found
	if filesLen := len(files); filesLen >= 1 {
		logger.Info("Found files:", color.BrightYellow(filesLen))

		// Set lower threads count if files count less then passed threads count
		if filesLen < options.Threads {
			options.Threads = filesLen
		}
	} else {
		logger.Fatal(color.BrightRed("Files for processing was not found"))
	}

	// Print files list (for verbose mode)
	if options.Verbose {
		logger.Verbose("\nFiles list:")
		for _, filePath := range files {
			logger.Verbose("  >", color.Blue(filePath))
		}
		logger.Verbose()
	}

	logger.Info("Start", color.BrightYellow(options.Threads), "threads")

	// Create channel with file paths
	channel := make(chan string, len(files))

	// Create a wait group <https://nathanleclaire.com/blog/2014/02/15/how-to-wait-for-all-goroutines-to-finish-executing-before-continuing/>
	var wg sync.WaitGroup

	wg.Add(options.Threads)

	// Fill up the channel with file paths (async)
	for _, filePath := range files {
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
				if quotaUsage, err := getQuotaUsage(tinypngClient.GetClient()); err == nil {
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
	if quotaUsage, err := getQuotaUsage(tinypngClient.GetClient()); err == nil {
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
		if err := compressFile(&imageData, filePath); err == nil {
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

// Get service used quota value
func getQuotaUsage(client *tinypngClient.Client) (int, error) {
	// If you know better way for getting current quota usage - please, make an issue in current repository
	if response, err := client.Request(http.MethodPost, "/shrink", nil); err == nil {
		if count, err := strconv.Atoi(response.Header["Compression-Count"][0]); err == nil {
			return count, nil
		} else {
			return -1, err
		}
	} else {
		return -1, err
	}
}

// Compress image using tinypng.com service. Important: API key must be set before function calling
func compressFile(buffer *[]byte, out string) error {
	if source, err := tinypngClient.FromBuffer(*buffer); err != nil {
		return err
	} else {
		if err := source.ToFile(out); err != nil {
			return err
		}
	}

	return nil
}

// Convert targets into file path slice. If target points to the directory - directory files will be read and returned
// (with absolute path). If file - file absolute path will be returned. Any invalid value (path to the non-existing
// file - this entry will be skipped)
func targetsToFilePath(targets []string) (result []string, error error) {
	// Iterate passed targets
	for _, path := range targets {
		// Extract absolute path to the target
		if absPath, err := filepath.Abs(path); err == nil {
			// If file stats extracted successful
			if info, err := os.Stat(absPath); err == nil {
				switch mode := info.Mode(); {
				case mode.IsDir(): // If directory found - run files iterator inside it
					if files, err := ioutil.ReadDir(absPath); err == nil {
						for _, file := range files {
							if file.Mode().IsRegular() {
								if abs, err := filepath.Abs(absPath + "/" + file.Name()); err == nil {
									result = append(result, abs)
								}
							}
						}
					} else {
						return result, err
					}

				case mode.IsRegular(): // If regular file found - append it into result
					result = append(result, absPath)
				}
			}
		} else {
			return result, err
		}
	}

	return result, nil
}

// Make files slice filtering using extensions slice. Extension can be combined (delimiter is ",")
func filterFilesUsingExtensions(files []string, extensions *[]string) (result []string) {
	const delimiter = ","

	for _, path := range files {
		for _, extension := range *extensions {
			if strings.Contains(extension, delimiter) {
				for _, subExtension := range strings.Split(extension, delimiter) {
					if strings.HasSuffix(path, subExtension) {
						result = append(result, path)
					}
				}
			} else {
				if strings.HasSuffix(path, extension) {
					result = append(result, path)
				}
			}
		}
	}

	return result
}

package main

import (
	"errors"
	tinypngClient "github.com/gwpp/tinify-go/tinify"
	flags "github.com/jessevdk/go-flags"
	color "github.com/logrusorgru/aurora"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	AppVersion = "0.0.1"
)

type Options struct {
	Verbose        bool     `short:"v" description:"Show verbose debug information"`
	ShowVersion    bool     `short:"V" long:"version" description:"Show version and exit"`
	FileExtensions []string `short:"e" long:"ext" default:"jpg,JPG,jpeg,JPEG,png,PNG" description:"Target file extensions"`
	ApiKey         string   `short:"k" long:"api-key" env:"TINYPNG_API_KEY" description:"API key <https://tinypng.com/dashboard/api>"`
	Threads        int      `short:"t" long:"threads" default:"5" description:"Threads processing count"`
	Targets        struct {
		Path []string `positional-arg-name:"files-and-directories"`
	} `positional-args:"yes" required:"true"`
}

var (
	options   Options
	errorsLog = log.New(os.Stderr, "", 0)
	infoLog   = log.New(os.Stdout, "", 0)
)

func main() {
	var parser = flags.NewParser(&options, flags.Default)

	// Check passed parameters (flags)
	if _, err := parser.Parse(); err != nil {
		if flagsErr, ok := err.(*flags.Error); ok && flagsErr.Type == flags.ErrHelp {
			os.Exit(0)
		} else {
			parser.WriteHelp(os.Stdout)
			os.Exit(1)
		}
	}

	// Show application version and exit, if flag `-V` passed
	if options.ShowVersion == true {
		infoLog.Printf("Version: %s\n", AppVersion)
		os.Exit(0)
	}

	// Check API key
	if key := strings.TrimSpace(options.ApiKey); len(key) >= 1 {
		if options.Verbose {
			infoLog.Println("API key:", color.BrightYellow(key))
		}
		tinypngClient.SetKey(key)
	} else {
		errorsLog.Fatal(color.BrightRed("tinypng.com API key is not provided"))
	}

	// Check threads count
	if options.Threads <= 0 {
		errorsLog.Fatal(color.BrightRed("Threads count cannot be less then 1"))
	}

	var files []string

	// Try to get files list
	files, _ = targetsToFilePath(options.Targets.Path)
	files = filterFilesUsingExtensions(files, &options.FileExtensions)

	// Check for files found
	if filesLen := len(files); filesLen >= 1 {
		infoLog.Println("Found files:", color.BrightYellow(filesLen))
	} else {
		errorsLog.Fatal(color.BrightRed("Files for processing was not found"))
	}

	// Print files list (for verbose mode)
	if options.Verbose {
		infoLog.Println("\nFiles list:")
		for _, filePath := range files {
			infoLog.Println("  >", color.Blue(filePath))
		}
		infoLog.Println()
	}

	infoLog.Println("Start", color.BrightYellow(options.Threads), "threads")

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
						errorsLog.Println(color.BrightRed(err))
					}
				} else {
					break
				}
			}
		}()
	}

	wg.Wait()
}

// Main function - compress file
func processFile(filePath string) error {
	var (
		tmpFileName = filePath + ".tmp"
		logColors   = [7]color.Color{
			color.RedFg, color.GreenFg, color.YellowFg, color.BlueFg, color.MagentaFg, color.CyanFg, color.WhiteFg,
		}
		randLogColor = logColors[(rand.New(rand.NewSource(time.Now().UnixNano()))).Intn(len(logColors))]
		logger       = log.New(infoLog.Writer(), color.Sprintf(color.Colorize("[%s] ", randLogColor|color.BoldFm), filePath), infoLog.Flags()|log.Ltime)
	)

	if options.Verbose {
		logger.Printf("Make file copy to %s\n", tmpFileName)
	}

	// Make original file copy
	if size, err := copyFile(filePath, tmpFileName); err == nil {
		if options.Verbose {
			logger.Printf("Copied file size: %d bytes\n", size)
		}

		logger.Println("Compressing file (upload and download back)..")

		// Compress file copy
		if err := compressFile(tmpFileName, tmpFileName); err == nil {
			logger.Println("File compressed and download successful")

			// Remove original file
			if err := os.Remove(filePath); err == nil {
				if err := os.Rename(tmpFileName, filePath); err == nil {
					if options.Verbose {
						logger.Println("Original file replaced with compressed temporary")
					}

					if info, err := os.Stat(filePath); err == nil {
						logger.Printf("Compression ratio: %0.1f%%\n", math.Abs(float64(info.Size()-size)/float64(size)*100))
					}
				} else {
					logger.Println("Cannot rename temporary file to original filename", err)

					return err
				}
			} else {
				logger.Println("Cannot remove original file", err)

				return err
			}
		} else {
			logger.Println("Error while compressing file", err)

			if err := os.Remove(tmpFileName); err == nil {
				logger.Println("Cannot remove temporary file", err)

				return err
			}

			return err
		}
	} else {
		return err
	}

	return nil
}

// Compress image using tinypng.com service. Important: API key must be set before function calling
func compressFile(in string, out string) error {
	if source, err := tinypngClient.FromFile(in); err != nil {
		return err
	} else {
		if err := source.ToFile(out); err != nil {
			return err
		}
	}

	return nil
}

// Copy file and return count of copied bytes and optionally error
func copyFile(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, errors.New(src + " is not a regular file")
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}

	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()

	nBytes, err := io.Copy(destination, source)

	return nBytes, err
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

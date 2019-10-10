package main

import (
	flags "github.com/jessevdk/go-flags"
	color "github.com/logrusorgru/aurora"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	AppVersion = "0.0.1"
)

type Options struct {
	Verbose        bool     `short:"v" description:"Show verbose debug information"`
	ShowVersion    bool     `short:"V" long:"version" description:"Show version and exit"`
	FileExtensions []string `short:"e" long:"ext" default:"jpg,JPG,jpeg,JPEG,png,PNG" description:"Target file extensions"`
	ApiKey         string   `short:"k" long:"api-key" env:"TINYPNG_API_KEY" description:"API key <https://tinypng.com/dashboard/api>"`
	Threads        int      `short:"t" long:"threads" description:"Threads processing count"`
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

	var apiKey string

	// Check API key
	if key := strings.TrimSpace(options.ApiKey); len(key) >= 1 {
		apiKey = key
	} else {
		errorsLog.Fatal(color.BrightRed("tinypng.com API key is not provided"))
	}

	var files []string

	// Try to get files list
	files, _ = targetsToFilePath(options.Targets.Path)

	// Check for files found
	if filesLen := len(files); filesLen >= 1 {
		infoLog.Println("Found files:", color.BrightYellow(filesLen))
	} else {
		errorsLog.Fatal(color.BrightRed("Files for processing was not found"))
	}

	infoLog.Printf(apiKey)

	// @todo: Write code
}

// Convert targets into file path slice. If target points to the directory - directory files will be read and returned
// (with absolute path). If file - file absolute path will be returned. Any invalid value (path to the non-existing
// file - this entry will be skipped)
func targetsToFilePath(targets []string) (patch []string, error error) {
	var result []string

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

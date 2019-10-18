package main

import (
	"errors"
	"fmt"
	"github.com/briandowns/spinner"
	"github.com/olekukonko/tablewriter"
	"io"
	"io/ioutil"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"sync"
	"syscall"
	"time"
)

// Task for processing.
type Task struct {
	FilePath   string
	ShouldStop bool // This bit means that goroutine should exit instead job processing
}

// Tasks processing result.
type TaskResult struct {
	FilePath     string
	OriginalSize int64
	ResultSize   int64
	Ratio        float64
}

// Tasks processor.
type Tasks struct {
	Targets    *Targets
	WG         sync.WaitGroup
	Spin       *spinner.Spinner
	Errors     []error
	maxErrors  int
	currentPos int
	maxPos     int
	ch         chan Task
	thCount    int
	Results    []TaskResult
}

// Callable for calling on working break.
type OnWorkingBreak func()

// Create new tasks processor.
func NewTasks(targets *Targets, thCount int, maxErrors int) *Tasks {
	res := Tasks{
		Targets:   targets,
		Spin:      spinner.New(spinner.CharSets[14], 150*time.Millisecond),
		maxPos:    len(targets.Files),
		ch:        make(chan Task, thCount),
		thCount:   thCount,
		maxErrors: maxErrors,
	}

	res.Spin.Prefix = " "

	return &res
}

// Fill up tasks queue (run this function async).
func (t *Tasks) FillUpTasks() {
	//defer func() {
	//	if r := recover(); r != nil {
	//		// panic: send on closed channel
	//	}
	//}()

	// Push tasks for a processing
	for _, filePath := range t.Targets.Files {
		t.ch <- Task{FilePath: filePath}
	}
	// And send exit signals at the end
	for i := 0; i < t.thCount; i++ {
		t.ch <- Task{ShouldStop: true}
	}
}

// Start queue workers.
func (t *Tasks) StartWorkers() {
	for i := 0; i < t.thCount; i++ {
		t.WG.Add(1)

		go func(workerNum int) {
			defer t.WG.Done()

			for {
				task := <-t.ch

				if !task.ShouldStop {
					t.currentPos++
					t.Spin.Suffix = fmt.Sprintf(
						" %0.1f%% (%d of %d) Compressing file [%s]…",
						math.Abs(float64(t.currentPos*100)/float64(t.maxPos)),
						t.currentPos,
						t.maxPos,
						filepath.Base(task.FilePath),
					)

					// Read image into buffer
					if imageData, err := ioutil.ReadFile(task.FilePath); err == nil {
						var originalFileLen = int64(len(imageData))

						// Compress image and overwrite original file
						if _, err := compressor.CompressBuffer(&imageData, task.FilePath); err == nil {
							// Get file info
							if info, err := os.Stat(task.FilePath); err == nil {
								t.Results = append(t.Results, TaskResult{
									FilePath:     task.FilePath,
									OriginalSize: originalFileLen,
									ResultSize:   info.Size(),
									Ratio:        math.Abs(float64(info.Size()-originalFileLen) / float64(originalFileLen) * 100),
								})
							}
						} else {
							t.Errors = append(t.Errors,
								errors.New("Cannot compress file \""+filepath.Base(task.FilePath)+"\": remote error"),
							)
						}
					} else {
						t.Errors = append(t.Errors, err)
					}
				} else {
					break
				}

				if t.maxErrors > 0 && len(t.Errors) >= t.maxErrors {
					break
				}
			}
		}(i)
	}
}

// Wait until all workers complete queue jobs.
func (t *Tasks) Wait(onBreak OnWorkingBreak) int {
	// Make a channel for system signals
	signals := make(chan os.Signal, 1)

	// "Subscribe" for system signals
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Listen for signal
	go func() {
		switch s := <-signals; s {
		// On SIGUSR1 just exit
		case syscall.SIGUSR1:
			break
		// In any another way - call the action
		default:
			if t.Spin.Active() {
				t.Spin.Stop()
			}
			onBreak()
			os.Exit(1)
		}
	}()

	defer close(signals)

	t.Spin.Start()
	t.WG.Wait()

	// Stop subscribing and say for goroutine to exit
	signal.Stop(signals)
	signals <- syscall.SIGUSR1

	t.Spin.Stop()

	return len(t.Errors)
}

// Print processing results.
func (t *Tasks) PrintResults(std io.Writer) {
	if len(t.Results) > 0 {
		var totalSaved int64
		table := tablewriter.NewWriter(std)

		for _, res := range t.Results {
			table.Append([]string{
				filepath.Base(res.FilePath),
				strconv.FormatInt(res.OriginalSize/1024, 10) + " Kb",
				strconv.FormatInt(res.ResultSize/1024, 10) + " Kb",
				fmt.Sprintf("%0.2f%%", math.Abs(res.Ratio)),
				strconv.FormatInt((res.OriginalSize-res.ResultSize)/1024, 10) + " Kb",
			})
			totalSaved = totalSaved + (res.OriginalSize - res.ResultSize)
		}

		table.SetHeader([]string{"File Name", "Original Size", "Compressed", "Compress Ratio", "Saved"})
		table.SetFooter([]string{"", "", "", "Total saved", strconv.FormatInt(totalSaved/1024, 10) + " Kb"})

		table.Render()
	}
}

// Print processing errors.
func (t *Tasks) PrintErrors(std io.Writer) {
	if len(t.Errors) > 0 {
		table := tablewriter.NewWriter(std)
		table.SetColWidth(80)

		for i, err := range t.Errors {
			table.Append([]string{
				strconv.Itoa(i + 1),
				err.Error(),
			})
		}

		table.SetHeader([]string{"#", "Error details"})

		table.Render()
	}
}

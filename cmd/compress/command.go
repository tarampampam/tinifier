package compress

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
	"tinifier/cmd/shared"
	"tinifier/tinypng"

	"github.com/dustin/go-humanize"
)

const tinypngRequestTimeout time.Duration = time.Second * 60

type Command struct {
	shared.WithAPIKey
	FileExtensions fileExtensions `short:"e" long:"ext" default:"jpg,JPG,jpeg,JPEG,png,PNG" description:"Image file extensions"` //nolint:lll
	Threads        int            `short:"t" long:"threads" default:"5" description:"Threads count"`
	Targets        struct {
		Path targets `positional-arg-name:"files-and-directories" required:"true"`
	} `positional-args:"yes"`
}

// Execute command.
func (cmd *Command) Execute(_ []string) error {
	// get tasks for processing
	tasks, err := cmd.getTasks(cmd.FileExtensions.GetAll(), cmd.Targets.Path.Expand())
	if err != nil {
		return err
	}

	var (
		tasksWg     = sync.WaitGroup{}
		tasksChan   = make(chan task, cmd.Threads)
		resultsWg   = sync.WaitGroup{}
		resultsChan = make(chan result)
		ossChan     = make(chan os.Signal, 1) // channel for operational system signals
		tiny        = tinypng.NewClient(cmd.APIKey.String(), tinypngRequestTimeout)
	)

	// "subscribe" for system signals
	signal.Notify(ossChan, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// fill-up tasks channel
	go func() {
		for _, task := range *tasks {
			tasksChan <- task
		}

		close(tasksChan)
	}()

	// create context
	ctx, cancel := context.WithCancel(context.Background())

	// start workers
	for i := 0; i < cmd.Threads; i++ {
		tasksWg.Add(1)

		go cmd.work(ctx, tiny, tasksChan, resultsChan, &tasksWg)
	}

	// listen for system signals in separate goroutine
	go func() {
		<-ossChan
		cancel()
	}()

	resultsWg.Add(1)

	go cmd.readResults(resultsChan, len(*tasks), &resultsWg)

	tasksWg.Wait()
	close(resultsChan)
	resultsWg.Wait()

	return nil
}

func (cmd *Command) getTasks(extensions []string, targets []string) (*[]task, error) {
	var tasks []task

	// create extensions map for fast checking
	extensionsMap := make(map[string]bool)
	for _, ext := range extensions {
		extensionsMap[ext] = true
	}

	for i, filePath := range targets {
		if _, ok := extensionsMap[strings.Trim(filepath.Ext(filePath), ". ")]; ok {
			tasks = append(tasks, task{num: uint32(i), filePath: filePath})
		}
	}

	if len(tasks) == 0 {
		return nil, errors.New("there is no files for a work")
	}

	return &tasks, nil
}

func (cmd *Command) work(ctx context.Context, c *tinypng.Client, tasks chan task, res chan result, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done(): // if `cancel()` executed
			return

		default:
			task, isOpened := <-tasks // get task

			if !isOpened {
				return
			}

			result := result{filePath: task.filePath}

			// open source file for reading
			if readFile, openReadErr := os.OpenFile(task.filePath, os.O_RDONLY, 0); openReadErr == nil {
				if resp, compressErr := c.Compress(ctx, readFile); compressErr == nil {
					result.fileType = resp.Output.Type
					result.originalSizeBytes = resp.Input.Size
					result.compressedSizeBytes = resp.Output.Size

					// open source file for writing
					if writeFile, openWriteErr := os.OpenFile(readFile.Name(), os.O_WRONLY|os.O_TRUNC, 0644); openWriteErr == nil {
						if _, copyErr := io.Copy(writeFile, resp.Compressed); copyErr != nil {
							result.error = copyErr
						}

						writeFile.Close()
					} else {
						result.error = openWriteErr
					}
				} else {
					result.error = compressErr
				}

				readFile.Close()
			} else {
				result.error = openReadErr
			}

			res <- result
		}
	}
}

// readResults reads tasks processing results and show some stats for user.
func (cmd *Command) readResults(results chan result, total int, wg *sync.WaitGroup) { //nolint:funlen
	// calcPercentageDiff calculate difference between passed values in percentage representation
	calcPercentageDiff := func(from, to float64) float64 {
		if to <= 0 {
			return 0
		}

		return math.Abs(((from - to) / to) * 100) //nolint:gomnd
	}

	// create UI elements
	progressBar := newProgress("Compress files", total, os.Stderr)
	table := newTable(os.Stdout)

	// define counter for summary stats
	var (
		totalOriginalBytes uint64
		totalCompressedBytes,
		totalSavedBytes int64
	)

	defer func() {
		// stop progress bar and render table on exit
		_ = progressBar.Clear()

		table.Render()

		wg.Done()
	}()

	// set table headers
	table.SetHeader([]string{"File Name", "Type", "Size Difference", "Saved"})

	// read for messages with results
	for {
		result, isOpened := <-results

		// when channel closes - we should break our infinite loop
		if !isOpened {
			break
		}

		basePath := filepath.Base(result.filePath)

		// increment progress bar state
		_ = progressBar.Add(1)

		if result.error == nil {
			// calculate stat values
			diffBytes := int64(result.originalSizeBytes - result.compressedSizeBytes)
			diffPercentage := calcPercentageDiff(float64(result.compressedSizeBytes), float64(result.originalSizeBytes))
			totalOriginalBytes += result.originalSizeBytes
			totalSavedBytes += diffBytes
			totalCompressedBytes += int64(result.compressedSizeBytes)

			// append a row in a table
			table.Append([]string{
				basePath,
				result.fileType,
				fmt.Sprintf(
					"%s  →  %s",
					humanize.IBytes(result.originalSizeBytes),
					humanize.IBytes(result.compressedSizeBytes),
				),
				fmt.Sprintf(
					"%s (-%0.2f%%)",
					humanize.IBytes(uint64(diffBytes)),
					diffPercentage,
				),
			})
		} else {
			errorMessage := result.error.Error()

			if errors.Unwrap(result.error) == context.Canceled {
				errorMessage = "Task canceled"
			}

			table.Append([]string{filepath.Base(result.filePath), result.fileType, "ERROR", errorMessage})
		}
	}

	// append summary stats into table
	table.SetFooter([]string{"", "", "Total saved", fmt.Sprintf("%s (-%0.2f%%)",
		humanize.IBytes(uint64(totalSavedBytes)),
		calcPercentageDiff(float64(totalCompressedBytes), float64(totalOriginalBytes)),
	)})
}

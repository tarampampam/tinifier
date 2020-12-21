package pipeline

import (
	"context"
	"sync"
)

// compressor compress the image, that described in the Task. On successful compression it returns TaskResult with
// compression details, or an error on any fuck-up.
type compressor interface {
	// Compress the image.
	Compress(context.Context, Task) (*TaskResult, error)
}

type (
	TaskHandler   func(Task)       // TaskHandler pre/post task processing
	ResultHandler func(TaskResult) // ResultHandler processes task result
	ErrorHandler  func(TaskError)  // ErrorHandler processes task error
)

// CompressingPipeline is main tasks pipeline structure.
type CompressingPipeline struct {
	ctx          context.Context
	tasks        []Task
	compressor   compressor
	preWorkerRun TaskHandler
	onResult     ResultHandler
	onError      ErrorHandler
}

type (
	// Task describes all required image properties (like path to the file) for compressing.
	Task struct {
		FilePath string
	}

	// TaskResult describes successful compression results.
	TaskResult struct {
		FileType       string
		FilePath       string
		OriginalSize   uint64 // in bytes
		CompressedSize uint64 // in bytes
		UsedQuota      uint64 // aka "CompressionCount"
	}

	// TaskError contains all information about task compressing error.
	TaskError struct {
		Task  Task
		Error error
	}
)

// NewCompressingPipeline creates new pipeline.
func NewCompressingPipeline(
	ctx context.Context,
	tasks []Task,
	compressor compressor,
	onResult ResultHandler,
	onError ErrorHandler,
	options ...CompressingPipelineOption,
) CompressingPipeline {
	p := CompressingPipeline{
		ctx:        ctx,
		tasks:      tasks,
		compressor: compressor,
		onResult:   onResult,
		onError:    onError,
	}

	for i := 0; i < len(options); i++ {
		options[i](&p)
	}

	return p
}

// Run the compression pipeline.
//
// Goroutines working schema:
//
// 	                         |---------|       results     |-----------------|
// 	                         | |---------|   ------------> | results watcher |
// 	 |-----------|   tasks   |-| |---------|               |-----------------|
// 	 | scheduler |  -------->  |-| |---------|
// 	 |-----------|               |-| |---------|  errors   |----------------|
// 	                               |-| workers | --------> | errors watcher |
// 	                                 |---------|           |----------------|
//
func (p *CompressingPipeline) Run(workersCount uint8) <-chan struct{} {
	var (
		queue     = make(chan Task, workersCount)
		resultsCh = make(chan TaskResult, workersCount)
		errorsCh  = make(chan TaskError)
		done      = make(chan struct{})

		workersWg, collectorsWg sync.WaitGroup
	)

	// run scheduler
	go func() {
		defer close(queue)
		p.runScheduler(queue)
	}()

	// run workers
	workersWg.Add(int(workersCount))
	for i := uint8(0); i < workersCount; i++ { //nolint:wsl
		go func() {
			defer workersWg.Done()

			p.runWorker(queue, resultsCh, errorsCh)
		}()
	}

	collectorsWg.Add(1)
	// run results watcher
	go func() {
		defer collectorsWg.Done()
		p.runResultsWatcher(resultsCh)
	}()

	collectorsWg.Add(1)
	// run errors watcher
	go func() {
		defer collectorsWg.Done()
		p.runErrorsWatcher(errorsCh)
	}()

	go func() {
		workersWg.Wait()
		close(resultsCh)
		close(errorsCh)
		collectorsWg.Wait()

		done <- struct{}{}
	}()

	return done
}

// runScheduler fill-up tasks queue.
func (p *CompressingPipeline) runScheduler(queue chan<- Task) {
	for i := 0; i < len(p.tasks); i++ {
		select {
		case <-p.ctx.Done():
			return

		case queue <- p.tasks[i]:
			continue
		}
	}
}

// runWorker reads tasks from the queue and call compressor for image compressing. Results will be published into
// required channels.
func (p *CompressingPipeline) runWorker(queue <-chan Task, results chan<- TaskResult, errors chan<- TaskError) {
	for {
		select {
		case <-p.ctx.Done():
			return

		case task, isOpened := <-queue:
			if p.ctx.Err() != nil || !isOpened {
				return
			}

			if p.preWorkerRun != nil {
				p.preWorkerRun(task)
			}

			if result, err := p.compressor.Compress(p.ctx, task); err != nil {
				errors <- TaskError{Task: task, Error: err}
			} else {
				results <- *result
			}
		}
	}
}

// runResultsWatcher reads compression results and call the onResult handler.
func (p *CompressingPipeline) runResultsWatcher(results <-chan TaskResult) {
	for {
		result, isOpened := <-results
		if !isOpened {
			return
		}

		p.onResult(result)
	}
}

// runResultsWatcher reads compression errors and call the onError handler.
func (p *CompressingPipeline) runErrorsWatcher(errors <-chan TaskError) {
	for {
		taskErr, isOpened := <-errors
		if !isOpened {
			return
		}

		p.onError(taskErr)
	}
}

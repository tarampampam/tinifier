package pool

import (
	"context"
	"sync"
)

type (
	FileInfo interface {
		Size() uint64
		Type() string
	}

	Worker interface {
		Upload(ctx context.Context, filePath string) (url string, info FileInfo, err error)
		Download(ctx context.Context, url, toFilePath string) (info FileInfo, err error)
		CopyContent(fromFilePath, toFilePath string) error
		RemoveFile(filePath string) error
	}

	PreRunner interface {
		PreTaskRun(task Task)
	}
)

type (
	Task struct {
		FilePath   string
		TaskNumber uint // from 1 to TasksCount
		TasksCount uint // summary task count
	}

	Result struct {
		Task Task

		FileType       string
		FilePath       string
		OriginalSize   uint64 // in bytes
		CompressedSize uint64 // in bytes

		Err error // if Err is not nil - some error occurred during task processing
	}

	Pool struct {
		ctx    context.Context
		worker Worker
	}
)

func NewPool(ctx context.Context, worker Worker) Pool {
	return Pool{
		ctx:    ctx,
		worker: worker,
	}
}

func (p *Pool) Run(filePaths []string, workersCount uint8) <-chan Result {
	var (
		queue   = make(chan Task, workersCount)
		results = make(chan Result)

		wg sync.WaitGroup
	)

	// run scheduler
	go func() {
		p.runScheduler(queue, filePaths)
		close(queue)
	}()

	wg.Add(int(workersCount))
	// run workers
	for i := uint8(0); i < workersCount; i++ {
		go func() {
			p.runWorker(queue, results)
			wg.Done()
		}()
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

func (p *Pool) runScheduler(queue chan<- Task, filePaths []string) {
	for i, l := uint(0), uint(len(filePaths)); i < l; i++ {
		if p.ctx.Err() != nil {
			return
		}

		select {
		case <-p.ctx.Done():
			return

		case queue <- Task{FilePath: filePaths[i], TaskNumber: i + 1, TasksCount: l}:
		}
	}
}

func (p *Pool) runWorker(queue <-chan Task, results chan<- Result) { //nolint:funlen
	for {
		if err := p.ctx.Err(); err != nil {
			return
		}

		select {
		case <-p.ctx.Done():
			return

		case task, isOpened := <-queue: // TODO move logic into separate function (defer remove tmp file)
			if !isOpened {
				return
			}

			if err := p.ctx.Err(); err != nil {
				results <- Result{Task: task, Err: err}

				return
			}

			if w, ok := p.worker.(PreRunner); ok {
				w.PreTaskRun(task)
			}

			url, uplInfo, uplErr := p.worker.Upload(p.ctx, task.FilePath)
			if uplErr != nil {
				results <- Result{Task: task, Err: uplErr}

				continue
			}

			tempFilePath := p.generateTempFilePathFor(task.FilePath)

			dlInfo, dlErr := p.worker.Download(p.ctx, url, tempFilePath)
			if dlErr != nil {
				_ = p.worker.RemoveFile(tempFilePath) // cleanup anyway
				results <- Result{Task: task, Err: dlErr}

				continue
			}

			if err := p.worker.CopyContent(tempFilePath, task.FilePath); err != nil {
				_ = p.worker.RemoveFile(tempFilePath) // cleanup anyway
				results <- Result{Task: task, Err: err}

				continue
			}

			if err := p.worker.RemoveFile(tempFilePath); err != nil {
				results <- Result{Task: task, Err: err}

				continue
			}

			results <- Result{
				Task:           task,
				FileType:       dlInfo.Type(),
				FilePath:       task.FilePath,
				OriginalSize:   uplInfo.Size(),
				CompressedSize: dlInfo.Size(),
			}
		}
	}
}

func (p *Pool) generateTempFilePathFor(filePath string) string { return filePath + ".tiny" }

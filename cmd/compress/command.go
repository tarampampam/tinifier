package compress

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"tinifier/cmd/shared"

	log "github.com/sirupsen/logrus"
)

type Command struct {
	shared.WithAPIKey
	Threads int `short:"t" long:"threads" default:"5" description:"Threads processing count"`
}

type (
	task struct {
		num uint32
	}
)

// Follows `flags.Commander` interface (required for commands handling).
func (*Command) Execute(_ []string) error { return nil }

// Handle `serve` command.
func (cmd *Command) Handle(l *log.Logger, _ []string) error {
	var (
		totalSavedKb int64
		wg           = sync.WaitGroup{}
		tasks        = make(chan task)
		oss          = make(chan os.Signal, 1) // channel for operational system signals
	)

	// "subscribe" for system signals
	signal.Notify(oss, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// create context
	ctx, cancel := context.WithCancel(context.Background())

	// fill-up tasks channel
	go cmd.pushTasks(l, ctx, tasks)

	// start workers
	for i := 0; i < cmd.Threads; i++ {
		wg.Add(1)
		go cmd.work(l, ctx, tasks, &totalSavedKb, &wg)
	}

	// listen for system signals in separate goroutine
	go func() {
		s := <-oss
		l.WithField("signal", s).Warn("Caught stopping signal")
		cancel()
	}()

	wg.Wait()

	l.Infof("totalSavedKb = %d", totalSavedKb)

	return nil
}

func (cmd *Command) pushTasks(l *log.Logger, ctx context.Context, tasks chan task) {
	defer close(tasks)

	var input []task

	for i := 0; i < 10; i++ { // @todo: just for a test - generate slice with some tasks
		input = append(input, task{num: uint32(i)})
	}

	for _, task := range input {
		select {
		case <-ctx.Done(): // if `cancel()` executed
			l.Debug("stopping tasks pushing")
			return

		default:
			l.WithField("num", task.num).Debug("push new task", task)
			tasks <- task
		}
	}
}

func (cmd *Command) work(l *log.Logger, ctx context.Context, tasks chan task, totalSavedKb *int64, wg *sync.WaitGroup) {
	defer wg.Done()

	for {
		select {
		case <-ctx.Done(): // if `cancel()` executed
			l.Debug("stopping worker")
			return

		default:
			task, isOpened := <-tasks // get task

			if !isOpened {
				l.Debug("channel is closed")
				return
			}

			time.Sleep(1 * time.Second)
			l.WithField("num", task.num).Debug("worker process task", task)
			atomic.AddInt64(totalSavedKb, 1)
		}
	}
}

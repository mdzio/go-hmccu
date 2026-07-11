package util

import (
	"errors"
	"sync"
	"time"
)

var (
	// ErrCanceled is returned from functions that were canceled prematurely.
	ErrCanceled = errors.New("Canceled")
)

type Context interface {
	// Done returns a channel that is closed when work should be canceled.
	Done() <-chan struct{}

	// IsDone returns true, when work should be canceled.
	IsDone() bool

	// Sleep pauses the execution for the specified duration. If the context is
	// canceled, Sleep returns immediately ErrCanceled.
	Sleep(time.Duration) error
}

type context struct {
	done chan struct{}
}

func (c *context) Done() <-chan struct{} {
	return c.done
}

func (c *context) IsDone() bool {
	// a read on closed channel succeeds immediately
	select {
	case <-c.done:
		return true
	default:
		return false
	}
}

func (c *context) Sleep(d time.Duration) error {
	t := time.NewTimer(d)
	select {
	case <-t.C:
		// time is up
		return nil
	case <-c.done:
		// cancel received, clean up timer
		if !t.Stop() {
			<-t.C
		}
		return ErrCanceled
	}
}

// DaemonFunc runs the specified function as daemon. Calling the returned
// function signals that the daemon function should be canceled and waits until
// the daemon function returns.
func DaemonFunc(f func(Context)) func() {
	// size of 1 does not block go routine, if the function is completed before
	// it is cancelled.
	term := make(chan struct{}, 1)
	ctx := &context{
		done: make(chan struct{}),
	}
	go func() {
		defer func() { term <- struct{}{} }()
		f(ctx)
	}()
	return func() {
		close(ctx.done)
		<-term
	}
}

// DaemonPool runs any number of functions as daemon. All functions can be
// canceled simultaneously with Close.
type DaemonPool struct {
	once sync.Once
	ctx  *context
	wg   sync.WaitGroup
}

// Run runs the specified function as daemon. The specified function can spawn
// additional daemon functions.
func (d *DaemonPool) Run(f func(Context)) {
	d.init()
	d.wg.Add(1)
	go func() {
		defer d.wg.Done()
		f(d.ctx)
	}()
}

// Close signals to all running daemon functions that they should be canceled
// and waits until all daemon functions return.
func (d *DaemonPool) Close() {
	d.init()
	close(d.ctx.done)
	d.wg.Wait()
}

func (d *DaemonPool) init() {
	d.once.Do(func() {
		d.ctx = &context{
			done: make(chan struct{}),
		}
	})
}

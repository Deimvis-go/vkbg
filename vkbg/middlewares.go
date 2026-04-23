package vkbg

import (
	"fmt"
	"runtime/debug"
	"time"

	"go.uber.org/zap"
)

// TODO: move to bgmw

// NOTE: middlewares providing something must be idempotent
// when possible (e.g. middleware providing some value into context)

func WithLogger(zapLg *zap.SugaredLogger) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		zapLg.Infow("Task run started", "task_id", c.TaskId(), "start_ts", c.RunStartTime().Unix())
		err := next(c)
		if err != nil {
			zapLg.Errorw("Task run failed", "task_id", c.TaskId(), "error", err, "stack", c.ErrorStack())
		} else {
			if c.Aborted() {
				zapLg.Infow("Task run aborted", "task_id", c.TaskId(), "reason", c.AbortReason())
			} else {
				zapLg.Infow("Task run completed", "task_id", c.TaskId(), "start_ts", c.RunStartTime().Unix())
			}
		}
		return err
	}
}

func WithRecovery(logger *zap.SugaredLogger) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) (ret error) {
		defer func() {
			if r := recover(); r != nil {
				stack := string(stack(3))
				if err, ok := r.(error); ok {
					logger.Infow("Recovered from panic", "error", err.Error(), "stack", stack)
					ret = err
				} else {
					logger.Infow("Recovered from panic", "error", r, "stack", stack)
					ret = fmt.Errorf("%v", r)
				}
			}
		}()
		return next(c)
	}
}

func WithTimeout(timeout time.Duration) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		c, cancel := c.WithTimeout(timeout)

		panicked := make(chan interface{}, 1)
		finished := make(chan struct{}, 1)
		var err error
		go func() {
			defer func() {
				if p := recover(); p != nil {
					panicked <- fmt.Errorf("%s\nOriginal stack:\n%s", p, string(debug.Stack()))
				}
			}()
			defer cancel()
			err = next(c)
			finished <- struct{}{}
		}()

		select {
		case p := <-panicked:
			panic(p)
		case <-finished:
			return err
		case <-time.After(timeout):
			cancel()
			c.Abort("timeout")
			return nil
		}
	}
}

func WithInterval(interval time.Duration) TaskMiddlewareFn {
	lastRunTime := time.Unix(0, 0)
	firstRun := true
	return func(c *Context, next TaskAction) error {
		if c.Value(IgnoreInterval{}) != nil {
			return next(c)
		}
		var err error
		timePassed := time.Since(lastRunTime)
		if firstRun || timePassed >= interval {
			err = next(c)
			if err != nil {
				return err
			}
			if c.Aborted() {
				return nil
			}
			c.InvokeAt(c.RunStartTime().Add(interval))
			lastRunTime = c.RunStartTime()
			firstRun = false
		} else {
			c.InvokeAt(time.Now().Add(interval - timePassed))
			err = nil
		}
		return err
	}
}

func WithIntervalFn(readyFn func(*Context) (bool, error)) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		if c.Value(IgnoreInterval{}) != nil {
			return next(c)
		}
		ready, err := readyFn(c)
		if err != nil {
			return err
		}
		if !ready {
			c.Abort("run was called too early")
			return nil
		}
		return next(c)
	}
}

type ContextFlag struct{}
type IgnoreInterval ContextFlag

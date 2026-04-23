package vkbg

import (
	"context"
	"time"
)

func NewContext(ctx context.Context, taskId string, runId string, runStartTime time.Time) *Context {
	return &Context{Context: ctx, taskId: taskId, runId: runId, runStartTime: runStartTime}
}

type Context struct {
	context.Context

	taskId       string
	runId        string
	runStartTime time.Time

	aborted        bool
	abortReason    string
	errorStack     string
	nextInvokeTime *time.Time
}

func (c *Context) TaskId() string {
	return c.taskId
}

func (c *Context) RunId() string {
	return c.runId
}

// StartTime returns the time when current run started.
func (c *Context) RunStartTime() time.Time {
	return c.runStartTime
}

func (c *Context) InvokeAt(t time.Time) {
	c.nextInvokeTime = &t
}

func (c *Context) Abort(reason string) {
	c.aborted = true
	c.abortReason = reason
}

func (c *Context) Aborted() bool {
	return c.aborted
}

func (c *Context) AbortReason() string {
	return c.abortReason
}

func (c *Context) ErrorStack() string {
	return c.errorStack
}

func (c *Context) SaveErrorStack() {
	c.errorStack = string(stack(3))
}

func (c *Context) WithTimeout(timeout time.Duration) (*Context, context.CancelFunc) {
	ccopy := c.clone()
	var cancel context.CancelFunc
	ccopy.Context, cancel = context.WithTimeout(ccopy.Context, timeout)
	return ccopy, cancel
}

func (c *Context) clone() *Context {
	ccopy := Context{
		Context:        context.WithValue(c.Context, contextParentCtxKey{}, c),
		taskId:         c.taskId,
		runId:          c.runId,
		runStartTime:   c.runStartTime,
		aborted:        c.aborted,
		abortReason:    c.abortReason,
		nextInvokeTime: c.nextInvokeTime,
	}
	return &ccopy
}

type contextParentCtxKey struct{}

package vkbg

import "fmt"

func NewTaskCallStack(mws []TaskMiddlewareFn, action TaskAction) *TaskCallStack {
	tcs := &TaskCallStack{
		mws:    mws,
		action: action,
		i:      -1,
	}
	return tcs
}

type TaskCallStack struct {
	mws    []TaskMiddlewareFn
	action TaskAction

	i int
}

// Invoke unwraps call stack.
// It automatically calls Context's Abort() method if middleware didn't,
// so Aborted() method will reflect whether task was actually performed.
func (tcs *TaskCallStack) Invoke(c *Context) error {
	tcs.i = -1
	return tcs.Next(c)
}

func (tcs *TaskCallStack) Next(c *Context) error {
	var err error
	tcs.i++
	if tcs.i < len(tcs.mws) {
		err = tcs.mws[tcs.i](c, tcs.Next)
	} else {
		err = tcs.action(c)
	}
	if tcs.i < len(tcs.mws) && !c.Aborted() {
		abortMwFnName := GetFnName(tcs.mws[tcs.i])
		c.Abort(fmt.Sprintf(`middleware #%d aborted unexpectedly (%s)`, tcs.i, abortMwFnName))
	}
	return err
}

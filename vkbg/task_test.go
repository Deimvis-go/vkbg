package vkbg

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTask(t *testing.T) {
	id := "task_id"
	mw := func(c *Context, next TaskAction) error { return nil }
	mws := []TaskMiddlewareFn{mw}
	actionCalled := false
	action := func(c *Context) error {
		actionCalled = true
		return nil
	}
	task := NewTask(id, action, mws...)
	require.Equal(t, id, task.Id())
	require.Equal(t, len(mws), len(task.Middlewares()))
	err := task.Run(nil)
	require.Nil(t, err)
	require.True(t, true, actionCalled)
}

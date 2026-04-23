package vkbg

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

func NewSimpleScheduler(mws ...TaskMiddlewareFn) *SimpleScheduler {
	return &SimpleScheduler{middlewares: mws, tasks: make(map[string]Task), tasksInvokeTime: make(map[string]time.Time)}
}

type SimpleScheduler struct {
	middlewares []TaskMiddlewareFn
	tasks       map[string]Task

	tasksInvokeTime map[string]time.Time
}

func (ss *SimpleScheduler) Schedule(t Task) error {
	if _, ok := ss.tasks[t.Id()]; ok {
		return fmt.Errorf("task with id=`%s` already exists", t.Id())
	}
	ss.tasks[t.Id()] = t
	return nil
}

func (ss *SimpleScheduler) MustSchedule(t Task) {
	Must0(ss.Schedule(t))
}

func (ss *SimpleScheduler) LoopInBackground() context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())
	go ss.Loop(ctx)
	return cancel
}

func (ss *SimpleScheduler) Loop(ctx context.Context) {
	wg := &sync.WaitGroup{}
	for id := range ss.tasks {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			ss.LoopTask(ctx, id)
		}(id)
	}
	wg.Wait()
}

func (ss *SimpleScheduler) LoopTask(ctx context.Context, id string) {
	ticker := time.NewTicker(5 * time.Second)
	for {
		MustTrue(HasKey(ss.tasks, id))
		invokeTime, ok := ss.tasksInvokeTime[id]
		if !ok || time.Since(invokeTime) > 0 {
			// TODO: support max_parallel > 1
			_ = ss.runTask(ctx, id)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (ss *SimpleScheduler) GetTask(id string) (Task, bool) {
	task, ok := ss.tasks[id]
	return task, ok
}

func (ss *SimpleScheduler) HasTask(id string) bool {
	_, ok := ss.tasks[id]
	return ok
}

func (ss *SimpleScheduler) RunTask(ctx context.Context, id string) error {
	return ss.runTask(ctx, id)
}

// runTask starts the task wrapped with all corresponding middlewares.
func (ss *SimpleScheduler) runTask(ctx context.Context, id string) error {
	t, ok := ss.tasks[id]
	MustTrue(ok)

	c := NewContext(ctx, id, uuid.New().String(), time.Now())
	mws := append(ss.middlewares, t.Middlewares()...)
	tcs := NewTaskCallStack(mws, t.Run)
	err := tcs.Invoke(c)
	if c.nextInvokeTime != nil {
		ss.tasksInvokeTime[id] = *c.nextInvokeTime
	}
	return err
}

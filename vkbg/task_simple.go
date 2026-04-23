package vkbg

import "time"

func NewSimpleTask(id string, action TaskAction, interval time.Duration, mws ...TaskMiddlewareFn) Task {
	mws = append(mws, WithInterval(interval))
	return NewTask(id, action, mws...)
}

package vkbg

import (
	"fmt"
)

type IdempotencyVerifier interface {
	Verify(c *Context, idempotencyKey string) (bool, error)
	Submit(c *Context, idempotencyKey string) error
	Rollback(c *Context, idempotencyKey string) error
}

// onPanicRollback runs rollback if code is panicking and re-panics.
// It must be called directly via defer.
func onPanicRollback(rollback func() error) {
	if r := recover(); r != nil {
		err := rollback()
		if err != nil {
			switch rr := r.(type) {
			case error:
				r = fmt.Errorf("%w\nwDuring handling of the above exception, another exception occurred:\n%w", rr, err)
			default:
				r = fmt.Errorf("panic(%v)\nwDuring handling of the above exception, another exception occurred:\n%w", rr, err)
			}
		}
		panic(r)
	}
}

func WithIdempotency(key string, iv IdempotencyVerifier) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		ok, err := iv.Verify(c, key)
		if err != nil {
			return err
		}
		if !ok {
			if !c.Aborted() {
				c.Abort("idempotency verification failed")
			}
			return nil
		}
		defer onPanicRollback(func() error { return iv.Rollback(c, key) })
		err = next(c)
		if err != nil {
			rErr := iv.Rollback(c, key)
			if rErr != nil {
				err = fmt.Errorf("%w\nwDuring handling of the above exception, another exception occurred:\n%w", err, rErr)
			}
			return err
		}
		if c.Aborted() {
			rErr := iv.Rollback(c, key)
			if rErr != nil {
				return fmt.Errorf("failed to rollback idempotency mechanism: %w", rErr)
			}
			return nil
		}
		err = iv.Submit(c, key)
		return err
	}
}

func WithIdempotencyFn(keyFn func(*Context) (string, error), iv IdempotencyVerifier) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		key, err := keyFn(c)
		if err != nil {
			return err
		}
		return WithIdempotency(key, iv)(c, next)
	}
}

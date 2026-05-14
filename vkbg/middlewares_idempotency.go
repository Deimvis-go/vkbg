package vkbg

import (
	"fmt"

	"github.com/Deimvis/go-ext/go1.25/ext"
)

type IdempotencyVerifier interface {
	Verify(c *Context, idempotencyKey string) (bool, error)
	Submit(c *Context, idempotencyKey string) error
	Rollback(c *Context, idempotencyKey string) error
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
		defer ext.OnPanicX(func(any) error { return iv.Rollback(c, key) })
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

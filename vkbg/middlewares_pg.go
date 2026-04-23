package vkbg

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func WithPGConn(pool *pgxpool.Pool) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		_, ok := c.Value(PgConnKey{}).(*pgxpool.Conn)
		if ok {
			return next(c)
		}

		conn, err := pool.Acquire(c)
		if err != nil {
			return err
		}
		defer conn.Release()
		c.Context = context.WithValue(c.Context, PgConnKey{}, conn)
		return next(c)
	}
}

// WithLoggerPG is a logging middleware
// that inserts logs in Postgres table.
// WithPGConn middleware is required to be called earlier.
// TODO: support more flexible configuration.
func WithLoggerPG(pool *pgxpool.Pool) TaskMiddlewareFn {
	Must(pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS "task_run_log" ("run_id" TEXT, "task_id" TEXT, "action_type" TEXT, "ts" BIGINT, "context" JSONB)`))
	return func(c *Context, next TaskAction) error {
		conn, ok := c.Value(PgConnKey{}).(*pgxpool.Conn)
		if !ok {
			prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGConn")
			return fmt.Errorf("failed to extract pg_conn from context: %s", prompt)
		}
		log := func(l *TaskRunLog) {
			_, err := conn.Exec(c, `INSERT INTO "task_run_log" ("run_id", "task_id", "action_type", "ts", "context") VALUES ($1, $2, $3, $4, $5)`, UnwrapStruct(l)...)
			MustNil(err, "failed to log into pg: %s", err)
		}

		log(NewTaskRunLog(c, RunEventStarted))

		err := next(c)

		if err != nil {
			log(NewTaskRunLog(c, RunEventFailed, "error", err.Error()))
			return err
		}
		if c.Aborted() {
			log(NewTaskRunLog(c, RunEventAborted, "abort_reason", c.AbortReason()))
			return nil
		}
		log(NewTaskRunLog(c, RunEventCompleted))
		return nil
	}
}

// WithPGLastRunStartTs sets last_run_start_ts to ctx.
// It will update last_run_start_ts only if task was completed
// (wasn't aborted and finished without an error).
// WithPGConn middleware is required to be called earlier.
func WithPGLastRunStartTs(pool *pgxpool.Pool) TaskMiddlewareFn {
	Must(pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS "task_last_run" ("task_id" TEXT PRIMARY KEY, "start_ts" BIGINT)`))
	return func(c *Context, next TaskAction) error {
		conn, ok := c.Value(PgConnKey{}).(*pgxpool.Conn)
		if !ok {
			prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGConn")
			return fmt.Errorf("failed to extract pg_conn from context: %s", prompt)
		}
		var lastRunStartTs int64
		err := conn.QueryRow(c, `SELECT "start_ts" FROM "task_last_run" WHERE "task_id" = $1`, c.TaskId()).Scan(&lastRunStartTs)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				lastRunStartTs = NoLastRun
			} else {
				return err
			}
		}

		c.Context = context.WithValue(c.Context, PGLastRunStartTsCtxKey{}, lastRunStartTs)
		err = next(c)
		if err != nil {
			return err
		}
		if c.Aborted() {
			return nil
		}

		_, err = conn.Exec(c, `INSERT INTO "task_last_run" ("task_id", "start_ts") VALUES ($1, $2) ON CONFLICT ("task_id") DO UPDATE SET "start_ts" = GREATEST("task_last_run"."start_ts", $2)`, c.TaskId(), c.RunStartTime().Unix())
		return err
	}
}

// WithIntervalPGSinceLastRunStartTs is an itnerval middleware
// that is based on Postgres database and last_run_start_ts.
// WithPGLastRunStartTs middleware is required to be called earlier.
func WithIntervalPGSinceLastRunStartTs(interval time.Duration) TaskMiddlewareFn {
	return func(c *Context, next TaskAction) error {
		if c.Value(IgnoreInterval{}) != nil {
			return next(c)
		}

		lastRunStartTs, ok := c.Value(PGLastRunStartTsCtxKey{}).(int64)
		if !ok {
			prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGLastRunStartTs")
			return fmt.Errorf("failed to extract last_run_start_ts from context: %s", prompt)
		}

		timePassed := time.Since(time.Unix(lastRunStartTs, 0))
		if lastRunStartTs != NoLastRun && timePassed < interval {
			timeLeft := interval - timePassed
			c.Abort(fmt.Sprintf("run was called too early (time left: %s)", timeLeft))
			c.InvokeAt(time.Now().Add(timeLeft))
			return nil
		}

		err := next(c)
		if err != nil {
			return err
		}
		if c.Aborted() {
			return nil
		}
		c.InvokeAt(c.RunStartTime().Add(interval))
		return nil
	}
}

// IdempotencyKeyFnPGLastRunStartTs is an idempotency key function
// that is based on task_id + last_run_start_ts.
// WithPGLastRunStartTs middleware is requried to be called earlier.
func IdempotencyKeyFnPGLastRunStartTs(c *Context) (string, error) {
	lastRunStartTs, ok := c.Value(PGLastRunStartTsCtxKey{}).(int64)
	if !ok {
		prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGLastRunStartTs")
		return "", fmt.Errorf("failed to extract last_run_start_ts from context: %s", prompt)
	}
	return fmt.Sprintf("%s_%d", c.TaskId(), lastRunStartTs), nil
}

func NewIdempotencyVerifierPG(idempotencyTimeout time.Duration, pool *pgxpool.Pool) *IdempotencyVerifierPG {
	Must(pool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS "task_run_idempotency" ("run_id" TEXT PRIMARY KEY, "task_id" TEXT, "idempotency_key" TEXT, "timeout_ts" BIGINT, "completed" BOOLEAN)`))
	return &IdempotencyVerifierPG{idempotencyTimeout: idempotencyTimeout}
}

// IdempotencyVerifierPG is an implementation for idemoptency verification
// using Postgres database.
// WithPGConn middleware is required to be called earlier.
type IdempotencyVerifierPG struct {
	idempotencyTimeout time.Duration
}

func (iv *IdempotencyVerifierPG) Verify(c *Context, idempotencyKey string) (bool, error) {
	conn, ok := c.Value(PgConnKey{}).(*pgxpool.Conn)
	if !ok {
		prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGConn")
		return false, fmt.Errorf("failed to extract pg_conn from context: %s", prompt)
	}

	tx := Must(conn.Begin(c))
	defer tx.Rollback(c)
	Must0(advLock(c, tx, idempotencyKey))

	var maxTimeoutTs *int64
	var anyCompleted *bool
	err := tx.QueryRow(c, `SELECT MAX("timeout_ts") as "max_timeout_ts", bool_or("completed") as "any_completed" FROM "task_run_idempotency" WHERE "task_id" = $1 AND "idempotency_key" = $2`, c.TaskId(), idempotencyKey).Scan(&maxTimeoutTs, &anyCompleted)
	if err != nil && !IsNoRows(err) {
		return false, err
	}
	Invariant(err == nil || IsNoRows(err))
	hasRows := (err == nil && maxTimeoutTs != nil && anyCompleted != nil)
	if hasRows {
		if *anyCompleted {
			// rerun immediately - either start next iteration or find out time to sleep before next iteration,
			// because current iteration is already completed
			c.Abort("task is already completed")
			c.InvokeAt(time.Unix(0, 0))
			return false, nil
		} else if time.Now().Unix() < *maxTimeoutTs {
			c.Abort("task is currently running")
			c.InvokeAt(time.Unix(*maxTimeoutTs, 0))
			return false, nil
		}
	}

	// no valid reason to stop (no one started yet or one who started reached his timeout)
	timeoutTs := time.Now().Add(iv.idempotencyTimeout).Unix()
	_, err = tx.Exec(c, `INSERT INTO "task_run_idempotency" ("run_id", "task_id", "idempotency_key", "timeout_ts", "completed") VALUES ($1, $2, $3, $4, $5)`, c.RunId(), c.TaskId(), idempotencyKey, timeoutTs, false)
	if err != nil {
		return false, err
	}
	tx.Commit(c)
	return true, nil
}

func (iv *IdempotencyVerifierPG) Submit(c *Context, idempotencyKey string) error {
	conn, ok := c.Value(PgConnKey{}).(*pgxpool.Conn)
	if !ok {
		prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGConn")
		return fmt.Errorf("failed to extract pg_conn from context: %s", prompt)
	}
	// TODO: last run ts mw should provide an API to other mws to perform extra queries in a single transaction and avoid inconsistencies
	_, err := conn.Exec(c, `UPDATE "task_run_idempotency" SET "completed" = true WHERE "run_id" = $1 AND "task_id" = $2 AND "idempotency_key" = $3`, c.RunId(), c.TaskId(), idempotencyKey)
	return err
}

func (iv *IdempotencyVerifierPG) Rollback(c *Context, idempotencyKey string) error {
	conn, ok := c.Value(PgConnKey{}).(*pgxpool.Conn)
	if !ok {
		prompt := fmt.Sprintf(enableMiddlewarePromptFormat, "WithPGConn")
		return fmt.Errorf("failed to extract pg_conn from context: %s", prompt)
	}
	_, err := conn.Exec(c, `DELETE FROM "task_run_idempotency" WHERE "run_id" = $1 AND "task_id" = $2 AND "idempotency_key" = $3`, c.RunId(), c.TaskId(), idempotencyKey)
	return err
}

func advLock(ctx context.Context, tx pgx.Tx, key string) error {
	_, err := tx.Exec(ctx, "SELECT pg_advisory_xact_lock( hashtext($1) );", key)
	return err
}

func IsNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

var NoLastRun int64 = -1

type PgConnKey struct{}
type PGLastRunStartTsCtxKey struct{}

var enableMiddlewarePromptFormat = "please, enable %s middleware and re-check middleware order"

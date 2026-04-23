package vkbg

import (
	"fmt"
	"time"

	"github.com/go-playground/validator/v10"
)

type TaskRunLog struct {
	RunId   string `validate:"required"`
	TaskId  string `validate:"required"`
	Event   string `validate:"oneof=started failed aborted completed"`
	Ts      int64  `validate:"gt=0"`
	Context map[string]interface{}
}

func NewTaskRunLog(c *Context, event string, keysAndValues ...interface{}) *TaskRunLog {
	if len(keysAndValues)%2 != 0 {
		panic(fmt.Errorf("invalid number of arguments for key-value pairs (%d)", len(keysAndValues)))
	}
	ctx := make(map[string]interface{})
	for i := 0; i < len(keysAndValues); i += 2 {
		key := keysAndValues[i]
		value := keysAndValues[i+1]
		keyStr, ok := key.(string)
		if !ok {
			panic(fmt.Errorf("key `%v` has type %T, but string expected", key, key))
		}
		ctx[keyStr] = value
	}
	log := &TaskRunLog{
		RunId:   c.RunId(),
		TaskId:  c.TaskId(),
		Event:   event,
		Ts:      time.Now().Unix(),
		Context: ctx,
	}
	return log
}

var (
	RunEventStarted   = "started"
	RunEventFailed    = "failed"
	RunEventAborted   = "aborted"
	RunEventCompleted = "completed"
)

func Validate(obj interface{}) error {
	return val.Struct(obj)
}

var val = validator.New(validator.WithRequiredStructEnabled())

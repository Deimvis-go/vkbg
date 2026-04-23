package vkbg

import (
	"fmt"
	"log"
	"os"
	"reflect"
	"strings"
)

// TODO: use go-ext

func HasKey[K comparable, V any](m map[K]V, k K) bool {
	_, ok := m[k]
	return ok
}

// https://github.com/stretchr/testify/blob/v1.9.0/assert/assertions.go#L280
func FormatMsgAndArgs(msgAndArgs ...interface{}) string {
	if len(msgAndArgs) == 0 || msgAndArgs == nil {
		return ""
	}
	if len(msgAndArgs) == 1 {
		msg := msgAndArgs[0]
		if msgAsStr, ok := msg.(string); ok {
			return msgAsStr
		}
		return fmt.Sprintf("%+v", msg)
	}
	if len(msgAndArgs) > 1 {
		return fmt.Sprintf(msgAndArgs[0].(string), msgAndArgs[1:]...)
	}
	return ""
}

func Must[T any](v T, err error) T {
	if err != nil {
		panic(err)
	}
	return v
}

func Must0(err error) {
	if err != nil {
		panic(err)
	}
}

func MustTrue(v bool, msgAndArgs ...interface{}) {
	if !v {
		msg := FormatMsgAndArgs(msgAndArgs...)
		panic(msg)
	}
}

func MustNil(e error, msgAndArgs ...interface{}) {
	if e != nil {
		msg := FormatMsgAndArgs(msgAndArgs...)
		panic(msg)
	}
}

func Contains[T comparable](s []T, v T) bool {
	for _, val := range s {
		if val == v {
			return true
		}
	}
	return false
}

func IsDebugEnv() bool {
	trueOpts := []string{"1", "true"}
	return Contains(trueOpts, strings.ToLower(os.Getenv("DEBUG")))
}

// Invariant fails fast in debug environment and does nothing otherwise
func Invariant(v bool, msgAndArgs ...interface{}) {
	if !IsDebugEnv() {
		return
	}
	if !v {
		msg := FormatMsgAndArgs(msgAndArgs...)
		errMsg := fmt.Sprintf("invariant failed: %s", msg)
		log.Fatalln(errMsg)
	}
}

// UnwrapStruct returns slice of values of each field.
// It unwraps embedded fields and returns its subfields
// ignoring embedded fields themselves.
// It ignores unexported (lowercase) fields.
// The order of returned struct field values is guaranteed
// to match the order in which fields were declared in the struct definition.
func UnwrapStruct(s interface{}) []interface{} {
	if s == nil {
		return nil
	}
	v := reflect.ValueOf(s)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	visibleFields := reflect.VisibleFields(v.Type())
	var fields []interface{}
	for _, f := range visibleFields {
		if f.Anonymous || !f.IsExported() {
			continue
		}
		fields = append(fields, v.FieldByIndex(f.Index).Interface())
	}
	return fields
}

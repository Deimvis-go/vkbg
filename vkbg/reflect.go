package vkbg

import (
	"reflect"
	"runtime"
	"strings"
)

func GetFnName(fn interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
}

func GetBaseFnName(fn interface{}) string {
	name := GetFnName(fn)
	parts := strings.Split(name, ".")
	return parts[len(parts)-1]
}

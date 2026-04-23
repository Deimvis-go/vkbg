package vkbg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHasKey(t *testing.T) {
	testCases := []struct {
		m        map[string]int
		key      string
		expected bool
	}{
		{
			make(map[string]int),
			"something",
			false,
		},
		{
			map[string]int{"a": 1, "b": 2},
			"a",
			true,
		},
		{
			map[string]int{"a": 1, "b": 2},
			"b",
			true,
		},
		{
			map[string]int{"a": 1, "b": 2},
			"c",
			false,
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			require.Equal(t, tc.expected, HasKey(tc.m, tc.key))
		})
	}
}

func TestFormatMsgAndArgs(t *testing.T) {
	type args = []interface{}
	testCases := []struct {
		msgAndArgs args
		expected   string
	}{
		{
			args{},
			"",
		},
		{
			args{""},
			"",
		},
		{
			args{"kek"},
			"kek",
		},
		{
			args{"%s", "kek"},
			"kek",
		},
		{
			args{"%d + %s = ERROR", 2, "2"},
			"2 + 2 = ERROR",
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual := FormatMsgAndArgs(tc.msgAndArgs...)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func TestUnwrapStruct(t *testing.T) {
	testCases := []struct {
		title    string
		obj      interface{}
		expected []interface{}
	}{
		{
			"nil",
			nil,
			nil,
		},
		{
			"plain struct",
			a{Key: 42, Value: "hello"},
			[]interface{}{
				42,
				"hello",
			},
		},
		{
			"pointer to plain struct",
			&a{Key: 42, Value: "hello"},
			[]interface{}{
				42,
				"hello",
			},
		},
		{
			"embedded struct",
			b{a: a{Key: 42, Value: "hello"}, Other: 0},
			[]interface{}{
				42,
				"hello",
				0,
			},
		},
		{
			"exported struct",
			c{A: a{Key: 42, Value: "hello"}, Other: 0},
			[]interface{}{
				a{Key: 42, Value: "hello"},
				0,
			},
		},
		{
			"unexported struct",
			d{a: a{Key: 42, Value: "hello"}, Other: 0},
			[]interface{}{
				0,
			},
		},
		{
			"recursively embedded struct",
			e{b: b{a: a{Key: 42, Value: "hello"}, Other: 0}, Other2: 1},
			[]interface{}{
				42,
				"hello",
				0,
				1,
			},
		},
	}
	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			actual := UnwrapStruct(tc.obj)
			require.Equal(t, tc.expected, actual)
		})
	}
}

type a struct {
	Key   int
	Value string
}

type b struct {
	a
	Other int
}

type c struct {
	A     a
	Other int
}

type d struct {
	a     a
	Other int
}

type e struct {
	b
	Other2 int
}

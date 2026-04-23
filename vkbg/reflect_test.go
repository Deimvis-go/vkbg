package vkbg

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetBaseFnName(t *testing.T) {
	testCases := []struct {
		title    string
		fn       interface{}
		expected string
	}{
		{
			"foo",
			foo,
			"foo",
		},
		{
			"bar",
			bar,
			"bar",
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("%s (%d)", tc.title, i), func(t *testing.T) {
			actual := GetBaseFnName(tc.fn)
			require.Equal(t, tc.expected, actual)
		})
	}
}

func foo()          {}
func bar(int) error { return nil }

package main

import (
	"testing"
)

func TestResults(t *testing.T) {
	var cases = []struct {
		expected int
		input    string
	}{
		{
			expected: 265,
			input:    "2 3 + 56 * 7 8 + -",
		},

		{
			expected: -27795,
			input:    "3 1 2 + 3 4 5 + * + 10 34 * - 45 + 9000 - *",
		},

		{
			expected: 36598,
			input:    "3 1 2 3 4 5 10 34 -45 90 - * + + * + * - +",
		},

		{
			expected: 24,
			input:    "4 2 * 3 *",
		},

		{
			expected: 6,
			input:    "4 rrrr tt 5+ 4t4t 2 %%% 3 ))) *",
		},
	}
	for _, item := range cases {
		result, err := Calculate(item.input)
		if !(result == item.expected) {
			t.Error("expected", item.expected, "have", result)
		} else if err != nil {
			t.Error("unexpected error")
		}
	}
}
func TestErrors(t *testing.T) {
	var cases = []struct {
		input string
	}{
		{
			input: "+",
		},

		{
			input: "*+-+ rrrrr4= - - -",
		},

		{
			input: "3 4 + 5 6 - ",
		},

		{
			input: "3 4 + 5 6 - - -",
		},

		{
			input: "+ 4 5 + 4",
		},
	}
	for _, item := range cases {
		_, err := Calculate(item.input)

		if err == nil {
			t.Error("expected an error")
		}
	}
}

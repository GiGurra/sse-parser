package sse_parser

import (
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	// test cases
	tests := []struct {
		name     string
		input    string
		expected []Message
	}{
		{
			name: "basic-case",
			input: "event: message\n" +
				"data: hello [END]\n\n" +
				"event: message\n" +
				"data: world [END]\n\n",
			expected: []Message{
				{
					Event: "message",
					Data:  "hello [END]",
				},
				{
					Event: "message",
					Data:  "world [END]",
				},
			},
		},
		//{
		//	name: "case2",
		//},
		//{
		//	name: "case3",
		//},
	}

	parser := NewParser(func(dataBytes string) bool {
		return strings.HasSuffix(dataBytes, "[END]")
	})

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.Reset()
			result := parser.Add(tt.input)
			result = append(result, parser.Finish()...)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d messages, got %d", len(tt.expected), len(result))
			}
		})
	}
}

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
			input: "event:message\n" +
				"data:hello [END]\n\n" +
				"event:message\n" +
				"data:world [END]\n\n",
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
		{
			name: "truncated end",
			input: "event:message\n" +
				"data:hello [END]\n\n" +
				"event:message\n" +
				"data:worl\n\n",
			expected: []Message{
				{
					Event: "message",
					Data:  "hello [END]",
				},
			},
		},
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

			for i, msg := range result {
				if msg.Event != tt.expected[i].Event {
					t.Errorf("expected event %s, got %s", tt.expected[i].Event, msg.Event)
				}
				if msg.Data != tt.expected[i].Data {
					t.Errorf("expected data %s, got %s", tt.expected[i].Data, msg.Data)
				}
			}
		})
	}
}

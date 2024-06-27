package sse_parser

import (
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	// test cases
	tests := []struct {
		name     string
		inputs   []string
		expected []Message
	}{
		{
			name: "basic-case",
			inputs: []string{"event:message\n" +
				"data:hello [END]\n\n" +
				"event:message\n" +
				"data:world [END]",
			},
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
			name: "basic-case-with-extra-newlines",
			inputs: []string{"event:message\n" +
				"data:hello [END]\n\n" +
				"event:message\n" +
				"data:world [END]\n\n\n\n\n\n",
			},
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
			inputs: []string{"event:message\n" +
				"data:hello [END]\n\n" +
				"event:message\n" +
				"data:worl",
			},
			expected: []Message{
				{
					Event: "message",
					Data:  "hello [END]",
				},
			},
		},
		{
			name: "truncated end fixed",
			inputs: []string{"event:message\n" +
				"data:hello [END]\n\n" +
				"event:message\n" +
				"data:worl",
				"d [END]",
			},
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
	}

	parser := NewParser(func(dataBytes string) bool {
		return strings.HasSuffix(dataBytes, "[END]")
	})

	// run tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser.Reset()
			result := []Message{}
			for _, input := range tt.inputs {
				result = append(result, parser.Add(input)...)
			}
			result = append(result, parser.Finish()...)

			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d messages, got %d", len(tt.expected), len(result))
			}

			for i, msg := range result {
				if msg.Event != tt.expected[i].Event {
					t.Fatalf("expected event %s, got %s", tt.expected[i].Event, msg.Event)
				}
				if msg.Data != tt.expected[i].Data {
					t.Fatalf("expected data %s, got %s", tt.expected[i].Data, msg.Data)
				}
			}
		})
	}
}

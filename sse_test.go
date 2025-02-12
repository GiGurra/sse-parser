package sse_parser

import (
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestParserChan(t *testing.T) {
	inputBuffer := "event:message\ndata:hello [END]\n\n"
	parser := NewParser(func(dataBytes string) bool {
		return strings.HasSuffix(dataBytes, "[END]")
	})
	msgChan := parser.Stream(strings.NewReader(inputBuffer), 100)
	recvd := []Message{}
	for msg := range msgChan {
		recvd = append(recvd, msg)
	}

	if len(recvd) != 1 {
		t.Fatalf("expected 1 message, got %d", len(recvd))
	}

	if recvd[0].Event != "message" {
		t.Fatalf("expected event message, got %s", recvd[0].Event)
	}

	if recvd[0].Data != "hello [END]" {
		t.Fatalf("expected data hello [END], got %s", recvd[0].Data)
	}
}

func TestLongStream(t *testing.T) {
	nMessages := 1000
	builder := strings.Builder{}
	for i := 0; i < nMessages; i++ {
		builder.WriteString("event:message\ndata:hello " + strconv.Itoa(i) + " [END]\n\n")
	}

	parser := NewParser(func(dataBytes string) bool {
		return strings.HasSuffix(dataBytes, "[END]")
	})
	msgChan := parser.Stream(strings.NewReader(builder.String()), 100)

	recvd := []Message{}
	for msg := range msgChan {
		recvd = append(recvd, msg)
	}

	fmt.Printf("received %d messages\n", len(recvd))
	if len(recvd) != nMessages {
		t.Fatalf("expected %d messages, got %d", nMessages, len(recvd))
	}
}

type ReaderImpl struct {
	byteChan chan byte
}

func (r *ReaderImpl) Read(p []byte) (int, error) {
	n := 0
	for i := range p {
		select {
		case b := <-r.byteChan:
			p[i] = b
			n++
		default:
			// no more bytes available
			return n, nil
		}
	}
	return n, nil
}

func TestGradualStream(t *testing.T) {
	nMessages := 1000
	inputStream := make(chan byte, 1024)
	reader := &ReaderImpl{
		byteChan: inputStream,
	}
	parser := NewParser(func(dataBytes string) bool {
		return strings.HasSuffix(dataBytes, "[END]")
	})
	nReceved := 0
	messageStream := parser.Stream(reader, 100)
	for i := 0; i < nMessages; i++ {
		slog.Info(fmt.Sprintf("sending message %d", i))
		message := "event:message\ndata:hello " + strconv.Itoa(i) + " [END]\n\n"
		for _, b := range message {
			inputStream <- byte(b)
		}
		select {
		case msg := <-messageStream:
			nReceved++
			slog.Info(fmt.Sprintf("received message %d", i))
			if msg.Data != "hello "+strconv.Itoa(i)+" [END]" {
				t.Fatalf("expected data hello %d [END], got %s", i, msg.Data)
			}
		//timeout case
		case <-time.After(1 * time.Second):
			t.Fatalf("timeout")
		}
	}
	close(inputStream)

	if nReceved != nMessages {
		t.Fatalf("expected %d messages, got %d", nMessages, nReceved)
	}
}

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
		{
			name: "find valid messages in bad data",
			inputs: []string{
				// 2 ok messages
				"event:message\n" +
					"data:hello [END]\n\n" +
					"event:message\n" +
					"data:worl",
				"d [END]",
				// bad message
				"event:message\n" +
					"data:world ",
				"\n\n",
				// ok message (just data)
				"data:justdata [END]",
				"\n\n",
				// bad message
				"garbage\n",
				"\n\n",
				// 1 ok message
				"event:message\n",
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
				{
					Event: "",
					Data:  "justdata [END]",
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

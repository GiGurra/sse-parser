package sse_parser

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type Parser struct {
	buffer         strings.Builder
	dataCompleteFn func(dataBytes string) bool
}

func NewParser(completeFn func(dataBytes string) bool) *Parser {
	return &Parser{
		dataCompleteFn: completeFn,
	}
}

type Message struct {
	Event string
	Data  string
}

func (p *Parser) Reset() {
	p.buffer.Reset()
}

func (p *Parser) doParseSingle(all string) (Message, bool) {
	lineInPart := strings.Split(all, "\n")

	if len(lineInPart) == 0 {
		return Message{}, false
	}

	firstLine := lineInPart[0]
	event := ""
	data := ""
	if strings.HasPrefix(firstLine, "event:") {
		// normal
		if len(lineInPart) < 2 {
			return Message{}, false
		}

		dataLines := strings.Join(lineInPart[1:], "\n")

		// Can either start with "event:" or "data:"
		if !strings.HasPrefix(dataLines, "data:") {
			return Message{}, false
		}

		event = strings.TrimPrefix(firstLine, "event:")
		data = strings.TrimPrefix(dataLines, "data:")

	} else if strings.HasPrefix(firstLine, "data:") {
		// just data (damn google "sse")
		dataLines := strings.Join(lineInPart[0:], "\n")
		data = strings.TrimPrefix(dataLines, "data:")
	} else {
		return Message{}, false
	}

	if p.dataCompleteFn == nil || p.dataCompleteFn(data) {
		return Message{
			Event: event,
			Data:  data,
		}, true
	} else {
		return Message{}, false
	}
}

func (p *Parser) doParseAll(isFinish bool) []Message {
	allInBuffer := p.buffer.String()

	parts := strings.Split(allInBuffer, "\n\n")
	p.buffer.Reset()

	// if all lines are empty, we can just return
	stringsAllEmpty := func(lines []string) bool {
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				return false
			}
		}
		return true
	}
	if stringsAllEmpty(parts) {
		return []Message{}
	}

	// All parts except the last must be a valid message.
	// The last part may or may not yet be complete
	messages := []Message{}
	for i, part := range parts {

		if i+1 == len(parts) {
			// we are at the last part, and we are not forcing the last line
			// so we can skip this part
			continue
		}

		message, ok := p.doParseSingle(part)
		if !ok {
			slog.Error(fmt.Sprintf("Invalid message: %s, skipping", part))
			continue
		}

		messages = append(messages, message)
	}

	lastPart := parts[len(parts)-1]
	if stringsAllEmpty([]string{lastPart}) {
		return messages
	}

	if isFinish || p.dataCompleteFn != nil {
		lastMessage, ok := p.doParseSingle(lastPart)
		if !ok {
			if isFinish {
				slog.Error(fmt.Sprintf("Invalid last message piece: %s, skipping", lastPart))
			} else {
				p.buffer.WriteString(lastPart) // put it back to parse later when we have more data
			}
			return messages
		}
		return append(messages, lastMessage)
	}

	return messages
}

func (p *Parser) Add(data string) []Message {

	// replace all \r\n with \n
	data = strings.ReplaceAll(data, "\r\n", "\n")
	p.buffer.WriteString(data)

	return p.doParseAll(false)
}

func (p *Parser) Finish() []Message {
	return p.doParseAll(true)
}

type Writer struct {
	parser *Parser
	ch     chan Message
}

var _ io.Writer = &Writer{}

func (w *Writer) Write(p []byte) (n int, err error) {
	messages := w.parser.Add(string(p))
	for _, message := range messages {
		w.ch <- message
	}
	return len(p), nil
}

func (w *Writer) Finish() {
	for _, message := range w.parser.Finish() {
		w.ch <- message
	}
}

func (p *Parser) Stream(reader io.Reader, msgBufSize int) <-chan Message {
	ch := make(chan Message, msgBufSize)
	writer := &Writer{parser: p, ch: ch}

	go func() {
		defer close(ch)
		defer writer.Finish()
		_, err := io.Copy(writer, reader)
		if err != nil {
			slog.Error(fmt.Sprintf("Error reading from stream: %v", err))
		}
	}()
	return ch
}

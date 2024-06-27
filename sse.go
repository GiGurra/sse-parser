package sse_parser

import (
	"fmt"
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
	if len(lineInPart) < 2 {
		return Message{}, false
	}
	eventLine := lineInPart[0]
	dataLines := strings.Join(lineInPart[1:], "\n")

	if !strings.HasPrefix(eventLine, "event:") {
		return Message{}, false
	}
	if !strings.HasPrefix(dataLines, "data:") {
		return Message{}, false
	}

	event := strings.TrimPrefix(eventLine, "event:")
	data := strings.TrimPrefix(dataLines, "data:")

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

# SSE Parser

NOTE: Archived. Not using this anymore. Thankfully nobody seems to want SSE anymore :D 

This project provides a simple Server-Sent Events (SSE) parser written in Go. The parser is designed to handle streaming
data and extract messages based on specific event and data patterns.

## Features

- **Incremental Parsing**: Add data incrementally and parse messages as they become complete.
- **Optional Completion Logic**: Optionally define custom logic to determine when a message is complete.
- **Error Handling**: Skips invalid messages and logs errors.

## Installation

To use this parser in your project, you need to have Go installed. Then, you can import the package:

```sh
go get github.com/GiGurra/sse-parser
```

## Usage

### Creating a Parser

You can create a new parser with or without a completion function:

```go
import "github.com/GiGurra/sse-parser"

// Without a completion function
parser := sse_parser.NewParser(nil)

// With a completion function (optional)
parser := sse_parser.NewParser(func (data string) bool {
return strings.HasSuffix(data, "[END]")
})
```

### Adding Data

You can add data to the parser incrementally:

```go
messages := parser.Add("event:message\n" +
"data:hello\n\n" +
"event:message\n" +
"data:world")
```

### Finishing Parsing

To finalize parsing and get any remaining messages:

```go
finalMessages := parser.Finish()
```

### Example

```go
package main

import (
	"fmt"
	"github.com/GiGurra/sse-parser"
)

func main() {
	// Create a parser without a completion function
	parser := sse_parser.NewParser(nil)

	data := "event:message\n" +
		"data:hello\n\n" +
		"event:message\n" +
		"data:world"

	messages := parser.Add(data)
	messages = append(messages, parser.Finish()...)

	for _, msg := range messages {
		fmt.Printf("Event: %s, Data: %s\n", msg.Event, msg.Data)
	}
}

```

## Streaming data

You can use a `Parser` to convert an `io.Reader` into a `<-chan Message`:

```go
package main

import (
	"fmt"
	"github.com/GiGurra/sse-parser"
	"strings"
)

func main() {
	// Create a parser without a completion function
	parser := sse_parser.NewParser(nil)

	data := "event:message\n" +
		"data:hello\n\n" +
		"event:message\n" +
		"data:world"

	reader := strings.NewReader(data)
	bufSize := 100
	messages := parser.Stream(reader, bufSize)

	for msg := range messages {
		fmt.Printf("Event: %s, Data: %s\n", msg.Event, msg.Data)
	}
}

```

## Behavior

- If no completion function is provided, the parser will consider a message complete when it encounters a double
  newline (`\n\n`).
- If a completion function is provided, it will be used to determine when a message's data is complete. WARNING: `\n\n`
  still indicates a hard separation between messages, so the completion function is only really necessary to handle
  cases where we have received the full data but not the `\n\n` yet.

## Testing

The project includes a set of tests to verify the parser's functionality. You can run the tests using:

```sh
go test
```

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request for any improvements or bug fixes.

## Future improvements

* The completion should be able to override the default behavior of considering a message complete when encountering a
  double newline.

---

Feel free to reach out if you have any questions or need further assistance. Happy coding!

package yaml

import (
	"io"
)

func yaml_insert_token(parser *yaml_parser_t, pos int, token *yaml_token_t) {
	// Check if we can move the queue at the beginning of the buffer.
	if parser.tokens_head > 0 && len(parser.tokens) == cap(parser.tokens) {
		if parser.tokens_head != len(parser.tokens) {
			copy(parser.tokens, parser.tokens[parser.tokens_head:])
		}

		parser.tokens = parser.tokens[:len(parser.tokens)-parser.tokens_head]
		parser.tokens_head = 0
	}

	parser.tokens = append(parser.tokens, *token)

	if pos < 0 {
		return
	}

	copy(parser.tokens[parser.tokens_head+pos+1:], parser.tokens[parser.tokens_head+pos:])
	parser.tokens[parser.tokens_head+pos] = *token
}

// Create a new parser object.
func yaml_parser_initialize(parser *yaml_parser_t) bool {
	*parser = yaml_parser_t{
		raw_buffer: make([]byte, 0, input_raw_buffer_size),
		buffer:     make([]byte, 0, input_buffer_size),
	}

	return true
}

// Reader read handler.
func yaml_reader_read_handler(parser *yaml_parser_t, buffer []byte) (n int, err error) {
	return parser.input_reader.Read(buffer)
}

// Set a file input.
func yaml_parser_set_input_reader(parser *yaml_parser_t, r io.Reader) {
	if parser.read_handler != nil {
		panic("must set the input source only once")
	}

	parser.read_handler = yaml_reader_read_handler
	parser.input_reader = r
}

// Destroy an event object.
func yaml_event_delete(event *yaml_event_t) {
	*event = yaml_event_t{}
}

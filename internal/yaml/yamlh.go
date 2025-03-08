package yaml

import (
	"fmt"
	"io"
)

// The version directive data.
type yaml_version_directive_t struct {
	major int8 // The major version number.
	minor int8 // The minor version number.
}

// The tag directive data.
type yaml_tag_directive_t struct {
	handle []byte // The tag handle.
	prefix []byte // The tag prefix.
}

type yaml_encoding_t int

// The stream encoding.
const (
	// Let the parser choose the encoding.
	yaml_ANY_ENCODING yaml_encoding_t = iota

	yaml_UTF8_ENCODING    // The default UTF-8 encoding.
	yaml_UTF16LE_ENCODING // The UTF-16-LE encoding with BOM.
	yaml_UTF16BE_ENCODING // The UTF-16-BE encoding with BOM.
)

type yaml_error_type_t int

// Many bad things could happen with the parser and emitter.
const (
	// No error is produced.
	yaml_NO_ERROR yaml_error_type_t = iota
	_
	yaml_READER_ERROR  // Cannot read or decode the input stream.
	yaml_SCANNER_ERROR // Cannot scan the input stream.
	yaml_PARSER_ERROR  // Cannot parse the input stream.
)

// The pointer position.
type yaml_mark_t struct {
	index  int // The position index.
	line   int // The position line.
	column int // The position column.
}

// Node Styles

type yaml_style_t int8

type yaml_scalar_style_t yaml_style_t

// Scalar styles.
const (
	// Let the emitter choose the style.
	_ yaml_scalar_style_t = 0

	yaml_PLAIN_SCALAR_STYLE         yaml_scalar_style_t = 1 << iota // The plain scalar style.
	yaml_SINGLE_QUOTED_SCALAR_STYLE                                 // The single-quoted scalar style.
	yaml_DOUBLE_QUOTED_SCALAR_STYLE                                 // The double-quoted scalar style.
	yaml_LITERAL_SCALAR_STYLE                                       // The literal scalar style.
	yaml_FOLDED_SCALAR_STYLE                                        // The folded scalar style.
)

type yaml_sequence_style_t yaml_style_t

// Sequence styles.
const (
	// Let the emitter choose the style.
	_                         yaml_sequence_style_t = iota
	yaml_BLOCK_SEQUENCE_STYLE                       // The block sequence style.
	yaml_FLOW_SEQUENCE_STYLE                        // The flow sequence style.
)

type yaml_mapping_style_t yaml_style_t

// Mapping styles.
const (
	// Let the emitter choose the style.
	_                        yaml_mapping_style_t = iota
	yaml_BLOCK_MAPPING_STYLE                      // The block mapping style.
	yaml_FLOW_MAPPING_STYLE                       // The flow mapping style.
)

// Tokens

type yaml_token_type_t int

// Token types.
const (
	// An empty token.
	_ yaml_token_type_t = iota

	yaml_STREAM_START_TOKEN // A STREAM-START token.
	yaml_STREAM_END_TOKEN   // A STREAM-END token.

	yaml_VERSION_DIRECTIVE_TOKEN // A VERSION-DIRECTIVE token.
	yaml_TAG_DIRECTIVE_TOKEN     // A TAG-DIRECTIVE token.
	yaml_DOCUMENT_START_TOKEN    // A DOCUMENT-START token.
	yaml_DOCUMENT_END_TOKEN      // A DOCUMENT-END token.

	yaml_BLOCK_SEQUENCE_START_TOKEN // A BLOCK-SEQUENCE-START token.
	yaml_BLOCK_MAPPING_START_TOKEN  // A BLOCK-SEQUENCE-END token.
	yaml_BLOCK_END_TOKEN            // A BLOCK-END token.

	yaml_FLOW_SEQUENCE_START_TOKEN // A FLOW-SEQUENCE-START token.
	yaml_FLOW_SEQUENCE_END_TOKEN   // A FLOW-SEQUENCE-END token.
	yaml_FLOW_MAPPING_START_TOKEN  // A FLOW-MAPPING-START token.
	yaml_FLOW_MAPPING_END_TOKEN    // A FLOW-MAPPING-END token.

	yaml_BLOCK_ENTRY_TOKEN // A BLOCK-ENTRY token.
	yaml_FLOW_ENTRY_TOKEN  // A FLOW-ENTRY token.
	yaml_KEY_TOKEN         // A KEY token.
	yaml_VALUE_TOKEN       // A VALUE token.

	yaml_ALIAS_TOKEN  // An ALIAS token.
	yaml_ANCHOR_TOKEN // An ANCHOR token.
	yaml_TAG_TOKEN    // A TAG token.
	yaml_SCALAR_TOKEN // A SCALAR token.
)

// The token structure.
type yaml_token_t struct {
	// The token type.
	typ yaml_token_type_t

	// The start/end of the token.
	start_mark, end_mark yaml_mark_t

	// The stream encoding (for yaml_STREAM_START_TOKEN).
	encoding yaml_encoding_t

	// The alias/anchor/scalar value or tag/tag directive handle
	// (for yaml_ALIAS_TOKEN, yaml_ANCHOR_TOKEN, yaml_SCALAR_TOKEN, yaml_TAG_TOKEN, yaml_TAG_DIRECTIVE_TOKEN).
	value []byte

	// The tag suffix (for yaml_TAG_TOKEN).
	suffix []byte

	// The tag directive prefix (for yaml_TAG_DIRECTIVE_TOKEN).
	prefix []byte

	// The scalar style (for yaml_SCALAR_TOKEN).
	style yaml_scalar_style_t

	// The version directive major/minor (for yaml_VERSION_DIRECTIVE_TOKEN).
	major, minor int8
}

// Events

type yaml_event_type_t int8

// Event types.
const (
	// An empty event.
	yaml_NO_EVENT yaml_event_type_t = iota

	yaml_STREAM_START_EVENT   // A STREAM-START event.
	yaml_STREAM_END_EVENT     // A STREAM-END event.
	yaml_DOCUMENT_START_EVENT // A DOCUMENT-START event.
	yaml_DOCUMENT_END_EVENT   // A DOCUMENT-END event.
	yaml_ALIAS_EVENT          // An ALIAS event.
	yaml_SCALAR_EVENT         // A SCALAR event.
	yaml_SEQUENCE_START_EVENT // A SEQUENCE-START event.
	yaml_SEQUENCE_END_EVENT   // A SEQUENCE-END event.
	yaml_MAPPING_START_EVENT  // A MAPPING-START event.
	yaml_MAPPING_END_EVENT    // A MAPPING-END event.
	yaml_TAIL_COMMENT_EVENT
)

var eventStrings = []string{
	yaml_NO_EVENT:             "none",
	yaml_STREAM_START_EVENT:   "stream start",
	yaml_STREAM_END_EVENT:     "stream end",
	yaml_DOCUMENT_START_EVENT: "document start",
	yaml_DOCUMENT_END_EVENT:   "document end",
	yaml_ALIAS_EVENT:          "alias",
	yaml_SCALAR_EVENT:         "scalar",
	yaml_SEQUENCE_START_EVENT: "sequence start",
	yaml_SEQUENCE_END_EVENT:   "sequence end",
	yaml_MAPPING_START_EVENT:  "mapping start",
	yaml_MAPPING_END_EVENT:    "mapping end",
	yaml_TAIL_COMMENT_EVENT:   "tail comment",
}

func (e yaml_event_type_t) String() string {
	if e < 0 || int(e) >= len(eventStrings) {
		return fmt.Sprintf("unknown event %d", e)
	}

	return eventStrings[e]
}

// The event structure.
type yaml_event_t struct {

	// The event type.
	typ yaml_event_type_t

	// The start and end of the event.
	start_mark, end_mark yaml_mark_t

	// The document encoding (for yaml_STREAM_START_EVENT).
	encoding yaml_encoding_t

	// The version directive (for yaml_DOCUMENT_START_EVENT).
	version_directive *yaml_version_directive_t

	// The list of tag directives (for yaml_DOCUMENT_START_EVENT).
	tag_directives []yaml_tag_directive_t

	// The comments
	head_comment []byte
	line_comment []byte
	foot_comment []byte

	// The anchor (for yaml_SCALAR_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT, yaml_ALIAS_EVENT).
	anchor []byte

	// The tag (for yaml_SCALAR_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT).
	tag []byte

	// The scalar value (for yaml_SCALAR_EVENT).
	value []byte

	// Is the document start/end indicator implicit, or the tag optional?
	// (for yaml_DOCUMENT_START_EVENT, yaml_DOCUMENT_END_EVENT, yaml_SEQUENCE_START_EVENT,
	// yaml_MAPPING_START_EVENT, yaml_SCALAR_EVENT).
	implicit bool

	// Is the tag optional for any non-plain style? (for yaml_SCALAR_EVENT).
	quoted_implicit bool

	// The style (for yaml_SCALAR_EVENT, yaml_SEQUENCE_START_EVENT, yaml_MAPPING_START_EVENT).
	style yaml_style_t
}

func (e *yaml_event_t) scalar_style() yaml_scalar_style_t     { return yaml_scalar_style_t(e.style) }
func (e *yaml_event_t) sequence_style() yaml_sequence_style_t { return yaml_sequence_style_t(e.style) }
func (e *yaml_event_t) mapping_style() yaml_mapping_style_t   { return yaml_mapping_style_t(e.style) }

// Nodes

const (
	_            = "tag:yaml.org,2002:null"      // The tag !!null with the only possible value: null.
	_            = "tag:yaml.org,2002:bool"      // The tag !!bool with the values: true and false.
	yaml_STR_TAG = "tag:yaml.org,2002:str"       // The tag !!str for string values.
	_            = "tag:yaml.org,2002:int"       // The tag !!int for integer values.
	_            = "tag:yaml.org,2002:float"     // The tag !!float for float values.
	_            = "tag:yaml.org,2002:timestamp" // The tag !!timestamp for date and time values.

	yaml_SEQ_TAG = "tag:yaml.org,2002:seq" // The tag !!seq is used to denote sequences.
	yaml_MAP_TAG = "tag:yaml.org,2002:map" // The tag !!map is used to denote mapping.

	// Not in original libyaml.
	_ = "tag:yaml.org,2002:binary"
	_ = "tag:yaml.org,2002:merge"

	_ = yaml_STR_TAG // The default scalar tag is !!str.
	_ = yaml_SEQ_TAG // The default sequence tag is !!seq.
	_ = yaml_MAP_TAG // The default mapping tag is !!map.
)

// The prototype of a read handler.
//
// The read handler is called when the parser needs to read more bytes from the
// source. The handler should write not more than size bytes to the buffer.
// The number of written bytes should be set to the size_read variable.
//
// [in,out]   data        A pointer to an application data specified by
//
//	yaml_parser_set_input().
//
// [out]      buffer      The buffer to write the data from the source.
// [in]       size        The size of the buffer.
// [out]      size_read   The actual number of bytes read from the source.
//
// On success, the handler should return 1.  If the handler failed,
// the returned value should be 0. On EOF, the handler should set the
// size_read to 0 and return 1.
type yaml_read_handler_t func(parser *yaml_parser_t, buffer []byte) (n int, err error)

// This structure holds information about a potential simple key.
type yaml_simple_key_t struct {
	possible     bool        // Is a simple key possible?
	required     bool        // Is a simple key required?
	token_number int         // The number of the token.
	mark         yaml_mark_t // The position mark.
}

// The states of the parser.
type yaml_parser_state_t int

const (
	yaml_PARSE_STREAM_START_STATE yaml_parser_state_t = iota

	yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE           // Expect the beginning of an implicit document.
	yaml_PARSE_DOCUMENT_START_STATE                    // Expect DOCUMENT-START.
	yaml_PARSE_DOCUMENT_CONTENT_STATE                  // Expect the content of a document.
	yaml_PARSE_DOCUMENT_END_STATE                      // Expect DOCUMENT-END.
	yaml_PARSE_BLOCK_NODE_STATE                        // Expect a block node.
	yaml_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE // Expect a block node or indentless sequence.
	yaml_PARSE_FLOW_NODE_STATE                         // Expect a flow node.
	yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE        // Expect the first entry of a block sequence.
	yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE              // Expect an entry of a block sequence.
	yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE         // Expect an entry of an indentless sequence.
	yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE           // Expect the first key of a block mapping.
	yaml_PARSE_BLOCK_MAPPING_KEY_STATE                 // Expect a block mapping key.
	yaml_PARSE_BLOCK_MAPPING_VALUE_STATE               // Expect a block mapping value.
	yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE         // Expect the first entry of a flow sequence.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE               // Expect an entry of a flow sequence.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE   // Expect a key of an ordered mapping.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE // Expect a value of an ordered mapping.
	yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE   // Expect the and of an ordered mapping entry.
	yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE            // Expect the first key of a flow mapping.
	yaml_PARSE_FLOW_MAPPING_KEY_STATE                  // Expect a key of a flow mapping.
	yaml_PARSE_FLOW_MAPPING_VALUE_STATE                // Expect a value of a flow mapping.
	yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE          // Expect an empty value of a flow mapping.
	yaml_PARSE_END_STATE                               // Expect nothing.
)

// The parser structure.
//
// All members are internal. Manage the structure using the
// yaml_parser_ family of functions.
type yaml_parser_t struct {

	// Error handling

	error yaml_error_type_t // Error type.

	problem string // Error description.

	// The byte about which the problem occurred.
	problem_offset int
	problem_value  int
	problem_mark   yaml_mark_t

	// The error context.
	context      string
	context_mark yaml_mark_t

	// Reader stuff

	read_handler yaml_read_handler_t // Read handler.

	input_reader io.Reader // File input data.

	eof bool // EOF flag

	buffer     []byte // The working buffer.
	buffer_pos int    // The current position of the buffer.

	unread int // The number of unread characters in the buffer.

	newlines int // The number of line breaks since last non-break/non-blank character

	raw_buffer     []byte // The raw buffer.
	raw_buffer_pos int    // The current position of the buffer.

	encoding yaml_encoding_t // The input encoding.

	offset int         // The offset of the current position (in bytes).
	mark   yaml_mark_t // The mark of the current position.

	// Comments

	head_comment []byte // The current head comments
	line_comment []byte // The current line comments
	foot_comment []byte // The current foot comments
	tail_comment []byte // Foot comment that happens at the end of a block.
	stem_comment []byte // Comment in item preceding a nested structure (list inside list item, etc)

	comments      []yaml_comment_t // The folded comments for all parsed tokens
	comments_head int

	// Scanner stuff

	stream_start_produced bool // Have we started to scan the input stream?
	stream_end_produced   bool // Have we reached the end of the input stream?

	flow_level int // The number of unclosed '[' and '{' indicators.

	tokens          []yaml_token_t // The tokens queue.
	tokens_head     int            // The head of the tokens queue.
	tokens_parsed   int            // The number of tokens fetched from the queue.
	token_available bool           // Does the tokens queue contain a token ready for dequeueing.

	indent  int   // The current indentation level.
	indents []int // The indentation levels stack.

	simple_key_allowed bool                // May a simple key occur at the current position?
	simple_keys        []yaml_simple_key_t // The stack of simple keys.
	simple_keys_by_tok map[int]int         // possible simple_key indexes indexed by token_number

	// Parser stuff

	state          yaml_parser_state_t    // The current parser state.
	states         []yaml_parser_state_t  // The parser states stack.
	marks          []yaml_mark_t          // The stack of marks.
	tag_directives []yaml_tag_directive_t // The list of TAG directives.
}

type yaml_comment_t struct {
	scan_mark  yaml_mark_t // Position where scanning for comments started
	token_mark yaml_mark_t // Position after which tokens will be associated with this comment
	start_mark yaml_mark_t // Position of '#' comment mark
	end_mark   yaml_mark_t // Position where comment terminated

	head []byte
	line []byte
	foot []byte
}

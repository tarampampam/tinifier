package yaml

import (
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"
	"sync"
	"unicode/utf8"
)

// The Unmarshaler interface may be implemented by types to customize their
// behavior when being unmarshaled from a YAML document.
type Unmarshaler interface {
	UnmarshalYAML(value *Node) error
}

type obsoleteUnmarshaler interface {
	UnmarshalYAML(unmarshal func(any) error) error
}

// A Decoder reads and decodes YAML values from an input stream.
type Decoder struct {
	parser      *parser
	knownFields bool
}

// NewDecoder returns a new decoder that reads from r.
//
// The decoder introduces its own buffering and may read
// data from r beyond the YAML values requested.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		parser: newParserFromReader(r),
	}
}

// Decode reads the next YAML-encoded value from its input
// and stores it in the value pointed to by v.
//
// See the documentation for Unmarshal for details about the
// conversion of YAML into a Go value.
func (dec *Decoder) Decode(v any) (err error) {
	d := newDecoder()
	d.knownFields = dec.knownFields

	defer handleErr(&err)

	node := dec.parser.parse()
	if node == nil {
		return io.EOF
	}

	out := reflect.ValueOf(v)
	if out.Kind() == reflect.Ptr && !out.IsNil() {
		out = out.Elem()
	}

	d.unmarshal(node, out)

	if len(d.terrors) > 0 {
		return &TypeError{d.terrors}
	}

	return nil
}

// Decode decodes the node and stores its data into the value pointed to by v.
//
// See the documentation for Unmarshal for details about the
// conversion of YAML into a Go value.
func (n *Node) Decode(v any) (err error) {
	d := newDecoder()

	defer handleErr(&err)

	out := reflect.ValueOf(v)
	if out.Kind() == reflect.Ptr && !out.IsNil() {
		out = out.Elem()
	}

	d.unmarshal(n, out)

	if len(d.terrors) > 0 {
		return &TypeError{d.terrors}
	}

	return nil
}

func handleErr(err *error) {
	if v := recover(); v != nil {
		if e, ok := v.(yamlError); ok {
			*err = e.err
		} else {
			panic(v)
		}
	}
}

type yamlError struct {
	err error
}

func fail(err error) {
	panic(yamlError{err})
}

func failf(format string, args ...any) {
	panic(yamlError{fmt.Errorf("yaml: "+format, args...)})
}

// A TypeError is returned by Unmarshal when one or more fields in
// the YAML document cannot be properly decoded into the requested
// types. When this error is returned, the value is still
// unmarshaled partially.
type TypeError struct {
	Errors []string
}

func (e *TypeError) Error() string {
	return fmt.Sprintf("yaml: unmarshal errors:\n  %s", strings.Join(e.Errors, "\n  "))
}

type Kind uint32

const (
	DocumentNode Kind = 1 << iota
	SequenceNode
	MappingNode
	ScalarNode
	AliasNode
)

type Style uint32

const (
	TaggedStyle Style = 1 << iota
	DoubleQuotedStyle
	SingleQuotedStyle
	LiteralStyle
	FoldedStyle
	FlowStyle
)

// Node represents an element in the YAML document hierarchy. While documents
// are typically encoded and decoded into higher level types, such as structs
// and maps, Node is an intermediate representation that allows detailed
// control over the content being decoded or encoded.
//
// It's worth noting that although Node offers access into details such as
// line numbers, colums, and comments, the content when re-encoded will not
// have its original textual representation preserved. An effort is made to
// render the data plesantly, and to preserve comments near the data they
// describe, though.
//
// Values that make use of the Node type interact with the yaml package in the
// same way any other type would do, by encoding and decoding yaml data
// directly or indirectly into them.
//
// For example:
//
//	var person struct {
//	        Name    string
//	        Address yaml.Node
//	}
//	err := yaml.Unmarshal(data, &person)
//
// Or by itself:
//
//	var person Node
//	err := yaml.Unmarshal(data, &person)
type Node struct {
	// Kind defines whether the node is a document, a mapping, a sequence,
	// a scalar value, or an alias to another node. The specific data type of
	// scalar nodes may be obtained via the ShortTag and LongTag methods.
	Kind Kind

	// Style allows customizing the appearance of the node in the tree.
	Style Style

	// Tag holds the YAML tag defining the data type for the value.
	// When decoding, this field will always be set to the resolved tag,
	// even when it wasn't explicitly provided in the YAML content.
	// When encoding, if this field is unset the value type will be
	// implied from the node properties, and if it is set, it will only
	// be serialized into the representation if TaggedStyle is used or
	// the implicit tag diverges from the provided one.
	Tag string

	// Value holds the unescaped and unquoted representation of the value.
	Value string

	// Anchor holds the anchor name for this node, which allows aliases to point to it.
	Anchor string

	// Alias holds the node that this alias points to. Only valid when Kind is AliasNode.
	Alias *Node

	// Content holds contained nodes for documents, mappings, and sequences.
	Content []*Node

	// HeadComment holds any comments in the lines preceding the node and
	// not separated by an empty line.
	HeadComment string

	// LineComment holds any comments at the end of the line where the node is in.
	LineComment string

	// FootComment holds any comments following the node and before empty lines.
	FootComment string

	// Line and Column hold the node position in the decoded YAML text.
	// These fields are not respected when encoding the node.
	Line   int
	Column int
}

// IsZero returns whether the node has all of its fields unset.
func (n *Node) IsZero() bool {
	return n.Kind == 0 &&
		n.Style == 0 &&
		n.Tag == "" &&
		n.Value == "" &&
		n.Anchor == "" &&
		n.Alias == nil &&
		n.Content == nil &&
		n.HeadComment == "" &&
		n.LineComment == "" &&
		n.FootComment == "" &&
		n.Line == 0 &&
		n.Column == 0
}

// LongTag returns the long form of the tag that indicates the data type for
// the node. If the Tag field isn't explicitly defined, one will be computed
// based on the node properties.
func (n *Node) LongTag() string {
	return longTag(n.ShortTag())
}

// ShortTag returns the short form of the YAML tag that indicates data type for
// the node. If the Tag field isn't explicitly defined, one will be computed
// based on the node properties.
func (n *Node) ShortTag() string {
	if n.indicatedString() {
		return strTag
	}

	if n.Tag == "" || n.Tag == "!" {
		switch n.Kind {
		case MappingNode:
			return mapTag
		case SequenceNode:
			return seqTag
		case AliasNode:
			if n.Alias != nil {
				return n.Alias.ShortTag()
			}
		case ScalarNode:
			tag, _ := resolve("", n.Value)

			return tag
		case 0:
			// Special case to make the zero value convenient.
			if n.IsZero() {
				return nullTag
			}
		}

		return ""
	}

	return shortTag(n.Tag)
}

func (n *Node) indicatedString() bool {
	return n.Kind == ScalarNode &&
		(shortTag(n.Tag) == strTag ||
			(n.Tag == "" || n.Tag == "!") && n.Style&(SingleQuotedStyle|DoubleQuotedStyle|LiteralStyle|FoldedStyle) != 0)
}

// SetString is a convenience function that sets the node to a string value
// and defines its style in a pleasant way depending on its content.
func (n *Node) SetString(s string) {
	n.Kind = ScalarNode
	if utf8.ValidString(s) {
		n.Value = s
		n.Tag = strTag
	} else {
		n.Value = encodeBase64(s)
		n.Tag = binaryTag
	}

	if strings.Contains(n.Value, "\n") {
		n.Style = LiteralStyle
	}
}

// --------------------------------------------------------------------------
// Maintain a mapping of keys to structure field indexes

// The code in this section was copied from mgo/bson.

// structInfo holds details for the serialization of fields of
// a given struct.
type structInfo struct {
	FieldsMap  map[string]fieldInfo
	FieldsList []fieldInfo

	// InlineMap is the number of the field in the struct that
	// contains an inline map, or -1 if there's none.
	InlineMap int

	// InlineUnmarshalers holds indexes to inlined fields that
	// contain unmarshaler values.
	InlineUnmarshalers [][]int
}

type fieldInfo struct {
	Key       string
	Num       int
	OmitEmpty bool
	Flow      bool
	// Id holds the unique field identifier, so we can cheaply
	// check for field duplicates without maintaining an extra map.
	Id int

	// Inline holds the field index if the field is part of an inlined struct.
	Inline []int
}

var structMap = make(map[reflect.Type]*structInfo)
var fieldMapMutex sync.RWMutex
var unmarshalerType reflect.Type

func init() {
	var v Unmarshaler
	unmarshalerType = reflect.ValueOf(&v).Elem().Type()
}

func getStructInfo(st reflect.Type) (*structInfo, error) {
	fieldMapMutex.RLock()
	sinfo, found := structMap[st]
	fieldMapMutex.RUnlock()

	if found {
		return sinfo, nil
	}

	n := st.NumField()
	fieldsMap := make(map[string]fieldInfo)
	fieldsList := make([]fieldInfo, 0, n)
	inlineMap := -1
	inlineUnmarshalers := [][]int(nil)

	for i := 0; i != n; i++ {
		field := st.Field(i)
		if field.PkgPath != "" && !field.Anonymous {
			continue // Private field
		}

		info := fieldInfo{Num: i}

		tag := field.Tag.Get("yaml")
		if tag == "" && !strings.Contains(string(field.Tag), ":") {
			tag = string(field.Tag)
		}

		if tag == "-" {
			continue
		}

		inline := false

		fields := strings.Split(tag, ",")
		if len(fields) > 1 {
			for _, flag := range fields[1:] {
				switch flag {
				case "omitempty":
					info.OmitEmpty = true
				case "flow":
					info.Flow = true
				case "inline":
					inline = true
				default:
					return nil, fmt.Errorf("unsupported flag %q in tag %q of type %s", flag, tag, st)
				}
			}

			tag = fields[0]
		}

		if inline {
			switch field.Type.Kind() {
			case reflect.Map:
				if inlineMap >= 0 {
					return nil, errors.New("multiple ,inline maps in struct " + st.String())
				}

				if field.Type.Key() != reflect.TypeOf("") {
					return nil, errors.New("option ,inline needs a map with string keys in struct " + st.String())
				}

				inlineMap = info.Num
			case reflect.Struct, reflect.Ptr:
				ftype := field.Type
				for ftype.Kind() == reflect.Ptr {
					ftype = ftype.Elem()
				}

				if ftype.Kind() != reflect.Struct {
					return nil, errors.New("option ,inline may only be used on a struct or map field")
				}

				if reflect.PointerTo(ftype).Implements(unmarshalerType) {
					inlineUnmarshalers = append(inlineUnmarshalers, []int{i})
				} else {
					sinfo, err := getStructInfo(ftype)
					if err != nil {
						return nil, err
					}

					for _, index := range sinfo.InlineUnmarshalers {
						inlineUnmarshalers = append(inlineUnmarshalers, append([]int{i}, index...))
					}

					for _, finfo := range sinfo.FieldsList {
						if _, found := fieldsMap[finfo.Key]; found {
							msg := "duplicated key '" + finfo.Key + "' in struct " + st.String()

							return nil, errors.New(msg)
						}

						if finfo.Inline == nil {
							finfo.Inline = []int{i, finfo.Num}
						} else {
							finfo.Inline = append([]int{i}, finfo.Inline...)
						}

						finfo.Id = len(fieldsList)
						fieldsMap[finfo.Key] = finfo
						fieldsList = append(fieldsList, finfo)
					}
				}
			default:
				return nil, errors.New("option ,inline may only be used on a struct or map field")
			}

			continue
		}

		if tag != "" {
			info.Key = tag
		} else {
			info.Key = strings.ToLower(field.Name)
		}

		if _, found = fieldsMap[info.Key]; found {
			msg := "duplicated key '" + info.Key + "' in struct " + st.String()

			return nil, errors.New(msg)
		}

		info.Id = len(fieldsList)
		fieldsList = append(fieldsList, info)
		fieldsMap[info.Key] = info
	}

	sinfo = &structInfo{
		FieldsMap:          fieldsMap,
		FieldsList:         fieldsList,
		InlineMap:          inlineMap,
		InlineUnmarshalers: inlineUnmarshalers,
	}

	fieldMapMutex.Lock()
	structMap[st] = sinfo
	fieldMapMutex.Unlock()

	return sinfo, nil
}

// IsZeroer is used to check whether an object is zero to
// determine whether it should be omitted when marshaling
// with the omitempty flag. One notable implementation
// is time.Time.
type IsZeroer interface {
	IsZero() bool
}

package container

import (
	"bytes"
	"encoding/json"
	"encoding/json/jsontext"
	"errors"
)

// Field is one key of a structured log message.
type Field[V any] struct {
	Key   string
	Value V
}

// Fields is a JSON object that keeps its keys in the order they were logged.
// Log lines carry a handful of keys, so a slice with a linear Get is cheaper
// than any map. Decoded with V = any, nested objects are Fields[any] too, so
// order holds at every depth.
type Fields[V any] []Field[V]

// Get returns the value for key.
func (f Fields[V]) Get(key string) (V, bool) {
	for _, field := range f {
		if field.Key == key {
			return field.Value, true
		}
	}
	var zero V
	return zero, false
}

// Set replaces the value for key in place, or appends it if key is new.
func (f *Fields[V]) Set(key string, value V) {
	for i := range *f {
		if (*f)[i].Key == key {
			(*f)[i].Value = value
			return
		}
	}
	*f = append(*f, Field[V]{Key: key, Value: value})
}

// Log lines are not always well-formed: duplicate keys keep the last value
// and invalid UTF-8 becomes U+FFFD, matching encoding/json v1.
var decodeOptions = []jsontext.Options{
	jsontext.AllowDuplicateNames(true),
	jsontext.AllowInvalidUTF8(true),
}

var errNotObject = errors.New("fields: not a JSON object")

// ParseFields decodes a JSON object. JSON null returns nil Fields and no error;
// any other non-object is an error.
func ParseFields(b []byte) (Fields[any], error) {
	dec := jsontext.NewDecoder(bytes.NewBuffer(b), decodeOptions...)
	var f Fields[any]
	if err := f.UnmarshalJSONFrom(dec); err != nil {
		return nil, err
	}
	if _, err := dec.ReadToken(); err == nil {
		return nil, errors.New("fields: trailing data after object")
	}
	return f, nil
}

func (f *Fields[V]) UnmarshalJSON(b []byte) error {
	dec := jsontext.NewDecoder(bytes.NewBuffer(b), decodeOptions...)
	return f.UnmarshalJSONFrom(dec)
}

func (f *Fields[V]) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	tok, err := dec.ReadToken()
	if err != nil {
		return err
	}
	switch tok.Kind() {
	case 'n':
		*f = nil
		return nil
	case '{':
	default:
		return errNotObject
	}
	out := make(Fields[V], 0, 8)
	for dec.PeekKind() != '}' {
		keyTok, err := dec.ReadToken()
		if err != nil {
			return err
		}
		key := keyTok.String() // the token is voided by the next read
		value, err := decodeValue(dec)
		if err != nil {
			return err
		}
		v, ok := value.(V)
		if !ok && value != nil {
			return &json.UnmarshalTypeError{Value: kindName(value)}
		}
		out.Set(key, v)
	}
	if _, err := dec.ReadToken(); err != nil {
		return err
	}
	*f = out
	return nil
}

func decodeValue(dec *jsontext.Decoder) (any, error) {
	switch dec.PeekKind() {
	case '{':
		var nested Fields[any]
		if err := nested.UnmarshalJSONFrom(dec); err != nil {
			return nil, err
		}
		return nested, nil
	case '[':
		if _, err := dec.ReadToken(); err != nil {
			return nil, err
		}
		arr := []any{}
		for dec.PeekKind() != ']' {
			v, err := decodeValue(dec)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		_, err := dec.ReadToken()
		return arr, err
	}
	tok, err := dec.ReadToken()
	if err != nil {
		return nil, err
	}
	switch tok.Kind() {
	case '"':
		return tok.String(), nil
	case '0':
		return tok.Float()
	case 't', 'f':
		return tok.Bool(), nil
	case 'n':
		return nil, nil
	}
	return nil, dec.SkipValue()
}

func kindName(v any) string {
	switch v.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "bool"
	case []any:
		return "array"
	default:
		return "object"
	}
}

func (f Fields[V]) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	enc := jsontext.NewEncoder(&buf)
	if err := f.MarshalJSONTo(enc); err != nil {
		return nil, err
	}
	// The encoder ends every top-level value with a newline.
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}

func (f Fields[V]) MarshalJSONTo(enc *jsontext.Encoder) error {
	if f == nil {
		return enc.WriteToken(jsontext.Null)
	}
	if err := enc.WriteToken(jsontext.BeginObject); err != nil {
		return err
	}
	for _, field := range f {
		if err := enc.WriteToken(jsontext.String(field.Key)); err != nil {
			return err
		}
		if err := encodeValue(enc, field.Value); err != nil {
			return err
		}
	}
	return enc.WriteToken(jsontext.EndObject)
}

func encodeValue(enc *jsontext.Encoder, v any) error {
	switch v := v.(type) {
	case nil:
		return enc.WriteToken(jsontext.Null)
	case string:
		return enc.WriteToken(jsontext.String(v))
	case float64:
		return enc.WriteToken(jsontext.Float(v))
	case bool:
		return enc.WriteToken(jsontext.Bool(v))
	case Fields[any]:
		return v.MarshalJSONTo(enc)
	case Fields[string]:
		return v.MarshalJSONTo(enc)
	case []any:
		if err := enc.WriteToken(jsontext.BeginArray); err != nil {
			return err
		}
		for _, item := range v {
			if err := encodeValue(enc, item); err != nil {
				return err
			}
		}
		return enc.WriteToken(jsontext.EndArray)
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return err
		}
		return enc.WriteValue(b)
	}
}

// Map converts f to a plain map, nested objects included, for consumers such
// as expr and text/template that index by key.
func (f Fields[V]) Map() map[string]any {
	m := make(map[string]any, len(f))
	for _, field := range f {
		m[field.Key] = plainValue(field.Value)
	}
	return m
}

func plainValue(v any) any {
	switch v := v.(type) {
	case Fields[any]:
		return v.Map()
	case Fields[string]:
		return v.Map()
	case []any:
		out := make([]any, len(v))
		for i, item := range v {
			out[i] = plainValue(item)
		}
		return out
	}
	return v
}

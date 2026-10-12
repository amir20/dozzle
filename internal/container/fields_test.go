package container

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFieldsKeepsOrderAtEveryDepth(t *testing.T) {
	in := `{"z":1,"a":{"y":true,"b":null,"c":[{"q":"x","p":2}]},"m":"s"}`
	f, err := ParseFields([]byte(in))
	require.NoError(t, err)

	out, err := json.Marshal(f)
	require.NoError(t, err)
	assert.Equal(t, in, string(out))

	nested, ok := f.Get("a")
	require.True(t, ok)
	assert.IsType(t, Fields[any]{}, nested)
}

func TestParseFieldsValueTypes(t *testing.T) {
	f, err := ParseFields([]byte(`{"n":1.5,"b":false,"s":"x","null":null,"arr":[]}`))
	require.NoError(t, err)
	n, _ := f.Get("n")
	assert.Equal(t, 1.5, n)
	b, _ := f.Get("b")
	assert.Equal(t, false, b)
	v, ok := f.Get("null")
	assert.True(t, ok)
	assert.Nil(t, v)
	arr, _ := f.Get("arr")
	assert.Equal(t, []any{}, arr)
}

func TestParseFieldsDuplicateKeyKeepsFirstPositionLastValue(t *testing.T) {
	f, err := ParseFields([]byte(`{"a":1,"b":2,"a":3}`))
	require.NoError(t, err)
	assert.Equal(t, Fields[any]{{"a", 3.0}, {"b", 2.0}}, f)
}

func TestParseFieldsInvalidUTF8(t *testing.T) {
	f, err := ParseFields([]byte("{\"msg\":\"bad \xff byte\"}"))
	require.NoError(t, err)
	msg, _ := f.Get("msg")
	assert.Equal(t, "bad � byte", msg)
}

func TestParseFieldsRejects(t *testing.T) {
	for _, in := range []string{`[1,2]`, `"str"`, `123`, `{"a":1} {"b":2}`, `{"a":`} {
		_, err := ParseFields([]byte(in))
		assert.Error(t, err, in)
	}
}

func TestParseFieldsNull(t *testing.T) {
	f, err := ParseFields([]byte(`null`))
	require.NoError(t, err)
	assert.Nil(t, f)

	f, err = ParseFields([]byte(`{}`))
	require.NoError(t, err)
	assert.NotNil(t, f)
}

func TestFieldsStringRejectsNonString(t *testing.T) {
	var f Fields[string]
	assert.Error(t, json.Unmarshal([]byte(`{"a":1}`), &f))
	require.NoError(t, json.Unmarshal([]byte(`{"a":"1"}`), &f))
	assert.Equal(t, Fields[string]{{"a", "1"}}, f)
}

func TestFieldsMapIsDeep(t *testing.T) {
	f, err := ParseFields([]byte(`{"a":{"b":[{"c":1}]}}`))
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		"a": map[string]any{"b": []any{map[string]any{"c": 1.0}}},
	}, f.Map())
}

func TestFieldsSet(t *testing.T) {
	var f Fields[string]
	f.Set("a", "1")
	f.Set("b", "2")
	f.Set("a", "3")
	assert.Equal(t, Fields[string]{{"a", "3"}, {"b", "2"}}, f)
}

const benchLine = `{"time":"2024-06-02T14:30:42.123Z","level":"info","logger":"http","msg":"request completed","method":"GET","path":"/api/v1/containers","status":200,"duration_ms":12.5,"user":{"id":42,"roles":["admin","dev"]},"trace_id":"4bf92f3577b34da6a3ce929d0e0e4736"}`

func BenchmarkParseFields(b *testing.B) {
	data := []byte(benchLine)
	b.ReportAllocs()
	for b.Loop() {
		if _, err := ParseFields(data); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFieldsMarshal(b *testing.B) {
	f, _ := ParseFields([]byte(benchLine))
	b.ReportAllocs()
	for b.Loop() {
		if _, err := json.Marshal(f); err != nil {
			b.Fatal(err)
		}
	}
}

// Logfmt values and prefix words hold raw container bytes, which may not be
// UTF-8. They must still encode, through Fields directly and through v1 as
// part of a LogEvent, with bad bytes as U+FFFD.
func TestFieldsMarshalInvalidUTF8(t *testing.T) {
	f := Fields[string]{{Key: "msg", Value: "caf\xe9"}, {Key: "k\xff", Value: "v"}}

	out, err := f.MarshalJSON()
	require.NoError(t, err)
	assert.Equal(t, "{\"msg\":\"caf�\",\"k�\":\"v\"}", string(out))

	event, err := json.Marshal(LogEvent{Message: f})
	require.NoError(t, err)
	assert.Contains(t, string(event), "\"m\":{\"msg\":\"caf�\"")

	anyOut, err := json.Marshal(Fields[any]{{Key: "nested", Value: Fields[any]{{Key: "m", Value: "\xe9"}}}})
	require.NoError(t, err)
	assert.Equal(t, "{\"nested\":{\"m\":\"�\"}}", string(anyOut))
}

func TestFieldsCompact(t *testing.T) {
	assert.Equal(t,
		Fields[string]{{"a", "3"}, {"b", "2"}},
		Fields[string]{{"a", "1"}, {"b", "2"}, {"a", "3"}}.Compact(),
	)

	// Past the index threshold the result must be the same.
	var big Fields[int]
	for i := range 100 {
		big = append(big, Field[int]{Key: fmt.Sprint(i % 40), Value: i})
	}
	compacted := big.Compact()
	require.Len(t, compacted, 40)
	for i, field := range compacted {
		assert.Equal(t, fmt.Sprint(i), field.Key)
		last := 80 + i
		if i >= 20 {
			last = 40 + i
		}
		assert.Equal(t, last, field.Value, "key %s keeps its last value", field.Key)
	}
}

// A single huge line (k8s does not cap line length) must not go quadratic in
// its key count.
func TestParseFieldsManyKeysIsLinear(t *testing.T) {
	var sb strings.Builder
	sb.WriteByte('{')
	for i := range 100_000 {
		if i > 0 {
			sb.WriteByte(',')
		}
		fmt.Fprintf(&sb, `"k%d":1`, i)
	}
	sb.WriteString(`,"k0":2}`)

	start := time.Now()
	f, err := ParseFields([]byte(sb.String()))
	require.NoError(t, err)
	assert.Less(t, time.Since(start), 2*time.Second)
	require.Len(t, f, 100_000)
	assert.Equal(t, Field[any]{Key: "k0", Value: 2.0}, f[0])
}

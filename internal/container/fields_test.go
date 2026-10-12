package container

import (
	"encoding/json"
	"testing"

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

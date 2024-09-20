package docker

import (
	"encoding/json"
	"reflect"
	"testing"

	orderedmap "github.com/wk8/go-ordered-map/v2"
)

func TestParseLog(t *testing.T) {

	tests := []struct {
		name    string
		log     string
		want    *orderedmap.OrderedMap[string, string]
		wantErr bool
	}{
		{
			name: "Valid logfmt log",
			log:  `time="2024-06-02T14:30:42Z" level=debug msg="container e23e04da2cb9 started"`,
			want: orderedmap.New[string, string](
				orderedmap.WithInitialData(
					orderedmap.Pair[string, string]{Key: "time", Value: "2024-06-02T14:30:42Z"},
					orderedmap.Pair[string, string]{Key: "level", Value: "debug"},
					orderedmap.Pair[string, string]{Key: "msg", Value: "container e23e04da2cb9 started"},
				),
			),
			wantErr: false,
		},
		{
			name:    "Random test with equal sign",
			log:     "foo bar=baz",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Invalid log with key without value",
			log:     "key1=value1 key2=",
			want:    nil,
			wantErr: true,
		},
		{
			name: "Valid log",
			log:  "key1=value1 key2=value2",
			want: orderedmap.New[string, string](
				orderedmap.WithInitialData(
					orderedmap.Pair[string, string]{Key: "key1", Value: "value1"},
					orderedmap.Pair[string, string]{Key: "key2", Value: "value2"},
				),
			),
			wantErr: false,
		},
		{
			name:    "Invalid log with unclosed quotes",
			log:     "key1=\"value1 key2=value2",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "Plain text log",
			log:     "foo bar baz",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseLogFmt(tt.log)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseLogFmt() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				jsonGot, _ := json.MarshalIndent(got, "", "  ")
				jsonWant, _ := json.MarshalIndent(tt.want, "", "  ")
				t.Errorf("ParseLogFmt() = %v, want %v", string(jsonGot), string(jsonWant))
			}
		})
	}
}

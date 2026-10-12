package logparse

import (
	"encoding/json"
	"github.com/amir20/dozzle/internal/container"
	"reflect"
	"testing"
)

func TestParseLog(t *testing.T) {
	tests := []struct {
		name    string
		log     string
		want    container.Fields[string]
		wantErr bool
	}{
		{
			name: "Valid logfmt log",
			log:  `time="2024-06-02T14:30:42Z" level=debug msg="container e23e04da2cb9 started"`,
			want: container.Fields[string]{
				{Key: "time", Value: "2024-06-02T14:30:42Z"},
				{Key: "level", Value: "debug"},
				{Key: "msg", Value: "container e23e04da2cb9 started"},
			},
			wantErr: false,
		},
		{
			name:    "Random test with equal sign",
			log:     "foo bar=baz",
			want:    nil,
			wantErr: true,
		},
		{
			name: "Valid log with key and trailing no value",
			log:  "key1=value1 key2=",
			want: container.Fields[string]{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: ""},
			},
			wantErr: false,
		},
		{
			name: "Valid log with key and no values",
			log:  "key1=value1 key2= key3=bar",
			want: container.Fields[string]{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: ""},
				{Key: "key3", Value: "bar"},
			},
			wantErr: false,
		},
		{
			name: "Valid log",
			log:  "key1=value1 key2=value2",
			want: container.Fields[string]{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
			wantErr: false,
		},
		{
			name: "Escaped quotes and backslashes in quoted values",
			log:  `level=info msg="failed to open \"config.yml\"" path="C:\\temp"`,
			want: container.Fields[string]{
				{Key: "level", Value: "info"},
				{Key: "msg", Value: `failed to open "config.yml"`},
				{Key: "path", Value: `C:\temp`},
			},
			wantErr: false,
		},
		{
			name: "Quoted value with undecodable escape is kept as-is",
			log:  `msg="foo \q bar"`,
			want: container.Fields[string]{
				{Key: "msg", Value: `foo \q bar`},
			},
			wantErr: false,
		},
		{
			name:    "Broken format with unexpected quotes",
			log:     `key1=value"1"= key2="value2"`,
			want:    nil,
			wantErr: true,
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

package docker

import (
	"testing"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/assert"
)

func Test_calculateMemUsageUnixNoCache(t *testing.T) {
	type args struct {
		mem container.MemoryStats
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		{
			name: "with cgroup v1",
			args: args{
				mem: container.MemoryStats{
					Usage: 100,
					Stats: map[string]uint64{
						"total_inactive_file": 1,
					},
				},
			},
			want: 99,
		},
		{
			name: "with cgroup v2",
			args: args{
				mem: container.MemoryStats{
					Usage: 100,
					Stats: map[string]uint64{
						"inactive_file": 2,
					},
				},
			},
			want: 98,
		},
		{
			name: "without cgroup",
			args: args{
				mem: container.MemoryStats{
					Usage: 100,
				},
			},
			want: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, calculateMemUsageUnixNoCache(tt.args.mem), "calculateMemUsageUnixNoCache(%v)", tt.args.mem)
		})
	}
}

package deploy

import (
	"context"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// ParseCompose parses raw compose YAML bytes into a Project.
// The project name is used as a prefix for resource names (networks, volumes, containers).
func ParseCompose(ctx context.Context, data []byte, projectName string) (*types.Project, error) {
	return loader.LoadWithContext(ctx, types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{Content: data},
		},
	}, func(opts *loader.Options) {
		opts.SetProjectName(projectName, true)
		opts.SkipResolveEnvironment = true
	}, loader.WithSkipValidation)
}

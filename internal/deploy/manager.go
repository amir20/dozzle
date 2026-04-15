package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
)

const composeFilename = "compose.yaml"

// Version represents a single commit in a project's git history.
type Version struct {
	Hash    string
	Message string
	Time    time.Time
}

// Manager manages git-backed compose projects under a data directory.
// Each project lives in its own git repository at {dataDir}/{projectName}.
type Manager struct {
	deployer *Deployer
	dataDir  string
}

// NewManager creates a manager that stores projects under dataDir.
func NewManager(cli DockerClient, dataDir string) *Manager {
	return &Manager{
		deployer: NewDeployer(cli),
		dataDir:  dataDir,
	}
}

func (m *Manager) projectDir(name string) string {
	return filepath.Join(m.dataDir, name)
}

// CreateProject initializes a new project: creates a git repo, writes the
// compose file, commits, and deploys.
func (m *Manager) CreateProject(ctx context.Context, name string, composeYAML []byte) error {
	dir := m.projectDir(name)
	if _, err := os.Stat(dir); err == nil {
		return fmt.Errorf("project %q already exists", name)
	}

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating project directory: %w", err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("initializing git repo: %w", err)
	}

	composePath := filepath.Join(dir, composeFilename)
	if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("writing compose file: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("getting worktree: %w", err)
	}

	if _, err := wt.Add(composeFilename); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("staging compose file: %w", err)
	}

	if _, err := wt.Commit("Initial deployment", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Dozzle Deploy",
			Email: "deploy@dozzle.dev",
			When:  time.Now(),
		},
	}); err != nil {
		os.RemoveAll(dir)
		return fmt.Errorf("committing: %w", err)
	}

	project, err := ParseCompose(composeYAML, name)
	if err != nil {
		return fmt.Errorf("parsing compose file: %w", err)
	}

	if err := m.deployer.Deploy(ctx, project, nil); err != nil {
		return fmt.Errorf("deploying: %w", err)
	}

	log.Info().Str("project", name).Msg("Project created and deployed")
	return nil
}

// UpdateConfig writes a new compose file, commits the change, and redeploys.
// Status updates (pull, start, stop, etc.) are sent to the optional channel.
func (m *Manager) UpdateConfig(ctx context.Context, name string, composeYAML []byte, status chan<- StatusUpdate) error {
	dir := m.projectDir(name)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return fmt.Errorf("opening project %q: %w", name, err)
	}

	composePath := filepath.Join(dir, composeFilename)
	if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
		return fmt.Errorf("writing compose file: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	if _, err := wt.Add(composeFilename); err != nil {
		return fmt.Errorf("staging compose file: %w", err)
	}

	wtStatus, err := wt.Status()
	if err != nil {
		return fmt.Errorf("checking git status: %w", err)
	}

	if !wtStatus.IsClean() {
		if _, err := wt.Commit("Update config", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Dozzle Deploy",
				Email: "deploy@dozzle.dev",
				When:  time.Now(),
			},
		}); err != nil {
			return fmt.Errorf("committing: %w", err)
		}
	}

	project, err := ParseCompose(composeYAML, name)
	if err != nil {
		return fmt.Errorf("parsing compose file: %w", err)
	}

	if err := m.deployer.Deploy(ctx, project, status); err != nil {
		return fmt.Errorf("deploying: %w", err)
	}

	log.Info().Str("project", name).Msg("Config updated and deployed")
	return nil
}

// ListVersions returns the git commit history for a project, newest first.
func (m *Manager) ListVersions(name string) ([]Version, error) {
	dir := m.projectDir(name)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("opening project %q: %w", name, err)
	}

	logIter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	var versions []Version
	err = logIter.ForEach(func(c *object.Commit) error {
		versions = append(versions, Version{
			Hash:    c.Hash.String(),
			Message: strings.TrimSpace(c.Message),
			Time:    c.Author.When,
		})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("iterating log: %w", err)
	}

	return versions, nil
}

// RollbackVersion restores the compose file from a previous commit, creates
// a new rollback commit, and redeploys. Supports both full and short commit hashes.
func (m *Manager) RollbackVersion(ctx context.Context, name string, commitHash string, status chan<- StatusUpdate) error {
	dir := m.projectDir(name)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return fmt.Errorf("opening project %q: %w", name, err)
	}

	commitObj, err := resolveCommit(repo, commitHash)
	if err != nil {
		return fmt.Errorf("resolving commit: %w", err)
	}

	tree, err := commitObj.Tree()
	if err != nil {
		return fmt.Errorf("reading tree: %w", err)
	}

	file, err := tree.File(composeFilename)
	if err != nil {
		return fmt.Errorf("reading compose file from commit: %w", err)
	}

	content, err := file.Contents()
	if err != nil {
		return fmt.Errorf("reading file contents: %w", err)
	}

	composeYAML := []byte(content)
	composePath := filepath.Join(dir, composeFilename)
	if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
		return fmt.Errorf("writing compose file: %w", err)
	}

	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}

	if _, err := wt.Add(composeFilename); err != nil {
		return fmt.Errorf("staging compose file: %w", err)
	}

	shortHash := commitObj.Hash.String()[:12]

	wtStatus, err := wt.Status()
	if err != nil {
		return fmt.Errorf("checking git status: %w", err)
	}

	if !wtStatus.IsClean() {
		if _, err := wt.Commit(fmt.Sprintf("Rollback to %s", shortHash), &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Dozzle Deploy",
				Email: "deploy@dozzle.dev",
				When:  time.Now(),
			},
		}); err != nil {
			return fmt.Errorf("committing rollback: %w", err)
		}
	}

	project, err := ParseCompose(composeYAML, name)
	if err != nil {
		return fmt.Errorf("parsing compose file: %w", err)
	}

	if err := m.deployer.Deploy(ctx, project, status); err != nil {
		return fmt.Errorf("deploying: %w", err)
	}

	log.Info().Str("project", name).Str("rollback_to", shortHash).Msg("Rolled back and deployed")
	return nil
}

// resolveCommit finds a commit by full or short hash prefix.
func resolveCommit(repo *git.Repository, hash string) (*object.Commit, error) {
	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}

	var match *object.Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if c.Hash.String() == hash || strings.HasPrefix(c.Hash.String(), hash) {
			if match != nil {
				return fmt.Errorf("ambiguous commit prefix %q", hash)
			}
			match = c
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, fmt.Errorf("commit %q not found", hash)
	}
	return match, nil
}

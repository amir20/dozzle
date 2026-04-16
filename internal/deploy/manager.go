package deploy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/rs/zerolog/log"
)

const (
	composeFilename = "compose.yaml"
	// DefaultStacksDir is the default on-disk location for git-backed compose projects.
	DefaultStacksDir = "./data/stacks"
)

// Version represents a single commit in a project's git history.
type Version struct {
	Hash    string
	Message string
	Time    time.Time
}

// Manager manages git-backed compose projects under a data directory.
// Each project lives in its own git repository at {dataDir}/{projectName}.
//
// All operations on the same project serialize through a per-project mutex,
// so concurrent Deploy/Rollback/ListVersions calls for one project run one at
// a time while different projects still run in parallel.
type Manager struct {
	deployer *Deployer
	dataDir  string
	locks    sync.Map // project name -> *sync.Mutex
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

// lockProject acquires the project's lock and returns an unlock func for defer.
func (m *Manager) lockProject(name string) func() {
	actual, _ := m.locks.LoadOrStore(name, &sync.Mutex{})
	mu := actual.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Deploy creates a new git-backed project if it doesn't exist, or updates the
// existing one, then redeploys. Status updates are sent to the optional channel.
func (m *Manager) Deploy(ctx context.Context, name string, composeYAML []byte, status chan<- StatusUpdate) error {
	defer m.lockProject(name)()

	dir := m.projectDir(name)
	_, err := os.Stat(dir)
	switch {
	case errors.Is(err, os.ErrNotExist):
		return m.createLocked(ctx, name, composeYAML, status)
	case err != nil:
		return fmt.Errorf("stat project %q: %w", name, err)
	default:
		return m.updateLocked(ctx, name, composeYAML, status)
	}
}

func (m *Manager) createLocked(ctx context.Context, name string, composeYAML []byte, status chan<- StatusUpdate) error {
	dir := m.projectDir(name)
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

	if err := m.deployer.Deploy(ctx, project, status); err != nil {
		return fmt.Errorf("deploying: %w", err)
	}

	log.Info().Str("project", name).Msg("Project created and deployed")
	return nil
}

func (m *Manager) updateLocked(ctx context.Context, name string, composeYAML []byte, status chan<- StatusUpdate) error {
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

// Remove tears down a deployed project: stops and removes all containers,
// removes project-labeled networks, and deletes the on-disk project directory
// (including git history). If removeVolumes is true, project-labeled named
// volumes are also deleted — this is destructive and should be opt-in.
func (m *Manager) Remove(ctx context.Context, name string, removeVolumes bool, status chan<- StatusUpdate) error {
	defer m.lockProject(name)()

	dir := m.projectDir(name)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("project %q does not exist", name)
	} else if err != nil {
		return fmt.Errorf("stat project %q: %w", name, err)
	}

	composeYAML, err := os.ReadFile(filepath.Join(dir, composeFilename))
	if err != nil {
		return fmt.Errorf("reading compose file: %w", err)
	}
	project, err := ParseCompose(composeYAML, name)
	if err != nil {
		return fmt.Errorf("parsing compose file: %w", err)
	}

	if err := m.deployer.Remove(ctx, project, removeVolumes, status); err != nil {
		return fmt.Errorf("removing project: %w", err)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing project directory: %w", err)
	}

	log.Info().Str("project", name).Bool("removed_volumes", removeVolumes).Msg("Project removed")
	return nil
}

// ListVersions returns the git commit history for a project, newest first.
func (m *Manager) ListVersions(name string) ([]Version, error) {
	defer m.lockProject(name)()

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
	defer m.lockProject(name)()

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

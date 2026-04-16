package deploy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/rs/zerolog/log"
)

const (
	composeFilename = "compose.yaml"
	// DefaultStacksDir is the default on-disk location for git-backed compose projects.
	DefaultStacksDir = "./data/stacks"

	commitAuthorName  = "Dozzle Deploy"
	commitAuthorEmail = "deploy@dozzle.dev"
)

// projectNameRegexp follows the compose project-name rules: lowercase letters,
// digits, dashes, and underscores only. Anything else is rejected to prevent
// path traversal and to stay consistent with compose-go's own validation.
var projectNameRegexp = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project name is required")
	}
	if !projectNameRegexp.MatchString(name) {
		return fmt.Errorf("invalid project name %q: must match [a-z0-9][a-z0-9_-]*", name)
	}
	return nil
}

// Version represents a single commit in a project's git history.
type Version struct {
	Hash    string
	Message string
	Time    time.Time
}

// Manager manages git-backed compose projects under a data directory.
// Each project lives in its own git repository at {dataDir}/{projectName}.
//
// All operations on the same project serialize through a per-project RWMutex:
// concurrent write ops (Deploy/Rollback/Remove) for one project run one at a
// time, while ListVersions can proceed in parallel. Different projects run
// fully in parallel.
type Manager struct {
	deployer *Deployer
	dataDir  string
	locks    sync.Map // project name -> *sync.RWMutex
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

func (m *Manager) projectLock(name string) *sync.RWMutex {
	actual, _ := m.locks.LoadOrStore(name, &sync.RWMutex{})
	return actual.(*sync.RWMutex)
}

// Deploy creates a new git-backed project if it doesn't exist, or updates the
// existing one, then redeploys. Status updates are sent to the optional channel.
func (m *Manager) Deploy(ctx context.Context, name string, composeYAML []byte, status chan<- StatusUpdate) error {
	if err := validateProjectName(name); err != nil {
		return err
	}
	mu := m.projectLock(name)
	mu.Lock()
	defer mu.Unlock()

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

	cleanup := func(stage string, err error) error {
		if rmErr := os.RemoveAll(dir); rmErr != nil {
			log.Warn().Err(rmErr).Str("project", name).Msg("Failed to clean up project directory after setup error")
		}
		return fmt.Errorf("%s: %w", stage, err)
	}

	repo, err := git.PlainInit(dir, false)
	if err != nil {
		return cleanup("initializing git repo", err)
	}

	composePath := filepath.Join(dir, composeFilename)
	if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
		return cleanup("writing compose file", err)
	}

	if err := commitCompose(repo, "Initial deployment"); err != nil {
		return cleanup("committing", err)
	}

	project, err := ParseCompose(ctx, composeYAML, name)
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
	existing, readErr := os.ReadFile(composePath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading existing compose file: %w", readErr)
	}
	if !bytes.Equal(existing, composeYAML) {
		if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
			return fmt.Errorf("writing compose file: %w", err)
		}
		if err := commitCompose(repo, "Update config"); err != nil {
			return err
		}
	}

	project, err := ParseCompose(ctx, composeYAML, name)
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
	if err := validateProjectName(name); err != nil {
		return err
	}
	mu := m.projectLock(name)
	mu.Lock()
	defer mu.Unlock()
	// Drop the lock entry on successful removal so a churn of ephemeral project
	// names doesn't leak a per-project RWMutex forever.
	defer m.locks.Delete(name)

	dir := m.projectDir(name)
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("project %q does not exist", name)
	} else if err != nil {
		return fmt.Errorf("stat project %q: %w", name, err)
	}

	if err := m.deployer.Remove(ctx, name, removeVolumes, status); err != nil {
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
	if err := validateProjectName(name); err != nil {
		return nil, err
	}
	mu := m.projectLock(name)
	mu.RLock()
	defer mu.RUnlock()

	dir := m.projectDir(name)
	repo, err := git.PlainOpen(dir)
	if err != nil {
		return nil, fmt.Errorf("opening project %q: %w", name, err)
	}

	logIter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, fmt.Errorf("reading log: %w", err)
	}

	versions := make([]Version, 0, 32)
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
	if err := validateProjectName(name); err != nil {
		return err
	}
	if commitHash == "" {
		return fmt.Errorf("commit_hash is required")
	}
	mu := m.projectLock(name)
	mu.Lock()
	defer mu.Unlock()

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

	reader, err := file.Reader()
	if err != nil {
		return fmt.Errorf("opening file reader: %w", err)
	}
	composeYAML, err := readAllAndClose(reader)
	if err != nil {
		return fmt.Errorf("reading file contents: %w", err)
	}

	composePath := filepath.Join(dir, composeFilename)
	existing, readErr := os.ReadFile(composePath)
	if readErr != nil && !errors.Is(readErr, os.ErrNotExist) {
		return fmt.Errorf("reading existing compose file: %w", readErr)
	}

	shortHash := shortID(commitObj.Hash.String())
	if !bytes.Equal(existing, composeYAML) {
		if err := os.WriteFile(composePath, composeYAML, 0o644); err != nil {
			return fmt.Errorf("writing compose file: %w", err)
		}
		if err := commitCompose(repo, fmt.Sprintf("Rollback to %s", shortHash)); err != nil {
			return err
		}
	}

	project, err := ParseCompose(ctx, composeYAML, name)
	if err != nil {
		return fmt.Errorf("parsing compose file: %w", err)
	}

	if err := m.deployer.Deploy(ctx, project, status); err != nil {
		return fmt.Errorf("deploying: %w", err)
	}

	log.Info().Str("project", name).Str("rollback_to", shortHash).Msg("Rolled back and deployed")
	return nil
}

// commitCompose stages compose.yaml and commits with the given message. It
// assumes the caller holds the per-project write lock. No-op when nothing is
// staged (caller is expected to skip entirely in that case, but this is
// defensive in case staging detects no diff).
func commitCompose(repo *git.Repository, message string) error {
	wt, err := repo.Worktree()
	if err != nil {
		return fmt.Errorf("getting worktree: %w", err)
	}
	if _, err := wt.Add(composeFilename); err != nil {
		return fmt.Errorf("staging compose file: %w", err)
	}
	_, err = wt.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  commitAuthorName,
			Email: commitAuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	return nil
}

// readAllAndClose reads the reader to completion and closes it.
func readAllAndClose(r interface {
	Read([]byte) (int, error)
	Close() error
}) ([]byte, error) {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(r)
	closeErr := r.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return buf.Bytes(), nil
}

// resolveCommit finds a commit by full or short hash prefix.
func resolveCommit(repo *git.Repository, hash string) (*object.Commit, error) {
	// Full-hash fast path: skip the log walk entirely.
	if len(hash) == 40 {
		c, err := repo.CommitObject(plumbing.NewHash(hash))
		if err == nil {
			return c, nil
		}
		// fall through to the prefix walk in case the caller passed a 40-char
		// string that doesn't resolve (unlikely but not impossible).
	}

	iter, err := repo.Log(&git.LogOptions{})
	if err != nil {
		return nil, err
	}

	var match *object.Commit
	err = iter.ForEach(func(c *object.Commit) error {
		if strings.HasPrefix(c.Hash.String(), hash) {
			if match != nil {
				return fmt.Errorf("ambiguous commit prefix %q", hash)
			}
			match = c
			// Keep walking to detect ambiguity.
		}
		return nil
	})
	if err != nil && !errors.Is(err, storer.ErrStop) {
		return nil, err
	}
	if match == nil {
		return nil, fmt.Errorf("commit %q not found", hash)
	}
	return match, nil
}

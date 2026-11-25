package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile_CreatesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")

	require.NoError(t, WriteFile(path, func(w io.Writer) error {
		_, err := w.Write([]byte("hello"))
		return err
	}))

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "hello", string(data))
}

func TestWriteFile_FailedEncoderKeepsOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	require.NoError(t, os.WriteFile(path, []byte("original"), 0600))

	err := WriteFile(path, func(w io.Writer) error {
		w.Write([]byte("partial"))
		return errors.New("encoder blew up")
	})
	assert.Error(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "original", string(data))
}

func TestWriteFile_ConcurrentWritesDontInterleave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "notifications.yml")

	var wg sync.WaitGroup
	for i := range 50 {
		wg.Go(func() {
			// Different lengths, so an interleaved write would leave a stale tail.
			content := strings.Repeat(fmt.Sprint(i%10), 100+i*37)
			assert.NoError(t, WriteFile(path, func(w io.Writer) error {
				_, err := w.Write([]byte(content))
				return err
			}))
		})
	}
	wg.Wait()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NotEmpty(t, data)
	assert.Equal(t, strings.Repeat(string(data[0]), len(data)), string(data))
}

// A save that rendered first must also land first, or a stale render overwrites a newer one.
func TestWriteFile_RenderIsSerialized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	entered, release := make(chan struct{}), make(chan struct{})
	secondRendered := make(chan struct{})

	var wg sync.WaitGroup
	wg.Go(func() {
		assert.NoError(t, WriteFile(path, func(w io.Writer) error {
			close(entered)
			<-release
			_, err := w.Write([]byte("old"))
			return err
		}))
	})
	<-entered
	wg.Go(func() {
		assert.NoError(t, WriteFile(path, func(w io.Writer) error {
			close(secondRendered)
			_, err := w.Write([]byte("new"))
			return err
		}))
	})

	select {
	case <-secondRendered:
		t.Error("second render ran while the first save was still in progress")
	case <-time.After(100 * time.Millisecond):
	}
	close(release)
	wg.Wait()

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "new", string(data))
}

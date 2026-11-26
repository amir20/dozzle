package cloud

import (
	"os"
	"path/filepath"
	"testing"
)

// Every NewClient in this package's tests would otherwise write
// ./data/install-id into the source tree.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "dozzle-install-id")
	if err != nil {
		panic(err)
	}
	installIDPath = filepath.Join(dir, "install-id")
	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func TestLoadInstallIDIsStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data", "install-id")
	first := loadInstallID(path)
	if first == "" {
		t.Fatal("empty install id")
	}
	if again := loadInstallID(path); again != first {
		t.Fatalf("install id changed across loads: %q then %q", first, again)
	}
	if other := loadInstallID(filepath.Join(t.TempDir(), "install-id")); other == first {
		t.Fatal("two data folders got the same install id")
	}
}

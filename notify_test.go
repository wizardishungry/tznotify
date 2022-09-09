package tznotify_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	. "jonwillia.ms/tznotify"
)

const testTimeout = 1 * time.Second

func TestGlobal(t *testing.T) {
	w, err := New()
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(w.Close)

	select {
	case err := <-w.Errors():
		t.Fatalf("Errors: %v", err)
	case loc := <-w.Locations():
		t.Logf("location is %v", loc)
	case <-time.After(testTimeout):
		t.Logf("timeout")
	}
}
func TestUpdates(t *testing.T) {
	path, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	t.Cleanup(func() { os.RemoveAll(path) })

	symlink := filepath.Join([]string{path, "localtime"}...)
	symlink2 := filepath.Join([]string{path, "localtime2"}...)

	const (
		firstLink  = "/var/db/timezone/zoneinfo/America/New_York" // mac specific?
		secondLink = "/var/db/timezone/zoneinfo/America/Phoenix"
		thirdLink  = "/var/db/timezone/zoneinfo/America/Denver"
	)

	err = os.Symlink(firstLink, symlink)
	if err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	w, err := NewFromPath(symlink)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(w.Close)

	select {
	case err := <-w.Errors():
		t.Fatalf("Errors: %v", err)
	case loc := <-w.Locations():
		t.Logf("location is %v", loc)
	case <-time.After(testTimeout):
		t.Fatalf("timeout")
	}

	// "atomic" symlink
	err = os.Symlink(secondLink, symlink2)
	if err != nil {
		t.Fatalf("Symlink: %v", err)
	}
	err = os.Rename(symlink2, symlink)
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}

	select {
	case err := <-w.Errors():
		t.Fatalf("Errors: %v", err)
	case loc := <-w.Locations():
		t.Logf("location is %v", loc)
	case <-time.After(testTimeout):
		t.Fatalf("timeout")
	}

	// non atomic symlink (ln -sf)
	err = os.Remove(symlink)
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	select { // should not update
	case err := <-w.Errors(): // should error
		t.Logf("Errors: %v", err)
	case loc := <-w.Locations():
		t.Fatalf("location is %v", loc)
	case <-time.After(testTimeout):
		t.Fatalf("timeout")
	}

	err = os.Symlink(thirdLink, symlink)
	if err != nil {
		t.Fatalf("Symlink: %v", err)
	}

	timeout := time.After(testTimeout)
LOOP:
	for {
		select {
		case err := <-w.Errors():
			t.Logf("Errors: %v", err)
		case loc := <-w.Locations():
			t.Logf("location is %v", loc)
			break LOOP
		case <-timeout:
			t.Fatalf("timeout")
		}
	}

}

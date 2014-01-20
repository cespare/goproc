package proc

import (
	"testing"
)

func TestMounts(t *testing.T) {
	mounts, err := Mounts()
	if err != nil {
		t.Fatal(err)
	}
	// Check that there's a / entry.
	found := false
	for _, entry := range mounts {
		if entry.File == "/" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Could not locate / in mounts")
	}
}

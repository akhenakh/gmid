package library

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCopyInto(t *testing.T) {
	srcDir := t.TempDir()
	root := t.TempDir()
	src := filepath.Join(srcDir, "Song.mid")
	writeFile(t, src, "MThd")

	dst, copied, err := CopyInto(src, root)
	if err != nil {
		t.Fatal(err)
	}
	if !copied {
		t.Fatal("expected a copy")
	}
	if filepath.Dir(dst) != root {
		t.Fatalf("dst %q not in %q", dst, root)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "MThd" {
		t.Fatalf("copied data %q", data)
	}

	again, copied, err := CopyInto(src, root)
	if err != nil {
		t.Fatal(err)
	}
	if !copied || again == dst {
		t.Fatalf("collision should make a new name, got %q (copied=%v)", again, copied)
	}
	if filepath.Base(again) != "Song (2).mid" {
		t.Fatalf("collision name %q", filepath.Base(again))
	}
}

func TestCopyIntoAlreadyInside(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "Inside.kar")
	writeFile(t, src, "data")

	dst, copied, err := CopyInto(src, root)
	if err != nil {
		t.Fatal(err)
	}
	if copied || dst != src {
		t.Fatalf("inside library should be a no-op, got %q copied=%v", dst, copied)
	}
}

func TestCopyIntoRejectsUnsupported(t *testing.T) {
	src := filepath.Join(t.TempDir(), "notes.txt")
	writeFile(t, src, "data")
	if _, _, err := CopyInto(src, t.TempDir()); err == nil {
		t.Fatal("expected an unsupported-type error")
	}
}

func TestCopyIntoRejectsDirectory(t *testing.T) {
	src := t.TempDir()
	if _, _, err := CopyInto(src, t.TempDir()); err == nil {
		t.Fatal("expected a directory error")
	}
}

func TestCopyIntoLowercaseExtension(t *testing.T) {
	src := filepath.Join(t.TempDir(), "Upper.MIDI")
	writeFile(t, src, "data")
	if _, copied, err := CopyInto(src, t.TempDir()); err != nil || !copied {
		t.Fatalf("uppercase extension: copied=%v err=%v", copied, err)
	}
}

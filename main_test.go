package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func writeTemp(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.ini")
	if err := os.WriteFile(path, []byte(contents), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestProcessFileCheckFlagsUnformatted(t *testing.T) {
	path := writeTemp(t, "[a]\nkey=value\n")

	err := processFile(path, false, false, false, true, false)
	if !errors.Is(err, errNotFormatted) {
		t.Fatalf("got err %v, want errNotFormatted", err)
	}

	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != "[a]\nkey=value\n" {
		t.Errorf("-check modified the file: %q", got)
	}
}

func TestProcessFileCheckAcceptsFormatted(t *testing.T) {
	path := writeTemp(t, "[a]\nkey = value\n")

	if err := processFile(path, false, false, false, true, false); err != nil {
		t.Errorf("got err %v, want nil for already-formatted input", err)
	}
}

func TestProcessFileCheckIgnoresWrite(t *testing.T) {
	contents := "[a]\nkey=value\n"
	path := writeTemp(t, contents)

	if err := processFile(path, true, false, false, true, false); !errors.Is(err, errNotFormatted) {
		t.Fatalf("got err %v, want errNotFormatted", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Errorf("-check wrote changes despite -w: %q", got)
	}
}

func TestProcessFileDupeCheckFindsDuplicates(t *testing.T) {
	contents := "[a]\nkey=1\nkey=2\n"
	path := writeTemp(t, contents)

	err := processFile(path, false, false, false, false, true)
	if !errors.Is(err, errDuplicateKeys) {
		t.Fatalf("got err %v, want errDuplicateKeys", err)
	}

	got, readErr := os.ReadFile(path)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(got) != contents {
		t.Errorf("-dupe-check modified the file: %q", got)
	}
}

func TestProcessFileDupeCheckAcceptsUniqueKeys(t *testing.T) {
	path := writeTemp(t, "[a]\nkey=1\nother=2\n")

	if err := processFile(path, false, false, false, false, true); err != nil {
		t.Errorf("got err %v, want nil when no keys repeat", err)
	}
}

func TestProcessFileDupeCheckIgnoresWriteAndSort(t *testing.T) {
	contents := "[a]\nkey=1\nkey=2\n"
	path := writeTemp(t, contents)

	if err := processFile(path, true, true, false, false, true); !errors.Is(err, errDuplicateKeys) {
		t.Fatalf("got err %v, want errDuplicateKeys", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Errorf("-dupe-check wrote changes despite -w: %q", got)
	}
}

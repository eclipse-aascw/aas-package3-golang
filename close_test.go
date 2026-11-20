// SPDX-FileCopyrightText: 2026 aas-package3-golang AUTHORS
//
// SPDX-License-Identifier: MIT

package aasx

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// createPackageWithSpec creates a valid package at pth with a single spec part.
func createPackageWithSpec(t *testing.T, pth string) {
	t.Helper()
	pkg, err := NewPackaging().Create(pth)
	if err != nil {
		t.Fatalf("Failed to create package: %v", err)
	}
	part, err := pkg.PutPart(mustParseURL("/aasx/data.json"), "text/json", []byte("{}"))
	if err != nil {
		t.Fatalf("Failed to put part: %v", err)
	}
	if err := pkg.MakeSpec(part); err != nil {
		t.Fatalf("Failed to make spec: %v", err)
	}
	if err := pkg.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
}

func readSpecURIs(t *testing.T, pth string) []string {
	t.Helper()
	pkg, err := NewPackaging().OpenRead(pth)
	if err != nil {
		t.Fatalf("Failed to open package for reading: %v", err)
	}
	defer pkg.Close()
	specs, err := pkg.Specs()
	if err != nil {
		t.Fatalf("Failed to read specs: %v", err)
	}
	result := make([]string, 0, len(specs))
	for _, spec := range specs {
		result = append(result, spec.URI.String())
	}
	return result
}

// backdate sets the modification time of pth into the past so that any
// subsequent write is observable.
func backdate(t *testing.T, pth string) time.Time {
	t.Helper()
	past := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(pth, past, past); err != nil {
		t.Fatalf("Failed to set file times: %v", err)
	}
	return past
}

func assertUntouched(t *testing.T, pth string, content []byte, mtime time.Time) {
	t.Helper()
	info, err := os.Stat(pth)
	if err != nil {
		t.Fatalf("Failed to stat: %v", err)
	}
	if !info.ModTime().Equal(mtime) {
		t.Errorf("Expected modification time %v, got %v", mtime, info.ModTime())
	}
	got, err := os.ReadFile(pth)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Error("Expected the file content to be left as-is")
	}
}

func TestCloseOfCreatedPackageWritesValidPackage(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()
	pth := filepath.Join(tmpdir, "new.aasx")

	pkg, err := NewPackaging().Create(pth)
	if err != nil {
		t.Fatalf("Failed to create package: %v", err)
	}
	if err := pkg.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	if specs := readSpecURIs(t, pth); len(specs) != 0 {
		t.Errorf("Expected no specs, got %v", specs)
	}
}

func TestCloseFlushesModifications(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()
	pth := filepath.Join(tmpdir, "package.aasx")
	createPackageWithSpec(t, pth)

	pkg, err := NewPackaging().OpenReadWrite(pth)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	part, err := pkg.PutPart(mustParseURL("/aasx/other.json"), "text/json", []byte("{}"))
	if err != nil {
		t.Fatalf("Failed to put part: %v", err)
	}
	if err := pkg.MakeSpec(part); err != nil {
		t.Fatalf("Failed to make spec: %v", err)
	}
	if err := pkg.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}

	if specs := readSpecURIs(t, pth); len(specs) != 2 {
		t.Errorf("Expected two specs after close, got %v", specs)
	}
}

func TestCloseWithoutModificationsLeavesFileAlone(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()
	pth := filepath.Join(tmpdir, "package.aasx")
	createPackageWithSpec(t, pth)
	content, err := os.ReadFile(pth)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	mtime := backdate(t, pth)

	pkg, err := NewPackaging().OpenReadWrite(pth)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	specs, err := pkg.Specs()
	if err != nil || len(specs) != 1 {
		t.Fatalf("Expected one spec, got %v (err: %v)", specs, err)
	}

	// Operations which do not change anything must not mark the package modified.
	if err := pkg.MakeSpec(specs[0]); err != nil {
		t.Fatalf("Failed to make spec: %v", err)
	}
	if err := pkg.UnsetThumbnail(); err != nil {
		t.Fatalf("Failed to unset thumbnail: %v", err)
	}
	if err := pkg.DeletePart(&Part{URI: mustParseURL("/does/not/exist")}); err != nil {
		t.Fatalf("Failed to delete part: %v", err)
	}

	if err := pkg.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
	assertUntouched(t, pth, content, mtime)
}

func TestCloseAfterFlushDoesNotWriteAgain(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()
	pth := filepath.Join(tmpdir, "package.aasx")
	createPackageWithSpec(t, pth)

	pkg, err := NewPackaging().OpenReadWrite(pth)
	if err != nil {
		t.Fatalf("Failed to open: %v", err)
	}
	if _, err := pkg.PutPart(
		mustParseURL("/aasx/other.json"), "text/json", []byte("{}")); err != nil {
		t.Fatalf("Failed to put part: %v", err)
	}
	if err := pkg.Flush(); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
	content, err := os.ReadFile(pth)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	mtime := backdate(t, pth)

	if err := pkg.Close(); err != nil {
		t.Fatalf("Failed to close: %v", err)
	}
	assertUntouched(t, pth, content, mtime)
}

func TestDiscardDropsModifications(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()
	pth := filepath.Join(tmpdir, "package.aasx")
	createPackageWithSpec(t, pth)
	content, err := os.ReadFile(pth)
	if err != nil {
		t.Fatalf("Failed to read: %v", err)
	}
	mtime := backdate(t, pth)

	failingModification := func() error {
		pkg, err := NewPackaging().OpenReadWrite(pth)
		if err != nil {
			return err
		}
		defer pkg.Discard()

		specs, err := pkg.Specs()
		if err != nil {
			return err
		}
		if err := pkg.UnmakeSpec(specs[0]); err != nil {
			return err
		}
		return errors.New("something went wrong halfway")
	}
	if err := failingModification(); err == nil {
		t.Fatal("Expected an error")
	}

	assertUntouched(t, pth, content, mtime)
	if specs := readSpecURIs(t, pth); len(specs) != 1 {
		t.Errorf("Expected the spec to survive, got %v", specs)
	}
}

func TestDiscardAfterCloseIsNoOp(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()
	pth := filepath.Join(tmpdir, "package.aasx")
	createPackageWithSpec(t, pth)

	successfulModification := func() error {
		pkg, err := NewPackaging().OpenReadWrite(pth)
		if err != nil {
			return err
		}
		defer pkg.Discard()

		specs, err := pkg.Specs()
		if err != nil {
			return err
		}
		if err := pkg.UnmakeSpec(specs[0]); err != nil {
			return err
		}
		return pkg.Close()
	}
	if err := successfulModification(); err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if specs := readSpecURIs(t, pth); len(specs) != 0 {
		t.Errorf("Expected no specs, got %v", specs)
	}
}

type failingWriteStream struct {
	readWriteSeeker
	fail bool
}

func (s *failingWriteStream) Write(p []byte) (int, error) {
	if s.fail {
		return 0, errors.New("write failed")
	}
	return s.readWriteSeeker.Write(p)
}

func TestCloseKeepsPackageOpenIfFlushFails(t *testing.T) {
	stream := &failingWriteStream{readWriteSeeker: readWriteSeeker{buf: &bytes.Buffer{}}}
	pkg, err := NewPackaging().CreateInStream(stream)
	if err != nil {
		t.Fatalf("Failed to create package: %v", err)
	}
	if _, err := pkg.PutPart(
		mustParseURL("/aasx/data.json"), "text/json", []byte("{}")); err != nil {
		t.Fatalf("Failed to put part: %v", err)
	}

	stream.fail = true
	if err := pkg.Close(); err == nil {
		t.Fatal("Expected Close to fail")
	}

	// The package is still usable, so the caller can retry.
	if _, err := pkg.Parts(); err != nil {
		t.Fatalf("Expected the package to remain open, got %v", err)
	}

	stream.fail = false
	if err := pkg.Close(); err != nil {
		t.Fatalf("Expected the retried close to succeed, got %v", err)
	}
	if stream.buf.Len() == 0 {
		t.Error("Expected the package to be written on the retried close")
	}
	if _, err := pkg.Parts(); !errors.Is(err, ErrPackageClosed) {
		t.Errorf("Expected ErrPackageClosed, got %v", err)
	}
}

// SPDX-FileCopyrightText: 2026 aas-package3-golang AUTHORS
//
// SPDX-License-Identifier: MIT

package aasx

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
)

// truncatableReadWriteSeeker is an in-memory stream which supports truncation.
type truncatableReadWriteSeeker struct {
	readWriteSeeker
}

func (s *truncatableReadWriteSeeker) Truncate(size int64) error {
	s.buf.Truncate(int(size))
	return nil
}

// shrinkPackageInStream puts a large part into a package created in stream,
// flushes, deletes the part and flushes again. It returns the size of the
// stream after the first flush and the error of the second flush.
func shrinkPackageInStream(t *testing.T, stream io.ReadWriteSeeker) (int64, error) {
	t.Helper()
	pkg, err := NewPackaging().CreateInStream(stream)
	if err != nil {
		t.Fatalf("Failed to create package: %v", err)
	}
	defer pkg.Close()

	// Incompressible content so that the shrink is substantial.
	content := make([]byte, 64*1024)
	for i := range content {
		content[i] = byte((i * 7919) ^ (i >> 3))
	}
	part, err := pkg.PutPart(
		mustParseURL("/aasx/large.bin"), "application/octet-stream", content)
	if err != nil {
		t.Fatalf("Failed to put part: %v", err)
	}
	if err := pkg.Flush(); err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}
	sizeBefore, err := stream.Seek(0, io.SeekEnd)
	if err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	if err := pkg.DeletePart(part); err != nil {
		t.Fatalf("Failed to delete part: %v", err)
	}
	return sizeBefore, pkg.Flush()
}

func assertOnlyOriginPart(t *testing.T, stream io.ReadSeeker) {
	t.Helper()
	pkg, err := NewPackaging().OpenReadFromStream(stream)
	if err != nil {
		t.Fatalf("Failed to open the flushed package: %v", err)
	}
	defer pkg.Close()
	parts, err := pkg.Parts()
	if err != nil {
		t.Fatalf("Failed to list parts: %v", err)
	}
	for _, part := range parts {
		if part.URI.String() == "/aasx/large.bin" {
			t.Errorf("Expected the deleted part to be gone, got parts %v", parts)
		}
	}
}

func TestFlushTruncatesFileStream(t *testing.T) {
	tmpdir, cleanup := temporaryDirectory(t)
	defer cleanup()

	file, err := os.Create(filepath.Join(tmpdir, "package.aasx"))
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	defer file.Close()

	sizeBefore, err := shrinkPackageInStream(t, file)
	if err != nil {
		t.Fatalf("Failed to flush the shrunk package: %v", err)
	}

	info, err := file.Stat()
	if err != nil {
		t.Fatalf("Failed to stat: %v", err)
	}
	if info.Size() >= sizeBefore {
		t.Errorf("Expected the file to shrink below %d bytes, got %d", sizeBefore, info.Size())
	}
	assertOnlyOriginPart(t, file)
}

func TestFlushTruncatesInMemoryStream(t *testing.T) {
	stream := &truncatableReadWriteSeeker{readWriteSeeker{buf: &bytes.Buffer{}}}

	sizeBefore, err := shrinkPackageInStream(t, stream)
	if err != nil {
		t.Fatalf("Failed to flush the shrunk package: %v", err)
	}

	if int64(stream.buf.Len()) >= sizeBefore {
		t.Errorf(
			"Expected the stream to shrink below %d bytes, got %d", sizeBefore, stream.buf.Len())
	}
	assertOnlyOriginPart(t, bytes.NewReader(stream.buf.Bytes()))
}

func TestFlushRefusesToShrinkNonTruncatableStream(t *testing.T) {
	stream := &readWriteSeeker{buf: &bytes.Buffer{}}

	sizeBefore, err := shrinkPackageInStream(t, stream)
	if !errors.Is(err, ErrStreamNotTruncatable) {
		t.Fatalf("Expected ErrStreamNotTruncatable, got %v", err)
	}

	// The previous package must be left intact.
	if int64(stream.buf.Len()) != sizeBefore {
		t.Errorf("Expected the stream to keep %d bytes, got %d", sizeBefore, stream.buf.Len())
	}
	pkg, err := NewPackaging().OpenReadFromStream(bytes.NewReader(stream.buf.Bytes()))
	if err != nil {
		t.Fatalf("Expected the previous package to remain readable: %v", err)
	}
	defer pkg.Close()
	if _, err := pkg.MustPart(mustParseURL("/aasx/large.bin")); err != nil {
		t.Errorf("Expected the previous package to keep its part: %v", err)
	}
}

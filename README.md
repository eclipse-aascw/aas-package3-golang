# aas-package3-golang

[![Test](https://github.com/aas-core-works/aas-package3-golang/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/aas-core-works/aas-package3-golang/actions/workflows/test.yml)
[![Check style](https://github.com/aas-core-works/aas-package3-golang/actions/workflows/check-style.yml/badge.svg)](https://github.com/aas-core-works/aas-package3-golang/actions/workflows/check-style.yml)
[![Coverage Status](https://coveralls.io/repos/github/aas-core-works/aas-package3-golang/badge.svg?branch=main)](https://coveralls.io/github/aas-core-works/aas-package3-golang?branch=main)
[![Go Reference](https://pkg.go.dev/badge/github.com/aas-core-works/aas-package3-golang/v2.svg)](https://pkg.go.dev/github.com/aas-core-works/aas-package3-golang/v2)

Aas-package3-golang is a library for reading and writing packaged file format of an [Asset Administration Shell (AAS)] in Go.

[Asset Administration Shell (AAS)]: https://www.plattform-i40.de/PI40/Redaktion/DE/Downloads/Publikation/Details_of_the_Asset_Administration_Shell_Part1_V3.html

## Status

The library is thoroughly tested and meant to be used in production.

The library is written in pure Go with no external dependencies beyond the standard library. Go 1.18 or later is required.

## Documentation

The full documentation is available at [doc/index.md](doc/index.md).

### Teaser

Here are short snippets to demonstrate how you can use the library.

To create and write to a package:

```go
import (
    "net/url"
    aasx "github.com/aas-core-works/aas-package3-golang/v2"
)

// General packaging handler to be shared across the program
packaging := aasx.NewPackaging()

// Create a package
specContent := []byte(`{"aas": "..."}`)
thumbnailContent := []byte{...} // PNG bytes
supplementaryContent := []byte{...} // PDF bytes

pkg, err := packaging.Create("/path/to/some/file.aasx")
if err != nil {
    log.Fatal(err)
}
// Discard is a no-op after Close, so it only drops changes on early exits.
defer pkg.Discard()

specURI, _ := url.Parse("/aasx/some-company/data.json")
spec, _ := pkg.PutPart(specURI, "text/json", specContent)
pkg.MakeSpec(spec)

thumbURI, _ := url.Parse("/some-thumbnail.png")
thumb, _ := pkg.PutPart(thumbURI, "image/png", thumbnailContent)
pkg.SetThumbnail(thumb)

supplURI, _ := url.Parse("/aasx-suppl/some-company/some-manual.pdf")
suppl, _ := pkg.PutPart(supplURI, "application/pdf", supplementaryContent)
pkg.RelateSupplementaryToSpec(suppl, spec)

// Close flushes the changes and closes the package.
if err := pkg.Close(); err != nil {
    log.Fatal(err)
}
```

To read from the package:

```go
import (
    "net/url"
    aasx "github.com/aas-core-works/aas-package3-golang/v2"
)

// General packaging handler to be shared across the program
packaging := aasx.NewPackaging()

// Read from the package
pkg, err := packaging.OpenRead("/path/to/some/file.aasx")
if err != nil {
    log.Fatal(err)
}
defer pkg.Close()

// Read the specs
specsByContentType, _ := pkg.SpecsByContentType()
if jsonSpecs, ok := specsByContentType["text/json"]; ok {
    spec := jsonSpecs[0]
    specContent, _ := spec.ReadAllBytes()
    // Do something with the spec content
}

// Read the thumbnail
thumbnail, _ := pkg.Thumbnail()
if thumbnail != nil {
    thumbnailContent, _ := thumbnail.ReadAllBytes()
    // Do something with the thumbnail content
}

// Read a supplementary file
supplURI, _ := url.Parse("/aasx-suppl/some-company/some-manual.pdf")
suppl, err := pkg.MustPart(supplURI)
if err != nil {
    log.Fatal(err)
}
supplementaryContent, _ := suppl.ReadAllBytes()
```

Please see the full documentation at [doc/index.md](doc/index.md) for more details.

## Installation

```bash
go get github.com/aas-core-works/aas-package3-golang/v2
```

## API Overview

### Types

| Type | Description |
| ------ | ------------- |
| `Packaging` | Factory for opening and creating AASX packages |
| `PackageRead` | Read-only access to an AASX package |
| `PackageReadWrite` | Read and write access to an AASX package |
| `PackageWriter` | Append-only, bounded-memory package creation |
| `Part` | Represents a part within an AASX package |

### Packaging Methods

| Method | Description |
| -------- | ------------- |
| `Create(path)` | Create a new AASX package at the given path |
| `CreateInStream(stream)` | Create a new AASX package in a stream; if the stream lacks `Truncate`, shrinking the package fails with `ErrStreamNotTruncatable` |
| `CreateInTruncatableStream(stream)` | Same, but the stream must implement `Truncate` (type-safe) |
| `CreateWriter(writer)` | Create an append-only package directly on an `io.Writer` |
| `OpenRead(path, options...)` | Lazily open an AASX package for reading |
| `OpenReadFromStream(stream, options...)` | Lazily open from an `io.ReadSeeker` |
| `OpenReadFromReaderAt(reader, size, options...)` | Lazily open from an `io.ReaderAt` |
| `OpenReadWrite(path)` | Open an AASX package for read/write |
| `OpenReadWriteFromStream(stream)` | Open an AASX package from a stream for read/write; if the stream lacks `Truncate`, shrinking the package fails with `ErrStreamNotTruncatable` |
| `OpenReadWriteFromTruncatableStream(stream)` | Same, but the stream must implement `Truncate` (type-safe) |

### PackageRead Methods

| Method | Description |
| -------- | ------------- |
| `Specs()` | List all AAS spec parts |
| `SpecsByContentType()` | List specs grouped by MIME type |
| `IsSpec(part)` | Check if a part is a spec |
| `SupplementariesFor(spec)` | List supplementary parts for a spec |
| `SupplementaryRelationships()` | List all supplementary relationships |
| `Parts()` | List all parts in the package, including parts no relationship targets |
| `FindPart(uri)` | Find a part by URI (returns nil if not found) |
| `MustPart(uri)` | Get a part by URI (returns error if not found) |
| `Thumbnail()` | Get the package thumbnail |
| `Close()` | Close the package |

### PackageReadWrite Methods

Inherits all `PackageRead` methods, plus:

| Method | Description |
| -------- | ------------- |
| `PutPart(uri, contentType, content)` | Write a part to the package |
| `PutPartFromStream(uri, contentType, stream)` | Write a part from a stream |
| `DeletePart(part)` | Remove a part from the package |
| `MakeSpec(part)` | Mark a part as a spec |
| `UnmakeSpec(part)` | Remove spec relationship |
| `RelateSupplementaryToSpec(supplementary, spec)` | Create supplementary relationship |
| `UnrelateSupplementaryFromSpec(supplementary, spec)` | Remove supplementary relationship |
| `SetThumbnail(part)` | Set the package thumbnail |
| `UnsetThumbnail()` | Remove the package thumbnail |
| `Flush()` | Write the package now, e.g., in the middle of a session |
| `Close()` | Flush if modified, then close the package |
| `Discard()` | Close without flushing, dropping unflushed changes |

### Bounded-memory I/O

Read-only packages keep ZIP payloads compressed until `Part.Stream`, `ReadAllBytes`, or
`ReadAllText` is called. Keep the package open until all part streams are closed. CRC and
decompression errors can therefore be reported while reading a part rather than by
`OpenRead`.

Use reader options such as `WithMaxPartCount`, `WithMaxOPCMetadataBytes`,
`WithMaxPartExpandedBytes`, and `WithMaxTotalExpandedBytes` for untrusted packages.
All limits default to zero, which means unlimited.

For bounded-memory output, use `CreateWriter` and finish with `PackageWriter.Close`.
The writer is append-only: every part is consumed before `PutPartFromStream` returns,
and relationships plus content types are emitted during `Close`. The destination is
caller-owned and may contain a partial ZIP if an operation fails. The mutable
`PackageReadWrite` API remains available when overwrite and delete operations are needed.

## Versioning

The name of the library indicates the supported version of the [Asset Administration Shell (AAS)].

In case of `aas-package3-golang`, this means that Version 3 of the [Asset Administration Shell (AAS)] is supported.

We follow [Semantic Versioning] to version the library.
The version X.Y.Z indicates:

[Semantic Versioning]: http://semver.org/spec/v1.0.0.html

* X is the major version (backward-incompatible),
* Y is the minor version (backward-compatible), and
* Z is the patch version (backward-compatible bug fix).

## Change Log

### v2.2.0 (2026-11-26)

* Flush on close unless discarded (#21)

  `PackageReadWrite.Close` now flushes the package if it has been modified,
  and the new `Discard` closes without flushing. If you `defer pkg.Close()`,
  an early return now saves partial edits; defer `pkg.Discard()` instead and
  return `pkg.Close()` once all edits succeed. See
  [Migrating from v2.1.0 and Earlier](doc/getting-started/installation.md#migrating-from-v210-and-earlier).

* Truncate streams when flushing a smaller package (#22)

  Flushing a package that shrank, e.g., after deleting a part, used to leave
  stale bytes at the end of the stream and corrupt the package. Streams that
  implement `Truncate(int64) error`, such as `*os.File`, are now truncated.
  For other streams, the flush fails with `ErrStreamNotTruncatable` and leaves
  the previous package intact. `CreateInTruncatableStream` and
  `OpenReadWriteFromTruncatableStream` let the compiler check that the stream
  can be truncated.

### v2.1.0 (2026-10-13)

* Add `PackageRead.Parts` to enumerate package parts (#15)

  Parts which no relationship targets, such as the supplementary files of
  the IDTA-01005 example package, could not be discovered before. `Parts`
  lists all parts sorted by URI, excluding the OPC metadata entries.

* Compare part names independent of percent-encoding (#16)

  Parts with percent-encoded names, as the OPC standard requires, could not be
  found before. Part names are now compared independent of encoding and ASCII
  case, as ISO/IEC 29500-2 requires. Parts read from a package are written
  back under their original ZIP entry name, and `PackageWriter` now writes
  percent-encoded entry names.

### v2.0.0 (2026-07-20)

* Make AASX read/write stream APIs lazy and bounded-memory (#14)

  Read-only packages load payloads lazily and support configurable limits for
  untrusted input. `OpenReadFromReaderAt` reads from random-access sources,
  and the append-only `PackageWriter` streams parts to an `io.Writer` with
  bounded memory. The module path is now
  `github.com/aas-core-works/aas-package3-golang/v2`; see
  [Migrating from v1](doc/getting-started/installation.md#migrating-from-v1).

### v1.0.0 (2026-05-27)

This is the first stable release. The release candidates stood the test of
time, so the code is the same as in v1.0.0-rc.2.

### v1.0.0-rc.2 (2026-05-21)

* Support deprecated AASX relationship URIs (#8)

  Relationship types with the deprecated prefix
  `http://www.admin-shell.io/aasx/relationships/` are now recognized as
  their preferred `http://admin-shell.io/aasx/relationships/` counterparts.

* Improve path normalization (#10)

  ZIP entries with Windows path separators are read correctly, and part paths
  are normalized consistently.

### v1.0.0-rc1 (2026-03-05)

This is the initial release, ready for first integration tests.

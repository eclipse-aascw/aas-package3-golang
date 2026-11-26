# Change Log

## v2.2.0 (2026-11-26)

* Flush on close unless discarded (#21)

  `PackageReadWrite.Close` now flushes the package if it has been modified,
  and the new `Discard` closes without flushing. If you `defer pkg.Close()`,
  an early return now saves partial edits; defer `pkg.Discard()` instead and
  return `pkg.Close()` once all edits succeed. See
  [Migrating from v2.1.0 and Earlier](getting-started/installation.md#migrating-from-v210-and-earlier).

* Truncate streams when flushing a smaller package (#22)

  Flushing a package that shrank, e.g., after deleting a part, used to leave
  stale bytes at the end of the stream and corrupt the package. Streams that
  implement `Truncate(int64) error`, such as `*os.File`, are now truncated.
  For other streams, the flush fails with `ErrStreamNotTruncatable` and leaves
  the previous package intact. `CreateInTruncatableStream` and
  `OpenReadWriteFromTruncatableStream` let the compiler check that the stream
  can be truncated.

## v2.1.0 (2026-10-13)

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

## v2.0.0 (2026-07-20)

* Make AASX read/write stream APIs lazy and bounded-memory (#14)

  Read-only packages load payloads lazily and support configurable limits for
  untrusted input. `OpenReadFromReaderAt` reads from random-access sources,
  and the append-only `PackageWriter` streams parts to an `io.Writer` with
  bounded memory. The module path is now
  `github.com/aas-core-works/aas-package3-golang/v2`; see
  [Migrating from v1](getting-started/installation.md#migrating-from-v1).

## v1.0.0 (2026-05-27)

This is the first stable release. The release candidates stood the test of
time, so the code is the same as in v1.0.0-rc.2.

## v1.0.0-rc.2 (2026-05-21)

* Support deprecated AASX relationship URIs (#8)

  Relationship types with the deprecated prefix
  `http://www.admin-shell.io/aasx/relationships/` are now recognized as
  their preferred `http://admin-shell.io/aasx/relationships/` counterparts.

* Improve path normalization (#10)

  ZIP entries with Windows path separators are read correctly, and part paths
  are normalized consistently.

## v1.0.0-rc1 (2026-03-05)

This is the initial release, ready for first integration tests.

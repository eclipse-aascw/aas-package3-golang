# Installation

The library is available as a Go module.

You can add it to your project using `go get`:

```bash
go get github.com/aas-core-works/aas-package3-golang/v2
```

Then import it in your code:

```go
import aasx "github.com/aas-core-works/aas-package3-golang/v2"
```

## Requirements

- Go 1.18 or later
- No external dependencies (standard library only)

## Migrating from v2.1.0 and Earlier

`PackageReadWrite.Close` now flushes the package if it has been modified since it was
opened or last flushed. Up to v2.1.0, `Close` did not flush, so unflushed changes were
dropped. If you `defer pkg.Close()`, an early return or panic now saves partial edits.

Defer `Discard` instead, and explicitly return `Close` once all edits succeed.

Old pattern:

```go
pkg, err := packaging.OpenReadWrite(path)
if err != nil {
    return err
}
defer pkg.Close()
// ... modify pkg, returning early on errors ...
return pkg.Flush()
```

New pattern:

```go
pkg, err := packaging.OpenReadWrite(path)
if err != nil {
    return err
}
defer pkg.Discard() // no-op after a successful Close
// ... modify pkg, returning early on errors ...
return pkg.Close() // flushes and closes
```

Likewise, if you relied on `Close` to drop in-memory edits, call `Discard` instead.
`PackageRead.Close` and `PackageWriter.Close` are unaffected.
See [Closing and Flushing](read-write.md#closing-and-flushing) for details.

## Migrating from v1

Version 2 uses Go semantic import versioning, so update imports to include `/v2`.
Read-only packages are now lazy: `OpenRead` keeps its file open until `Close`, and
payload CRC, decompression, or truncation errors can be returned when a part is read.
Consume and close all part streams before closing the package. Existing mutable
`PackageReadWrite` operations remain materialized and retain their v1 behavior.

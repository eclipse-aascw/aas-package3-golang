# Read-Writing

We show here how you can open a package for both read/write operations.

We are going to use an instance `packaging` of `Packaging` to interact with the library.
Please see [entry-point.md](entry-point.md) for more details.

## Creating a New Package

Creating a new package is straightforward:

```go
pkg, err := packaging.Create("/path/to/some/file.aasx")
if err != nil {
    log.Fatal(err)
}
defer pkg.Discard()

// ... modify the package ...

if err := pkg.Close(); err != nil {
    log.Fatal(err)
}
```

You can also create a package in a stream (for example, to write to a buffer or network connection):

```go
var buf bytes.Buffer
stream := ... // any io.ReadWriteSeeker

pkg, err := packaging.CreateInStream(stream)
if err != nil {
    log.Fatal(err)
}
defer pkg.Discard()

// ... modify the package ...

if err := pkg.Close(); err != nil {
    log.Fatal(err)
}
```

Please do not forget to close the package when done.
See [Closing and Flushing](#closing-and-flushing) for why we defer `Discard` rather than `Close`.

## Creating a Package with Bounded Memory

`Create` and `CreateInStream` provide mutable packages and retain part content until
`Flush`. If you only need to create a package once, use the append-only `PackageWriter`
to copy each part directly into the output ZIP:

```go
destination := ... // any io.Writer
writer, err := packaging.CreateWriter(destination)
if err != nil {
    log.Fatal(err)
}

specURI, _ := url.Parse("/aasx/data.json")
spec, err := writer.PutPartFromStream(specURI, "application/json", specReader)
if err != nil {
    log.Fatal(err)
}
if err := writer.MakeSpec(spec); err != nil {
    log.Fatal(err)
}

if err := writer.Close(); err != nil {
    log.Fatal(err)
}
```

Each stream is completely consumed before `PutPartFromStream` returns. You can then call
`MakeSpec`, `RelateSupplementaryToSpec(supplementary, spec)`, or `SetThumbnail` with the
returned lightweight part handle. `Close` writes relationships and content types and
finalizes the ZIP.

The writer does not close `destination`. It does not support overwrite, delete, unrelate,
or rollback; use `PackageReadWrite` when mutation is required. If an input, output, or
`Close` operation fails, the destination can contain a partial package and should be
discarded.

## Opening a Package for Read/Writing

Opening a package for read/writing is similar to how we open a package for reading:

```go
pkg, err := packaging.OpenReadWrite("/path/to/some/file.aasx")
if err != nil {
    log.Fatal(err)
}
defer pkg.Discard()

// ... modify the package ...

if err := pkg.Close(); err != nil {
    log.Fatal(err)
}
```

## Putting Parts Together

The [Open Packaging Conventions] format is based on parts and relationships.
The parts represent the pieces of data, while relationships model how these pieces relate to each other.

[Open Packaging Conventions]: https://en.wikipedia.org/wiki/Open_Packaging_Conventions

We wanted to encapsulate as much as possible the underlying format, but we decided to keep the structure of [Open Packaging Conventions].
This means that you need to first write a part, and then establish its relation to other parts (or package itself).

The parts are put to the package using `PutPart`:

```go
import "net/url"

uri, _ := url.Parse("/aasx/some-company/data.json")
content := []byte("{}")

part, err := pkg.PutPart(uri, "text/json", content)
if err != nil {
    log.Fatal(err)
}
```

You can also use streams:

```go
import "net/url"

uri, _ := url.Parse("/aasx/data.json")
stream := ... // any io.Reader

part, err := pkg.PutPartFromStream(uri, "text/json", stream)
if err != nil {
    log.Fatal(err)
}
```

The `PutPart` function returns a `*Part` so that you can easily chain it with other functions (see below for examples).

### Overwriting & Deleting

If a part already exists at the given URI, it is silently overwritten.
Therefore you need to be careful when you overwrite a part and make sure that the relationships are updated accordingly.

This also applies when deleting the parts.
Since it would be unfortunately inefficient to enforce consistency, the library indeed allows you to make your package inconsistent (where relationships point to dangling parts).

To delete a part:

```go
err := pkg.DeletePart(part)
if err != nil {
    log.Fatal(err)
}
```

### Specs

You establish a part as a spec by calling `MakeSpec`:

```go
uri, _ := url.Parse("/aasx/some-company/data.json")
content := []byte("{}")

part, err := pkg.PutPart(uri, "text/json", content)
if err != nil {
    log.Fatal(err)
}

err = pkg.MakeSpec(part)
if err != nil {
    log.Fatal(err)
}
```

To remove a spec relationship:

```go
err := pkg.UnmakeSpec(part)
if err != nil {
    log.Fatal(err)
}
```

## Supplementary Parts

Similar to how you make parts into specs, you relate supplementary parts to the spec parts with `RelateSupplementaryToSpec`:

```go
specURI, _ := url.Parse("/aasx/data.xml")
specContent := []byte("<aas>...</aas>")
spec, err := pkg.PutPart(specURI, "application/xml", specContent)
if err != nil {
    log.Fatal(err)
}
pkg.MakeSpec(spec)

supplURI, _ := url.Parse("/aasx-suppl/manual.pdf")
supplContent := ... // PDF bytes
supplementary, err := pkg.PutPart(supplURI, "application/pdf", supplContent)
if err != nil {
    log.Fatal(err)
}

err = pkg.RelateSupplementaryToSpec(supplementary, spec)
if err != nil {
    log.Fatal(err)
}
```

The relation can also be undone:

```go
err := pkg.UnrelateSupplementaryFromSpec(supplementary, spec)
if err != nil {
    log.Fatal(err)
}
```

## Thumbnail

There can be only one thumbnail per package.

Similar to specs, you set a thumbnail relation from a package to a part:

```go
thumbURI, _ := url.Parse("/thumbnail.png")
thumbContent := ... // PNG bytes

thumbnail, err := pkg.PutPart(thumbURI, "image/png", thumbContent)
if err != nil {
    log.Fatal(err)
}

err = pkg.SetThumbnail(thumbnail)
if err != nil {
    log.Fatal(err)
}
```

If you want to remove the thumbnail, call:

```go
err := pkg.UnsetThumbnail()
if err != nil {
    log.Fatal(err)
}
```

## Closing and Flushing

The changes are kept in memory until they are flushed to the file or stream.
The behavior follows how SQLite connections handle commits:

* `Close` flushes the package if it has been modified since it was opened or last flushed, and then closes it.
  If nothing has been modified, the file or stream is left untouched, i.e., it is neither rewritten nor is its modification time bumped.
* `Discard` closes the package *without* flushing, so the unflushed changes are dropped.
  It is a no-op if the package has already been closed.
* `Flush` writes the package immediately, which is useful for saving in the middle of a session.

We recommend to defer `Discard` and call `Close` explicitly at the end, analogous to `Rollback` and `Commit` of `sql.Tx`.
This way, a half-modified package never reaches the disk when you return early with an error or panic:

```go
func addSpec(packaging *aasx.Packaging, path string, content []byte) error {
    pkg, err := packaging.OpenReadWrite(path)
    if err != nil {
        return err
    }
    defer pkg.Discard()

    uri, _ := url.Parse("/aasx/data.json")
    part, err := pkg.PutPart(uri, "text/json", content)
    if err != nil {
        return err // nothing is written
    }
    if err := pkg.MakeSpec(part); err != nil {
        return err // nothing is written, not even the part
    }

    return pkg.Close() // flushes and closes
}
```

If the flush in `Close` fails, the package remains open and the error is returned, so that you can retry or `Discard`.

To flush without closing:

```go
err := pkg.Flush()
if err != nil {
    log.Fatal(err)
}
```

## Concurrency

Please be careful if you read parts while you are writing.
The library uses internal locking for individual operations, but you need to handle higher-level synchronization yourself if performing complex operations across multiple goroutines.

Additionally, be careful to re-read the proper nuggets of information if you changed them.
For example, capturing the groups of specs by content type becomes invalid if you add a new spec:

```go
specsByContentType, _ := pkg.SpecsByContentType()

for contentType, specs := range specsByContentType {
    // If you call PutPart with a different content type here...
    pkg.PutPart(...)

    // specsByContentType is now stale and needs to be re-read.
}
```

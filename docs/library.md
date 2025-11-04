# Library Usage

Borg can also be used as a Go library. The public API is exposed under the `pkg` directory. Import paths use the module `github.com/Snider/Borg`.

Note: This documentation describes usage only; functionality remains unchanged.

## Collecting a GitHub repo into a DataNode

```
package main

import (
    "log"
    "os"

    "github.com/Snider/Borg/pkg/datanode"
    borggithub "github.com/Snider/Borg/pkg/github"
)

func main() {
    // Create a DataNode writer (uncompressed example)
    dn, err := datanode.NewFileDataNodeWriter("repo.dat")
    if err != nil { log.Fatal(err) }
    defer dn.Close()

    client := borggithub.NewDefaultClient(nil) // uses http.DefaultClient
    if err := borggithub.CollectRepo(client, "https://github.com/Snider/Borg", dn); err != nil {
        log.Fatal(err)
    }
}
```

## Collecting a Website

```
package main

import (
    "log"
    "github.com/Snider/Borg/pkg/datanode"
    "github.com/Snider/Borg/pkg/website"
)

func main() {
    dn, err := datanode.NewFileDataNodeWriter("website.dat")
    if err != nil { log.Fatal(err) }
    defer dn.Close()

    if err := website.Collect("https://example.com", 1, dn); err != nil {
        log.Fatal(err)
    }
}
```

## PWA Collection

```
package main

import (
    "log"
    "github.com/Snider/Borg/pkg/datanode"
    "github.com/Snider/Borg/pkg/pwa"
)

func main() {
    dn, err := datanode.NewFileDataNodeWriter("pwa.dat")
    if err != nil { log.Fatal(err) }
    defer dn.Close()

    if err := pwa.Collect("https://squoosh.app", dn); err != nil {
        log.Fatal(err)
    }
}
```

## Logging

The package `pkg/logger` provides helpers for configuring output. See `pkg/logger` tests for examples.

## Notes

- Import paths throughout this repository already use the module path and should work when consumed via `go get github.com/Snider/Borg@latest`.
- The exact function names may differ; consult GoDoc/pkg.go.dev for up-to-date signatures based on the current code.

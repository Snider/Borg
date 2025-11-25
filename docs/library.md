# Library Usage

Borg can also be used as a Go library. The public API is exposed under the `pkg` directory. Import paths use the module `github.com/Snider/Borg`.

## Collecting a GitHub repo into a DataNode

```go
package main

import (
    "log"
    "os"

    "github.com/Snider/Borg/pkg/vcs"
)

func main() {
    // Clone and package the repository
    cloner := vcs.NewGitCloner()
    dn, err := cloner.CloneGitRepository("https://github.com/Snider/Borg", nil)
    if err != nil {
        log.Fatal(err)
    }

    // Save to disk
    tarBytes, err := dn.ToTar()
    if err != nil {
        log.Fatal(err)
    }
    if err := os.WriteFile("repo.dat", tarBytes, 0644); err != nil {
        log.Fatal(err)
    }
}
```

## Collecting a Website

```go
package main

import (
    "log"
    "os"

    "github.com/Snider/Borg/pkg/website"
)

func main() {
    // Download and package the website
    // 1 is the depth (0 = just the page, 1 = page + links on page)
    dn, err := website.DownloadAndPackageWebsite("https://example.com", 1, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Save to disk
    tarBytes, err := dn.ToTar()
    if err != nil {
        log.Fatal(err)
    }
    if err := os.WriteFile("website.dat", tarBytes, 0644); err != nil {
        log.Fatal(err)
    }
}
```

## PWA Collection

```go
package main

import (
    "log"
    "os"

    "github.com/Snider/Borg/pkg/pwa"
)

func main() {
    client := pwa.NewPWAClient()
    pwaURL := "https://squoosh.app"

    // Find the manifest
    manifestURL, err := client.FindManifest(pwaURL)
    if err != nil {
        log.Fatal(err)
    }

    // Download and package the PWA
    dn, err := client.DownloadAndPackagePWA(pwaURL, manifestURL, nil)
    if err != nil {
        log.Fatal(err)
    }

    // Save to disk
    tarBytes, err := dn.ToTar()
    if err != nil {
        log.Fatal(err)
    }
    if err := os.WriteFile("pwa.dat", tarBytes, 0644); err != nil {
        log.Fatal(err)
    }
}
```

## Logging

The package `pkg/logger` provides helpers for configuring output. See `pkg/logger` tests for examples.

## Notes

- Import paths throughout this repository already use the module path and should work when consumed via `go get github.com/Snider/Borg@latest`.
- The exact function names may differ; consult GoDoc/pkg.go.dev for up-to-date signatures based on the current code.

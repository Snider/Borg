# Concurrency and Race Condition Audit

## Introduction

This document outlines the findings of a concurrency and race condition audit performed on the data collection operations of the codebase, with a specific focus on the website downloader located in `pkg/website/website.go`.

## Analysis of `pkg/website/website.go`

The current implementation of the `Downloader` in `pkg/website/website.go` is **sequential**. The `crawl` function recursively downloads pages and assets in a single thread. There are no goroutines or other concurrent mechanisms used for the download process.

## Race Condition Analysis

Due to the sequential nature of the code, there are **no race conditions** present in the current implementation.

To verify this, the Go race detector was run on the `pkg/website` package, and it reported no race conditions.

```
$ go test -race ./pkg/website
ok  	github.com/Snider/Borg/pkg/website	1.160s
```

## Recommendations for Future Concurrency

If the `Downloader` were to be refactored to use concurrency (e.g., a worker pool) to speed up downloads, several potential race conditions would need to be addressed. The `Downloader` struct contains shared state that would be accessed by multiple goroutines.

Specifically, the following fields would be vulnerable to race conditions:

*   `visited map[string]bool`: Concurrent reads and writes to this map without a lock would cause a race condition.
*   `errors []error`: Appending to this slice from multiple goroutines concurrently is not safe.
*   `dn *datanode.DataNode`: The `AddData` method on the `DataNode` would likely be called concurrently, and its internal implementation would need to be thread-safe.

To mitigate these risks, a `sync.Mutex` should be added to the `Downloader` struct and used to protect all access (reads and writes) to these shared fields.

---

## Discovered Race Condition in PWA Downloader

A project-wide test run using `go test -race ./...` revealed a data race in the Progressive Web App (PWA) downloader.

### Location of the Race

The race condition occurs in `pkg/pwa/pwa.go` during the concurrent download of assets. The race is triggered because multiple goroutines call the `AddData` method on the same `*datanode.DataNode` instance simultaneously. The `AddData` method, defined in `pkg/datanode/datanode.go`, is not thread-safe.

### Race Detector Output

The following output from the race detector clearly indicates a write-write race on the internal map of the `DataNode`.

```
==================
WARNING: DATA RACE
Write at 0x00c0002e6540 by goroutine 163:
  runtime.mapassign_faststr()
      /home/jules/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64/src/internal/runtime/maps/runtime_faststr_swiss.go:263 +0x0
  github.com/Snider/Borg/pkg/datanode.(*DataNode).AddData()
      /app/pkg/datanode/datanode.go:95 +0x1c4
  github.com/Snider/Borg/pkg/pwa.(*pwaClient).DownloadAndPackagePWA.func1()
      /app/pkg/pwa/pwa.go:220 +0xab8
  github.com/Snider/Borg/pkg/pwa.(*pwaClient).DownloadAndPackagePWA.gowrap2()
      /app/pkg/pwa/pwa.go:325 +0x57

Previous write at 0x00c0002e6540 by goroutine 162:
  runtime.mapassign_faststr()
      /home/jules/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.0.linux-amd64/src/internal/runtime/maps/runtime_faststr_swiss.go:263 +0x0
  github.com/Snider/Borg/pkg/datanode.(*DataNode).AddData()
      /app/pkg/datanode/datanode.go:95 +0x1c4
  github.com/Snider/Borg/pkg/pwa.(*pwaClient).DownloadAndPackagePWA.func1()
      /app/pkg/pwa/pwa.go:220 +0xab8
  github.com/Snider/Borg/pkg/pwa.(*pwaClient).DownloadAndPackagePWA.gowrap2()
      /app/pkg/pwa/pwa.go:325 +0x57
==================
```

### Analysis and Recommendation

The `DownloadAndPackagePWA` function launches multiple goroutines to fetch assets in parallel. However, the shared `DataNode`'s `AddData` method modifies its internal map without any synchronization mechanism. This is a classic data race.

It is recommended to add a `sync.Mutex` to the `DataNode` struct in `pkg/datanode/datanode.go` to ensure that all read and write operations on its internal data structures are thread-safe.

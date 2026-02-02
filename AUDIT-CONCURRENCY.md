# Concurrency and Race Condition Audit

## Introduction

This document outlines the findings of a concurrency and race condition audit performed on the data collection operations of the codebase.

## Analysis of `pkg/website/website.go`

The current implementation of the `Downloader` in `pkg/website/website.go` is **sequential**. The `crawl` function recursively downloads pages and assets in a single thread. There are no goroutines or other concurrent mechanisms used for the download process. Due to its sequential nature, there are **no race conditions** in this package.

## Discovered Race Condition in PWA Downloader

A project-wide test run using `go test -race ./...` revealed a data race in the Progressive Web App (PWA) downloader.

### Location of the Race

The race condition occurs in `pkg/pwa/pwa.go` during the concurrent download of assets. The race is triggered because multiple goroutines call the `AddData` method on the same `*datanode.DataNode` instance simultaneously. The `AddData` method, defined in `pkg/datanode/datanode.go`, is not thread-safe.

### Race Detector Output

```
==================
WARNING: DATA RACE
Write at 0x00c0002e6540 by goroutine 163:
  runtime.mapassign_faststr()
  github.com/Snider/Borg/pkg/datanode.(*DataNode).AddData()
      /app/pkg/datanode/datanode.go:95 +0x1c4
  github.com/Snider/Borg/pkg/pwa.(*pwaClient).DownloadAndPackagePWA.func1()
      /app/pkg/pwa/pwa.go:220 +0xab8
==================
```

### Analysis and Recommendation

The `DownloadAndPackagePWA` function launches multiple goroutines to fetch assets in parallel. However, the shared `DataNode`'s `AddData` method modifies its internal map without any synchronization mechanism.

It is recommended to add a `sync.Mutex` to the `DataNode` struct in `pkg/datanode/datanode.go` to ensure that all read and write operations on its internal data structures are thread-safe.

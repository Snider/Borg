# Usage

This document explains how to use the Borg Data Collector.

## `collect git`

The `collect git` command is used to clone a git repository and store it in a DataNode.

### Collect a single repository

```bash
borg collect git --uri https://github.com/torvalds/linux.git --output linux.dat
```

### Collect all repositories for a user

```bash
borg collect git --user torvalds --output /path/to/output/dir
```

## `collect website`

The `collect website` command is used to crawl a website and store it in a DataNode.

### Example

```bash
borg collect website --uri https://tldp.org/
```

## `serve`

The `serve` command is used to serve a DataNode file.

### Example

```bash
borg serve --file linux.borg
```

# imgmkr

A command-line utility for creating test OCI images with configurable layers and sizes.

## Overview

imgmkr creates mock OCI images for testing purposes. It allows you to create images with varying numbers of layers and layer sizes without needing to implement actual code or curate real artifacts.

## Requirements

- Go 1.21 or later
- Finch or Docker (finch preferred, docker as fallback)

## Installation

```bash
go install github.com/jlbutler/imgmkr@latest
```

Or build from source:

```bash
git clone https://github.com/jlbutler/imgmkr.git
cd imgmkr
go build
```

## Usage

```bash
imgmkr --layer-sizes [sizes] [--tmpdir-prefix [path]] [--max-concurrent [int]] [--mock-fs] [--max-depth [int]] [--target-files [int]] repo:tag
```

### Parameters

- `--layer-sizes`: Required. Comma-separated list of layer sizes. Supports various formats:
  - Bytes: `8150`, `8B`, `8b`, `8byte`, `8bytes`
  - Kilobytes: `512KB`, `512kb`, `512K`, `512k`
  - Megabytes: `1MB`, `1mb`, `1M`, `1m`
  - Gigabytes: `2GB`, `2gb`, `2G`, `2g`
  - Decimal values: `1.5MB`, `2.75GB`
  - The number of layers is automatically inferred from this list.
- `--tmpdir-prefix`: Optional. Directory prefix for temporary build files. If not specified, uses the system default temp directory. Useful for very large images that might exceed tmpfs capacity.
- `--max-concurrent`: Optional. Maximum number of layers to create concurrently (default: 5). Higher values may speed up creation but use more system resources.
- `--mock-fs`: Optional. Create mock filesystem structure with multiple files and directories instead of single large files per layer.
- `--max-depth`: Optional. Maximum directory depth for mock filesystem (default: 3). Only used with --mock-fs.
- `--target-files`: Optional. Target number of files per layer for mock filesystem (default: calculated based on layer size). Only used with --mock-fs.
- `repo:tag`: Required. Repository and tag for the built image.

### Examples

Create an image with 3 layers of different sizes:

```bash
imgmkr --layer-sizes 10MB,50MB,100MB myrepo:latest
```

Create an image with 2 layers, one small and one large:

```bash
imgmkr --layer-sizes 1MB,1GB test-image:v1
```

Create a very large image using rootfs instead of tmpfs:

```bash
imgmkr --layer-sizes 1GB,2GB,5GB --tmpdir-prefix /tmp large-image:v1
```

Create an image with mock filesystem structure instead of single files:

```bash
imgmkr --layer-sizes 500MB,1GB --mock-fs realistic-image:v1
```

Create a complex mock filesystem with custom depth and file count:

```bash
imgmkr --layer-sizes 1GB --mock-fs --max-depth 4 --target-files 200 complex-image:v1
```

## How It Works

1. Creates a temporary build directory
2. Generates mock data files of specified sizes for each layer (with real-time progress tracking)
3. Creates a Dockerfile that adds each layer
4. Builds the image using finch (preferred) or docker (fallback)
5. Cleans up temporary files after building

## Progress Tracking

imgmkr provides real-time progress updates during layer creation, including:
- Visual progress bar showing completion percentage
- Layer count progress (e.g., 3/5 layers)
- Data size progress (e.g., 2.5GB/5GB)
- Individual layer completion times
- Estimated time to completion (ETA)

This is especially useful when creating large images with multiple layers.

## Graceful Shutdown

imgmkr handles interruption signals (Ctrl+C) gracefully:
- Catches SIGINT and SIGTERM signals
- Cleans up temporary files and directories
- Provides clear feedback about cleanup operations
- Exits with appropriate status codes

If you need to stop a long-running operation, simply press Ctrl+C and imgmkr will clean up after itself.

## License

[MIT No Attribution License](LICENSE)

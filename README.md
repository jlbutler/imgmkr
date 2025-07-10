# imgmkr

A command-line utility for creating test OCI images with configurable layers and sizes.

## Overview

imgmkr creates mock OCI images for testing purposes. It allows you to create images with varying numbers of layers and layer sizes without needing to implement actual code or curate real artifacts.

## Requirements

- Go 1.21 or later
- Docker (on Linux) or Finch (on macOS)

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
imgmkr --layer-sizes [sizes] [--tmpdir-prefix [path]] [--max-concurrent [int]] repo:tag
```

### Parameters

- `--layer-sizes`: Required. Comma-separated list of layer sizes with KB, MB, or GB suffixes (e.g., 512KB,1MB,2GB). The number of layers is automatically inferred from this list.
- `--tmpdir-prefix`: Optional. Directory prefix for temporary build files. If not specified, uses the system default temp directory. Useful for very large images that might exceed tmpfs capacity.
- `--max-concurrent`: Optional. Maximum number of layers to create concurrently (default: 5). Higher values may speed up creation but use more system resources.
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

## How It Works

1. Creates a temporary build directory
2. Generates mock data files of specified sizes for each layer
3. Creates a Dockerfile that adds each layer
4. Builds the image using finch (on macOS) or docker (on Linux)
5. Cleans up temporary files after building

## License

[MIT No Attribution License](LICENSE)

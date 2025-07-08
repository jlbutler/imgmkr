# imgmkr

A command-line utility for creating test OCI images with configurable layers and sizes.

## Overview

imgmkr creates mock OCI images for testing purposes. It allows you to create images with varying numbers of layers and layer sizes without needing to implement actual code or curate real artifacts.

## Requirements

- Go 1.21 or later
- Docker (on Linux) or Finch (on macOS)

## Installation

```bash
go install github.com/yourusername/imgmkr@latest
```

Or build from source:

```bash
git clone https://github.com/yourusername/imgmkr.git
cd imgmkr
go build
```

## Usage

```bash
imgmkr --num-layers [int] --layer-sizes [int,..] repo:tag
```

### Parameters

- `--num-layers`: Required. Total number of layers to include in the image.
- `--layer-sizes`: Required. Comma-separated list of layer sizes with KB, MB, or GB suffixes (e.g., 512KB,1MB,2GB).
- `repo:tag`: Required. Repository and tag for the built image.

### Examples

Create an image with 3 layers of different sizes:

```bash
imgmkr --num-layers 3 --layer-sizes 10MB,50MB,100MB myrepo:latest
```

Create an image with 2 layers, one small and one large:

```bash
imgmkr --num-layers 2 --layer-sizes 1MB,1GB test-image:v1
```

## How It Works

1. Creates a temporary build directory
2. Generates mock data files of specified sizes for each layer
3. Creates a Dockerfile that adds each layer
4. Builds the image using finch (on macOS) or docker (on Linux)
5. Cleans up temporary files after building

## License

[MIT License](LICENSE)
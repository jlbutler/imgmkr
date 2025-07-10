# Project Specification

## 1. Project Overview

- **Project Name**: imgmkr
- **Version**: 0.1
- **Description**: A simple command line utility for creating test OCI images
- **Target Audience**: Anyone who wants mock images of varying sizes and layer configurations
- **Business Value**: Test utility

## 2. Requirements

### 2.1 Functional Requirements

This utility creates mock OCI images for testing purposes. This enables a set of images specific to use cases (e.g. one large image layer, several small image layers, combinations of large and small layers), without the need to actually implement code and curate real artifacts. All layers are populated with mock data, and may consist of very large files, or many small files to make up the requested layer size.

- Command line utility implemented in Go to create mock OCI images for test purposes
- Images can be creating with varying numbers of layers, and layers can be of differing sizes
- Takes required parameters for number of image layers, image layer sizes (in MB), and content layout per layer (default one large file per layer)
- Automates creation of layer content, generates a Dockerfile, and builds the image using finch, docker, or other utility for building and pushing images
- Creates single files to populate image layers based on the desired configuration, e.g. a 2GB file is added to the image as a single layer when a 2GB layer is specified.
- Once the image build is complete, it is in the finch images list, and the temporary data files and Dockerfile are all removed from the system.
- Image base layers are scratch layers, and each subsequent can be added with `ADD` command in Dockerfile to make up the requested total number of layers

### 2.2 Non-Functional Requirements

- **Performance**: The utility creates layers concurrently to optimize build times, especially for multi-layer images. Performance can be tuned via the --max-concurrent parameter.
- **Usability**: As a typical command line utility, it should have help and usage for the command and its options and arguments
- **Compatibility**: Images should be built for linux/amd64 by default

## 3. Technical Specifications

### 3.1 User Interface

imgmkr --layer-sizes [sizes] [--tmpdir-prefix [path]] [--max-concurrent [int]] repo:tag

- layer-sizes: required, comma-delineated list of image sizes in KB, MB, or GB (where KB is 1024 bytes, etc). The number of layers is automatically inferred from this list.
- tmpdir-prefix: optional, directory prefix for temporary build files (default: system temp dir)
- max-concurrent: optional, maximum number of layers to create concurrently (default: 5)

imgmkr will create a local image by creating mock data and creating a custom Dockerfile to achieve the desired image configuration. The number of layers is automatically inferred from the comma-delineated list of sizes specified by `layer-sizes`.

### 3.2 Technology Stack

This is a simple utility and doesn't require direct integration with other components, but other components and 3rd party libraries could be used. It does require docker, finch, or some other means of building images to be installed and present.

## 4. Implementation Details

### 4.1 Runtime

Once the utility is invoked, the options and arguments are checked for validity, then each of the stages below are executed (data generation, image spec creation, image creation, and cleanup). The utility should provide status messages regarding its progress.

### 4.2 Mock data generation

Each layer is populated by a single file of the size specified for that layer. The utility creates mock files filled with data to achieve the specified layer sizes. All layer content is created in a temporary build directory, organized by subdirectory per layer.

The temporary build directory location can be controlled via the `--tmpdir-prefix` option:

- **Default behavior**: Uses the system's default temporary directory (typically tmpfs on Linux/macOS)
- **Custom prefix**: When `--tmpdir-prefix` is specified, creates the build directory under the specified path (useful for very large images that might exceed tmpfs capacity)

For example:

/tmp/imgmkr-build-xyz/
layer1/1.00GB-file
layer2/4.00GB-file
layer3/2.00GB-file
layer4/512.00MB-file
layer5/128.00MB-file
layer6/512.00KB-file

### 4.3 Image specification

Once the layer data files are created, a Dockerfile is created in the temp build directory. The Dockerfile is a base image `FROM scratch`, followed by an `ADD` directive for each layer.

For example:

FROM scratch
ADD layer1
ADD layer2

etc

### 4.4 Image build

The image is then built using finch or docker. The utility tries finch first (preferred), then falls back to docker if finch is not available. The build will result in a local image tagged as specified by the command line argument `repo:tag`.

### 4.5 Cleanup

The temp build directory and all its contents are removed once the image is built.

## 5. Constraints and Assumptions

- The utility is not intended to be a full fledged image builder, but rather a simple utility for creating mock images for testing purposes.
- Files within mock layers are real files with data, not sparse files.
- Built images are OCI compliant and can be pushed to an OCI registry like ECR.
- Image entrypoints and configuration are not important, testing is focused on end-to-end image pull testing (layer transfer, layer unpack, checksum, and local store write).

## 5. Success Criteria

- Users can create one-off test images of varying layouts with a single invocation.

## 6. Future Enhancements

### 6.1 Configurable Temporary Directory ✅ IMPLEMENTED

The utility now supports configurable temporary directory location to handle very large images that might exceed tmpfs capacity. This enhancement includes:

- **Flexible Storage**: Users can specify where temporary build files are created
- **tmpfs vs rootfs**: Default uses system temp (tmpfs) for speed, custom prefix allows rootfs for capacity
- **Large Image Support**: Prevents "no space left on device" errors when creating multi-gigabyte test images
- **Backward Compatibility**: Maintains default behavior when no prefix is specified

**Usage**: `imgmkr --layer-sizes 1GB,2GB,5GB --tmpdir-prefix /tmp large-image:v1`

### 6.2 Concurrent Layer Creation ✅ IMPLEMENTED

The utility now supports concurrent layer creation to improve performance when building images with multiple layers. This enhancement includes:

- **Concurrent Processing**: Layers are created in parallel using goroutines and a worker pool pattern
- **Configurable Concurrency**: Users can control the level of concurrency with the `--max-concurrent` flag (default: 5)
- **Progress Reporting**: Real-time progress updates show when each layer completes, even when created concurrently
- **Error Handling**: If any layer creation fails, all remaining operations are stopped and the error is reported
- **Resource Management**: The worker pool prevents overwhelming the system with too many concurrent operations

**Usage**: `imgmkr --layer-sizes 1GB,2GB,1GB,500MB,100MB --max-concurrent 3 myimage:latest`

This enhancement significantly reduces build times for multi-layer images, especially when creating large layers that are I/O bound.

### 6.3 Mock rootfs file layouts ✅ IMPLEMENTED

The utility now provides a `--mock-fs` option that creates mock root file systems, adding various directories and mock files to fill up a given layer, vs single files. This enhancement includes:

- **Realistic Filesystem Structure**: Creates nested directories with multiple files of varying sizes
- **Configurable Depth**: Users can control directory nesting with `--max-depth` (default: 3)
- **File Count Control**: Users can specify target file count with `--target-files` or let the system calculate based on layer size
- **Random File Sizes**: Files range from 1KB to 512MB with random distribution
- **Size Accuracy**: Total file content matches the specified layer size

**Usage**: `imgmkr --layer-sizes 1GB --mock-fs --max-depth 4 --target-files 200 complex-image:v1`

Since test images are meant to exercise client image pull operations, including image layer unpack, decompression and unpack of layers will perform differently with one large single file vs a more realistic mock rootfs. This option creates such a mock file system rather than the default behavior of one large file.

### 6.4 Additional Future Enhancements

The utility could create images itself using the OCI SDK without calling out to another tool. This would remove the need for finch, docker, etc.

The utility could provide a `publish` option that implements an OCI registry client to publish to a provided registry by pushing the image.

The utility could move to using BuildPacks instead of Dockerfile, if that offers more flexibility or performance improvements.

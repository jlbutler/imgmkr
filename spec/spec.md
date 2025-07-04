# Project Specification

## 1. Project Overview
- **Project Name**: image-maker
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
- **Performance**: The utility should be fairly quick, but image creation performance is not a main consideration
- **Usability**: As a typical command line utility, it should have help and usage for the command and its options and arguments
- **Compatibility**: Images should be built for linux/amd64 by default

## 3. Technical Specifications

### 3.1 User Interface

image-maker --num-layers [int] --layer-sizes [int,..] repo:tag

- num-layers: required, total number of layers to include
- layer-sizes:  required, comma-delineated list of image sizes in KB, MB, or GB (where KB is 1024 bytes, etc)

image-maker will create a local image by creating mock data and creating a custom Dockerfile to achieve the desired image configuration. The number of layers is specified by `num-layers`, and the image layer sizes detailed by a comma-delineated list of sizes in MB specified by `layer-sizes`. 

### 3.2 Technology Stack

This is a simple utility and doesn't require direct integration with other components, but other components and 3rd party libraries could be used.  It does require docker, finch, or some other means of building images to be installed and present.

## 4. Implementation Details

### 4.1  Runtime

Once the utility is invoked, the options and arguments are checked for validity, then each of the stages below are executed (data generation, image spec creation, image creation, and cleanup). The utility should provide status messages regarding its progress.

### 4.2 Mock data generation

Each layer is populated by a single file of the size specified for that layer. The utility may implement a mock file creator, or simply use a utility like `mkfile` if that's just as good. All layer content is created in a temp build directory, organized by subdirectory per layer.

For example:

/tmp/foo-build/
  layer1/1GB-file
  layer2/4GB-file
  layer3/2GB-file
  layer4/512MB-file
  layer5/128MB-file
  layer6/512KB-file

### 4.3 Image specification

Once the layer data files are created, a Dockerfile is created in the temp build directory. The Dockerfile is a base image `FROM scratch`, followed by an `ADD` directive for each layer.

For example:

FROM scratch
ADD layer1
ADD layer2

etc

### 4.4 Image build

The image is then built using finch, docker, or other utility. On MacOS, use finch. On Linux, use docker. The build will result in a local image tagged as specified by the command line argument `repo:tag`.

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

The utility could create images itself using the OCI SDK without calling out to another tool. This would remove the need for finch, docker, etc.

The utility could provide a `publish` option that implements an OCI registry client to publish to a provided registry by pushing the image.

The utility could provide a `mock-fs` option that creates mock root file systems, adding various directories and mock files to fill up a given layer, vs single files.

The utility could move to using BuildPacks instead of Dockerfile, if that offers more flexibility or performance improvements.

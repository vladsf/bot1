# Makefile for building and pushing Docker images

# Define the image name and version
IMAGE_NAME := blackandwhite
IMAGE_VERSION := latest

# Define the Docker repository
DOCKER_REPO := vladsf

# Define the full image tag
IMAGE_TAG := $(DOCKER_REPO)/$(IMAGE_NAME):$(IMAGE_VERSION)

# Default target
all: build push

# Build the Docker image
build:
	docker build -t $(IMAGE_TAG) .

# Push the Docker image to the repository
push:
	docker push $(IMAGE_TAG)

# Clean up
clean:
	-docker rmi $(IMAGE_TAG)

.PHONY: all build push clean

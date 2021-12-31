NAME ?= mumoshu/wy
VERSION ?= latest
GO ?= go

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# default list of platforms for which multiarch image is built
# See https://www.docker.com/blog/faster-multi-platform-builds-dockerfile-cross-compilation-guide/
ifeq (${PLATFORMS}, )
	export PLATFORMS="linux/amd64,linux/arm64"
endif

# if IMG_RESULT is unspecified, by default the image will be pushed to registry
ifeq (${IMG_RESULT}, load)
	export PUSH_ARG="--load"
    # if load is specified, image will be built only for the build machine architecture.
    export PLATFORMS="local"
else ifeq (${IMG_RESULT}, cache)
	# if cache is specified, image will only be available in the build cache, it won't be pushed or loaded
	# therefore no PUSH_ARG will be specified
else
	export PUSH_ARG="--push"
endif

# Run go fmt against code
fmt:
	$(GO) fmt ./...

# Run go vet against code
vet:
	$(GO) vet ./...

build:
	$(GO) build .

docker-buildx: buildx
	export DOCKER_CLI_EXPERIMENTAL=enabled
	@if ! docker buildx ls | grep -q container-builder; then\
		docker buildx create --platform ${PLATFORMS} --name container-builder --use;\
	fi
	docker buildx build --platform ${PLATFORMS} \
		-t "${NAME}:${VERSION}" \
		-f Dockerfile \
		. ${PUSH_ARG}

OS_NAME := $(shell uname -s | tr A-Z a-z)

BUILDX_VERSION ?= 0.7.0

# find or download controller-gen
# download controller-gen if necessary
buildx:
ifeq (, $(shell [ -e ~/.docker/cli-plugins/docker-buildx ] && echo exists))
	@{ \
	set -e ;\
	BUILDX_TMP_DIR=$$(mktemp -d) ;\
	cd $$BUILDX_TMP_DIR ;\
	wget https://github.com/docker/buildx/releases/download/v$(BUILDX_VERSION)/buildx-v$(BUILDX_VERSION).$(OS_NAME)-amd64 ;\
	chmod a+x buildx-v$(BUILDX_VERSION).$(OS_NAME)-amd64 ;\
	mkdir -p ~/.docker/cli-plugins ;\
	mv buildx-v$(BUILDX_VERSION).$(OS_NAME)-amd64 ~/.docker/cli-plugins/docker-buildx ;\
	rm -rf $$BUILDX_TMP_DIR ;\
	}
BUILDX_BIN=~/.docker/cli-plugins/docker-buildx
else
BUILDX_BIN=~/.docker/cli-plugins/docker-buildx
endif

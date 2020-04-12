export CGO_ENABLED=0
export GO111MODULE=on

ATOMIX_REDIS_CONTROLLER_VERSION := latest

.PHONY: build


test: # @HELP run the unit tests and source code validation
test: build linters
	go test github.com/atomix/redis-storage-controller/pkg/...

coverage: # @HELP generate unit test coverage data
coverage: build linters license_check

build: # @HELP build the source code
build: deps
	GOOS=linux GOARCH=amd64 go build -o build/_output/redis-storage-controller ./cmd/redis-storage-controller

deps: # @HELP ensure that the required dependencies are in place
	go build -v ./...
	bash -c "diff -u <(echo -n) <(git diff go.mod)"
	bash -c "diff -u <(echo -n) <(git diff go.sum)"

linters: # @HELP examines Go source code and reports coding problems
	GOGC=75 golangci-lint run

license_check: # @HELP examine and ensure license headers exist
	@if [ ! -d "../build-tools" ]; then cd .. && git clone https://github.com/onosproject/build-tools.git; fi
	./../build-tools/licensing/boilerplate.py -v --rootdir=${CURDIR}

images: # @HELP build redis-storage Docker image
images: build
	docker build . -f build/redis-storage-controller/Dockerfile -t atomix/redis-storage-controller:${ATOMIX_REDIS_CONTROLLER_VERSION}

kind: images
	kind load docker-image atomix/redis-storage-controller:${ATOMIX_REDIS_CONTROLLER_VERSION}
	

all: test


clean: # @HELP remove all the build artifacts
	rm -rf ./build/_output ./vendor

help:
	@grep -E '^.*: *# *@HELP' $(MAKEFILE_LIST) \
    | sort \
    | awk ' \
        BEGIN {FS = ": *# *@HELP"}; \
        {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}; \
    '

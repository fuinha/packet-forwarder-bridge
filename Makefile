SHELL = bash

# Environment

GIT_BRANCH = $(or $(CI_BUILD_REF_NAME) ,`git rev-parse --abbrev-ref HEAD 2>/dev/null`)
GIT_COMMIT = $(or $(CI_BUILD_REF), `git rev-parse HEAD 2>/dev/null`)
BUILD_DATE = $(or $(CI_BUILD_DATE), `date -u +%Y-%m-%dT%H:%M:%SZ`)
GO_PATH = `echo $(GOPATH) | awk -F':' '{print $$1}'`
GO_SRC = `pwd | xargs dirname | xargs dirname | xargs dirname`

# All

.PHONY: all build-deps deps dev-deps test cover-clean cover-deps cover coveralls fmt vet build dev link docs clean docker

all: deps build

# Deps

build-deps:
	@command -v govendor > /dev/null || go get "github.com/kardianos/govendor"

deps: build-deps
	govendor sync -v

dev-deps: deps
	@command -v golint > /dev/null || go get github.com/golang/lint/golint

# Go Test

GO_FILES = $(shell find . -name "*.go" | grep -vE ".git|.env|vendor")
GO_PACKAGES = $(shell find . -name "*.go" | grep -vE ".git|.env|vendor" | sed 's:/[^/]*$$::' | sort | uniq)
GO_TEST_PACKAGES = $(shell find . -name "*_test.go" | grep -vE ".git|.env|vendor" | sed 's:/[^/]*$$::' | sort | uniq)

GO_COVER_FILE ?= coverage.out
GO_COVER_DIR ?= .cover
GO_COVER_FILES = $(patsubst ./%, $(GO_COVER_DIR)/%.out, $(shell echo "$(GO_TEST_PACKAGES)"))

test: $(GO_FILES)
	go test -v $(GO_TEST_PACKAGES)

cover-clean:
	rm -rf $(GO_COVER_DIR) $(GO_COVER_FILE)

cover-deps:
	@command -v goveralls > /dev/null || go get github.com/mattn/goveralls

cover: $(GO_COVER_FILE)

$(GO_COVER_FILE): cover-clean $(GO_COVER_FILES)
	echo "mode: set" > $(GO_COVER_FILE)
	cat $(GO_COVER_FILES) | grep -vE "mode: set" | sort >> $(GO_COVER_FILE)

$(GO_COVER_DIR)/%.out: %
	@mkdir -p "$(GO_COVER_DIR)/$<"
	go test -cover -coverprofile="$@" "./$<"

coveralls: cover-deps $(GO_COVER_FILE)
	goveralls -coverprofile=$(GO_COVER_FILE) -service=travis-ci -repotoken $$COVERALLS_TOKEN

fmt:
	[[ -z "`echo "$(GO_PACKAGES)" | xargs go fmt | tee -a /dev/stderr`" ]]

vet:
	echo $(GO_PACKAGES) | xargs go vet

lint:
	for pkg in `echo $(GO_PACKAGES)`; do golint $$pkg; done

# Go Build

RELEASE_DIR ?= build
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOEXE = $(shell GOOS=$(GOOS) GOARCH=$(GOARCH) go env GOEXE)
CGO_ENABLED ?= 0

DIST_FLAGS ?= -a -installsuffix cgo

splitfilename = $(subst ., ,$(subst -, ,$(subst $(RELEASE_DIR)/,,$1)))
GOOSfromfilename = $(word 4, $(call splitfilename, $1))
GOARCHfromfilename = $(word 5, $(call splitfilename, $1))
LDFLAGS = -ldflags "-w -X main.gitBranch=${GIT_BRANCH} -X main.gitCommit=${GIT_COMMIT} -X main.buildDate=${BUILD_DATE}"
GOBUILD = CGO_ENABLED=$(CGO_ENABLED) GOOS=$(call GOOSfromfilename, $@) GOARCH=$(call GOARCHfromfilename, $@) go build $(DIST_FLAGS) ${LDFLAGS} -tags "${TAGS}" -o "$@"

build: $(RELEASE_DIR)/lora-gateway-bridge-$(GOOS)-$(GOARCH)$(GOEXE)

$(RELEASE_DIR)/lora-gateway-bridge-%: $(GO_FILES)
	$(GOBUILD) ./cmd/lora-gateway-bridge/main.go

lora-gateway-bridge-dev: DIST_FLAGS=
lora-gateway-bridge-dev: CGO_ENABLED=1
lora-gateway-bridge-dev: $(RELEASE_DIR)/lora-gateway-bridge-$(GOOS)-$(GOARCH)$(GOEXE)

install:
	go install -v ./cmd/lora-gateway-bridge

dev: install lora-gateway-bridge-dev

GOBIN ?= $(GO_PATH)/bin

link: build
	ln -sf $(PWD)/$(RELEASE_DIR)/lora-gateway-bridge-$(GOOS)-$(GOARCH)$(GOEXE) $(GOBIN)/lora-gateway-bridge

# Clean

clean:
	[ -d $(RELEASE_DIR) ] && rm -rf $(RELEASE_DIR) || [ ! -d $(RELEASE_DIR) ]

# Docker

docker: GOOS=linux
docker: GOARCH=amd64
docker: $(RELEASE_DIR)/lora-gateway-bridge-linux-amd64
	docker build -t thethingsnetwork/lora-gateway-bridge -f Dockerfile .

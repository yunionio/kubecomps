ROOT_DIR = $(CURDIR)
OUTPUT_DIR = $(ROOT_DIR)/_output
BIN_DIR = $(OUTPUT_DIR)/bin
REPO_PREFIX = yunion.io/x/kubecomps

REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
	                   git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

BUILD_SCRIPT := $(ROOT_DIR)/build/build.sh

VERSION_PKG_PREFIX := yunion.io/x/pkg/util/version
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD)
GIT_COMMIT := $(shell git rev-parse --short HEAD)
GIT_VERSION := $(shell git describe --tags --abbrev=14 $(GIT_COMMIT)^{commit})
GIT_TREE_STATE := $(shell s=`git status --porcelain 2>/dev/null`; if [ -z "$$s"  ]; then echo "clean"; else echo "dirty"; fi)
BUILD_DATE := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')

LDFLAGS := "-w \
	-X $(VERSION_PKG_PREFIX).gitBranch=$(GIT_BRANCH) \
	-X $(VERSION_PKG_PREFIX).gitVersion=$(GIT_VERSION) \
	-X $(VERSION_PKG_PREFIX).gitCommit=$(GIT_COMMIT) \
	-X $(VERSION_PKG_PREFIX).gitTreeState=$(GIT_TREE_STATE) \
	-X $(VERSION_PKG_PREFIX).buildDate=$(BUILD_DATE)"


export GO111MODULE:=on
export GOPROXY:=direct
RELEASE_BRANCH:=release/4.0
GO_BUILD := go build -mod vendor -buildvcs=false -ldflags $(LDFLAGS)

CMDS := $(shell find $(ROOT_DIR)/cmd -mindepth 1 -maxdepth 1 -type d)

build: clean generate
	@for CMD in $(CMDS); do \
		echo build $$CMD; \
		$(GO_BUILD) -o $(BIN_DIR)/`basename $${CMD}` $$CMD; \
	done

embed-helm-pkgs:
	docker run --rm -v $(shell pwd):/app registry.cn-beijing.aliyuncs.com/yunionio/helm:v3.5.2 \
		/bin/bash -c "cd /app && /app/scripts/embed-helm-pkgs.sh"

generate: embed-helm-pkgs
	@git submodule init
	@git submodule update
	@go generate -mod vendor ./...
	@echo "[OK] files added to embed box!"

prepare_dir:
	@mkdir -p $(BIN_DIR)

mod:
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go get yunion.io/x/onecloud@$(RELEASE_BRANCH)
	GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go get yunion.io/x/sdnagent@$(RELEASE_BRANCH)
	#GOPROXY=$(GOPROXY) GONOSUMDB=yunion.io/x go get $(patsubst %,%@master,$(shell GO111MODULE=on go mod edit -print | sed -n -e 's|.*\(yunion.io/x/[a-z].*\) v.*|\1|p' | grep -v '/onecloud$$'))
	go mod tidy
	go mod vendor -v


cmd/%: prepare_dir
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

image: generate
	DEBUG=$(DEBUG) ARCH=$(ARCH) TAG=$(VERSION) REGISTRY=$(REGISTRY) $(ROOT_DIR)/scripts/docker_push.sh $(filter-out $@,$(MAKECMDGOALS))

clean:
	@rm -rf $(OUTPUT_DIR)

gen-swagger-check:
	which swagger || (GO111MODULE=off go get -u github.com/yunionio/go-swagger/cmd/swagger)
	which swagger-serve || (GO111MODULE=off go get -u yunion.io/x/code-generator/cmd/swagger-serve)

gen-swagger: gen-swagger-check
	mkdir -p ./_output/swagger
	./_output/bin/kube-swagger-gen \
		--input-dirs yunion.io/x/kubecomps/pkg/kubeserver/usages \
		--input-dirs yunion.io/x/kubecomps/pkg/kubeserver/models \
		--output-package \
		yunion.io/x/kubecomps/pkg/generated/kubeserver
	swagger generate spec -o ./_output/swagger/kubeserver.yaml --scan-models --work-dir=./pkg/generated/kubeserver

swagger-serve: gen-swagger
	swagger-serve generate -i ./_output/swagger/kubeserver.yaml \
		-o ./_output/swagger

.PHONY: build clean image mod generate embed-helm-pkgs

KS_BINS_DIR = $(OUTPUT_DIR)/binaries

sync-kubespray-bins:
	rsync -avP $(KS_BINS_DIR)/* a3-iso:/data/binaries


ifeq ($(UNAME), Linux)
XARGS_FLAGS = --no-run-if-empty
endif

fmt:
	@git ls-files --exclude '*' '*.go' \
		| grep -v '^vendor/' \
		| while read f; do \
			if ! grep -m1 -q '^// Code generated .* DO NOT EDIT\.$$' "$$f"; then \
				echo "$$f"; \
			fi ; \
		done \
		| xargs $(XARGS_FLAGS) gofmt -w

fmt-check: fmt
	@if git status --short | grep -E '^.M .*/[^.]+.go'; then \
		git diff | cat; \
		echo "$@: working tree modified (possibly by gofmt)" >&2 ; \
		false ; \
	fi

test:
	@go test $(GO_BUILD_FLAGS) $(shell go list ./...)

.PHONY: fmt fmt-check test

%:
	@:

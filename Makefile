ROOT_DIR = $(CURDIR)
OUTPUT_DIR = $(ROOT_DIR)/_output
BIN_DIR = $(OUTPUT_DIR)/bin
REPO_PREFIX = yunion.io/x/kubecomps

GO_BUILD := go build

CMDS := $(shell find $(ROOT_DIR)/cmd -mindepth 1 -maxdepth 1 -type d)

REGISTRY ?= "registry.cn-beijing.aliyuncs.com/yunionio"
VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
	                   git describe --match=$(git rev-parse --short=8 HEAD) --always --dirty --abbrev=8)

build: clean
	@for CMD in $(CMDS); do \
		echo build $$CMD; \
		$(GO_BUILD) -o $(BIN_DIR)/`basename $${CMD}` $$CMD; \
	done

prepare_dir:
	@mkdir -p $(BIN_DIR)

cmd/%: prepare_dir
	$(GO_BUILD) -o $(BIN_DIR)/$(shell basename $@) $(REPO_PREFIX)/$@

image:
	DEBUG=$(DEBUG) ARCH=$(ARCH) TAG=$(VERSION) REGISTRY=$(REGISTRY) $(ROOT_DIR)/scripts/docker_push.sh $(filter-out $@,$(MAKECMDGOALS))

clean:
	@rm -rf $(OUTPUT_DIR)

%:
	@:

.PHONY: build clean image

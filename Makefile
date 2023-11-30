default_target: all

.PHONY: default_target

VERSION=$(shell git describe --tags --always)
GIT_COMMIT=$(shell git rev-parse HEAD)
GIT_COMMIT_DATE=$(shell git log -n1 --pretty='format:%cd' --date=format:'%Y%m%d')
REPO=github.com/artela-network/artela
BUILD=./build

GOBIN ?= $$(go env GOPATH)/bin

ldflags = -X $(REPO)/version.AppVersion=$(VERSION) \
          -X $(REPO)/version.GitCommit=$(GIT_COMMIT) \
          -X $(REPO)/version.GitCommitDate=$(GIT_COMMIT_DATE)

ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
 ldflags += -w -s
endif
ldflags += $(LDFLAGS)
ldflags := $(strip $(ldflags))

BUILD_FLAGS := -tags "$(build_tags)" -ldflags '$(ldflags)'
# check for nostrip option
ifeq (,$(findstring nostrip,$(COSMOS_BUILD_OPTIONS)))
 BUILD_FLAGS += -trimpath
endif

# check if no optimization option is passed
# used for remote debugging
ifneq (,$(findstring nooptimization,$(COSMOS_BUILD_OPTIONS)))
  BUILD_FLAGS += -gcflags "all=-N -l"
endif

mkbuild:
	mkdir -p $(BUILD)

build: mkbuild
	go build -o $(BUILD)/. $(BUILD_FLAGS) ./...

build-linux: mkbuild
	GOOS=linux GOARCH=amd64 LEDGER_ENABLED=false $(MAKE) build

install:
	go install $(BUILD_FLAGS) ./...

all: build

build-testnet:
	docker build --platform linux/amd64 --no-cache --tag artela-network/artela ../. -f ./Dockerfile
	@if ! [ -f _testnet/node0/artelad/config/genesis.json ]; then docker run --platform linux/amd64 --rm -v $(CURDIR)/_testnet:/artela:Z artela-network/artela:latest "./artelad testnet init-files --chain-id artela_11820-1 --v 4 -o /artela --keyring-backend=test --starting-ip-address 172.16.10.2"; fi

create-testnet: remove-testnet build-testnet
	docker compose up -d

start-testnet:
ifeq ($(shell docker images -q artela-network/artela:latest 2> /dev/null),)
	@echo "nothing has changed."
	@echo "testnet is not created, run 'make create-testnet' instead."
else
	docker-compose up -d
endif

stop-testnet:
	docker compose stop

remove-testnet:
	docker compose down
ifneq ($(shell docker images -q artela-network/artela:latest 2> /dev/null),)
	docker rmi artela-network/artela:latest 
endif
	sudo rm -rf ./_testnet

clean:
	rm -rf ./build



###############################################################################
###                                Linting                                  ###
###############################################################################
golangci_lint_cmd=golangci-lint
golangci_version=v1.54.2

lint:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --timeout=10m

lint-fix:
	@echo "--> Running linter"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	@$(golangci_lint_cmd) run --fix --out-format=tab --issues-exit-code=0

.PHONY: lint lint-fix

format:
	@go install mvdan.cc/gofumpt@latest
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(golangci_version)
	find . -name '*.go' -type f -not -path "./vendor*" -not -path "*.git*" -not -path "./tests/mocks/*" -not -name "*.pb.go" -not -name "*.pb.gw.go" -not -name "*.pulsar.go" | xargs gofumpt -w -l
	$(golangci_lint_cmd) run --fix
.PHONY: format


test-unit:
	go test -v ./... -short
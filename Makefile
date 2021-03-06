PACKAGES=$(shell go list ./... | grep -v '/simulation')

VERSION := $(shell echo $(shell git describe --tags) | sed 's/^v//')
COMMIT := $(shell git log -1 --format='%H')

# TODO: Update the ldflags with the app, client & server names
ldflags = -X github.com/saiSunkari19/ibc-demo/version.Name=IBC-DEMO \
	-X github.com/saiSunkari19/ibc-demo/version.ServerName=assetd \
	-X github.com/saiSunkari19/ibc-demo/version.ClientName=assetcli \
	-X github.com/saiSunkari19/ibc-demo/version.Version=$(VERSION) \
	-X github.com/saiSunkari19/ibc-demo/version.Commit=$(COMMIT)

BUILD_FLAGS := -ldflags '$(ldflags)'

all: install

install: go.sum
		go install $(BUILD_FLAGS) ./cmd/assetd
		go install $(BUILD_FLAGS) ./cmd/assetcli

go.sum: go.mod
		@echo "--> Ensure dependencies have not been modified"
		GO111MODULE=on go mod verify

# Uncomment when you have some tests
# test:
# 	@go test -mod=readonly $(PACKAGES)

# look into .golangci.yml for enabling / disabling linters
lint:
	@echo "--> Running linter"
	@golangci-lint run
	@go mod verify
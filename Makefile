GO ?= go

.PHONY: sync-assets check-assets test build build-all

sync-assets:
	node scripts/sync-assets.mjs

check-assets:
	node scripts/sync-assets.mjs --check

test: check-assets
	cd go && $(GO) test ./...

build: sync-assets
	cd go && $(GO) build -o ../unleash .
	cd go && $(GO) build -o ../unleash-gpt ./cmd/unleash-gpt
	cd go && $(GO) build -o ../unleash-omp ./cmd/unleash-omp

build-all: sync-assets
	cd go && GOOS=linux GOARCH=amd64 $(GO) build -o ../unleash-linux-amd64 .
	cd go && GOOS=linux GOARCH=amd64 $(GO) build -o ../unleash-gpt-linux-amd64 ./cmd/unleash-gpt
	cd go && GOOS=linux GOARCH=amd64 $(GO) build -o ../unleash-omp-linux-amd64 ./cmd/unleash-omp

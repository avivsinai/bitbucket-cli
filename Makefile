GO ?= go
BIN_DIR ?= bin
CMD := ./cmd/bkt
SOURCES := $(shell find cmd internal pkg -name '*.go')

VERSION ?= $(shell \
	if git describe --tags --exact-match >/dev/null 2>&1; then \
		git describe --tags --exact-match; \
	else \
		short=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
		if git diff-index --quiet HEAD 2>/dev/null; then \
			echo "dev-$$short"; \
		else \
			echo "dev-$$short-dirty"; \
		fi; \
	fi \
)
COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo unknown)
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X github.com/avivsinai/bitbucket-cli/internal/build.versionFromLdflags=$(VERSION) \
	-X github.com/avivsinai/bitbucket-cli/internal/build.commitFromLdflags=$(COMMIT) \
	-X github.com/avivsinai/bitbucket-cli/internal/build.dateFromLdflags=$(BUILD_DATE)

.PHONY: build fmt lint test tidy sbom release snapshot clean sync-skills check-skills

build: $(BIN_DIR)/bkt

# Skill sync: .claude/skills/ is source of truth
sync-skills:
	@echo "Syncing skills from .claude/skills/ to .codex/skills/ and skills/..."
	@mkdir -p .codex/skills/bkt skills/bkt
	@cp -R .claude/skills/bkt/* .codex/skills/bkt/
	@cp -R .claude/skills/bkt/* skills/bkt/
	@echo "✓ Skills synced"

check-skills:
	@echo "Checking skill sync..."
	@diff -rq .claude/skills/bkt .codex/skills/bkt || (echo "❌ .codex/skills/bkt out of sync" && exit 1)
	@diff -rq .claude/skills/bkt skills/bkt || (echo "❌ skills/bkt out of sync" && exit 1)
	@echo "✓ Skills in sync"

$(BIN_DIR)/bkt: $(SOURCES) go.mod go.sum
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/bkt $(CMD)

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

sbom:
	@if ! command -v syft >/dev/null 2>&1; then \
		echo "syft not installed; install from https://github.com/anchore/syft" >&2; \
		exit 1; \
	fi
	syft dir:. -o cyclonedx-json=sbom.cdx.json

release:
	goreleaser release --clean

snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Run: brew install goreleaser"; exit 1; }
	goreleaser release --snapshot --clean --skip=publish

clean:
	rm -rf $(BIN_DIR) dist/

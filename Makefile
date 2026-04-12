GO ?= go
BIN_DIR ?= bin
CMD := ./cmd/bkt
SOURCES := $(shell find cmd internal pkg -name '*.go')
MACOS_CODESIGN_ID ?= io.github.avivsinai.bitbucket-cli

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
BKT_OAUTH_CLIENT_ID ?=
BKT_OAUTH_CLIENT_SECRET ?=
LDFLAGS := -s -w \
	-X github.com/avivsinai/bitbucket-cli/internal/build.versionFromLdflags=$(VERSION) \
	-X github.com/avivsinai/bitbucket-cli/internal/build.commitFromLdflags=$(COMMIT) \
	-X github.com/avivsinai/bitbucket-cli/internal/build.dateFromLdflags=$(BUILD_DATE) \
	-X github.com/avivsinai/bitbucket-cli/pkg/oauth.cloudClientID=$(BKT_OAUTH_CLIENT_ID) \
	-X github.com/avivsinai/bitbucket-cli/pkg/oauth.cloudClientSecret=$(BKT_OAUTH_CLIENT_SECRET)

.PHONY: build fmt lint test tidy sbom release snapshot clean check-skills check-generated-skill release-local generate-skill

build: $(BIN_DIR)/bkt

# Skill integrity: skills/ is canonical, .claude/skills/ and .agents/skills/ are symlinks
check-skills:
	@echo "Checking skill symlinks..."
	@test -L .claude/skills/bkt || (echo "❌ .claude/skills/bkt is not a symlink" && exit 1)
	@test -L .agents/skills/bkt || (echo "❌ .agents/skills/bkt is not a symlink" && exit 1)
	@test "$$(readlink .claude/skills/bkt)" = "../../skills/bkt" || (echo "❌ .claude/skills/bkt target is not ../../skills/bkt" && exit 1)
	@test "$$(readlink .agents/skills/bkt)" = "../../skills/bkt" || (echo "❌ .agents/skills/bkt target is not ../../skills/bkt" && exit 1)
	@diff -rq skills/bkt .claude/skills/bkt || (echo "❌ .claude/skills/bkt content mismatch" && exit 1)
	@echo "✓ Skill symlinks valid"

$(BIN_DIR)/bkt: $(SOURCES) go.mod go.sum
	@mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/bkt $(CMD)
	./scripts/codesign-macos.sh "$(BIN_DIR)/bkt" "$(MACOS_CODESIGN_ID)"

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

release-local:
	goreleaser release --clean

snapshot:
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Run: brew install goreleaser"; exit 1; }
	goreleaser release --snapshot --clean --skip=publish

clean:
	rm -rf $(BIN_DIR) dist/

generate-skill:
	$(GO) run ./cmd/docgen -o skills/bkt/rules

check-generated-skill:
	@GO="$(GO)" ./scripts/check-generated-skill.sh

release:
	@test -n "$(RELEASE_VERSION)" || (echo "usage: make release RELEASE_VERSION=X.Y.Z [RELEASE_DATE=YYYY-MM-DD] [RELEASE_SKIP_VERIFY=1] [RELEASE_ALLOW_EMPTY=1] [RELEASE_NO_AUTO_MERGE=1]" && exit 1)
	./scripts/release.sh "$(RELEASE_VERSION)" $(if $(RELEASE_DATE),--date $(RELEASE_DATE),) $(if $(RELEASE_SKIP_VERIFY),--skip-verify,) $(if $(RELEASE_ALLOW_EMPTY),--allow-empty,) $(if $(RELEASE_NO_AUTO_MERGE),--no-auto-merge,)

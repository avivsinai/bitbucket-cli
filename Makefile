GO ?= go
BIN_DIR ?= bin
BASH ?= bash
CMD := ./cmd/bkt
MACOS_CODESIGN_ID ?= io.github.avivsinai.bitbucket-cli

ifeq ($(OS),Windows_NT)
# The Windows recipes below use cmd.exe syntax, but GNU Make prefers sh.exe
# from PATH (e.g. Git for Windows) when present — pin the shell explicitly.
SHELL := cmd.exe
.SHELLFLAGS := /C
BIN_EXT := .exe
NULL_DEVICE := NUL
BUILD_DATE_VALUE := $(shell powershell -NoProfile -ExecutionPolicy Bypass -Command "(Get-Date).ToUniversalTime().ToString('yyyy-MM-ddTHH:mm:ssZ')")
MKDIR_P = if not exist "$(subst /,\,$(1))" mkdir "$(subst /,\,$(1))"
RM_RF = if exist "$(subst /,\,$(1))" rmdir /s /q "$(subst /,\,$(1))"
else
BIN_EXT :=
NULL_DEVICE := /dev/null
BUILD_DATE_VALUE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
MKDIR_P = mkdir -p "$(1)"
RM_RF = rm -rf "$(1)"
endif

BIN := $(BIN_DIR)/bkt$(BIN_EXT)
# Tracked and untracked (non-ignored) Go sources; $(wildcard) drops entries
# for files that are tracked but deleted from the working tree.
SOURCES := $(wildcard $(shell git ls-files --cached --others --exclude-standard -- "cmd/*.go" "internal/*.go" "pkg/*.go" 2>$(NULL_DEVICE)))
GIT_TAG := $(shell git describe --tags --exact-match 2>$(NULL_DEVICE))
GIT_COMMIT_SHORT := $(shell git rev-parse --short HEAD 2>$(NULL_DEVICE))
GIT_COMMIT := $(shell git rev-parse HEAD 2>$(NULL_DEVICE))
GIT_DIRTY := $(shell git diff-index --quiet HEAD -- 2>$(NULL_DEVICE) || echo dirty)

VERSION ?= $(if $(GIT_TAG),$(GIT_TAG),dev-$(if $(GIT_COMMIT_SHORT),$(GIT_COMMIT_SHORT),unknown)$(if $(GIT_DIRTY),-dirty,))
COMMIT ?= $(if $(GIT_COMMIT),$(GIT_COMMIT),unknown)
BUILD_DATE ?= $(if $(BUILD_DATE_VALUE),$(BUILD_DATE_VALUE),unknown)
BKT_OAUTH_CLIENT_ID ?=
BKT_OAUTH_CLIENT_SECRET ?=
LDFLAGS := -s -w \
	-X github.com/avivsinai/bitbucket-cli/internal/build.versionFromLdflags=$(VERSION) \
	-X github.com/avivsinai/bitbucket-cli/internal/build.commitFromLdflags=$(COMMIT) \
	-X github.com/avivsinai/bitbucket-cli/internal/build.dateFromLdflags=$(BUILD_DATE) \
	-X github.com/avivsinai/bitbucket-cli/pkg/oauth.cloudClientID=$(BKT_OAUTH_CLIENT_ID) \
	-X github.com/avivsinai/bitbucket-cli/pkg/oauth.cloudClientSecret=$(BKT_OAUTH_CLIENT_SECRET)

.PHONY: build fmt lint test tidy sbom release snapshot clean check-skills check-generated-skill release-local generate-skill nix-update-vendor-hash require-bash

build: $(BIN)

# Skill integrity: skills/ is canonical, .claude/skills/ and .agents/skills/ are symlinks
check-skills:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "$$ErrorActionPreference='Stop'; Write-Host 'Checking skill symlinks...'; foreach ($$link in @('.claude/skills/bkt','.agents/skills/bkt')) { $$item=Get-Item -LiteralPath $$link -Force; if (-not (($$item.Attributes -band [IO.FileAttributes]::ReparsePoint) -ne 0)) { Write-Error ($$link + ' is not a symlink'); exit 1 }; $$target=$$item.Target; if ($$target -is [array]) { $$target=$$target[0] }; if (($$target -ne '../../skills/bkt') -and ($$target -ne '..\..\skills\bkt')) { Write-Error ($$link + ' target is not ../../skills/bkt'); exit 1 } }; Write-Host 'Skill symlinks valid'"
else
	@echo "Checking skill symlinks..."
	@test -L .claude/skills/bkt || (echo "ERROR: .claude/skills/bkt is not a symlink" && exit 1)
	@test -L .agents/skills/bkt || (echo "ERROR: .agents/skills/bkt is not a symlink" && exit 1)
	@test "$$(readlink .claude/skills/bkt)" = "../../skills/bkt" || (echo "ERROR: .claude/skills/bkt target is not ../../skills/bkt" && exit 1)
	@test "$$(readlink .agents/skills/bkt)" = "../../skills/bkt" || (echo "ERROR: .agents/skills/bkt target is not ../../skills/bkt" && exit 1)
	@diff -rq skills/bkt .claude/skills/bkt || (echo "ERROR: .claude/skills/bkt content mismatch" && exit 1)
	@echo "Skill symlinks valid"
endif

$(BIN): $(SOURCES) go.mod go.sum
	@$(call MKDIR_P,$(BIN_DIR))
	$(GO) build -trimpath -ldflags "$(LDFLAGS)" -o "$(BIN)" $(CMD)
ifneq ($(OS),Windows_NT)
	./scripts/codesign-macos.sh "$(BIN)" "$(MACOS_CODESIGN_ID)"
endif

fmt:
	$(GO) fmt ./...

lint:
	golangci-lint run

test:
	$(GO) test ./...

tidy:
	$(GO) mod tidy

sbom:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "if (-not (Get-Command syft -ErrorAction SilentlyContinue)) { Write-Error 'syft not installed; install from https://github.com/anchore/syft'; exit 1 }"
else
	@if ! command -v syft >/dev/null 2>&1; then \
		echo "syft not installed; install from https://github.com/anchore/syft" >&2; \
		exit 1; \
	fi
endif
	syft dir:. -o cyclonedx-json=sbom.cdx.json

release-local:
	goreleaser release --clean

snapshot:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "if (-not (Get-Command goreleaser -ErrorAction SilentlyContinue)) { Write-Error 'goreleaser not installed. Install from https://goreleaser.com/install/'; exit 1 }"
else
	@command -v goreleaser >/dev/null 2>&1 || { echo "goreleaser not installed. Run: brew install goreleaser"; exit 1; }
endif
	goreleaser release --snapshot --clean --skip=publish

clean:
	@$(call RM_RF,$(BIN_DIR))
	@$(call RM_RF,dist)

generate-skill:
	$(GO) run ./cmd/docgen -o skills/bkt/rules

check-generated-skill: require-bash
ifeq ($(OS),Windows_NT)
	@set "GO=$(GO)" && "$(BASH)" ./scripts/check-generated-skill.sh
else
	@GO="$(GO)" "$(BASH)" ./scripts/check-generated-skill.sh
endif

nix-update-vendor-hash: require-bash
	"$(BASH)" ./scripts/update-nix-vendor-hash.sh

release: require-bash
ifeq ($(OS),Windows_NT)
	@if "$(RELEASE_VERSION)"=="" (echo usage: mingw32-make release RELEASE_VERSION=X.Y.Z [RELEASE_DATE=YYYY-MM-DD] [RELEASE_SKIP_VERIFY=1] [RELEASE_ALLOW_EMPTY=1] [RELEASE_NO_AUTO_MERGE=1] & exit /b 1)
	"$(BASH)" ./scripts/release.sh "$(RELEASE_VERSION)" $(if $(RELEASE_DATE),--date $(RELEASE_DATE),) $(if $(RELEASE_SKIP_VERIFY),--skip-verify,) $(if $(RELEASE_ALLOW_EMPTY),--allow-empty,) $(if $(RELEASE_NO_AUTO_MERGE),--no-auto-merge,)
else
	@test -n "$(RELEASE_VERSION)" || (echo "usage: make release RELEASE_VERSION=X.Y.Z [RELEASE_DATE=YYYY-MM-DD] [RELEASE_SKIP_VERIFY=1] [RELEASE_ALLOW_EMPTY=1] [RELEASE_NO_AUTO_MERGE=1]" && exit 1)
	"$(BASH)" ./scripts/release.sh "$(RELEASE_VERSION)" $(if $(RELEASE_DATE),--date $(RELEASE_DATE),) $(if $(RELEASE_SKIP_VERIFY),--skip-verify,) $(if $(RELEASE_ALLOW_EMPTY),--allow-empty,) $(if $(RELEASE_NO_AUTO_MERGE),--no-auto-merge,)
endif

require-bash:
ifeq ($(OS),Windows_NT)
	@powershell -NoProfile -ExecutionPolicy Bypass -Command "if (-not (Get-Command '$(BASH)' -ErrorAction SilentlyContinue)) { Write-Error '$(BASH) not installed; required for this target'; exit 1 }"
else
	@command -v "$(BASH)" >/dev/null 2>&1 || { echo "$(BASH) not installed; required for this target" >&2; exit 1; }
endif

# Documentation Audit Report

Generated: 2026-02-11 | Commit: dbb5c7d

## Executive Summary

| Metric | Count |
|--------|-------|
| Documents scanned | 4 (user-facing only) |
| Claims verified | ~95 |
| Verified TRUE | ~62 (65%) |
| **Verified FALSE** | **33 (35%)** |

Documents audited: `README.md`, `cmd/digest/skill/SKILL.md`, `code-review.md`, `CHARM_REMOVAL_PLAN.md`

Skipped (plans/internal): `docs/plans/*.md`, `missing-tests.md` (advisory, not factual claims)

---

## False Claims Requiring Fixes

### README.md

| Line | Claim | Reality | Severity | Fix |
|------|-------|---------|----------|-----|
| 11 | `digest feed discover` command exists | Discovery is integrated into `feed add`, not a separate command | Medium | Remove or rewrite as "Auto-discover built into `feed add`" |
| 76 | `make lint` target exists | Makefile has no `lint` target | Low | Remove from README or add target to Makefile |
| 86 | `digest feed discover https://example.com` | No `discover` subcommand; use `digest feed add` which auto-discovers | Medium | Update example to `digest feed add` |
| 95 | `digest sync` command | Command is named `digest fetch`, not `digest sync` | High | Change to `digest fetch` |
| 146 | Database at `~/.config/digest/digest.db` | Actual: `~/.local/share/digest/digest.db` (XDG_DATA_HOME) | High | Update path |
| 147 | Subscriptions at `~/.config/digest/feeds.opml` | Actual: `~/.local/share/digest/feeds.opml` | High | Update path |
| N/A | Missing commands | `folder add/list`, `mark-unread`, `open`, `setup`, `migrate`, `export` (YAML/Markdown) not documented | Medium | Add to CLI usage section |
| N/A | Missing `--no-discover` flag docs | `feed add` has `--no-discover` and `--title` flags | Low | Add to CLI examples |

### cmd/digest/skill/SKILL.md

| Line | Claim | Reality | Severity | Fix |
|------|-------|---------|----------|-----|
| 8 | "full-text search" capability | No search MCP tool or CLI command exists | High | Remove "full-text search" claim or implement search |
| 24 | `mcp__digest__list_articles` tool | Actual tool name: `list_entries` | High | Rename to `list_entries` |
| 27 | `mcp__digest__search_articles` tool | Does not exist | High | Remove entirely |
| 28 | `mcp__digest__refresh_feeds` tool | Actual tool name: `sync_feeds` | Medium | Rename to `sync_feeds` |
| 34 | `add_feed` takes `title` parameter | MCP tool only accepts `url` and `folder` | Medium | Update signature |
| 59 | `digest add` command | Actual: `digest feed add` | High | Fix command |
| 60 | `digest list` lists feeds | Actual: `digest feed list` (feeds) or `digest list` (entries) | High | Fix command |
| 61 | `digest articles` command | Actual: `digest list` | High | Fix command |
| 62 | `digest articles --all` | Actual: `digest list --all` | High | Fix command |
| 64 | `digest search "query"` | Does not exist | High | Remove |
| 65 | `digest refresh` command | Actual: `digest fetch` | High | Fix command |
| 73 | Data at `~/.local/share/digest/digest.db` with XDG_DATA_HOME | Path is correct but description is incomplete: now supports markdown backend too | Low | Add note about backend choice |

### code-review.md

| Line | Claim | Reality | Severity | Fix |
|------|-------|---------|----------|-----|
| 8 | "OPML as source of truth for subscriptions" | OPML is still used, but storage backends (SQLite, Markdown) now store feeds too | Low | Outdated architecture claim |
| 21 | References `internal/db/` package (entries.go, feeds.go) | Package does not exist; replaced by `internal/storage/` | High | Update all `internal/db/` references |
| 30 | Line references in `feeds.go` (Line 33-41) | File doesn't exist | High | Remove or update |
| 31 | Line references in `entries.go` (Line 212-228, Line 117-151) | File doesn't exist | High | Remove or update |
| 60 | "Beautiful colored output with `fatih/color`" | fatih/color is still used - CORRECT | - | - |
| 63 | "Line 88-95 in `read.go`: Good markdown rendering with graceful fallback" | glamour is no longer used; read.go uses plain text | Medium | Update |
| 76 | "Line 14 in `db.go`: Restrictive directory permissions" | `db.go` doesn't exist | High | Update reference |
| 85 | "Lines 198-245 in `test/integration_test.go`" | File doesn't exist | High | Remove reference |
| 90 | "Line 319-385 in `resources.go`: `calculateStats`" | Line numbers likely stale | Low | Verify/update |
| 103 | "Line 17-21 in `.goreleaser.yml`: Only builds for macOS due to CGO" | Builds for macOS, Linux, AND Windows; CGO_ENABLED=0 | High | Update - cross-platform is supported |
| 104 | "Missing configuration file support" | Config file (`~/.config/digest/config.json`) exists | High | Remove claim |

### CHARM_REMOVAL_PLAN.md

| Line | Claim | Reality | Severity | Fix |
|------|-------|---------|----------|-----|
| 6-7 | Lists charm and glamour as deps to REMOVE | Both already removed; plan is completed | Medium | Mark plan as completed or archive |
| 15 | `internal/charm/client.go` exists | Directory and files deleted | Medium | Plan completed |
| 16 | `cmd/digest/sync.go` exists | Renamed to `fetch.go` | Medium | Plan completed |
| 23 | `glamour.Render` used in read.go | glamour removed; using plain text | Medium | Plan completed |
| 149-150 | Files to CREATE: `internal/storage/migrations.go` | Created as `internal/storage/migrate.go` instead | Low | Minor naming difference |

---

## Pattern Summary

| Pattern | Count | Root Cause |
|---------|-------|------------|
| Dead file references (`internal/db/*`, `internal/charm/*`) | 9 | Major refactor (charm removal + storage abstraction) completed, docs not updated |
| Wrong CLI command names | 8 | SKILL.md appears auto-generated or written from different codebase version |
| Wrong data paths | 3 | Storage moved from `~/.config/` to `~/.local/share/` per XDG spec |
| Stale line number references | 5 | Code evolved but code-review.md not refreshed |
| Non-existent features claimed | 3 | `search_articles`, `digest discover`, `digest search` never implemented |
| Completed plan presented as TODO | 4 | CHARM_REMOVAL_PLAN.md fully executed but not archived |

---

## Pass 2: Gap Detection (Documented vs Actual)

### Undocumented Commands (exist in code, missing from docs)

| Command | File | Notes |
|---------|------|-------|
| `digest folder add <name>` | `cmd/digest/folder.go:18` | Folder management |
| `digest folder list` | `cmd/digest/folder.go:41` | List folders |
| `digest mark-unread <id>` | `cmd/digest/markunread.go:12` | Reverse mark-read |
| `digest open <id>` | `cmd/digest/open.go:15` | Open entry link in browser |
| `digest setup` | `cmd/digest/setup.go` | Interactive TUI setup wizard |
| `digest migrate` | `cmd/digest/migrate.go` | Migrate between storage backends |
| `digest install-skill` | `cmd/digest/skill.go:22` | Install Claude Code skill |
| `digest version` | `cmd/digest/version.go` | Show version info |
| `digest export --format yaml` | `cmd/digest/export.go` | YAML export (only OPML documented) |
| `digest export --format markdown` | `cmd/digest/export.go` | Markdown export |

### Undocumented Features

| Feature | Location | Notes |
|---------|----------|-------|
| Markdown storage backend | `internal/storage/markdown*.go` | Alternative to SQLite, uses mdstore |
| Backend selection config | `internal/config/config.go` | `config.json` with Backend field |
| TUI setup wizard | `internal/tui/setup.go` | Charmbracelet bubbletea-based |
| Migration between backends | `internal/storage/migrate.go` | SQLite <-> Markdown |
| `--force` flag on fetch | `cmd/digest/fetch.go` | Force re-fetch ignoring cache |
| `--no-mark` flag on read | `cmd/digest/read.go` | Read without marking as read |
| `--week` / `--yesterday` flags | `cmd/digest/list.go` | Additional date filters |

---

## Human Review Queue

- [ ] SKILL.md is severely outdated - nearly every CLI command and 3 MCP tool names are wrong. **Recommend full rewrite.**
- [ ] code-review.md references deleted packages (`internal/db/`, `internal/charm/`). **Recommend regenerating or deleting.**
- [ ] CHARM_REMOVAL_PLAN.md is fully completed. **Recommend archiving or deleting.**
- [ ] README.md needs moderate updates for CLI commands, storage paths, and missing features.
- [ ] Verify whether `search_articles` / full-text search should be implemented (FTS5 exists in SQLite schema but no tool/CLI exposes it).

---

## Recommended Priority

1. **Critical**: Rewrite SKILL.md (actively used by Claude Code, completely wrong)
2. **High**: Fix README.md CLI commands and data paths
3. **Medium**: Archive/delete completed CHARM_REMOVAL_PLAN.md
4. **Medium**: Regenerate or delete stale code-review.md
5. **Low**: Add documentation for undocumented commands and features

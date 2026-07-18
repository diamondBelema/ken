# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2025-07-18

### Added
- `ken notes <subject>` — interactive note management (create, edit, delete, search)
- `ken summaries <subject>` — summary management with subject-scoped summaries
- `ken read <subject>` — read plain markdown content from `notes/` directory
- Note taking in flashcard/quiz study (`n` key, non-interrupting, auto-linked to current context)
- Notes can link to concepts, cards, quizzes, other notes, or be unlinked
- Summaries: content-parsed (`## <id>:summary`) + user-created, both shown when they exist
- Concept initialization: concepts auto-loaded when running flashcards/quiz
- Markdown rendering via glamour for all user-facing content
- Mermaid diagram support: ASCII quick view + SVG export
- Diagram/Link fields in concept frontmatter
- Vim motions (j/k/gg/G) in list views
- Dashboard shows note/summary/read file counts

### Changed
- Parser now extracts Diagram, Link, and Summary fields from concept files
- Progress model includes Notes and Summaries collections with CRUD methods
- Flashcard/quiz TUI shows note input panel without interrupting study flow

## [0.2.0] - 2025-07-18

### Added
- Phase 5: `ken` bare command — dashboard with confidence spread per subject
- Phase 5: `ken progress [subject]` — per-concept confidence breakdown
- Phase 5: `ken stats` — placeholder for confidence trends (no data yet)
- Empty state handling: zero subjects, missing progress files

## [0.1.0] - 2025-07-18

### Added
- Phase 1: Cobra CLI scaffold with `ken subjects` command
- Phase 1: `internal/discovery` — scan `~/Documents/learn/subjects/`, count `.md` files per category
- Phase 1: Clear error message when learn directory is missing
- Phase 1.5: `internal/parser` — YAML frontmatter splitting + concept set parsing
- Phase 1.5: `internal/progress` — `Load`/`Save`/`InitConcepts` for progress state
- Phase 2: `internal/mastery` — BayesianConfidenceStrategyV2 port with exact constants
- Phase 2: 7 unit tests covering decay, anomaly tolerance, bounds, fixture sequence
- Phase 3: `internal/parser/flashcard.go` — flashcard set parsing with notes association
- Phase 3: `internal/study` — flashcard session loading with duplicate ID detection
- Phase 3: `internal/tui` — bubbletea flashcard study model (front/back flip, 5-level grading)
- Phase 3: `cmd/ken/flashcards.go` — `ken flashcards <subject>` command
- XDG-compliant state separation: `~/.local/share/ken/` for all writable state
- Content directory (`~/Documents/learn/subjects/`) is now 100% read-only

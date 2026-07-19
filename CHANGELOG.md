# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Concept detail view (`c` key) in flashcard and quiz study — shows concept name, description, summary, diagrams, links, note/summary counts
- Concept detail view uses glamour markdown rendering with scroll support (j/k/g/G)
- Shared helpers: `renderProgressBar`, `runeTruncate`, `buildConceptMap`, `lookupConcept`, `renderConceptDetail`, `renderUserNotes`
- Content summaries from concept files shown in summaries view with `[content]` label
- Notes display in flashcard/quiz views — shows all notes linked to the concept AND the specific card/quiz
- Note tab-cycling: attach to concept, specific card/quiz, or unlinked
- Notes page: link notes to concepts via tab-cycling when creating new notes
- Notes detail view shows linked entity

### Fixed
- Progress view scroll — overhead calculation was wrong (viewHeight-4 → viewHeight-6), causing beginning concepts to be cut off at terminal bottom
- Flashcard note input returns to correct state (front or back) instead of always returning to back
- Detail view scrolling in summaries with cached rendered lines

### Changed
- Flashcard/quiz constructors now accept `[]parser.Concept` for concept detail support
- Concept detail and summary detail use `render.RenderMarkdown` (glamour) for proper markdown rendering

### Removed
- Dead styles: `statusBarStyle`, `borderStyle`, `dashBadgeNeverStyle`, `dashConfBarFilled`, `dashConfBarEmpty`
- Dead types: `flashcardQuitMsg`
- Duplicate `renderProgressBar` methods from flashcard/quiz models (now shared)
- Duplicate `runeSafeTruncate` from progress model (now shared)

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

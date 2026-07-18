# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- XDG-compliant state separation: `~/.local/share/ken/` for all writable state
- Content directory (`~/Documents/learn/subjects/`) is now 100% read-only

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

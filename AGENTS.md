# AGENTS.md

## What this is

`ken` is a terminal-based spaced-repetition study harness. It reads flashcard, quiz, and concept sets from a folder of Markdown+YAML-frontmatter files, tracks mastery via a Bayesian confidence algorithm (ported from Expo/Claudy's `BayesianConfidenceStrategyV2`), and presents a TUI. The spec is the source of truth: `ken-spec.md`.

## Project state

All 5 build phases complete. CLI commands working: `ken` (dashboard), `ken subjects`, `ken flashcards <subject>`, `ken quiz <subject>`, `ken progress [subject]`, `ken stats`, `ken notes <subject>`, `ken summaries <subject>`, `ken read <subject>`. TUI renders with bubbletea. Mastery engine has 7 passing tests. New features implemented: notes, summaries, diagrams (external SVG + mermaid), links, markdown rendering, and content reading.

## Tech stack

- **Go** (module: `github.com/diamondBelema/ken`)
- **CLI:** Cobra
- **TUI:** Charmbracelet stack — `bubbletea` v1.3.10, `bubbles` v1.0.0, `lipgloss` v1.1.0
- **Markdown:** `glamour` v2 for rendering markdown in TUI
- **Diagrams:** `mermaigo` (ASCII), `go-mermaid` (SVG) — both integrated
- **Parsing:** `gopkg.in/yaml.v3` for YAML frontmatter

## Build & run

```bash
go build -o ken ./cmd/ken    # build binary
go run ./cmd/ken             # run directly
go vet ./...                 # static analysis
go test ./...                # unit tests
```

## Package layout

```
cmd/ken/
  main.go           # cobra root command + dashboard
  subjects.go       # ken subjects — list subjects with file counts
  flashcards.go     # ken flashcards <subject> — launch TUI study
  quiz.go           # ken quiz <subject> — launch quiz TUI
  progress.go       # ken progress [subject] — show progress
  stats.go          # ken stats — aggregate stats
  notes.go          # ken notes <subject> — manage user notes
  summaries.go      # ken summaries <subject> — manage summaries
  read.go           # ken read <subject> — read lecture notes/content
internal/
  discovery/        # scan ~/Documents/learn/subjects/
  mastery/          # Bayesian confidence engine + 7 tests
  parser/           # YAML frontmatter + markdown body parsing
  progress/         # progress state read/write (XDG data dir)
  study/            # study session logic (flashcard + quiz)
  tui/              # bubbletea models, views, update loops
  render/           # markdown rendering via glamour
  diagram/          # mermaid rendering wrapper
```

## Key conventions from the spec

- **Confidence, not SM-2.** Mastery lives on *concepts*, not cards. Cards/quizzes are evidence that updates a concept's Bayesian confidence score. The full Go algorithm is in `ken-spec.md` (lines 142–268) — it's copy-paste ready with exact constants: `decayRatePerDay = 0.95`, `inertia = 0.8`, `maxDailyDelta = 0.08`, confidence bounds `0.05`–`0.995`.
- **5-level grading, not 4.** Flashcard study presents Unknown/KnownLittle/KnownFairly/KnownWell/Mastered — don't remap a 4-button scheme onto this, the algorithm depends on the 5-level likelihood mapping.
- **`concept_id` is the link.** Cards and questions with a `concept_id` update that concept's confidence on grade. Without one, the card still studies but contributes no mastery signal.
- **IDs must be unique per subject** across all files — parser must error on collision at load time.
- **Unknown quiz types** (`mcq`, `true_false`, `fill_blank` are valid) → warn + skip, never crash.
- **Anomaly tolerance** in quiz grading: a miss from a concept with confidence > 0.75 uses likelihood 0.45 (assume slip), not 0.3 (true gap). Don't flatten this.
- **No generation, no network, no auth** — single local user, single machine.

## Data architecture — content vs state separation

**Content (read-only):** `~/Documents/learn/subjects/<subject>/`
- `concepts/*.md` — concept definitions with hierarchy, diagrams, links, summaries
- `flashcards/*.md` — flashcard sets
- `quizzes/*.md` — quiz sets
- Nothing in this tree is ever written to by ken.

**State (writable):** `~/.local/share/ken/`
- `<subject>.json` — per-subject progress (concepts, cards, quizzes, notes, summaries)
- `stats.json` — aggregate stats (Phase 5)
- Follows XDG Base Directory spec. Content directory stays 100% read-only.

This means:
- Git repos containing course materials stay clean (no progress files)
- Multiple users can study the same content independently
- Reinstalling/updating content never touches progress

## New features (in progress)

- **Notes:** User-created, never interrupt learning flow. Auto-linked to current context. Can link to other notes. Editable/deletable. Markdown content rendered with glamour.
- **Summaries:** Content-parsed (`## <id>:summary`) + user-created. Both shown when they exist. Scoped to concept/concepts/subject.
- **Diagrams:** Mermaid syntax, inline source or external file. ASCII quick view (mermaigo) + SVG export (go-mermaid).
- **Links:** URL + label + type, stored in content files.
- **Markdown rendering:** glamour v2 for all user-facing text content.

## When working on this repo

- Read `ken-spec.md` before writing any code. It defines exact file formats, CLI commands, the full mastery algorithm, acceptance criteria per phase, and out-of-scope items.
- Check `FEATURE-PLAN.md` for the new features plan.
- Build command: `go build -o ken ./cmd/ken` (entrypoint is `cmd/ken/`, not root).
- Tests: `go test ./...` — currently 7 tests in `internal/mastery/`.

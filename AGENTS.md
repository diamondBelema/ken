# AGENTS.md

## What this is

`ken` is a terminal-based spaced-repetition study harness. It reads flashcard, quiz, and concept sets from a folder of Markdown+YAML-frontmatter files, tracks mastery via a Bayesian confidence algorithm (ported from Expo/Claudy's `BayesianConfidenceStrategyV2`), and presents a TUI. The spec is the source of truth: `ken-spec.md`.

## Project state

Early scaffold. Only `main.go` has code (placeholder print). All `internal/` and `cmd/` directories are empty. Dependencies are declared in `go.mod` but nothing imports them yet.

## Tech stack

- **Go** (module: `github.com/diamondBelema/ken`)
- **TUI:** Charmbracelet stack — `bubbletea`, `bubbles`, `lipgloss`
- **Parsing:** `gopkg.in/yaml.v3` for YAML frontmatter
- No CLI framework declared yet (spec mentions cobra or plain `flag`)

## Build & run

```bash
go build -o ken ./cmd/ken    # build binary
go run ./cmd/ken             # run directly
go vet ./...                 # static analysis
go test ./...                # unit tests
```

No Makefile, CI, linter config, or test suite exists yet.

## Package layout

```
cmd/ken/
  main.go           # cobra root command
  subjects.go       # ken subjects — list subjects with file counts
  flashcards.go     # ken flashcards <subject> — launch TUI study
internal/
  discovery/        # scan ~/Documents/learn/subjects/
  mastery/          # Bayesian confidence engine (port of BayesianConfidenceStrategyV2)
  parser/           # YAML frontmatter + markdown body parsing
  progress/         # progress state read/write (XDG data dir)
  study/            # study session logic
  tui/              # bubbletea models, views, update loops
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
- `concepts/*.md` — concept definitions with hierarchy
- `flashcards/*.md` — flashcard sets
- `quizzes/*.md` — quiz sets
- Nothing in this tree is ever written to by ken.

**State (writable):** `~/.local/share/ken/`
- `<subject>.json` — per-subject progress (concepts, cards, quizzes)
- `stats.json` — aggregate stats (Phase 5)
- Follows XDG Base Directory spec. Content directory stays100% read-only.

This means:
- Git repos containing course materials stay clean (no progress files)
- Multiple users can study the same content independently
- Reinstalling/updating content never touches progress

## When working on this repo

- Read `ken-spec.md` before writing any code. It defines exact file formats, CLI commands, the full mastery algorithm, acceptance criteria per phase, and out-of-scope items.
- The build is phased (5 phases + a 1.5 in the spec). Check which phase is current before starting work.
- No tests exist yet. When tests are added, expect standard `go test ./...` behavior.

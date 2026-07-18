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
go build -o ken .        # build binary
go run .                 # run directly
go vet ./...             # static analysis
```

No Makefile, CI, linter config, or test suite exists yet.

## Package layout

```
cmd/ken/          # intended CLI entrypoint (currently empty)
internal/
  mastery/        # Bayesian confidence engine (port of BayesianConfidenceStrategyV2)
  parser/         # YAML frontmatter + markdown body parsing
  progress/       # progress.json read/write
  study/          # study session logic
  tui/            # bubbletea models, views, update loops
```

All `internal/` packages are empty. Code goes in `main.go` for now.

## Key conventions from the spec

- **Confidence, not SM-2.** Mastery lives on *concepts*, not cards. Cards/quizzes are evidence that updates a concept's Bayesian confidence score. The full Go algorithm is in `ken-spec.md` (lines 142–268) — it's copy-paste ready with exact constants: `decayRatePerDay = 0.95`, `inertia = 0.8`, `maxDailyDelta = 0.08`, confidence bounds `0.05`–`0.995`.
- **5-level grading, not 4.** Flashcard study presents Unknown/KnownLittle/KnownFairly/KnownWell/Mastered — don't remap a 4-button scheme onto this, the algorithm depends on the 5-level likelihood mapping.
- **`concept_id` is the link.** Cards and questions with a `concept_id` update that concept's confidence on grade. Without one, the card still studies but contributes no mastery signal.
- **`progress.json`** is the only file `ken` writes. It has three maps: `concepts` (confidence + last_reviewed_at), `cards` (reviews + last_grade), `quizzes` (attempts + correct + streak). Everything else under `subjects/` is read-only input.
- **IDs must be unique per subject** across all files — parser must error on collision at load time.
- **Unknown quiz types** (`mcq`, `true_false`, `fill_blank` are valid) → warn + skip, never crash.
- **Anomaly tolerance** in quiz grading: a miss from a concept with confidence > 0.75 uses likelihood 0.45 (assume slip), not 0.3 (true gap). Don't flatten this.
- **No generation, no network, no auth** — single local user, single machine.

## Data directory

`~/Documents/learn/` — subjects are discovered by scanning directories under `subjects/`. No registry file. Each subject can contain `concepts/`, `flashcards/`, and `quizzes/` subdirectories.

## When working on this repo

- Read `ken-spec.md` before writing any code. It defines exact file formats, CLI commands, the full mastery algorithm, acceptance criteria per phase, and out-of-scope items.
- The build is phased (5 phases + a 1.5 in the spec). Check which phase is current before starting work.
- No tests exist yet. When tests are added, expect standard `go test ./...` behavior.

# AGENTS.md

## What this is

`ken` is a terminal-based spaced-repetition study harness. It reads flashcard, quiz, and concept sets from a folder of Markdown+YAML-frontmatter files, tracks mastery via a Bayesian confidence algorithm (ported from Expo/Claudy's `BayesianConfidenceStrategyV2`), and presents a TUI. The spec is the source of truth: `ken-spec.md`.

## Project state

All 5 build phases complete. CLI commands working: `ken` (dashboard), `ken subjects`, `ken flashcards <subject>`, `ken quiz <subject>`, `ken progress [subject]`, `ken stats`, `ken notes <subject>`, `ken summaries <subject>`, `ken read <subject>`, `ken lint [subject]`. TUI renders with bubbletea. Mastery engine has 7 passing tests. Features implemented: notes, summaries, diagrams (external SVG + mermaid), links, markdown rendering, content reading. Registry/marketplace system: `ken search`, `ken add`, `ken list`, `ken remove`, `ken package`, `ken publish`. Cross-platform support (Linux + Windows). File locking prevents race conditions on concurrent installs.

## Tech stack

- **Go** (module: `github.com/diamondBelema/ken`)
- **CLI:** Cobra
- **TUI:** Charmbracelet stack — `bubbletea` v1.3.10, `bubbles` v1.0.0, `lipgloss` v1.1.0
- **Markdown:** `glamour` v2 for rendering markdown in TUI
- **Diagrams:** `mermaigo` (ASCII), `go-mermaid` (SVG) — both integrated
- **Parsing:** `gopkg.in/yaml.v3` for YAML frontmatter
- **Registry:** GitHub-hosted index (`diamondBelema/ken-registry`), HTTP tarball downloads (no git required on user machine)

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
  lint.go           # ken lint [subject] — validate content files
  add.go            # ken add <package> — install study package from registry
  search.go         # ken search <query> — search registry for packages
  list.go           # ken list — list installed packages
  remove.go         # ken remove <package> — uninstall a package
  package.go        # ken package — create package manifest from local content
  publish.go        # ken publish — publish packages to registry
  launchers.go      # shared launcher helpers (diagram opening, link opening)
internal/
  discovery/        # scan ~/Documents/learn/subjects/
  mastery/          # Bayesian confidence engine + 7 tests
  parser/           # YAML frontmatter + markdown body parsing + ken.yaml manifest
  progress/         # progress state read/write (XDG data dir)
  study/            # study session logic (flashcard + quiz)
  tui/              # bubbletea models, views, update loops
  render/           # markdown rendering via glamour
  diagram/          # mermaid rendering wrapper
  lint/             # content validation (parse errors, duplicate IDs, broken refs)
  registry/         # package registry (client, install, publish, state with file locking)
  system/           # cross-platform helpers (file opening, path resolution)
```

## Key conventions from the spec

- **Confidence, not SM-2.** Mastery lives on *concepts*, not cards. Cards/quizzes are evidence that updates a concept's Bayesian confidence score. The full Go algorithm is in `ken-spec.md` (lines 142–268) — it's copy-paste ready with exact constants: `decayRatePerDay = 0.95`, `inertia = 0.8`, `maxDailyDelta = 0.08`, confidence bounds `0.05`–`0.995`.
- **5-level grading, not 4.** Flashcard study presents Unknown/KnownLittle/KnownFairly/KnownWell/Mastered — don't remap a 4-button scheme onto this, the algorithm depends on the 5-level likelihood mapping.
- **`concept_id` is the link.** Cards and questions with a `concept_id` update that concept's confidence on grade. Without one, the card still studies but contributes no mastery signal.
- **IDs must be unique per subject** across all files — parser must error on collision at load time.
- **Unknown quiz types** (`mcq`, `true_false`, `fill_blank` are valid) → warn + skip, never crash.
- **Anomaly tolerance** in quiz grading: a miss from a concept with confidence > 0.75 uses likelihood 0.45 (assume slip), not 0.3 (true gap). Don't flatten this.
- **Study is offline.** The study loop (flashcards, quiz, progress) never touches the network. Only registry operations (search, add, publish) use the network.

## Data architecture — content vs state separation

**Content (read-only):** `~/Documents/learn/subjects/<subject>/`
- `concepts/*.md` — concept definitions with hierarchy, diagrams, links, summaries
- `flashcards/*.md` — flashcard sets
- `quizzes/*.md` — quiz sets
- `notes/` — raw readable content (lecture slides, textbook extracts)
- Nothing in this tree is ever written to by ken.

**State (writable):** `~/.local/share/ken/` (Linux) / `~/AppData/Local/ken/` (Windows)
- `<subject>.json` — per-subject progress (concepts, cards, quizzes, notes, summaries)
- `stats.json` — aggregate stats
- `registry.json` — installed packages state (locked via file lock during writes)
- Follows XDG Base Directory spec on Linux. Content directory stays 100% read-only.

This means:
- Git repos containing course materials stay clean (no progress files)
- Multiple users can study the same content independently
- Reinstalling/updating content never touches progress

## Registry / marketplace

Packages are hosted on GitHub. The registry index lives at `github.com/diamondBelema/ken-registry` (a single `registry.json`). Package content lives at `github.com/diamondBelema/ken-subjects` (all subjects in one repo).

- `ken search <query>` — search registry by name/description/tags
- `ken add <author/package>` — download and install a package (HTTP tarball, no git required)
- `ken list` — show installed packages
- `ken remove <author/package>` — uninstall and delete content
- `ken package` — generate `ken.yaml` manifest from local content
- `ken publish` — push packages to registry (requires `gh` CLI, creates PR)

Package ID format: `author/package` (e.g. `diamondBelema/nucleic-acid`). Manifest format: `ken.yaml` with id, name, version, author, description, license, subjects, concepts, flashcards, tags.

Concurrent installs are protected by file locking (`internal/registry/lock_unix.go`, `lock_windows.go`). The lock covers the entire load-modify-save cycle for `registry.json`.

## When working on this repo

- Read `ken-spec.md` before writing any code. It defines exact file formats, CLI commands, the full mastery algorithm, acceptance criteria per phase, and out-of-scope items.
- Check `FEATURE-PLAN.md` for the new features plan (completed).
- Build command: `go build -o ken ./cmd/ken` (entrypoint is `cmd/ken/`, not root).
- Tests: `go test ./...` — currently 7 tests in `internal/mastery/`.
- Website: `docs/` directory, served via GitHub Pages. `index.html` is the landing page, `docs.html` is the documentation.
- CI/CD: GitHub Actions workflow at `.github/workflows/release.yml` builds Linux + Windows binaries on tag push.

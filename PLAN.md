# ken — Full Implementation Plan (COMPLETED)

> **Status: All 5 phases complete.** This document is preserved for reference.

## Target Architecture (final state)

```
cmd/ken/
  main.go              # cobra root + subcommand registration + Execute()
  subjects.go          # ken subjects — list subjects with file counts
  flashcards.go        # ken flashcards <subject> — launch TUI study
  quiz.go              # ken quiz <subject> — launch TUI quiz
  progress.go          # ken progress [subject] — print confidence breakdown
  stats.go             # ken stats — print confidence trend + streak
internal/
  discovery/
    discovery.go       # DiscoverSubjects() — scan ~/Documents/learn/subjects/
  parser/
    parser.go          # SplitFrontmatter() — shared --- delimiter split + YAML unmarshal
    concept.go         # ParseConceptSet() — concept file → []Concept
    flashcard.go       # ParseFlashcardSet() — flashcard file → FlashcardSet + notes
    quiz.go            # ParseQuizSet() — quiz file → QuizSet, warn+skip bad questions
  mastery/
    mastery.go         # Bayesian confidence engine (spec lines 142–268, copy-paste)
    mastery_test.go    # unit tests per spec acceptance criteria
  progress/
    progress.go        # Load/Save/Create state files (~/.local/share/ken/), CRUD for concept/card/quiz state
  study/
    flashcards.go      # FlashcardSession — loads cards, drives study loop, writes progress
    quiz.go            # QuizSession — loads questions, drives quiz loop, writes progress
  tui/
    app.go             # root bubbletea model, command dispatch
    styles.go          # lipgloss theme (colors, spacing, borders)
    subjects.go        # subjects list view
    flashcards.go      # flashcard study view (front/back flip, 5-button grade)
    quiz.go            # quiz view (mcq/true_false/fill_blank rendering, feedback)
    dashboard.go       # bare `ken` — confidence spread, streak, nav
    progress.go        # per-concept confidence breakdown, decay highlighting
    stats.go           # confidence trend, streak history from stats.json
```

## Dependencies (already installed)

| Package | Purpose | Phase needed |
|---|---|---|
| `github.com/spf13/cobra` | CLI subcommand routing | 1 |
| `github.com/charmbracelet/bubbletea` | TUI framework | 3 |
| `github.com/charmbracelet/bubbles` | TUI components (keys, lists) | 3 |
| `github.com/charmbracelet/lipgloss` | TUI styling | 3 |
| `gopkg.in/yaml.v3` | YAML frontmatter parsing | 1.5 |

No additional `go get` needed.

---

## Phase 1 — Scaffold + Folder Discovery

### Files

| File | Action |
|---|---|
| `cmd/ken/main.go` | Create — cobra root command, `Execute()` |
| `cmd/ken/subjects.go` | Create — `subjects` cobra command, calls discovery, prints table |
| `internal/discovery/discovery.go` | Create — `DiscoverSubjects(dir) []SubjectInfo` |
| `main.go` | Delete — entrypoint moves to `cmd/ken/` |
| `.gitignore` | Create — ignore `ken` binary |

### Implementation

**`cmd/ken/main.go`** — cobra root:
- `rootCmd` with `Use: "ken"`, short description
- `Execute()` calls `rootCmd.Execute()`
- No `--help` customization needed (cobra generates it)

**`cmd/ken/subjects.go`** — subjects command:
- `subjectsCmd` added to root in `init()`
- Calls `discovery.DiscoverSubjects(learnDir + "/subjects")`
- Prints table: `name: N concepts, N flashcards, N quizzes`
- If discover returns error (missing dir), print message + exit 1

**`internal/discovery/discovery.go`**:
- `SubjectInfo struct { Name string; ConceptFiles, FlashcardFiles, QuizFiles int }`
- `DiscoverSubjects(subjectsDir string) ([]SubjectInfo, error)`
- `os.ReadDir` → filter dirs → count `.md` files in `concepts/`, `flashcards/`, `quizzes/`
- Missing `subjectsDir` → return `fmt.Errorf("learn directory not found at %s — create it and add subject folders", subjectsDir)`
- Missing subdirs (e.g., no `concepts/`) → count 0, don't error

### Build & run

```bash
go build -o ken ./cmd/ken    # note: ./cmd/ken, not .
./ken subjects
```

### Acceptance

1. Create test data:
```bash
mkdir -p ~/Documents/learn/subjects/biochemistry/{concepts,flashcards,quizzes}
mkdir -p ~/Documents/learn/subjects/physiology/{concepts,flashcards}
touch ~/Documents/learn/subjects/biochemistry/concepts/glycolysis.md
touch ~/Documents/learn/subjects/biochemistry/flashcards/glycolysis.md
touch ~/Documents/learn/subjects/biochemistry/flashcards/citric-acid-cycle.md
touch ~/Documents/learn/subjects/biochemistry/quizzes/glycolysis.md
touch ~/Documents/learn/subjects/physiology/concepts/homeostasis.md
touch ~/Documents/learn/subjects/physiology/flashcards/homeostasis.md
```
2. `./ken subjects` → prints:
```
biochemistry: 1 concepts, 2 flashcards, 1 quizzes
physiology: 1 concepts, 1 flashcards, 0 quizzes
```
3. `rm -rf ~/Documents/learn` → `./ken subjects` → prints clear error, not panic
4. `go vet ./...` → clean

---

## Phase 1.5 — Concept Parsing

### Files

| File | Action |
|---|---|
| `internal/parser/parser.go` | Create — `SplitFrontmatter(data []byte) (map[string]interface{}, string)` |
| `internal/parser/concept.go` | Create — `ParseConceptSet(data []byte) ([]Concept, error)` |
| `internal/progress/progress.go` | Create — `Load()`, `Save()`, `InitConcepts()` |

### Implementation

**`internal/parser/parser.go`**:
- `SplitFrontmatter` splits on `---` delimiters, returns YAML as `map[string]interface{}` + markdown body as string
- Empty frontmatter → error
- No `---` delimiters → error

**`internal/parser/concept.go`**:
- `Concept struct { ID, Name, ParentID string; Description string }`
- `ParseConceptSet(data []byte) ([]Concept, error)`
- Unmarshal frontmatter → `type: "concept_set"` check → extract `concepts` array
- Parse markdown body → `## <id>` sections → associate description text to concept by ID
- Unresolved `parent_id` → log warning + treat as root (don't error)
- Duplicate concept ID within file → error

**`internal/progress/progress.go`**:
- `Progress struct { FormatVersion int; Concepts map[string]ConceptState; Cards map[string]CardState; Quizzes map[string]QuizState }`
- `ConceptState struct { Confidence float64; LastReviewedAt *int64 }`
- `CardState struct { Reviews int; LastGrade string }`
- `QuizState struct { Attempts, Correct, Streak int }`
- `Load(path string) (*Progress, error)` — read JSON, create default if missing
- `Save(path string, p *Progress) error` — atomic write (write to temp + rename)
- `InitConcepts(p *Progress, concepts []Concept)` — set `confidence: 0.5`, `last_reviewed_at: nil` for any concept not yet in map

### Acceptance

1. Create test concept file:
```markdown
---
format_version: 1
type: concept_set
concepts:
  - id: c-glycolysis
    name: Glycolysis
    parent_id: null
  - id: c-pfk1
    name: Phosphofructokinase-1
    parent_id: c-glycolysis
  - id: c-hexokinase
    name: Hexokinase
    parent_id: c-glycolysis
---

## c-glycolysis
The metabolic pathway that breaks down glucose into pyruvate.

## c-pfk1
The rate-limiting, committed enzyme of glycolysis.
```
2. Parse → 3 concepts, c-pfk1 and c-hexokinase nest under c-glycolysis
3. Fresh `progress.json` created with all 3 concepts at confidence 0.5
4. Re-run → existing progress preserved, new concepts get 0.5

---

## Phase 2 — Mastery Engine

### Files

| File | Action |
|---|---|
| `internal/mastery/mastery.go` | Create — direct copy from spec lines 142–268 |
| `internal/mastery/mastery_test.go` | Create — unit tests per spec acceptance criteria |

### Implementation

Copy the Go code from `ken-spec.md` lines 142–268 verbatim into `internal/mastery/mastery.go`. The code is the spec — do not change constants, function signatures, or logic.

Exported functions: `UpdateFromFlashcard`, `UpdateFromQuiz`
Exported types: `ConceptState`, `ConfidenceLevel` (+ `Unknown`..`Mastered` constants)

### Tests (`mastery_test.go`)

1. **Mastered grade from 0.5** — confidence moves up, not past `maxDailyDelta` ceiling
2. **Quiz miss at high confidence (>0.75)** — uses likelihood 0.45 (anomaly tolerance), not 0.3
3. **Quiz miss at low confidence (≤0.75)** — uses likelihood 0.3
4. **Decay after multi-day gap** — confidence measurably decreases
5. **Decay above 0.8** — decays slower (exponent halved) vs below 0.8
6. **Confidence never exceeds bounds** — stays within [0.05, 0.995]
7. **Fixture test** — concept at 0.5, sequence of 5 events with known timestamps, verify final confidence within 1e-6

### Acceptance

```bash
go test ./internal/mastery/ -v   # all 7 tests pass
```

---

## Phase 3 — Flashcard Parsing + Study Mode

### Files

| File | Action |
|---|---|
| `internal/parser/flashcard.go` | Create — `ParseFlashcardSet(data []byte) (FlashcardSet, error)` |
| `internal/study/flashcards.go` | Create — `FlashcardSession` — loads cards, manages study state |
| `internal/tui/flashcards.go` | Create — bubbletea model for flashcard study |
| `internal/tui/styles.go` | Create — lipgloss theme |
| `internal/tui/app.go` | Create — root model, command dispatch |
| `cmd/ken/flashcards.go` | Create — `flashcards` cobra command |

### Implementation

**Parser**:
- `FlashcardSet struct { Set, Source string; Cards []Flashcard }`
- `FlashcardSet.Cards` has `ID, ConceptID, Front, Back, Tags []string, Notes string`
- `SplitFrontmatter` + YAML unmarshal + markdown `## Notes: <id>` association
- Duplicate card ID across subject → hard error naming both files

**Study session**:
- Loads all flashcard files for a subject
- Validates: duplicate IDs across files → error
- Returns ordered list of cards to study

**TUI model**:
- States: `showingFront`, `showingBack`, `showingGrades`, `finished`
- Front: shows card front + "press space to flip"
- Back: shows card back + notes if present + 5 grade buttons (1-5 keys)
- Grades: Unknown(1), KnownLittle(2), KnownFairly(3), KnownWell(4), Mastered(5)
- On grade: update `progress.json` (concept confidence if card has `concept_id`, always update card history)
- After last card: show summary, quit

**Cobra command**:
- `ken flashcards <subject>` — required arg
- Loads flashcard files, creates session, runs bubbletea program

### Acceptance

1. Create 5-card test set (3 with `concept_id`, 2 without)
2. Study all 5, grade each differently
3. Check `progress.json`:
   - 3 concepts updated (only cards with `concept_id`)
   - 5 cards updated (all cards)
4. Re-run → cards show review history

---

## Phase 4 — Quiz Mode

### Files

| File | Action |
|---|---|
| `internal/parser/quiz.go` | Create — `ParseQuizSet(data []byte) (QuizSet, error)` |
| `internal/study/quiz.go` | Create — `QuizSession` — loads questions, manages quiz state |
| `internal/tui/quiz.go` | Create — bubbletea model for quiz |
| `cmd/ken/quiz.go` | Create — `quiz` cobra command |

### Implementation

**Parser**:
- `QuizSet struct { Set string; Questions []Question }`
- `Question struct { ID, ConceptID, Type, Question, Explanation string; Options []string; Answer interface{} }`
- Unknown `type` → log warning, skip question (don't include in output)
- Missing `answer` field → log warning, skip question

**Quiz session**:
- Loads all quiz files for a subject
- Dispatches by type: mcq (show options, pick one), true_false (two buttons), fill_blank (text input)
- Running score: correct/total

**TUI model**:
- States: `answering`, `feedback`, `finished`
- Answering: shows question + options/input
- Feedback: correct/incorrect + explanation if present, auto-advance after delay or keypress
- Finished: final score, quit

**Cobra command**:
- `ken quiz <subject>` — required arg
- Loads quiz files, creates session, runs bubbletea program

### Acceptance

1. Quiz with all 3 types + 1 unknown type + 1 malformed (missing answer)
2. Unknown type skipped with warning
3. Malformed question skipped with warning
4. Score matches manual count
5. `progress.json` updated: quiz attempts/correct/streak, concept confidence (only for questions with `concept_id`)

---

## Phase 5 — Progress & Dashboard

### Files

| File | Action |
|---|---|
| `internal/tui/dashboard.go` | Create — bare `ken` dashboard model |
| `internal/tui/progress.go` | Create — progress view model |
| `internal/tui/stats.go` | Create — stats view model |
| `cmd/ken/progress.go` | Create — `progress` cobra command |
| `cmd/ken/stats.go` | Create — `stats` cobra command |
| `cmd/ken/main.go` | Modify — add bare command (no subcommand = dashboard) |

### Implementation

**Dashboard** (`ken` with no args):
- Overall confidence spread: count above 0.7 threshold
- Current streak (days studied consecutively)
- Quick nav: subject with most decay-eligible concepts (not reviewed in N days)
- Empty state: "No subjects found — add content to ~/Documents/learn/subjects/"

**Progress** (`ken progress [subject]`):
- Per-concept confidence, grouped by parent hierarchy
- Highlight: concepts not reviewed in N days (decay-eligible) vs recently reinforced
- If no subject arg → show all subjects

**Stats** (`ken stats`):
- Read `stats.json` from `~/Documents/learn/`
- Format: `{ "daily": { "2026-07-18": { "concepts_reviewed": 5, "avg_confidence_change": 0.03 } } }`
- Show confidence trend over time, streak history
- Append-only: each study session adds today's entry

**Cobra commands**:
- `ken` (no subcommand) → dashboard
- `ken progress [subject]` → progress view
- `ken stats` → stats view

### Acceptance

1. Zero subjects → dashboard shows empty state message
2. One subject with concepts → dashboard shows confidence spread
3. Multiple subjects → dashboard picks most decay-eligible
4. `ken progress biochemistry` → shows concept hierarchy with confidence
5. `ken stats` → shows trend (or "no data yet" if empty)
6. No divide-by-zero on accuracy with zero attempts

---

## Build Order

```
Phase 1 → Phase 1.5 → Phase 2 → Phase 3 → Phase 4 → Phase 5
```

Each phase is a separate commit. Each phase's acceptance criteria must pass before moving to the next.

## Verification Commands (every phase)

```bash
go vet ./...                    # static analysis
go test ./...                   # unit tests (Phase 2+)
go build -o ken ./cmd/ken       # build binary
./ken <command>                 # manual verification
```

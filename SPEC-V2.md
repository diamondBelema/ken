# Ken Spec V2 — Learning Operating System

> Ken is not a flashcard app. It's a learning operating system built on one principle: **concepts are the primary unit of knowledge.** Everything else — flashcards, quizzes, lecture notes, diagrams — is a resource attached to concepts. The spec covers the three-layer learning system (familiarity → reflection → deep reading) and the Wails desktop app.

---

# Part 1: Three-Layer Learning System

## Architecture Overview

Ken's domain model:

```
Subject
  └── Knowledge Graph (concepts + hierarchy)
        └── Learning Resources (flashcards, quizzes, notes, diagrams)
```

Three learning layers, each with its own command, interaction, and cognitive purpose:

| Layer | Command | Cognitive Function | What It Answers |
|-------|---------|-------------------|-----------------|
| **Familiarity** | `ken map` | "I've seen this." | What exists and how does it connect? |
| **Reflection** | `ken reflect` | "I can explain this." | Why does this exist and what's its role? |
| **Deep Reading** | `ken read` | "I've engaged with the source material." | What does the lecture/textbook actually say? |
| **Mastery** | `ken flashcards` / `ken quiz` | "I can recall it under pressure." | Can I retrieve this fact on demand? |

The **learning flow** is ordered:

```
Map → Reflect → Read → Flashcards → Quiz
```

Why reflect BEFORE reading: first you try explaining, then you discover what you missed. That's a stronger learning loop than reading first. Reading after reflection means you read with purpose — you know what you don't know.

The **dashboard** (`ken` bare command) shows readiness across all layers plus course group progress.

```
ken                        # Dashboard
ken map <subject>          # Familiarity layer
ken reflect <subject>      # Reflection layer
ken read <subject> [notes...]  # Deep reading layer
ken flashcards <subject>   # Mastery: flashcard drills
ken quiz <subject>         # Mastery: quiz drills
ken notes <subject>        # User note management
ken stats                  # Aggregate stats
ken lint [subject]         # Content validation
```

### Commands Removed

| Old Command | Fate |
|-------------|------|
| `ken summaries <subject>` | **Removed** — absorbed into `ken map` |
| `ken progress [subject]` | **Removed** — absorbed into dashboard |
| `ken subjects` | **Removed** — dashboard shows subjects now |

### Link Feature Removed

The `links` field in concept frontmatter and the `l` key binding are removed. Proven unused.

---

## Domain Model — Concept as Core

The concept is the heart of Ken. Everything attaches to it.

```go
type Concept struct {
    ID          string
    Name        string
    ParentID    string
    Summary     string        // one-liner for familiarity
    Explanation string        // full explanation for reflection
    Diagrams    []Diagram
}

type Diagram struct {
    ID     string
    Label  string
    Source string   // inline mermaid (optional)
    File   string   // path to .mmd file (optional)
}
```

Flashcards aren't concepts. They're **resources** attached to concepts. Same for quizzes, lecture sections, diagrams, videos, examples. In V1 we keep the existing flashcard/quiz model (separate files with `concept_id` references). The resource interface is the future direction:

```go
// Future (Phase 2+) — not built in V1
type Resource interface {
    ResourceID() string
    ConceptID() string
}
// FlashcardResource, QuizResource, LectureSectionResource,
// DiagramResource, VideoResource, FormulaResource, etc.
```

This means a concept can eventually have:

```
ATP
  ├── Summary
  ├── 5 flashcards
  ├── 2 quizzes
  ├── 1 lecture section
  ├── 3 diagrams
  ├── 1 animation
  ├── 7 examples
  └── 1 mnemonic
```

without changing the model. V1 keeps the current file-based approach; the resource interface is the architectural direction.

---

## Per-Layer Progress Model

Each learning layer has its own state. No mixing.

```go
type ConceptProgress struct {
    Familiarity  FamiliarityState
    Reflection   ReflectionState
    Mastery      MasteryState
}

type FamiliarityState struct {
    Seen bool       // expanded in ken map at least once
}

type ReflectionState struct {
    Count int       // number of reflections completed
    LastAt *int64   // timestamp of last reflection
}

type MasteryState struct {
    Confidence    float64   // Bayesian confidence (0.05–0.995)
    LastReviewedAt *int64   // timestamp of last flashcard/quiz review
}
```

### progress.json — V2 Schema

```json
{
  "format_version": 2,
  "concepts": {
    "c-glycolysis": {
      "familiarity": { "seen": true },
      "reflection": { "count": 3, "last_at": 1752835200 },
      "mastery": { "confidence": 0.62, "last_reviewed_at": 1752835200 }
    }
  },
  "cards": {
    "bch-001": { "reviews": 4, "last_grade": "known_fairly" }
  },
  "quizzes": {
    "bch-q001": { "attempts": 3, "correct": 2, "streak": 1 }
  },
  "notes": {
    "n-1": {
      "content": "ATP is the energy currency of the cell...",
      "linked_to": { "type": "concept", "id": "c-atp" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    }
  }
}
```

**Key design decision:** Understanding/Reflection state does NOT affect mastery confidence. They are different signals:
- **Familiarity** = "I've seen this concept in the tree" (boolean)
- **Reflection** = "I've typed an explanation for this concept" (count, no grading)
- **Mastery** = "I can recall this fact reliably" (graded, Bayesian)

Mixing them would make the dashboard lie. A concept with high familiarity and completed reflection but low mastery confidence means "I've seen it and explained it but can't recall it yet" — that's honest.

---

## Course Groups

Large thematic sections within a subject (e.g., "Bioenergetics", "Enzymes"). NOT temporal — structural divisions of the course.

### File Format

Location: `~/Documents/learn/subjects/<subject>/groups.yaml`

```yaml
groups:
  - id: bioenergetics
    name: "Bioenergetics"
    concepts:
      - c-atp
      - c-oxidative-phosphorylation
      - c-chemiosmosis
      - c-electron-transport-chain
  - id: enzymes
    name: "Enzymes"
    concepts:
      - c-enzyme-kinetics
      - c-michaelis-menten
      - c-allosteric-regulation
  - id: carb-metabolism
    name: "Carbohydrate Metabolism"
    concepts:
      - c-glycolysis
      - c-tca-cycle
      - c-gluconeogenesis
```

**Rules:**
- `id`: unique string, kebab-case
- `name`: human-readable display name
- `concepts`: array of concept IDs (can appear in multiple groups)
- Ordered by appearance in file (study order)
- No `groups.yaml` = no groups (all concepts ungrouped)

### CLI Usage

```bash
ken map biochem --group enzymes             # familiarity scoped
ken reflect biochem --group carb-metabolism  # reflection scoped
ken flashcards biochem --group enzymes       # mastery scoped
ken quiz biochem --group enzymes             # quiz scoped
```

### Parser

```go
type CourseGroup struct {
    ID       string   `yaml:"id"`
    Name     string   `yaml:"name"`
    Concepts []string `yaml:"concepts"`
}

type GroupsFile struct {
    Groups []CourseGroup `yaml:"groups"`
}
```

New package: `internal/groups/` with `Load(subjectDir string) ([]CourseGroup, error)`.

---

## Layer 1: `ken map` — Familiarity

Purpose: build a mental map of the course. Walk the concept tree, see summaries, understand the hierarchy. No interaction beyond expanding/collapsing.

### Behavior

- Loads the global concept tree for the subject
- Renders tree with expand/collapse per concept
- **Collapsed**: concept name + 1-2 line one-liner (first line of summary)
- **Expanded**: full concept summary
- No summary? Show description instead
- Status: `[✓ familiar]` / `[— not started]` based on `familiarity.seen`

### Tree Rendering

```
Biochemistry — Concept Map
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

▼ Carbohydrate Metabolism
  ▼ c-glycolysis  Glycolysis [✓ familiar]
  │ Glycolysis is the first step of cellular respiration,
  │ occurring in the cytoplasm. It converts glucose to two
  │ molecules of pyruvate, producing a net gain of 2 ATP.
  │
  │   ► c-pfk1  Phosphofructokinase-1 [✓ familiar]
  │   ► c-hexokinase  Hexokinase [— not started]
  │   ► c-pyruvate-kinase  Pyruvate Kinase [— not started]
  │
  ▼ c-tca-cycle  TCA Cycle [— not started]
  │ The citric acid cycle oxidizes acetyl-CoA to CO₂...
  │
  │   ► c-idh-complex  Isocitrate Dehydrogenase [— not started]

▼ Lipid Metabolism
  ► c-fatty-acid-oxidation  Beta-Oxidation [— not started]
  ► c-lipogenesis  Lipogenesis [— not started]
```

- `▼` = expanded (showing summary)
- `►` = collapsed (showing one-liner)
- Group filter: `--group` shows only concepts in that group + their ancestors

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Move between concepts |
| `enter` | Expand / collapse |
| `g` | Open group filter |
| `/` | Search concepts |
| `d` | View diagram (ASCII inline) |
| `Esc` | Back to dashboard |
| `?` | Help |

---

## Layer 2: `ken reflect` — Reflection

Purpose: self-explanation prompts. Type your explanation of "why does this exist," "what feeds into it," then see the canonical answer. No grading — the act of typing is the learning. Renamed from "Understanding" because: understanding sounds measurable, reflection sounds intentional. You're not measuring. You're prompting.

### Behavior

- Loads the global concept tree
- Randomizes concept order (Fisher-Yates shuffle each session)
- Presents one concept at a time with a prompt
- User types explanation (multi-line textarea)
- On submit: shows canonical answer (from summary or description)
- User can optionally save typed explanation as a note (`w` key)
- `n` moves to next concept

### The Prompt

Generated from concept tree structure — no AI at runtime:

- **Root concepts**: "Explain what [name] is and why it matters in [subject]."
- **Child concepts**: "Explain how [name] relates to [parent]. What role does it play?"
- **Leaf concepts**: "Explain [name] in the context of [parent]. What does it do and why?"

### The Interaction

```
ken reflect biochem --group bioenergetics

  Reflection: Bioenergetics
  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  Explain the role of ATP in bioenergetics.

  > _                                        [type here]

  ──────────────────────────────────────
  j/n: next  s: summary  w: save note  Esc: quit
```

After submit:

```
  Explain the role of ATP in bioenergetics.

  > ATP is the energy currency of the cell. It stores
  > energy in its phosphate bonds and releases it when
  > hydrolyzed to ADP + Pi.

  Canonical answer:
  ──────────────────────────────────────
  ATP (adenosine triphosphate) is the primary energy
  carrier in all living organisms. It stores energy in
  high-energy phosphoanhydride bonds. Hydrolysis releases
  ~30.5 kJ/mol, driving endergonic reactions.

  j/n: next  s: summary  w: save note  Esc: quit
```

### Note Saving

- Press `w` to save typed explanation as a note
- Tagged to concept: `{ "type": "concept", "id": "c-atp" }`
- Saved to `progress.json` under `notes` map
- If user doesn't press `w`, explanation is discarded

### Data Model

Reflection state tracked per concept:

```json
"reflection": { "count": 3, "last_at": 1752835200 }
```

- `count`: number of reflections completed (incremented each time `w` is pressed)
- `last_at`: timestamp of last saved reflection
- Does NOT affect mastery confidence

### Keybindings

| Key | Action |
|-----|--------|
| `type` | Write explanation |
| `enter` | Submit → show canonical answer |
| `w` | Save as note (tagged to concept) |
| `n` | Next concept (random) |
| `p` | Previous concept |
| `s` | See concept summary |
| `d` | View diagram |
| `Esc` | Back to dashboard |
| `?` | Help |

---

## Layer 3: `ken read` — Deep Reading

Purpose: read lecture notes, textbook extracts. Content is AI-tagged with concept annotations. The reading view shows the note's internal concept tree with status from the global concept tree.

### Document Parsing Model

Internally, notes are parsed into a structured document model — not raw markdown:

```go
type Document struct {
    Title    string
    Sections []Section
}

type Section struct {
    Heading    string
    Level      int           // heading depth (1-6)
    ConceptID  string        // concept tag (if present)
    Blocks     []Block
    Children   []Section     // nested sections
}

type Block struct {
    Type            string    // "paragraph", "bullet", "table", "code", "formula"
    Content         string
    ConceptRefs     []string  // inline concept references
}
```

The TUI renders this as an expandable tree. The desktop renders it as a split pane (tree left, content right). Same data, different renderers. This is why the document model matters — it decouples content from presentation.

### Two Types of Content

| Content Type | Location | Writable | Purpose |
|-------------|----------|----------|---------|
| **Lecture notes** | `notes/` directory | No | AI-tagged content for deep reading |
| **User notes** | `progress.json` | Yes | Personal notes, reflection explanations |

`ken read` reads lecture notes. `ken notes` manages user notes.

### Note Selection

```bash
ken read biochem                    # opens note selector
ken read biochem lecture-3 lecture-4 # loads specific notes
```

### Concept Tagging Model

Tags are AI-generated during content creation. Two tag types:

1. **Heading tags** — on markdown headings, define the tree structure
2. **Paragraph tags** — on any content block, inline navigation markers

### Tagged Note Format

```markdown
# Glycolysis [c-glycolysis]

Glycolysis is the metabolic pathway that converts glucose into
pyruvate. It occurs in the cytoplasm and consists of 10 steps.

## Regulation [c-pfk1]

PFK-1 is the key regulatory enzyme. It catalyzes the committed
step — phosphorylation of fructose-6-phosphate.

Citrate [c-citrate] from TCA cycle inhibits PFK-1. AMP [c-amp]
activates it when energy is low.

## Energy Yield [c-glycolysis]

Net yield per glucose: 2 ATP and 2 NADH.
```

**AI tagging rules:**
- Every heading gets a concept tag by default
- AI can also tag paragraphs, list items, any content block
- Tags: `[c-concept-id]` at end of heading or inline in text
- Heading hierarchy = note-local tree (can differ from global tree)
- AI tags at subtree level — broad sections tag the parent concept
- Same concept can appear multiple times in one note (different sections)

### Reading View

```
Lecture 3 — Glycolysis Steps [note 1 of 2]
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

▼ c-glycolysis  Glycolysis [✓ familiar] [✓ reflected] [0.82]
│ Glycolysis is the metabolic pathway that converts glucose
│ into pyruvate. It occurs in the cytoplasm.

  ▼ c-pfk1  Regulation [✓ familiar] [✓ reflected]
  │ PFK-1 is the key regulatory enzyme. It catalyzes the
  │ committed step.
  │
  │   [c-citrate] Citrate from TCA cycle inhibits PFK-1
  │   [c-amp] AMP activates it when energy is low

  ► c-glycolysis  Energy Yield [✓ familiar] [— reflected]

─── Lecture 4 — TCA Cycle [note 2 of 2] ───

▼ c-tca-cycle  TCA Cycle [— familiar] [— reflected]
│ The citric acid cycle oxidizes acetyl-CoA to CO₂...
│
│   ► c-idh-complex  Isocitrate Dehydrogenase [— familiar]
```

### Cross-Note Concept Hopping

Press `c` on a concept → list of all instances across loaded notes → select to jump. Compare what different sources say about the same concept.

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Move between tree nodes |
| `enter` | Expand / collapse heading |
| `s` | Jump to concept's global summary |
| `c` | Hop to same concept in another note |
| `d` | View diagram |
| `/` | Search within loaded notes |
| `tab` | Switch between notes |
| `Esc` | Back to dashboard |
| `?` | Help |

---

## Dashboard

Main landing screen. Shows readiness across all layers plus course group progress.

### Layout

```
Ken — Dashboard
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Biochemistry (BCH 234)
  Familiarity  [████████░░░░] 48/74    Reflection  [██████░░░░░░] 31/74
  Mastery      [███████░░░░░] avg 0.68 Due: 12 flashcards, 5 quizzes

  ▼ Course Groups
    Bioenergetics                    [██████████] 94% ready
    Enzymes                          [██████░░░░] 58% ready
    Carbohydrate Metabolism          [████░░░░░░] 38% ready
    Lipid Metabolism                 [░░░░░░░░░░]  0% ready

Cell Biology (BIO 301)
  Familiarity  [███░░░░░░░░░] 18/60    Reflection  [█░░░░░░░░░░░]  6/60
  Mastery      [██░░░░░░░░░░] avg 0.31 Due: 34 flashcards, 12 quizzes

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
 [f] flashcards  [t] quiz  [m] map  [v] reflect  [r] read  [s] stats
```

### Readiness Calculation

Per-group readiness (weighted):
- Familiarity: 30% (concepts with `familiarity.seen: true`)
- Reflection: 30% (concepts with `reflection.count > 0`)
- Mastery: 40% (average confidence normalized to 0-100%)

### Keybindings

| Key | Action |
|-----|--------|
| `j` / `k` | Move between subjects |
| `enter` | Expand / collapse course groups |
| `f` | Flashcards |
| `t` | Quiz |
| `m` | Map |
| `v` | Reflect |
| `r` | Read |
| `s` | Stats |
| `Esc` | Quit |
| `?` | Help |

---

## Concept File Format

### Frontmatter

```yaml
---
format_version: 1
type: concept_set
concepts:
  - id: c-glycolysis
    name: Glycolysis
    parent_id: null
    diagrams:
      - id: glycolysis-pathway
        label: "Glycolysis Pathway"
        source: |
          graph TD
            A[Glucose] --> B[Pyruvate]
---
```

**Removed from V1:** `links` array (link feature removed).

### Body Sections

```markdown
## c-glycolysis
The metabolic pathway that breaks down glucose into pyruvate.

## c-glycolysis:summary
Glycolysis is the first step of cellular respiration...
```

Unchanged. The `## <id>:summary` section is the one-liner for `ken map` and the canonical answer for `ken reflect`.

---

## Global Keybinding Reference

### Navigation (all list/tree views)

| Key | Action |
|-----|--------|
| `j` / `down` | Move down |
| `k` / `up` | Move up |
| `enter` | Select / expand / collapse |
| `Esc` | Back / close |
| `?` | Help overlay |

### Per-View Keybindings

| Key | Dashboard | map | reflect | read | Flashcards | Quiz | Notes | Stats |
|-----|-----------|-----|---------|------|------------|------|-------|-------|
| `f` | flashcards | — | — | — | — | — | — | — |
| `t` | quiz | — | — | — | — | — | — | — |
| `m` | map | — | — | — | — | — | — | — |
| `v` | reflect | — | — | — | — | — | — | — |
| `r` | read | — | — | — | — | — | — | — |
| `s` | stats | — | summary | summary | — | — | — | — |
| `n` | — | — | next | — | — | — | new | — |
| `p` | — | — | prev | — | — | — | — | — |
| `w` | — | — | save note | — | — | — | — | — |
| `d` | — | diagram | diagram | diagram | diagram | diagram | — | — |
| `c` | — | — | — | hop | — | — | — | — |
| `g` | — | group | — | — | — | — | — | — |
| `/` | — | search | — | search | — | — | search | — |
| `tab` | — | — | — | switch note | — | — | — | — |
| `e` | — | — | — | — | — | — | edit | — |
| `x` | — | — | — | — | — | — | delete | — |
| `space` | — | — | — | — | flip | — | — | — |
| `1`-`5` | — | — | — | — | grade | — | — | — |
| `a`-`d` | — | — | — | — | — | answer | — | — |
| `q` | quit | quit | quit | quit | quit | quit | quit | quit |

---

## Content Creation Updates

### 1. Add Course Groups Section
Document `groups.yaml` format. Groups are structural, not temporal.

### 2. Add Note Tagging Convention
Two tag types: heading tags + paragraph tags. AI tags at subtree level.

### 3. Remove Links Section
Remove all `links` field documentation.

### 4. Add Reflection Prompts Section
Document prompt templates for `ken reflect`.

### 5. Rename Understanding → Reflection
Throughout all documentation.

### 6. Update Flow Documentation
Map → Reflect → Read → Flashcards → Quiz. Reflect comes before Read.

### 7. Update Checklist
Remove link checks. Add course group validation. Add note tagging conventions.

---

## Build Phases

### Phase 6 — Course Groups
- `groups.yaml` parser (`internal/groups/`)
- `--group` flag on `ken map`, `ken reflect`, `ken flashcards`, `ken quiz`
- Dashboard group display
- **Acceptance:** groups load; `--group` filters concepts; dashboard shows per-group readiness

### Phase 7 — Ken Map (Familiarity)
- Tree rendering with expand/collapse
- Summary display (one-liner collapsed, full expanded)
- `familiarity.seen` flag in progress.json
- `g` group filter, `/` search, `d` diagrams
- **Acceptance:** tree renders; expanding shows summary; first expansion sets seen; group filter works

### Phase 8 — Ken Reflect (Reflection)
- Single-concept prompt display
- Textarea for typing explanation
- Canonical answer on submit
- Note saving with `w`
- Randomized concept order
- `reflection.count` tracking in progress.json
- **Acceptance:** prompts display; typing works; answer shows; note saves; concepts randomized; count increments

### Phase 9 — Ken Read (Deep Reading)
- Multi-note selector (checklist + CLI args)
- Document parser (heading tags + paragraph tags → Section/Block model)
- Note-local tree rendering
- Status indicators from global concept state
- Cross-note concept hopping
- **Acceptance:** notes load with concept tags; tree renders; status correct; hopping works

### Phase 10 — Dashboard Redesign
- Three-layer progress display
- Course group breakdown (expandable)
- Scrollable subject list
- Keybindings for all sub-commands
- **Acceptance:** three-layer progress shows; groups display; scrolling works; keybindings launch commands

### Phase 11 — Command Cleanup
- Remove `ken summaries`, `ken progress`, `ken subjects`
- Remove `links` from parser, `l` key from views
- Remove `v` key (diagrams consolidated to `d`)
- Rename `ken active` → `ken reflect`
- Update AGENTS.md and CONTENT-CREATION.md
- **Acceptance:** removed commands show clear message; no link code remains; docs updated

---

## Future Directions (Phase 2+)

### Event-Centric Architecture

Instead of mutating progress directly, record learning events:

```go
type LearningEvent struct {
    Time       int64
    SubjectID  string
    ConceptID  string
    Type       string  // "familiarity_seen", "reflection_saved", "flashcard_reviewed", "quiz_answered"
    Payload    map[string]interface{}
}
```

Derive `ConceptProgress` from events (or periodically materialize for performance). Benefits:
- Complete learning history
- Undo/replay capability
- Future analytics ("what caused mastery to improve?")
- Ability to change confidence algorithm without losing history

### Learning Resource Interface

```go
type Resource interface {
    ResourceID() string
    ConceptID() string
}
// FlashcardResource, QuizResource, LectureSectionResource,
// DiagramResource, VideoResource, FormulaResource,
// ExampleResource, ClinicalPearlResource, MnemonicResource
```

Any new resource type attaches to concepts without changing the model.

---

# Part 2: Wails Desktop App

## Architecture

```
Go Backend (domain logic)
  ├── mastery, parser, progress, study, groups
  └── Wails bindings (IPC)
        └── Svelte Frontend (UI only)
              └── render() → call backend() → display result()
```

**Critical rule:** the frontend never contains business logic. No Bayesian equations. No scheduling. No parsing. No progress calculations. Only render, call backend, display result. This makes it possible to later have CLI, Desktop, Android, Web — all sharing the same learning engine.

### What Reuses Existing Go Code

| Package | Reused? | Notes |
|---------|---------|-------|
| `internal/mastery` | **Yes** | Bayesian confidence engine |
| `internal/parser` | **Yes** | YAML + markdown parsing |
| `internal/progress` | **Yes** | Progress state read/write |
| `internal/discovery` | **Yes** | Subject scanning |
| `internal/study` | **Yes** | Flashcard + quiz logic |
| `internal/diagram` | **Yes** | Mermaid rendering |
| `internal/lint` | **Yes** | Content validation |
| `internal/groups` | **Yes** | New from Phase 6 |
| `internal/tui` | **No** | TUI-specific |
| `internal/render` | **Partial** | Web uses different renderer |
| `internal/system` | **Yes** | Cross-platform helpers |
| `internal/registry` | **Yes** | Package registry |

### What Needs New Code

| Component | Language | Purpose |
|-----------|----------|---------|
| Wails bindings | Go | IPC layer |
| Frontend UI | Svelte | All views |
| State management | TypeScript | Client-side state |
| Markdown rendering | JavaScript | marked/markdown-it |
| Mermaid rendering | JavaScript | mermaid.js |

## Project Structure

```
desktop/
├── main.go
├── app.go                     # binding methods
├── wails.json
├── frontend/
│   ├── src/
│   │   ├── App.svelte
│   │   ├── lib/
│   │   │   ├── api.ts         # Wails runtime bindings
│   │   │   ├── stores.ts
│   │   │   └── types.ts
│   │   └── components/
│   │       ├── Dashboard.svelte
│   │       ├── MapView.svelte
│   │       ├── ReflectView.svelte
│   │       ├── ReadView.svelte
│   │       ├── FlashcardView.svelte
│   │       ├── QuizView.svelte
│   │       ├── NotesView.svelte
│   │       ├── ConceptTree.svelte
│   │       ├── NoteTree.svelte
│   │       └── KeybindingHelp.svelte
```

## Go Backend — Wails Bindings

```go
type App struct {
    progressDir string
    contentDir  string
}

func (a *App) ListSubjects() ([]discovery.SubjectInfo, error)
func (a *App) LoadSubject(subject string) (*parser.SubjectData, error)
func (a *App) LoadGroups(subject string) ([]groups.CourseGroup, error)
func (a *App) LoadProgress(subject string) (*progress.Progress, error)
func (a *App) SaveProgress(subject string, p *progress.Progress) error
func (a *App) UpdateFromFlashcard(subject, conceptID string, grade mastery.ConfidenceLevel) error
func (a *App) UpdateFromQuiz(subject, conceptID string, wasCorrect bool) error
func (a *App) StartFlashcards(subject, groupID string) (*study.FlashcardSession, error)
func (a *App) GradeCard(sessionID, cardID string, grade mastery.ConfidenceLevel) error
func (a *App) StartQuiz(subject, groupID string) (*study.QuizSession, error)
func (a *App) AnswerQuestion(sessionID, questionID string, answer interface{}) (*study.QuizResult, error)
func (a *App) CreateNote(subject, content string, linkedTo *progress.EntityRef) error
func (a *App) UpdateNote(subject, noteID, content string) error
func (a *App) DeleteNote(subject, noteID string) error
func (a *App) ListNotes(subject string, filter *NoteFilter) ([]progress.Note, error)
func (a *App) ListLectureNotes(subject string) ([]string, error)
func (a *App) ReadNote(subject, filename string) (string, error)
func (a *App) ParseTaggedNote(subject, filename string) (*TaggedNote, error)
func (a *App) LintSubject(subject string) (*lint.Report, error)
```

## Routing

```
#/                          → Dashboard
#/map/:subject              → Map
#/map/:subject/:group       → Map (group filtered)
#/reflect/:subject          → Reflect
#/reflect/:subject/:group   → Reflect (group filtered)
#/read/:subject             → Read (note selector)
#/read/:subject/:notes      → Read (specific notes)
#/flashcards/:subject       → Flashcards
#/flashcards/:subject/:group → Flashcards (group filtered)
#/quiz/:subject             → Quiz
#/quiz/:subject/:group      → Quiz (group filtered)
#/notes/:subject            → Notes
#/stats                     → Stats
```

## Desktop-Only Enhancements

- **Split panes**: tree left, content right
- **Multi-window**: multiple subjects simultaneously
- **Tray icon**: due cards count
- **Notifications**: "You have 12 cards due"
- **Global search**: across all subjects

## Build Phases (Desktop)

### Phase 12 — Wails Scaffold
- Initialize project, wire Go packages, basic routing
- **Acceptance:** app launches, placeholder dashboard, Go backend accessible

### Phase 13 — Dashboard + Subjects
- Subject list, three-layer progress, course groups
- **Acceptance:** dashboard shows subjects with progress; groups expand/collapse

### Phase 14 — Map + Reflect + Read
- Concept tree component, summary display, reflect view, read view
- **Acceptance:** all three views render; keyboard navigation works

### Phase 15 — Flashcards + Quiz
- Card flip, grade buttons, quiz answers, mermaid diagrams
- **Acceptance:** study works end to end; progress updates correctly

### Phase 16 — Notes + Stats + Polish
- Note editor, stats, help overlay, search, notifications
- **Acceptance:** note CRUD works; stats display; search finds concepts

---

## Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Desktop framework | Wails v2 | Reuses Go internals, web UI, single binary |
| Frontend | Svelte + Tailwind | Lightweight, great DX |
| Domain model | Concept-centric | Everything attaches to concepts |
| Progress model | Per-layer (Familiarity/Reflection/Mastery) | No mixing, honest dashboard |
| Learning flow | Map → Reflect → Read → Flashcards → Quiz | Reflect before read = stronger loop |
| Naming | Reflection (not Understanding) | Intentional, not measurable |
| Document model | Structured blocks (Section/Block) | Decouples content from presentation |
| Understanding state | Not saved | Fresh sessions prevent rubber-stamping |
| Familiarity tracking | Boolean flag | Simple, honest |
| Course groups | Single groups.yaml | Fastest to author |
| Note tagging | AI-generated, two tag types | Heading + paragraph |
| Link feature | Removed | Unused |
| `v`/`d` split | Consolidated into `d` | Context decides renderer |
| `q` key | Quit everywhere | Consistent |
| `v` key | Reflect (dashboard) | Avoids conflict with `q` |
| Business logic | Go backend only | Frontend renders, never computes |
| Event-centric | Phase 2 | Future: learning history, analytics |
| Resource interface | Phase 2 | Future: extensible resource types |

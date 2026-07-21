# Feature Plan: Notes, Summaries, Diagrams, Links (COMPLETED)

> **Status: All features implemented.** This document is preserved for reference.

## Overview

Five new features to make `ken` a full learning harness:

1. **Concept initialization** — auto-init concepts from content files when studying
2. **Note taking** — user-created freeform notes, never interrupts learning flow. User opens note input with a command; auto-linked to current context (card, concept, quiz). Notes can also link to other notes, or be unlinked. Editable and deletable.
3. **Summaries** — two sources:
   - **Content summaries** — parsed from markdown files (like concepts, flashcards, quizzes), part of the learning material
   - **User summaries** — created by the learner during study, stored in progress.json
   - Scoped to: a single concept, a list of concepts, or the whole subject
   - Both shown when they exist (content + user labeled separately)
4. **Diagrams** — mermaid syntax, inline source or external file reference. ASCII quick view + SVG export.
5. **Links** — reference and open websites/YouTube from concept files (REMOVED in three-layer redesign)

### Key distinction: notes vs summaries

| | Notes | Summaries |
|---|---|---|
| **Created by** | User only | User or content author |
| **Source** | progress.json | Content files + progress.json |
| **Purpose** | Personal annotations, reminders | Structured overviews of material |
| **Scope** | Any entity, other notes, or nothing | Concept, concepts, or subject |
| **Linking** | Freeform — link to anything including other notes | Scoped — tied to specific entities |
| **Editing** | Editable and deletable | Editable and deletable |

---

## Data Model — Notes

Notes are a **first-class collection** in progress.json. Freeform text, optionally linked to anything — including other notes.

```json
{
  "notes": {
    "n-1": {
      "content": "PFK-1 is inhibited by ATP and citrate, activated by AMP and F-2,6-BP",
      "linked_to": { "type": "concept", "id": "c-pfk1" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    },
    "n-2": {
      "content": "Watch this for a great visual explanation: https://youtube.com/watch?v=...",
      "linked_to": null,
      "created_at": 1752835200,
      "updated_at": 1752835200
    },
    "n-3": {
      "content": "This card is tricky — need to review more",
      "linked_to": { "type": "card", "id": "bch-001" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    },
    "n-4": {
      "content": "Missed this quiz question — review the enzyme naming convention",
      "linked_to": { "type": "quiz", "id": "bch-q001" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    },
    "n-5": {
      "content": "Also see the inhibition kinetics note — same pattern applies here",
      "linked_to": { "type": "note", "id": "n-1" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    }
  }
}
```

**`linked_to` types:**
- `{ "type": "concept", "id": "c-pfk1" }` — tied to one concept
- `{ "type": "card", "id": "bch-001" }` — tied to one flashcard
- `{ "type": "quiz", "id": "bch-q001" }` — tied to one quiz question
- `{ "type": "note", "id": "n-1" }` — tied to another note
- `null` — general note, not tied to anything

Notes are always user-created. The note input never interrupts the learning flow — user presses `n` when they want to jot something down. The note auto-links to whatever the user is currently viewing (card, concept, quiz question). User can optionally change the link target before saving.

---

## Data Model — Summaries

Summaries come from **two sources**:

### Source 1: Content summaries (read-only, parsed from markdown files)

Part of the learning material, authored alongside concepts. Stored in concept files:

```markdown
## c-glycolysis
The metabolic pathway that breaks down glucose into pyruvate.

## c-glycolysis:summary
Glycolysis is the first step of cellular respiration, occurring in the cytoplasm.
It converts glucose to pyruvate, yielding net 2 ATP and 2 NADH.
```

Parsed by the same `parser.ParseConceptSet` — each concept can have an optional `## <id>:summary` section. This is per-concept only.

### Source 2: User summaries (writable, stored in progress.json)

Created by the learner during study. More flexible scoping:

```json
{
  "summaries": {
    "s-1": {
      "title": "Glycolysis Overview",
      "content": "First step of cellular respiration. Key points: glucose → pyruvate, net 2 ATP.",
      "linked_to": { "type": "concept", "id": "c-glycolysis" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    },
    "s-2": {
      "title": "Enzyme Regulation",
      "content": "PFK-1 is the main control point, inhibited by ATP/citrate, activated by AMP.",
      "linked_to": { "type": "concepts", "ids": ["c-pfk1", "c-hexokinase", "c-pyruvate-kinase"] },
      "created_at": 1752835200,
      "updated_at": 1752835200
    },
    "s-3": {
      "title": "Biochemistry — Full Overview",
      "content": "Covers metabolic pathways: glycolysis, citric acid cycle, ETC...",
      "linked_to": { "type": "subject", "id": "biochemistry" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    }
  }
}
```

**`linked_to` types:**
- `{ "type": "concept", "id": "c-glycolysis" }` — summary of one concept
- `{ "type": "concepts", "ids": ["c-1", "c-2", "c-3"] }` — summary spanning multiple concepts
- `{ "type": "subject", "id": "biochemistry" }` — summary of the whole subject

### Display

When showing a concept's summaries:
1. If content summary exists (`## <id>:summary`) → show it labeled "Content Summary"
2. If user summary exists for that concept → show it labeled "User Summary"
3. Both shown if both exist — user summary does NOT replace content summary

---

## Content File Changes (read-only)

### Concept file additions

```yaml
---
format_version: 1
type: concept_set
concepts:
  - id: c-glycolysis
    name: Glycolysis
    parent_id: null
    diagrams:
      - path: diagrams/glycolysis.png
        label: "Glycolysis Pathway"
    links:
      - url: "https://www.youtube.com/watch?v=abc123"
        title: "Glycolysis Explained"
        type: youtube
      - url: "https://en.wikipedia.org/wiki/Glycolysis"
        title: "Wikipedia: Glycolysis"
        type: website
---

## c-glycolysis
The metabolic pathway that breaks down glucose into pyruvate.

## c-glycolysis:summary
Glycolysis is the first step of cellular respiration, occurring in the cytoplasm.
```

New frontmatter fields per concept:
- `diagrams`: optional array of `{path, label}` — path relative to subject directory
- `links`: optional array of `{url, title, type}` — type is "youtube", "website", or "reference"

New markdown body section:
- `## <id>:summary` — author-provided summary (separate from description)

---

## Diagram Rendering (Hybrid Approach)

### Syntax: Mermaid

All diagrams use Mermaid syntax — the most popular diagram-as-code format. Stored inline in concept file frontmatter.

### Content file format

```yaml
concepts:
  - id: c-glycolysis
    name: Glycolysis
    diagrams:
      - id: glycolysis-path
        label: "Glycolysis Pathway"
        source: |
          graph TD
            A[Glucose] --> B[G6P]
            B --> C[F6P]
            C --> D[F1,6BP]
            D --> E[G3P]
      - id: krebs-cycle
        label: "Krebs Cycle"
        file: diagrams/krebs.mmd
```

Two ways to provide mermaid source:
- **`source`** — inline mermaid definition in the YAML (small/medium diagrams)
- **`file`** — path to external `.mmd` file relative to subject directory (large diagrams)

If both `source` and `file` are provided, `source` takes precedence.

### Two rendering paths

| Path | Library | Output | When used |
|---|---|---|---|
| **Quick view** | `a-kaibu/mermaigo` | ASCII/Unicode | Inline in TUI, press `v` |
| **Full view** | `zkrebbekx/go-mermaid` | SVG file | Open externally, press `d` |

### Quick view (ASCII inline)

- Press `v` on a diagram in progress view or flashcard back
- `mermaigo.RenderText(mermaidSource)` → ASCII string
- Displayed directly in the TUI inside a styled box
- No external dependencies, instant rendering

### Full view (SVG export)

- Press `d` on a diagram
- `go-mermaid.Render(mermaidSource)` → SVG bytes
- Write to temp file, open with `xdg-open`
- Full visual fidelity, themed rendering

### New dependency

```bash
go get github.com/a-kaibu/mermaigo
go get github.com/zkrebbekx/go-mermaid
```

Both are pure Go — no Chrome, no Node.js, no external runtime.

### TUI key bindings (diagrams)

| Key | Context | Action |
|---|---|---|
| `v` | Progress view, flashcard back | Render ASCII diagram inline |
| `d` | Progress view, flashcard back | Export SVG and open externally |
| `escape` | ASCII diagram view | Close inline diagram |

---

## State File Changes (writable)

### progress.json — full schema

```json
{
  "format_version": 2,
  "concepts": {
    "c-pfk1": {
      "familiarity": {
        "seen": true
      },
      "reflection": {
        "count": 3,
        "last_at": 1752835200
      },
      "mastery": {
        "confidence": 0.62,
        "last_reviewed_at": 1752835200
      }
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
      "content": "PFK-1 is inhibited by ATP...",
      "linked_to": { "type": "concept", "id": "c-pfk1" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    }
  },
  "summaries": {
    "s-1": {
      "title": "Glycolysis Overview",
      "content": "Glycolysis is the first step...",
      "linked_to": { "type": "concept", "id": "c-glycolysis" },
      "created_at": 1752835200,
      "updated_at": 1752835200
    }
  }
}
```

---

## Parser Changes

### `internal/parser/concept.go`

**New structs:**
```go
type Diagram struct {
    ID     string
    Label  string
    Source string   // inline mermaid source (optional)
    File   string   // path to .mmd file relative to subject dir (optional)
}

type Concept struct {
    ID          string
    Name        string
    ParentID    string
    Description string
    Summary     string    // from ## <id>:summary section
    Diagrams    []Diagram
}
```

**Parsing changes:**
- Extract `diagrams` arrays from concept frontmatter
- Parse `## <id>:summary` sections alongside `## <id>` description sections
- Summary is optional — concept works without it

### `internal/progress/progress.go`

**New structs:**
```go
type Note struct {
    Content   string      `json:"content"`
    LinkedTo  *EntityRef  `json:"linked_to,omitempty"`
    CreatedAt int64       `json:"created_at"`
    UpdatedAt int64       `json:"updated_at"`
}

type Summary struct {
    Title     string      `json:"title"`
    Content   string      `json:"content"`
    LinkedTo  *EntityRef  `json:"linked_to"`
    CreatedAt int64       `json:"created_at"`
    UpdatedAt int64       `json:"updated_at"`
}

type EntityRef struct {
    Type string   `json:"type"`              // "concept", "concepts", "card", "quiz", "subject", "note"
    ID   string   `json:"id,omitempty"`      // single entity
    IDs  []string `json:"ids,omitempty"`     // multiple entities (for "concepts")
}

type FamiliarityState struct {
    Seen bool `json:"seen"`
}

type ReflectionState struct {
    Count  int    `json:"count"`
    LastAt *int64 `json:"last_at,omitempty"`
}

type MasteryState struct {
    Confidence     float64 `json:"confidence"`
    LastReviewedAt *int64  `json:"last_reviewed_at"`
}

type ConceptState struct {
    Familiarity FamiliarityState `json:"familiarity"`
    Reflection  ReflectionState  `json:"reflection"`
    Mastery     MasteryState     `json:"mastery"`
}

type Progress struct {
    FormatVersion int                    `json:"format_version"`
    Concepts      map[string]ConceptState `json:"concepts"`
    Cards         map[string]CardState    `json:"cards"`
    Quizzes       map[string]QuizState    `json:"quizzes"`
    Notes         map[string]Note         `json:"notes,omitempty"`
    Summaries     map[string]Summary      `json:"summaries,omitempty"`
}
```

---

## TUI Changes

### Note-taking flow (never interrupts learning)

Notes are **user-initiated** and never stop the learning flow:
- User presses `n` during flashcard study, quiz, or progress view
- Note input appears as a split/panel — the current view stays visible
- Note auto-links to whatever the user is currently viewing
- User can optionally change link target (cycle through: current card → concept → quiz → other note → nothing)
- Enter saves, esc cancels — user returns exactly where they were
- User can press `n` again anytime to take another note

Note editing/deletion:
- In `ken notes` view, `e` edits selected note, `x` deletes (with confirmation)
- In progress view, notes show with `e` to edit inline

### Flashcard study (`internal/tui/flashcards.go`)

After grading a card with a `concept_id`:
- Show concept summary if available (content summary and/or user summary)
- Show available diagrams for the concept
- Key bindings: `n` add note (auto-linked to card), `d` open diagram

New state: `fcNoteInput`
- Text input panel at bottom of screen — flashcard view stays visible above
- Auto-linked to current card
- User can cycle link target with `tab` before saving

### Quiz feedback (`internal/tui/quiz.go`)

After answering a question with `concept_id`:
- Show concept summary (content + user if both exist)
- Same note-taking options — `n` opens note input, auto-linked to quiz question

### Progress view (`internal/tui/progress.go`)

Enhanced per-concept display:
```
c-pfk1: 62% confident
  Description: The rate-limiting enzyme of glycolysis.
  Content Summary: PFK-1 commits glucose to glycolysis...
  User Summary: My take on PFK-1 regulation...
  Notes (2):
    - "Inhibited by ATP and citrate" (linked to concept) [e edit]
    - "Tricky card, review more" (linked to card bch-001) [e edit]
  Diagrams: glycolysis-pathway [v ascii] [d svg]
```

Subject-level summaries shown at the top.

### Dashboard (`internal/tui/dashboard.go`)

Show subject summary if available. Show note/summary counts.

### New: `ken notes <subject>`

List all notes for a subject, filterable:
- `ken notes <subject>` — all notes
- `ken notes <subject> --concept c-pfk1` — notes linked to a concept
- `ken notes <subject> --unlinked` — notes not linked to anything
- `j`/`k` navigate, `e` edit, `x` delete, `n` new note, `/` search, `f` filter

### New: `ken summaries <subject>`

List all summaries for a subject:
- Show subject-level summaries first
- Then per-concept summaries
- Then multi-concept summaries
- `j`/`k` navigate, `e` edit, `x` delete, `s` new summary

---

## Key Bindings (new)

### Vim motions (global)

| Key | Context | Action |
|---|---|---|
| `j` | Any list/selection | Move down |
| `k` | Any list/selection | Move up |
| `gg` | Any list/selection | Jump to top |
| `G` | Any list/selection | Jump to bottom |
| `q` | Any view | Quit/back |

### Feature-specific

| Key | Context | Action |
|---|---|---|
| `n` | Flashcard/quiz/progress | Open note input (auto-linked to current context) |
| `d` | Progress view, flashcard back | Open diagram in default viewer (`xdg-open`) |
| `v` | Progress view, flashcard back | Render ASCII diagram inline |
| `s` | Progress view | Add/edit summary for concept/subject |
| `e` | Notes/summaries view, progress view | Edit selected note/summary |
| `x` | Notes/summaries view | Delete selected note/summary (with confirmation) |
| `f` | Notes/summaries view | Filter by linked entity type |
| `/` | Notes/summaries view | Search/filter notes |
| `tab` | Note input | Cycle link target (card → concept → quiz → note → nothing) |
| `m` | Dashboard | Open ken map (familiarity layer) |
| `v` | Dashboard | Open ken reflect (reflection layer) |

---

## CLI Changes

```
ken notes <subject>              # list all notes (interactive TUI)
ken notes <subject> --concept X  # notes linked to concept X
ken notes <subject> --unlinked   # unlinked notes
ken map <subject>                # familiarity layer (concept tree)
ken map <subject> --group X      # filter to course group X
ken reflect <subject>            # reflection layer (self-explanation)
ken reflect <subject> c-pfk1     # reflect on specific concept
```

Existing commands enhanced:
- `ken flashcards` — `n` opens note input (never stops flow), show summaries/diagrams
- `ken quiz` — `n` opens note input after feedback, show summaries
- `ken read` — concept tree view with concept hopping (n/N/1-9)

---

## Implementation Order

### Step 1: AGENTS.md update
- Fix stale project state
- Update package layout
- Add new features

### Step 2: Concept initialization
- Parse `concepts/*.md` when `ken flashcards` or `ken quiz` runs
- Call `progress.InitConcepts` before study begins

### Step 3: Parser updates + diagram dependencies
- Add `Diagram`, `Link` structs
- Add `Summary` field to `Concept`
- Parse `## <id>:summary` sections
- Parse `diagrams` (with inline mermaid source) and `links` from frontmatter
- Install `mermaigo` and `go-mermaid` dependencies
- Create `internal/diagram/` package for rendering wrapper

### Step 4: Progress model updates
- Add `Note`, `Summary`, `EntityRef` structs
- Add `Notes` and `Summaries` maps to `Progress`
- Update `Load`/`Save` to handle new fields

### Step 5: Note taking in TUI (never interrupts flow)
- Add `fcNoteInput` state to FlashcardModel
- Text input panel at bottom — current view stays visible above
- Auto-link to current context (card/concept/quiz)
- Tab cycles link target before saving
- Enter saves, esc cancels — returns to exact position
- Support for note-to-note linking

### Step 6: Note editing/deletion
- Add edit state to `ken notes` view (`e` key)
- Add delete with confirmation (`x` key)
- Edit inline in progress view
- Update `progress.Save` to handle edits and deletes

### Step 6: Summary creation in TUI
- Allow creating summaries during progress review
- Scope selector: concept / concepts / subject
- Save to `progress.Summaries`

### Step 7: Enhanced progress view
- Show summaries, notes, diagrams, links per concept
- Key bindings for opening diagrams/links
- Show subject-level summaries

### Step 8: Notes and summaries commands
- `ken notes <subject>` — list/filter/edit/delete notes
- `ken summaries <subject>` — list summaries
- Add search (`/`) and filter (`f`) to notes view

### Step 9: Vim motions
- Add j/k navigation to all list views
- Add gg/G for top/bottom
- Add q for quit/back
- Apply to: progress view, notes view, summaries view, flashcard grade selector, quiz answer selector

### Step 10: Test with real content
- Create test concept file with diagrams, links, summaries
- Verify end-to-end flow

---

## File Changes Summary

| File | Changes |
|---|---|
| `AGENTS.md` | Fix stale info, add new features |
| `CONTENT-CREATION.md` | Add groups.yaml, note tagging, remove links |
| `internal/parser/concept.go` | Add Diagram (Source+File) structs, summary parsing |
| `internal/parser/document.go` | New: concept tag parsing for ken read |
| `internal/parser/notes.go` | Note file loading for ken read |
| `internal/progress/progress.go` | Add Note/Summary/EntityRef structs, V2 per-layer model |
| `internal/groups/groups.go` | New: course groups parser (groups.yaml) |
| `internal/diagram/diagram.go` | New: mermaid rendering wrapper (mermaigo + go-mermaid) |
| `internal/tui/flashcards.go` | Add note input panel (non-interrupting), show summaries/diagrams |
| `internal/tui/quiz.go` | Show summaries (content+user) after feedback, note input |
| `internal/tui/progress.go` | Show all details, add key bindings, ASCII diagram view, edit/delete |
| `internal/tui/notes.go` | New: notes list/filter/edit/delete view with vim motions |
| `internal/tui/read.go` | New: concept tree view with hopping, concept tags |
| `internal/tui/kenmap.go` | New: familiarity layer (concept tree with expand/collapse) |
| `internal/tui/reflect.go` | New: reflection layer (self-explanation with canonical answer) |
| `internal/tui/dashboard.go` | Three-layer progress display, course groups, updated keybindings |
| `internal/tui/styles.go` | Add styles for diagrams, notes, summaries |
| `cmd/ken/flashcards.go` | Add concept initialization |
| `cmd/ken/quiz.go` | Add concept initialization |
| `cmd/ken/notes.go` | New: ken notes command |
| `cmd/ken/read.go` | Update for concept tree + hopping |
| `cmd/ken/map.go` | New: ken map command (familiarity layer) |
| `cmd/ken/reflect.go` | New: ken reflect command (reflection layer) |

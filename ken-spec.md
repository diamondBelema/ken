# `ken` — Folder-Based Learning Harness

Agent-ready build spec. Target: opencode (Big Pickle). Language: **Go** (bubbletea + lipgloss + bubbles for the TUI). Single static binary, no external runtime deps.

## Core Principle

`ken` is a **reader**, not a generator. It has zero opinion about how its folder was populated — a hand-written note, an opencode session, a Claude Code export, or an exported Claudy vault should all work identically, as long as they conform to the file format below. The format *is* the product. Treat it as a versioned contract, not an implementation detail.

**Mastery lives on concepts, not on individual flashcards/questions.** This is the important shift from a plain SM-2 approach: flashcards and quiz questions are *evidence* that updates a concept's confidence score, mirroring Expo's `BayesianConfidenceStrategyV2`. A card with no `concept_id` still gets studied and graded, but has nothing to accumulate confidence into — it's a standalone drill, not a mastery-tracked unit. Attaching a `concept_id` is what makes a card "count" toward long-term tracking.

## Folder Structure

```
~/Documents/learn/
├── config.json
├── stats.json
└── subjects/
    ├── biochemistry/
    │   ├── concepts/
    │   │   └── glycolysis.md
    │   ├── flashcards/
    │   │   └── glycolysis.md
    │   ├── quizzes/
    │   │   └── glycolysis.md
    │   └── progress.json
    └── physiology/
```

- A "subject" is just a directory name under `subjects/`. No subject registry file needed — `ken` discovers subjects by scanning directories.
- `concepts/`, `flashcards/`, and `quizzes/` each hold one or more `.md` files. One file = one "set" (e.g. one lecture's worth of material).
- `progress.json` is the only file `ken` ever writes to inside a subject folder. Everything else is read-only input.

## File Format: Markdown + YAML Frontmatter

Structured/answerable fields live in frontmatter (unambiguous for the parser). Free-text explanation/hint content lives in the markdown body (human-readable, agent-writable without fighting escaping rules).

### Concepts — `concepts/glycolysis.md`

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
---

## c-glycolysis
The metabolic pathway that breaks down glucose into pyruvate, yielding a net 2 ATP.

## c-pfk1
The rate-limiting, committed enzyme of glycolysis.
```

- `parent_id` mirrors Claudy's `parent_concept` self-relation — lets concepts nest into a topic hierarchy.
- Every concept starts with `confidence: 0.5` (neutral prior) and `last_reviewed_at: null` the first time it's seen — these live in `progress.json`, not here; this file is the read-only concept definition.

### Flashcard set — `flashcards/glycolysis.md`

```markdown
---
format_version: 1
type: flashcard_set
set: Glycolysis
source: BCH 208-MBBS-GLYCOLYSIS.pptx
cards:
  - id: bch-001
    concept_id: c-pfk1
    front: What is the rate-limiting enzyme of glycolysis?
    back: Phosphofructokinase-1 (PFK-1)
    tags: [glycolysis, enzymes]
---

## Notes: bch-001
Commits glucose to glycolysis — the committed, irreversible step regulated allosterically by ATP/citrate.
```

- `concept_id` is optional. Present → grading this card updates that concept's confidence. Absent → card studies normally but contributes no mastery signal anywhere.
- One `## Notes: <id>` section per card is optional — becomes hint/explanation text.
- `id` must be unique within the subject, not just the file — `ken` errors loudly on collision at load time.

### Quiz set — `quizzes/glycolysis.md`

```markdown
---
format_version: 1
type: quiz_set
set: Glycolysis Quiz
questions:
  - id: bch-q001
    concept_id: c-pfk1
    type: mcq
    question: Which enzyme catalyzes the committed step?
    options: [Hexokinase, PFK-1, Pyruvate kinase, Aldolase]
    answer: 1
    explanation: PFK-1 is the committed step
  - id: bch-q002
    type: true_false
    question: Glycolysis occurs in mitochondria
    answer: false
  - id: bch-q003
    type: fill_blank
    question: "Glucose is phosphorylated to glucose-___-phosphate"
    answer: "6"
---
```

- `type` is one of `mcq`, `true_false`, `fill_blank`. Unknown types are skipped with a warning at load, never a hard crash.
- Same `concept_id` rule as flashcards.

### `progress.json` (owned entirely by `ken`, never hand-edited)

```json
{
  "format_version": 1,
  "concepts": {
    "c-pfk1": {
      "confidence": 0.62,
      "last_reviewed_at": 1752835200
    }
  },
  "cards": {
    "bch-001": { "reviews": 4, "last_grade": "known_fairly" }
  },
  "quizzes": {
    "bch-q001": { "attempts": 3, "correct": 2, "streak": 1 }
  }
}
```

- `concepts` is the mastery ledger — this is what the dashboard and `ken progress` actually report on.
- `cards`/`quizzes` are lightweight per-item history (review counts, last grade) for the study UI's own bookkeeping — not mastery, just "have I seen this card before and how'd it go."

## Confidence Algorithm — ported from `BayesianConfidenceStrategyV2` (Expo/Claudy)

Direct port of the Kotlin source, same constants. Confidence lives on the concept (range `0.05`–`0.995`, neutral prior `0.5`), and is updated from two evidence types: a flashcard grade or a quiz result.

```go
package mastery

import "math"

const (
	minConfidence   = 0.05
	maxConfidence   = 0.995
	decayRatePerDay = 0.95
	inertia         = 0.8
	maxDailyDelta   = 0.08
)

type ConfidenceLevel int

const (
	Unknown ConfidenceLevel = iota
	KnownLittle
	KnownFairly
	KnownWell
	Mastered
)

type ConceptState struct {
	Confidence     float64
	LastReviewedAt *int64 // unix seconds, nil if never reviewed
}

// applyDecay ports BayesianConfidenceStrategyV2.applyDecay — forgetting
// curve since last review, slower decay once confidence is already high.
func applyDecay(c ConceptState, now int64) ConceptState {
	if c.LastReviewedAt == nil {
		return c
	}
	days := float64(now-*c.LastReviewedAt) / (60 * 60 * 24)
	if days <= 0 {
		return c
	}
	rate := decayRatePerDay
	exponent := days
	if c.Confidence > 0.8 {
		exponent = days * 0.5
	}
	decayed := c.Confidence * math.Pow(rate, exponent)
	c.Confidence = clamp(decayed, minConfidence, maxConfidence)
	return c
}

// bayesianUpdate ports the prior/likelihood -> posterior step.
func bayesianUpdate(prior, likelihood float64) float64 {
	numerator := prior * likelihood
	denominator := numerator + (1-prior)*(1-likelihood)
	return numerator / denominator
}

// applyInertia blends prior and posterior so a single review can't swing
// confidence too fast.
func applyInertia(prior, posterior float64) float64 {
	return prior*inertia + posterior*(1-inertia)
}

// capDelta enforces the daily max-change ceiling.
func capDelta(prior, updated float64) float64 {
	delta := clamp(updated-prior, -maxDailyDelta, maxDailyDelta)
	return prior + delta
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func likelihoodForFlashcardGrade(level ConfidenceLevel) float64 {
	switch level {
	case Unknown:
		return 0.15
	case KnownLittle:
		return 0.35
	case KnownFairly:
		return 0.6
	case KnownWell:
		return 0.8
	case Mastered:
		return 0.95
	}
	return 0.15
}

// UpdateFromFlashcard ports updateFromFlashcard.
func UpdateFromFlashcard(c ConceptState, grade ConfidenceLevel, now int64) ConceptState {
	decayed := applyDecay(c, now)
	likelihood := likelihoodForFlashcardGrade(grade)
	posterior := bayesianUpdate(decayed.Confidence, likelihood)
	blended := applyInertia(decayed.Confidence, posterior)
	capped := capDelta(decayed.Confidence, blended)
	decayed.Confidence = clamp(capped, minConfidence, maxConfidence)
	decayed.LastReviewedAt = &now
	return decayed
}

// UpdateFromQuiz ports updateFromQuiz, including the anomaly-tolerance
// rule: a miss from an already-high-confidence concept is treated more
// gently than a miss from a low-confidence one (assume slip, not true gap).
func UpdateFromQuiz(c ConceptState, wasCorrect bool, now int64) ConceptState {
	decayed := applyDecay(c, now)
	var likelihood float64
	switch {
	case wasCorrect:
		likelihood = 0.95
	case decayed.Confidence > 0.75:
		likelihood = 0.45 // anomaly tolerance
	default:
		likelihood = 0.3
	}
	posterior := bayesianUpdate(decayed.Confidence, likelihood)
	blended := applyInertia(decayed.Confidence, posterior)
	capped := capDelta(decayed.Confidence, blended)
	decayed.Confidence = clamp(capped, minConfidence, maxConfidence)
	decayed.LastReviewedAt = &now
	return decayed
}
```

This is a direct, same-constants port — `decayRatePerDay = 0.95`, `inertia = 0.8`, `maxDailyDelta = 0.08`, confidence bounds `0.05`–`0.995` all match the Kotlin source exactly. If Expo's constants change later, update both places or the two engines will quietly drift.

### Grading UI implication

Expo's flashcard evidence uses a 5-level `ConfidenceLevel` (Unknown/KnownLittle/KnownFairly/KnownWell/Mastered), not SM-2's 4-button Again/Hard/Good/Easy. `ken`'s flashcard study screen should present these same 5 options so the semantics match exactly what the algorithm expects — don't remap a 4-button scheme onto a 5-level enum, that reintroduces exactly the kind of drift this port is trying to avoid.

## CLI Commands

```
ken                       # Dashboard: due-for-review concepts, current streak, overall confidence spread
ken subjects              # List subjects with concept/card/quiz counts
ken flashcards <subject>  # Study mode: flip cards, grade Unknown/KnownLittle/KnownFairly/KnownWell/Mastered
ken quiz <subject>        # Take quiz: instant feedback per question, final score
ken progress [subject]    # Concept-level confidence breakdown, all subjects or one
ken stats                 # Detailed stats (confidence trend, streak history)
```

## Build Phases (each phase = one agent handoff, acceptance criteria included)

### Phase 1 — Scaffold + Folder Discovery
- Go module, cobra (or plain flag parsing) for CLI arg routing, bubbletea wired for at least a placeholder screen.
- `ken subjects` scans `~/Documents/learn/subjects/`, lists directory names + counts concept/flashcard/quiz files inside each.
- **Acceptance:** running `ken subjects` against a folder with 2+ subjects prints correct names and file counts. Missing `~/Documents/learn` directory produces a clear message, not a panic.

### Phase 1.5 — Concept Parsing
- Parser for `concept_set` frontmatter: load `id`/`name`/`parent_id` per concept, body sections (`## <id>`) become description text.
- Build an in-memory concept tree per subject (unresolved `parent_id` = warn + treat as root, don't crash).
- On first encounter of a concept not yet in `progress.json`, initialize it with `confidence: 0.5`, `last_reviewed_at: null`.
- **Acceptance:** a concepts file with 2+ levels of nesting loads correctly; a fresh subject with no `progress.json` yet gets one created with neutral-confidence entries for every concept found.

### Phase 2 — Mastery Engine (port `BayesianConfidenceStrategyV2`)
- Implement the `mastery` package exactly as specified above — `UpdateFromFlashcard`, `UpdateFromQuiz`, and their shared helpers.
- Unit tests ported from the same logic: confidence starting at 0.5, one `Mastered` grade should move it up but not past the daily delta cap; a quiz miss from confidence > 0.75 should apply the anomaly-tolerance likelihood (0.45), not the harsh one (0.3); decay should measurably reduce confidence after a simulated multi-day gap, and less sharply above 0.8 confidence than below it.
- **Acceptance:** a small fixture (concept starting at 0.5, then a sequence of 5 mixed flashcard/quiz evidence events with known timestamps) produces confidence values matching hand-computed expected output within floating-point tolerance (1e-6).

### Phase 3 — Flashcard Parsing + Study Mode
- Parser: split YAML frontmatter from markdown body, unmarshal into a `FlashcardSet` struct, associate `## Notes: <id>` sections back to card IDs, carry optional `concept_id` through.
- Duplicate `id` across the subject → hard error naming both files.
- Study mode: bubbletea screen showing front, keypress to flip to back, five keys/buttons for the `ConfidenceLevel` grades. On grade: if the card has a `concept_id`, call `mastery.UpdateFromFlashcard` and write the result to `progress.json`'s `concepts` map; always update the card's own `reviews`/`last_grade` in `progress.json`'s `cards` map regardless of whether it has a concept.
- **Acceptance:** studying a 5-card set (mix of cards with and without `concept_id`) end to end updates `progress.json` correctly — concept confidence only changes for cards that had one, `cards` history updates for all of them.

### Phase 4 — Quiz Mode
- Parser for `quiz_set` frontmatter, dispatch by `type` (mcq/true_false/fill_blank), unknown type = warn + skip.
- Instant feedback per question, running score. On answer: if the question has a `concept_id`, call `mastery.UpdateFromQuiz`; always update `quizzes` attempt/correct/streak history.
- **Acceptance:** a quiz mixing all three question types runs to completion, score matches manual count, malformed question (missing `answer`) is skipped with a visible warning, and concept confidence updates only fire for questions carrying a `concept_id`.

### Phase 5 — Progress & Dashboard
- `ken progress [subject]`: per-concept confidence (grouped by parent where nesting exists), highlighting concepts that haven't been reviewed in N days (decay-eligible) vs recently reinforced ones.
- `ken` bare command: dashboard summarizing overall confidence spread (e.g. count of concepts above/below some threshold like 0.7), streak, quick nav into the subject with the most decay-eligible concepts.
- `ken stats`: confidence trend over time (requires logging daily snapshots — lightweight `stats.json` at the root, append-only by date).
- **Acceptance:** dashboard renders correctly with zero subjects (empty state), one subject, and many subjects — no divide-by-zero, no crash on a subject with concepts but zero flashcards/quizzes attached yet.

## Explicitly Out of Scope (v1)

- No generation logic of any kind — `ken` never calls an LLM. That stays entirely on the agent/exporter side.
- No live sync between `ken`'s `progress.json` and Claudy's actual mobile confidence state — same algorithm, but two independent data stores. Studying the same concept in both places produces two separate confidence tracks that never reconcile automatically.
- No network calls, no auth, no multi-user — single local user, single machine.

## Open Design Note (not blocking build)

Because the algorithm is now shared exactly with Expo/Claudy, the two `progress.json`-equivalent stores (Claudy's PocketBase confidence field, `ken`'s local `concepts` map) will compute *identically* given identical evidence sequences — they just don't feed each other. If cross-device reconciliation ever becomes wanted, the fact that both sides run the same update function is what makes that tractable later (same math, just needs a merge strategy for divergent evidence histories) — not something to build now, just worth knowing the door's open because the algorithms match exactly.

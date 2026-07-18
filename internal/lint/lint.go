package lint

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/diamondBelema/ken/internal/parser"
)

type Severity int

const (
	SeverityError   Severity = iota // blocks study, must fix
	SeverityWarning                 // works but likely a content mistake
	SeverityInfo                    // fyi, not necessarily wrong
)

type Issue struct {
	Severity Severity
	File     string // relative to subject dir, e.g. "concepts/biochem.md"
	ID       string // the concept/card/question ID involved, if any
	Message  string
}

type Report struct {
	Subject string
	Issues  []Issue
}

func (r Report) HasErrors() bool {
	for _, i := range r.Issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

func (r Report) CountBySeverity() (errors, warnings, infos int) {
	for _, i := range r.Issues {
		switch i.Severity {
		case SeverityError:
			errors++
		case SeverityWarning:
			warnings++
		case SeverityInfo:
			infos++
		}
	}
	return
}

type conceptRef struct {
	file      string
	itemID    string
	conceptID string
}

func LintSubject(subjectsDir, subject string) (Report, error) {
	subjectDir := filepath.Join(subjectsDir, subject)
	info, err := os.Stat(subjectDir)
	if err != nil || !info.IsDir() {
		return Report{}, fmt.Errorf("subject directory not found: %s", subjectDir)
	}

	report := Report{Subject: subject}

	// Per-namespace ID tracking: id -> list of files
	conceptIDs := map[string][]string{}
	cardIDs := map[string][]string{}
	questionIDs := map[string][]string{}

	// Parent map: conceptID -> parentID (for cycle detection)
	parentMap := map[string]string{}

	// All known concept IDs
	allConceptIDs := map[string]bool{}

	// concept_id references from flashcards/quizzes
	var conceptRefs []conceptRef

	// Concepts referenced by at least one card or question
	referencedConcepts := map[string]bool{}

	// ── Concepts ──────────────────────────────────────────────────────────
	conceptDir := filepath.Join(subjectDir, "concepts")
	for _, fname := range listMDFiles(conceptDir) {
		relPath := "concepts/" + fname
		data, err := os.ReadFile(filepath.Join(conceptDir, fname))
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  fmt.Sprintf("failed to read file: %v", err),
			})
			continue
		}

		cs, err := parser.ParseConceptSet(data)
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  err.Error(),
			})
			continue
		}

		for _, c := range cs.Concepts {
			allConceptIDs[c.ID] = true
			conceptIDs[c.ID] = append(conceptIDs[c.ID], relPath)
			if c.ParentID != "" {
				parentMap[c.ID] = c.ParentID
			}

			for _, d := range c.Diagrams {
				if d.Source == "" && d.File == "" {
					report.Issues = append(report.Issues, Issue{
						Severity: SeverityWarning,
						File:     relPath,
						ID:       c.ID,
						Message:  fmt.Sprintf("diagram '%s' has no source", d.ID),
					})
				} else if d.File != "" {
					diagPath := filepath.Join(subjectDir, d.File)
					if _, err := os.Stat(diagPath); os.IsNotExist(err) {
						report.Issues = append(report.Issues, Issue{
							Severity: SeverityError,
							File:     relPath,
							ID:       c.ID,
							Message:  fmt.Sprintf("diagram file not found: %s", d.File),
						})
					}
				}
			}

			for _, l := range c.Links {
				if l.URL == "" {
					report.Issues = append(report.Issues, Issue{
						Severity: SeverityError,
						File:     relPath,
						ID:       c.ID,
						Message:  "link with empty URL",
					})
				} else if !strings.HasPrefix(l.URL, "http://") && !strings.HasPrefix(l.URL, "https://") {
					report.Issues = append(report.Issues, Issue{
						Severity: SeverityWarning,
						File:     relPath,
						ID:       c.ID,
						Message:  fmt.Sprintf("link URL does not start with http:// or https://: %s", l.URL),
					})
				}
			}
		}
	}

	// ── Flashcards ────────────────────────────────────────────────────────
	flashcardDir := filepath.Join(subjectDir, "flashcards")
	for _, fname := range listMDFiles(flashcardDir) {
		relPath := "flashcards/" + fname
		data, err := os.ReadFile(filepath.Join(flashcardDir, fname))
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  fmt.Sprintf("failed to read file: %v", err),
			})
			continue
		}

		fs, err := parser.ParseFlashcardSet(data)
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  err.Error(),
			})
			continue
		}

		for _, c := range fs.Cards {
			cardIDs[c.ID] = append(cardIDs[c.ID], relPath)

			if c.ConceptID != "" {
				conceptRefs = append(conceptRefs, conceptRef{
					file:      relPath,
					itemID:    c.ID,
					conceptID: c.ConceptID,
				})
			}

			if strings.TrimSpace(c.Front) == "" {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					ID:       c.ID,
					Message:  "empty front field",
				})
			}
			if strings.TrimSpace(c.Back) == "" {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					ID:       c.ID,
					Message:  "empty back field",
				})
			}
		}
	}

	// ── Quizzes ───────────────────────────────────────────────────────────
	quizDir := filepath.Join(subjectDir, "quizzes")
	for _, fname := range listMDFiles(quizDir) {
		relPath := "quizzes/" + fname
		data, err := os.ReadFile(filepath.Join(quizDir, fname))
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  fmt.Sprintf("failed to read file: %v", err),
			})
			continue
		}

		raw, _, err := parser.SplitFrontmatter(data)
		if err != nil {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  err.Error(),
			})
			continue
		}

		typeStr, _ := raw["type"].(string)
		if typeStr != "quiz_set" {
			if typeStr == "" {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					Message:  "missing 'type' field",
				})
			} else {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					Message:  fmt.Sprintf("expected type 'quiz_set', got '%s'", typeStr),
				})
			}
			continue
		}

		questionsRaw, ok := raw["questions"].([]interface{})
		if !ok {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     relPath,
				Message:  "missing or invalid 'questions' field",
			})
			continue
		}

		for _, q := range questionsRaw {
			qm, ok := q.(map[string]interface{})
			if !ok {
				continue
			}

			id, _ := qm["id"].(string)
			if id == "" {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					Message:  "question missing required 'id' field",
				})
				continue
			}

			questionIDs[id] = append(questionIDs[id], relPath)

			conceptID, _ := qm["concept_id"].(string)
			if conceptID != "" {
				conceptRefs = append(conceptRefs, conceptRef{
					file:      relPath,
					itemID:    id,
					conceptID: conceptID,
				})
			}

			qType, _ := qm["type"].(string)
			switch qType {
			case "mcq", "true_false", "fill_blank":
				// valid
			default:
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					ID:       id,
					Message:  fmt.Sprintf("unknown quiz type '%s'", qType),
				})
				continue
			}

			questionText, _ := qm["question"].(string)
			if strings.TrimSpace(questionText) == "" {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					ID:       id,
					Message:  "empty question field",
				})
			}

			if _, exists := qm["answer"]; !exists {
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     relPath,
					ID:       id,
					Message:  "missing answer field",
				})
			}

			if qType == "mcq" {
				optsRaw, _ := qm["options"].([]interface{})
				if len(optsRaw) < 2 {
					report.Issues = append(report.Issues, Issue{
						Severity: SeverityError,
						File:     relPath,
						ID:       id,
						Message:  fmt.Sprintf("mcq question has fewer than 2 options (%d)", len(optsRaw)),
					})
				}
			}
		}
	}

	// ── Cross-file checks ─────────────────────────────────────────────────

	// Duplicate IDs within namespaces
	for id, files := range conceptIDs {
		if len(files) > 1 {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				ID:       id,
				Message:  fmt.Sprintf("duplicate concept ID in: %s", strings.Join(files, ", ")),
			})
		}
	}
	for id, files := range cardIDs {
		if len(files) > 1 {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				ID:       id,
				Message:  fmt.Sprintf("duplicate flashcard ID in: %s", strings.Join(files, ", ")),
			})
		}
	}
	for id, files := range questionIDs {
		if len(files) > 1 {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				ID:       id,
				Message:  fmt.Sprintf("duplicate question ID in: %s", strings.Join(files, ", ")),
			})
		}
	}

	// Orphaned concept_id references
	for _, ref := range conceptRefs {
		if !allConceptIDs[ref.conceptID] {
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     ref.file,
				ID:       ref.itemID,
				Message:  fmt.Sprintf("references missing concept '%s'", ref.conceptID),
			})
		} else {
			referencedConcepts[ref.conceptID] = true
		}
	}

	// Concepts with zero evidence
	for id := range allConceptIDs {
		if !referencedConcepts[id] {
			file := ""
			if files := conceptIDs[id]; len(files) > 0 {
				file = files[0]
			}
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityWarning,
				File:     file,
				ID:       id,
				Message:  fmt.Sprintf("concept '%s' has no cards or quiz questions — its confidence will never update from study", id),
			})
		}
	}

	// Broken parent_id + cycle detection (uses pre-built parentMap)
	for id, parentID := range parentMap {
		if !allConceptIDs[parentID] {
			file := ""
			if files := conceptIDs[id]; len(files) > 0 {
				file = files[0]
			}
			report.Issues = append(report.Issues, Issue{
				Severity: SeverityError,
				File:     file,
				ID:       id,
				Message:  fmt.Sprintf("parent_id '%s' does not match any known concept", parentID),
			})
			continue
		}

		// Cycle detection: walk parent chain using parentMap
		visited := map[string]bool{id: true}
		current := parentID
		cycle := []string{id, current}
		for {
			if visited[current] {
				file := ""
				if files := conceptIDs[id]; len(files) > 0 {
					file = files[0]
				}
				report.Issues = append(report.Issues, Issue{
					Severity: SeverityError,
					File:     file,
					ID:       id,
					Message:  fmt.Sprintf("parent cycle detected: %s", strings.Join(cycle, " → ")),
				})
				break
			}
			visited[current] = true
			nextParent, ok := parentMap[current]
			if !ok || nextParent == "" {
				break
			}
			cycle = append(cycle, nextParent)
			current = nextParent
		}
	}

	// Empty subject
	if len(listMDFiles(conceptDir)) == 0 && len(listMDFiles(flashcardDir)) == 0 && len(listMDFiles(quizDir)) == 0 {
		report.Issues = append(report.Issues, Issue{
			Severity: SeverityWarning,
			Message:  "subject has no content files (concepts, flashcards, quizzes)",
		})
	}

	return report, nil
}

func listMDFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".md" {
			files = append(files, e.Name())
		}
	}
	return files
}

package progress

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/diamondBelema/ken/internal/parser"
)

const stateDirName = "ken"

func StateDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".local", "share", stateDirName), nil
}

func SubjectPath(subject string) (string, error) {
	dir, err := StateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, subject+".json"), nil
}

type EntityRef struct {
	Type string   `json:"type"`
	ID   string   `json:"id,omitempty"`
	IDs  []string `json:"ids,omitempty"`
}

type Note struct {
	Content   string     `json:"content"`
	LinkedTo  *EntityRef `json:"linked_to,omitempty"`
	CreatedAt int64      `json:"created_at"`
	UpdatedAt int64      `json:"updated_at"`
}

type Summary struct {
	Title     string     `json:"title"`
	Content   string     `json:"content"`
	LinkedTo  *EntityRef `json:"linked_to"`
	CreatedAt int64      `json:"created_at"`
	UpdatedAt int64      `json:"updated_at"`
}

type Progress struct {
	FormatVersion int                    `json:"format_version"`
	Concepts      map[string]ConceptState `json:"concepts"`
	Cards         map[string]CardState    `json:"cards"`
	Quizzes       map[string]QuizState    `json:"quizzes"`
	Notes         map[string]Note         `json:"notes,omitempty"`
	Summaries     map[string]Summary      `json:"summaries,omitempty"`
	NextNoteID    int                     `json:"next_note_id"`
	NextSummaryID int                     `json:"next_summary_id"`
}

type ConceptState struct {
	Confidence     float64 `json:"confidence"`
	LastReviewedAt *int64  `json:"last_reviewed_at"`
}

type CardState struct {
	Reviews   int    `json:"reviews"`
	LastGrade string `json:"last_grade"`
}

type QuizState struct {
	Attempts int `json:"attempts"`
	Correct  int `json:"correct"`
	Streak   int `json:"streak"`
}

func Load(path string) (*Progress, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Progress{
				FormatVersion: 1,
				Concepts:      make(map[string]ConceptState),
				Cards:         make(map[string]CardState),
				Quizzes:       make(map[string]QuizState),
				Notes:         make(map[string]Note),
				Summaries:     make(map[string]Summary),
				NextNoteID:    1,
				NextSummaryID: 1,
			}, nil
		}
		return nil, fmt.Errorf("failed to read progress.json: %w", err)
	}

	var p Progress
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("failed to parse progress.json: %w", err)
	}

	if p.Concepts == nil {
		p.Concepts = make(map[string]ConceptState)
	}
	if p.Cards == nil {
		p.Cards = make(map[string]CardState)
	}
	if p.Quizzes == nil {
		p.Quizzes = make(map[string]QuizState)
	}
	if p.Notes == nil {
		p.Notes = make(map[string]Note)
	}
	if p.Summaries == nil {
		p.Summaries = make(map[string]Summary)
	}

	if p.NextNoteID == 0 {
		p.NextNoteID = computeMaxID(p.Notes, "n-") + 1
	}
	if p.NextSummaryID == 0 {
		p.NextSummaryID = computeMaxID(p.Summaries, "s-") + 1
	}

	return &p, nil
}

func computeMaxID[V any](m map[string]V, prefix string) int {
	maxNum := 0
	for id := range m {
		if strings.HasPrefix(id, prefix) {
			numStr := strings.TrimPrefix(id, prefix)
			if n, err := strconv.Atoi(numStr); err == nil && n > maxNum {
				maxNum = n
			}
		}
	}
	return maxNum
}

func Save(path string, p *Progress) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal progress: %w", err)
	}

	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

func InitConcepts(p *Progress, concepts []parser.Concept) {
	for _, c := range concepts {
		if _, exists := p.Concepts[c.ID]; !exists {
			p.Concepts[c.ID] = ConceptState{
				Confidence:     0.5,
				LastReviewedAt: nil,
			}
		}
	}
}

func ConceptIDs(p *Progress) []string {
	ids := make([]string, 0, len(p.Concepts))
	for id := range p.Concepts {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

func (p *Progress) AddNote(content string, linkedTo *EntityRef) string {
	id := fmt.Sprintf("n-%d", p.NextNoteID)
	p.NextNoteID++
	now := time.Now().Unix()
	p.Notes[id] = Note{
		Content:   content,
		LinkedTo:  linkedTo,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return id
}

func (p *Progress) EditNote(id, content string) error {
	note, exists := p.Notes[id]
	if !exists {
		return fmt.Errorf("note %s not found", id)
	}
	note.Content = content
	note.UpdatedAt = time.Now().Unix()
	p.Notes[id] = note
	return nil
}

func (p *Progress) DeleteNote(id string) error {
	if _, exists := p.Notes[id]; !exists {
		return fmt.Errorf("note %s not found", id)
	}
	delete(p.Notes, id)
	return nil
}

func (p *Progress) AddSummary(title, content string, linkedTo *EntityRef) string {
	id := fmt.Sprintf("s-%d", p.NextSummaryID)
	p.NextSummaryID++
	now := time.Now().Unix()
	p.Summaries[id] = Summary{
		Title:     title,
		Content:   content,
		LinkedTo:  linkedTo,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return id
}

func (p *Progress) EditSummary(id, title, content string) error {
	summary, exists := p.Summaries[id]
	if !exists {
		return fmt.Errorf("summary %s not found", id)
	}
	summary.Title = title
	summary.Content = content
	summary.UpdatedAt = time.Now().Unix()
	p.Summaries[id] = summary
	return nil
}

func (p *Progress) DeleteSummary(id string) error {
	if _, exists := p.Summaries[id]; !exists {
		return fmt.Errorf("summary %s not found", id)
	}
	delete(p.Summaries, id)
	return nil
}

func (p *Progress) NotesForConcept(conceptID string) []Note {
	var notes []Note
	for _, note := range p.Notes {
		if note.LinkedTo != nil && note.LinkedTo.Type == "concept" && note.LinkedTo.ID == conceptID {
			notes = append(notes, note)
		}
	}
	return notes
}

func (p *Progress) NotesForCard(cardID string) []Note {
	var notes []Note
	for _, note := range p.Notes {
		if note.LinkedTo != nil && note.LinkedTo.Type == "card" && note.LinkedTo.ID == cardID {
			notes = append(notes, note)
		}
	}
	return notes
}

func (p *Progress) NotesForQuiz(quizID string) []Note {
	var notes []Note
	for _, note := range p.Notes {
		if note.LinkedTo != nil && note.LinkedTo.Type == "quiz" && note.LinkedTo.ID == quizID {
			notes = append(notes, note)
		}
	}
	return notes
}

func (p *Progress) UnlinkedNotes() []Note {
	var notes []Note
	for _, note := range p.Notes {
		if note.LinkedTo == nil {
			notes = append(notes, note)
		}
	}
	return notes
}

func (p *Progress) SummariesForConcept(conceptID string) []Summary {
	var summaries []Summary
	for _, summary := range p.Summaries {
		if summary.LinkedTo != nil && summary.LinkedTo.Type == "concept" && summary.LinkedTo.ID == conceptID {
			summaries = append(summaries, summary)
		}
	}
	return summaries
}

func (p *Progress) SubjectSummaries(subject string) []Summary {
	var summaries []Summary
	for _, summary := range p.Summaries {
		if summary.LinkedTo != nil && summary.LinkedTo.Type == "subject" && summary.LinkedTo.ID == subject {
			summaries = append(summaries, summary)
		}
	}
	return summaries
}

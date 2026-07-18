package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/render"
)

type notesState int

const (
	notesList notesState = iota
	notesDetail
	notesEdit
	notesDeleteConfirm
	notesNew
)

type NotesModel struct {
	progress    *progress.Progress
	subject     string
	state       notesState
	notes       []progress.Note
	noteIDs     []string
	selected    int
	scrollTop   int
	viewWidth   int
	editInput   textinput.Model
	editID      string
	filter      string
	searchInput textinput.Model
}

func NewNotesModel(prog *progress.Progress, subject string) NotesModel {
	ei := textinput.New()
	ei.Placeholder = "Edit note..."
	ei.Focus()
	ei.CharLimit = 1000

	si := textinput.New()
	si.Placeholder = "Search notes..."
	si.CharLimit = 100

	return NotesModel{
		progress:    prog,
		subject:     subject,
		state:       notesList,
		editInput:   ei,
		searchInput: si,
	}
}

func (m NotesModel) Init() tea.Cmd {
	return nil
}

func (m NotesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case notesList:
		return m.updateList(msg)
	case notesDetail:
		return m.updateDetail(msg)
	case notesEdit:
		return m.updateEdit(msg)
	case notesDeleteConfirm:
		return m.updateDeleteConfirm(msg)
	case notesNew:
		return m.updateNew(msg)
	}
	return m, nil
}

func (m NotesModel) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "j", "down":
			if m.selected < len(m.notes)-1 {
				m.selected++
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
			}
		case "g":
			m.selected = 0
		case "G":
			m.selected = len(m.notes) - 1
		case "enter":
			if len(m.notes) > 0 {
				m.state = notesDetail
			}
		case "n":
			return m.startNew(), nil
		case "e":
			if len(m.notes) > 0 {
				return m.startEdit(), nil
			}
		case "x":
			if len(m.notes) > 0 {
				m.state = notesDeleteConfirm
				return m, nil
			}
		case "/":
			m.state = notesList
			return m, tea.Batch(m.searchInput.Focus(), nil)
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.refreshNotes()
				return m, nil
			}
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
	}
	return m, nil
}

func (m NotesModel) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc", "q":
			m.state = notesList
		case "e":
			return m.startEdit(), nil
		case "x":
			m.state = notesDeleteConfirm
		}
	}
	return m, nil
}

func (m NotesModel) updateEdit(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			content := m.editInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.EditNote(m.editID, content)
			}
			m.state = notesDetail
			m.editInput.SetValue("")
			m.refreshNotes()
			return m, nil
		case "esc":
			m.state = notesDetail
			m.editInput.SetValue("")
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m NotesModel) updateDeleteConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "y", "Y":
			if len(m.notes) > 0 {
				m.progress.DeleteNote(m.noteIDs[m.selected])
				m.refreshNotes()
				if m.selected >= len(m.notes) {
					m.selected = len(m.notes) - 1
				}
			}
			m.state = notesList
		case "n", "N", "esc":
			m.state = notesList
		}
	}
	return m, nil
}

func (m NotesModel) updateNew(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			content := m.editInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.AddNote(content, nil)
				m.refreshNotes()
			}
			m.state = notesList
			m.editInput.SetValue("")
			return m, nil
		case "esc":
			m.state = notesList
			m.editInput.SetValue("")
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m *NotesModel) startEdit() NotesModel {
	if len(m.notes) == 0 {
		return *m
	}
	note := m.notes[m.selected]
	m.state = notesEdit
	m.editID = m.noteIDs[m.selected]
	m.editInput.SetValue(note.Content)
	m.editInput.Focus()
	return *m
}

func (m *NotesModel) startNew() NotesModel {
	m.state = notesNew
	m.editInput.SetValue("")
	m.editInput.Focus()
	return *m
}

func (m *NotesModel) refreshNotes() {
	m.notes = nil
	m.noteIDs = nil

	for id, note := range m.progress.Notes {
		if m.filter != "" {
			if note.LinkedTo == nil || note.LinkedTo.Type != m.filter {
				continue
			}
		}
		if m.searchInput.Value() != "" {
			if !strings.Contains(strings.ToLower(note.Content), strings.ToLower(m.searchInput.Value())) {
				continue
			}
		}
		m.notes = append(m.notes, note)
		m.noteIDs = append(m.noteIDs, id)
	}

	sort.Slice(m.notes, func(i, j int) bool {
		return m.notes[i].CreatedAt > m.notes[j].CreatedAt
	})
}

func (m NotesModel) View() string {
	m.refreshNotes()

	var b strings.Builder

	b.WriteString(titleStyle.Render(fmt.Sprintf("Notes — %s", m.subject)))
	b.WriteString("\n")

	if m.filter != "" {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Filter: %s (esc to clear)", m.filter)))
		b.WriteString("\n")
	}

	switch m.state {
	case notesList:
		if len(m.notes) == 0 {
			b.WriteString(subtitleStyle.Render("No notes found. Press 'n' to create one."))
			b.WriteString("\n")
		} else {
			for i, note := range m.notes {
				prefix := "  "
				if i == m.selected {
					prefix = "→ "
				}

				linkLabel := "unlinked"
				if note.LinkedTo != nil {
					switch note.LinkedTo.Type {
					case "concept":
						linkLabel = fmt.Sprintf("concept: %s", note.LinkedTo.ID)
					case "card":
						linkLabel = fmt.Sprintf("card: %s", note.LinkedTo.ID)
					case "quiz":
						linkLabel = fmt.Sprintf("quiz: %s", note.LinkedTo.ID)
					case "note":
						linkLabel = fmt.Sprintf("note: %s", note.LinkedTo.ID)
					}
				}

				preview := note.Content
				if len(preview) > 60 {
					preview = preview[:60] + "..."
				}
				preview = strings.ReplaceAll(preview, "\n", " ")

				b.WriteString(fmt.Sprintf("%s%s [%s]\n", prefix, preview, linkLabel))
			}
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("j/k navigate • enter view • n new • e edit • x delete • / search • q quit"))

	case notesDetail:
		if len(m.notes) > 0 {
			note := m.notes[m.selected]
			b.WriteString(render.RenderMarkdown(note.Content, m.viewWidth-4))
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("e edit • x delete • esc back"))
		}

	case notesEdit:
		b.WriteString(noteInputHeaderStyle.Render("Edit Note"))
		b.WriteString("\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter save • esc cancel"))

	case notesDeleteConfirm:
		b.WriteString(gradeUnknownStyle.Render("Delete this note? (y/n)"))
		b.WriteString("\n")

	case notesNew:
		b.WriteString(noteInputHeaderStyle.Render("New Note (unlinked)"))
		b.WriteString("\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("enter save • esc cancel"))
	}

	return b.String()
}

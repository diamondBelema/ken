package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/parser"
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
	notesSearching
)

type NotesModel struct {
	progress     *progress.Progress
	concepts     []parser.Concept
	subject      string
	state        notesState
	notes        []progress.Note
	noteIDs      []string
	selected     int
	scrollTop    int
	viewWidth    int
	viewHeight   int
	editInput    textarea.Model
	editID       string
	filter       string
	searchInput  textarea.Model
	noteLinkedTo *progress.EntityRef
	noteCycleIdx int
}

func NewNotesModel(prog *progress.Progress, concepts []parser.Concept, subject string) NotesModel {
	ei := textarea.New()
	ei.Placeholder = "Type your note here..."
	ei.Focus()
	ei.CharLimit = 5000
	ei.SetWidth(70)
	ei.SetHeight(8)
	ei.ShowLineNumbers = false

	si := textarea.New()
	si.Placeholder = "Search notes..."
	si.CharLimit = 100
	si.SetWidth(40)
	si.SetHeight(1)

	return NotesModel{
		progress:    prog,
		concepts:    concepts,
		subject:     subject,
		state:       notesList,
		editInput:   ei,
		searchInput: si,
	}
}

func (m NotesModel) Init() tea.Cmd {
	return func() tea.Msg {
		return notesInitMsg{}
	}
}

type notesInitMsg struct{}

func (m NotesModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
	case notesInitMsg:
		m.refreshNotes()
		return m, nil
	}

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
	case notesSearching:
		return m.updateSearch(msg)
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
				m.clampScroll()
			}
		case "k", "up":
			if m.selected > 0 {
				m.selected--
				m.clampScroll()
			}
		case "g":
			m.selected = 0
			m.clampScroll()
		case "G":
			m.selected = len(m.notes) - 1
			m.clampScroll()
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
			m.state = notesSearching
			m.searchInput.SetValue("")
			return m, tea.Batch(m.searchInput.Focus(), nil)
		case "f":
			switch m.filter {
			case "":
				m.filter = "concept"
			case "concept":
				m.filter = "card"
			case "card":
				m.filter = "quiz"
			case "quiz":
				m.filter = "note"
			case "note":
				m.filter = ""
			}
			m.selected = 0
			m.scrollTop = 0
			m.refreshNotes()
			return m, nil
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.selected = 0
				m.scrollTop = 0
				m.refreshNotes()
				return m, nil
			}
			return m, tea.Quit
		case "q":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m NotesModel) updateSearch(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.state = notesList
			m.selected = 0
			m.scrollTop = 0
			m.refreshNotes()
			return m, nil
		case "esc":
			m.searchInput.SetValue("")
			m.state = notesList
			m.selected = 0
			m.scrollTop = 0
			m.refreshNotes()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	m.refreshNotes()
	return m, cmd
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
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc":
			content := m.editInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.EditNote(m.editID, "", content)
			}
			m.state = notesDetail
			m.editInput.SetValue("")
			m.refreshNotes()
			return m, nil
		case "ctrl+s":
			content := m.editInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.EditNote(m.editID, "", content)
			}
			m.state = notesDetail
			m.editInput.SetValue("")
			m.refreshNotes()
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
				m.clampScroll()
			}
			m.state = notesList
		case "n", "N", "esc":
			m.state = notesList
		}
	}
	return m, nil
}

func (m NotesModel) updateNew(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch msg.String() {
		case "esc":
			content := m.editInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.AddNote("", content, m.noteLinkedTo)
				m.refreshNotes()
			}
			m.state = notesList
			m.editInput.SetValue("")
			m.noteLinkedTo = nil
			m.noteCycleIdx = 0
			return m, nil
		case "ctrl+s":
			content := m.editInput.Value()
			if strings.TrimSpace(content) != "" {
				m.progress.AddNote("", content, m.noteLinkedTo)
				m.refreshNotes()
			}
			m.state = notesList
			m.editInput.SetValue("")
			m.noteLinkedTo = nil
			m.noteCycleIdx = 0
			return m, nil
		case "tab":
			m.cycleLinkTarget()
			return m, nil
		}
	}
	var cmd tea.Cmd
	m.editInput, cmd = m.editInput.Update(msg)
	return m, cmd
}

func (m *NotesModel) cycleLinkTarget() {
	targets := []*progress.EntityRef{nil}

	// Add concepts
	sorted := sortedConceptsByHierarchy(m.concepts)
	for _, c := range sorted {
		targets = append(targets, &progress.EntityRef{Type: "concept", ID: c.ID})
	}

	// Add existing notes
	for id := range m.progress.Notes {
		targets = append(targets, &progress.EntityRef{Type: "note", ID: id})
	}

	m.noteCycleIdx = (m.noteCycleIdx + 1) % len(targets)
	m.noteLinkedTo = targets[m.noteCycleIdx]
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

func (m *NotesModel) getLinkTargetLabel() string {
	if m.noteLinkedTo == nil {
		return "unlinked"
	}

	switch m.noteLinkedTo.Type {
	case "concept":
		// Find concept name
		for _, c := range m.concepts {
			if c.ID == m.noteLinkedTo.ID {
				conceptMap := buildConceptMap(m.concepts)
				d := conceptDepth(conceptMap, c.ID)
				indent := ""
				if d > 0 {
					indent = strings.Repeat("  ", d) + "├─ "
				}
				return indent + c.Name + " (" + c.ID + ")"
			}
		}
		return m.noteLinkedTo.ID
	case "note":
		// Find note title
		if note, ok := m.progress.Notes[m.noteLinkedTo.ID]; ok {
			title := note.Title
			if title == "" {
				title = truncate(note.Content, 40)
			}
			return title + " (" + m.noteLinkedTo.ID + ")"
		}
		return m.noteLinkedTo.ID
	default:
		return m.noteLinkedTo.ID
	}
}

func (m *NotesModel) startNew() NotesModel {
	m.state = notesNew
	m.editInput.SetValue("")
	m.editInput.Focus()
	m.noteLinkedTo = nil
	m.noteCycleIdx = 0
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

func (m *NotesModel) clampScroll() {
	visible := m.viewHeight - 6
	if visible < 1 {
		visible = 10
	}
	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}
	if m.selected >= m.scrollTop+visible {
		m.scrollTop = m.selected - visible + 1
	}
}

func (m NotesModel) View() string {
	if m.viewWidth == 0 {
		m.viewWidth = 80
	}

	var b strings.Builder

	header := titleStyle.Render(fmt.Sprintf("  notes · %s  ", m.subject))
	b.WriteString(header)
	b.WriteString("\n\n")

	switch m.state {
	case notesList:
		if m.filter != "" {
			b.WriteString(fmt.Sprintf("  %s\n\n", lipgloss.NewStyle().Foreground(colorAccent).Render(fmt.Sprintf("Filter: %s", m.filter))))
		}

		if len(m.notes) == 0 {
			empty := lipgloss.NewStyle().
				Foreground(colorMuted).
				Padding(4, 2).
				Render("No notes found.\n\n  Press 'n' to create one.")
			b.WriteString(empty)
		} else {
			visible := m.viewHeight - 6
			if visible < 1 {
				visible = 10
			}
			end := m.scrollTop + visible
			if end > len(m.notes) {
				end = len(m.notes)
			}

			for i := m.scrollTop; i < end; i++ {
				note := m.notes[i]
				linkLabel := "unlinked"
				if note.LinkedTo != nil {
					switch note.LinkedTo.Type {
					case "concept":
						linkLabel = fmt.Sprintf("→ %s", note.LinkedTo.ID)
					case "card":
						linkLabel = fmt.Sprintf("→ %s", note.LinkedTo.ID)
					case "quiz":
						linkLabel = fmt.Sprintf("→ %s", note.LinkedTo.ID)
					case "note":
						linkLabel = fmt.Sprintf("→ %s", note.LinkedTo.ID)
					}
				}

				title := note.Title
				if title == "" {
					title = truncate(note.Content, 60)
				}

				if i == m.selected {
					b.WriteString(listItemSelectedStyle.Render(fmt.Sprintf("  %s  %s", truncate(title, 50), linkLabel)))
					b.WriteString("\n")
				} else {
					b.WriteString(fmt.Sprintf("  %s  %s\n", listItemStyle.Render(truncate(title, 50)), lipgloss.NewStyle().Foreground(colorMuted).Render(linkLabel)))
				}
			}
		}
		b.WriteString("\n")
		if len(m.notes) > m.viewHeight-6 {
			b.WriteString(helpStyle.Render(fmt.Sprintf("  %d notes  ·  j/k navigate  ·  enter view  ·  n new  ·  e edit  ·  x delete  ·  / search  ·  f filter  ·  q quit", len(m.notes))))
		} else {
			b.WriteString(helpStyle.Render("  j/k navigate  ·  enter view  ·  n new  ·  e edit  ·  x delete  ·  / search  ·  f filter  ·  q quit"))
		}

	case notesSearching:
		b.WriteString("  ")
		b.WriteString(m.searchInput.View())
		b.WriteString("\n\n")
		if len(m.notes) > 0 {
			visible := m.viewHeight - 6
			if visible < 1 {
				visible = 10
			}
			end := m.scrollTop + visible
			if end > len(m.notes) {
				end = len(m.notes)
			}
			for i := m.scrollTop; i < end; i++ {
				note := m.notes[i]
				linkLabel := "unlinked"
				if note.LinkedTo != nil {
					linkLabel = fmt.Sprintf("→ %s", note.LinkedTo.ID)
				}
				title := note.Title
				if title == "" {
					title = truncate(note.Content, 60)
				}
				if i == m.selected {
					b.WriteString(listItemSelectedStyle.Render(fmt.Sprintf("  %s  %s", truncate(title, 50), linkLabel)))
					b.WriteString("\n")
				} else {
					b.WriteString(fmt.Sprintf("  %s  %s\n", listItemStyle.Render(truncate(title, 50)), lipgloss.NewStyle().Foreground(colorMuted).Render(linkLabel)))
				}
			}
		} else {
			b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render("  No matching notes"))
		}
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  enter confirm  ·  esc cancel"))

	case notesDetail:
		if len(m.notes) > 0 {
			note := m.notes[m.selected]
			if note.Title != "" {
				b.WriteString(titleStyle.Render(fmt.Sprintf("  %s", note.Title)))
				b.WriteString("\n\n")
			}
			b.WriteString(render.RenderMarkdown(note.Content, m.viewWidth-4))
			b.WriteString("\n")
			if note.LinkedTo != nil {
				b.WriteString(lipgloss.NewStyle().Foreground(colorMuted).Render(fmt.Sprintf("  linked to: %s %s", note.LinkedTo.Type, note.LinkedTo.ID)))
				b.WriteString("\n")
			}
			b.WriteString("\n")
			b.WriteString(helpStyle.Render("  e edit  ·  x delete  ·  esc back"))
		}

	case notesEdit:
		b.WriteString(noteInputHeaderStyle.Render("  edit note"))
		b.WriteString("\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  esc save  ·  enter newline  ·  ctrl+s save"))

	case notesDeleteConfirm:
		b.WriteString(lipgloss.NewStyle().
			Foreground(colorDanger).
			Bold(true).
			Render("  Delete this note? (y/n)"))
		b.WriteString("\n")

	case notesNew:
		linkLabel := "unlinked"
		if m.noteLinkedTo != nil {
			linkLabel = m.getLinkTargetLabel()
		}
		b.WriteString(noteInputHeaderStyle.Render(fmt.Sprintf("  new note  → %s", linkLabel)))
		b.WriteString("\n")
		b.WriteString(m.editInput.View())
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("  tab cycle link target  ·  esc save  ·  enter newline  ·  ctrl+s save"))
	}

	return b.String()
}

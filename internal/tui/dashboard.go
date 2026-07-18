package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diamondBelema/ken/internal/discovery"
	"github.com/diamondBelema/ken/internal/parser"
	"github.com/diamondBelema/ken/internal/progress"
	"github.com/diamondBelema/ken/internal/study"
)

type dashState int

const (
	dashBrowsing dashState = iota
	dashActionRow
	dashFiltering
)

type DashboardResult struct {
	Subject string
	Action  string
}

type DashboardModel struct {
	subjects       []discovery.SubjectInfo
	progData       map[string]*progress.Progress
	conceptData    map[string][]parser.Concept
	cardCounts     map[string]int
	quizCounts     map[string]int
	err            error
	width          int
	height         int
	state          dashState
	selected       int
	scrollTop      int
	actionSelected int
	filterInput    textinput.Model
	filterText     string
	result         DashboardResult
}

type dashboardQuitMsg struct {
	result DashboardResult
}

type dashboardLoadedMsg struct {
	subjects    []discovery.SubjectInfo
	progData    map[string]*progress.Progress
	conceptData map[string][]parser.Concept
	cardCounts  map[string]int
	quizCounts  map[string]int
}

type dashboardErrMsg struct {
	err error
}

func NewDashboardModel() DashboardModel {
	ti := textinput.New()
	ti.Placeholder = "filter subjects..."
	ti.Focus()
	ti.CharLimit = 40

	return DashboardModel{
		filterInput: ti,
	}
}

func (m DashboardModel) Result() DashboardResult {
	return m.result
}

func (m DashboardModel) Init() tea.Cmd {
	return func() tea.Msg {
		home, err := os.UserHomeDir()
		if err != nil {
			return dashboardErrMsg{err}
		}

		subjectsDir := filepath.Join(home, "Documents", "learn", "subjects")
		subjects, err := discovery.Discover(subjectsDir)
		if err != nil {
			return dashboardErrMsg{err}
		}

		progData := make(map[string]*progress.Progress)
		conceptData := make(map[string][]parser.Concept)
		cardCounts := make(map[string]int)
		quizCounts := make(map[string]int)
		for _, s := range subjects {
			progPath, err := progress.SubjectPath(s.Name)
			if err != nil {
				continue
			}
			prog, err := progress.Load(progPath)
			if err != nil {
				continue
			}
			progData[s.Name] = prog

			concepts, err := study.LoadConcepts(subjectsDir, s.Name)
			if err == nil {
				conceptData[s.Name] = concepts
				progress.InitConcepts(prog, concepts)
			}

			cardCounts[s.Name] = countFlashcards(subjectsDir, s.Name)
			quizCounts[s.Name] = countQuizzes(subjectsDir, s.Name)
		}

		return dashboardLoadedMsg{subjects: subjects, progData: progData, conceptData: conceptData, cardCounts: cardCounts, quizCounts: quizCounts}
	}
}

func (m DashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case dashboardLoadedMsg:
		m.subjects = msg.subjects
		m.progData = msg.progData
		m.conceptData = msg.conceptData
		m.cardCounts = msg.cardCounts
		m.quizCounts = msg.quizCounts
	case dashboardErrMsg:
		m.err = msg.err
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		return m.handleKey(msg)
	case dashboardQuitMsg:
		m.result = msg.result
		return m, tea.Quit
	}
	return m, nil
}

func (m DashboardModel) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.state {
	case dashBrowsing:
		return m.handleBrowsing(msg)
	case dashActionRow:
		return m.handleActionRow(msg)
	case dashFiltering:
		return m.handleFiltering(msg)
	}
	return m, nil
}

func (m DashboardModel) handleBrowsing(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	filtered := m.filteredSubjects()
	cols := m.columnCount()

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if cols > 1 {
			if m.selected+cols < len(filtered) {
				m.selected += cols
			} else if m.selected < len(filtered)-1 {
				m.selected = len(filtered) - 1
			}
		} else {
			if m.selected < len(filtered)-1 {
				m.selected++
			}
		}
		m.clampScroll()
	case "k", "up":
		if cols > 1 {
			if m.selected-cols >= 0 {
				m.selected -= cols
			} else {
				m.selected = 0
			}
		} else {
			if m.selected > 0 {
				m.selected--
			}
		}
		m.clampScroll()
	case "l", "right":
		if m.selected < len(filtered)-1 {
			m.selected++
		}
		m.clampScroll()
	case "h", "left":
		if m.selected > 0 {
			m.selected--
		}
		m.clampScroll()
	case "g", "home":
		m.selected = 0
		m.clampScroll()
	case "G", "end":
		m.selected = len(filtered) - 1
		m.clampScroll()
	case "/":
		m.state = dashFiltering
		m.filterInput.SetValue(m.filterText)
		m.filterInput.Focus()
		return m, textinput.Blink
	case "enter":
		if len(filtered) > 0 {
			m.state = dashActionRow
			m.actionSelected = 0
		}
	case "f":
		if len(filtered) > 0 {
			return m, m.launch(filtered[m.selected].Name, "flashcards")
		}
	case "t":
		if len(filtered) > 0 {
			return m, m.launch(filtered[m.selected].Name, "quiz")
		}
	case "n":
		if len(filtered) > 0 {
			return m, m.launch(filtered[m.selected].Name, "notes")
		}
	case "s":
		if len(filtered) > 0 {
			return m, m.launch(filtered[m.selected].Name, "summaries")
		}
	case "r":
		if len(filtered) > 0 {
			return m, m.launch(filtered[m.selected].Name, "read")
		}
	case "p":
		if len(filtered) > 0 {
			return m, m.launch(filtered[m.selected].Name, "progress")
		}
	}
	return m, nil
}

func (m DashboardModel) handleActionRow(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	actionCount := 6

	switch msg.String() {
	case "esc", "q":
		m.state = dashBrowsing
		return m, nil
	case "j", "right":
		m.actionSelected = (m.actionSelected + 1) % actionCount
	case "k", "left":
		m.actionSelected = (m.actionSelected - 1 + actionCount) % actionCount
	case "enter":
		filtered := m.filteredSubjects()
		if len(filtered) > 0 {
			subject := filtered[m.selected].Name
			action := actionNameForIndex(m.actionSelected)
			return m, m.launch(subject, action)
		}
	case "1":
		return m, m.launchFromAction(0)
	case "2":
		return m, m.launchFromAction(1)
	case "3":
		return m, m.launchFromAction(2)
	case "4":
		return m, m.launchFromAction(3)
	case "5":
		return m, m.launchFromAction(4)
	case "6":
		return m, m.launchFromAction(5)
	}
	return m, nil
}

func (m DashboardModel) launchFromAction(idx int) tea.Cmd {
	filtered := m.filteredSubjects()
	if len(filtered) == 0 {
		return nil
	}
	return m.launch(filtered[m.selected].Name, actionNameForIndex(idx))
}

func (m DashboardModel) handleFiltering(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = dashBrowsing
		m.filterInput.Blur()
		m.filterText = ""
		m.selected = 0
		m.clampScroll()
		return m, nil
	case "enter":
		m.state = dashBrowsing
		m.filterInput.Blur()
		m.selected = 0
		m.clampScroll()
		return m, nil
	case "j", "k":
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.filterText = m.filterInput.Value()
	m.selected = 0
	m.clampScroll()
	return m, cmd
}

func (m DashboardModel) launch(subject, action string) tea.Cmd {
	return func() tea.Msg {
		return dashboardQuitMsg{result: DashboardResult{Subject: subject, Action: action}}
	}
}

func (m DashboardModel) filteredSubjects() []discovery.SubjectInfo {
	if m.filterText == "" {
		return m.subjects
	}
	query := strings.ToLower(m.filterText)
	var filtered []discovery.SubjectInfo
	for _, s := range m.subjects {
		if strings.Contains(strings.ToLower(s.Name), query) {
			filtered = append(filtered, s)
		}
	}
	return filtered
}

func (m *DashboardModel) clampScroll() {
	visible := m.visibleRows()
	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}
	if m.selected >= m.scrollTop+visible*m.columnCount() {
		m.scrollTop = m.selected - visible*m.columnCount() + m.columnCount()
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m DashboardModel) visibleRows() int {
	if m.height == 0 {
		return 4
	}
	available := m.height - 10
	rows := available / 5
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m DashboardModel) columnCount() int {
	if m.width >= 120 {
		return 3
	}
	if m.width >= 72 {
		return 2
	}
	return 1
}

func (m DashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to exit.\n", m.err)
	}

	if m.width == 0 {
		m.width = 80
	}
	if m.height == 0 {
		m.height = 24
	}

	var b strings.Builder

	// Build content into a buffer to measure height
	var content strings.Builder
	content.WriteString(m.renderHeader())
	content.WriteString("\n")

	if m.state == dashFiltering || m.filterText != "" {
		filterLabel := dashFilterStyle.Render(" / ")
		filterLine := filterLabel + m.filterInput.View()
		content.WriteString(centerBlock(filterLine, m.width))
		content.WriteString("\n")
	}

	filtered := m.filteredSubjects()
	if len(filtered) == 0 {
		if m.filterText != "" {
			content.WriteString(dashDetailStyle.Render("  No subjects match your filter."))
			content.WriteString("\n")
		} else {
			content.WriteString(m.renderEmptyState())
		}
	} else {
		content.WriteString(m.renderGrid(filtered))
	}

	if m.state == dashActionRow && len(filtered) > 0 {
		content.WriteString(m.renderActionRow())
	}

	content.WriteString("\n")
	content.WriteString(centerBlock(m.renderHelpBar(), m.width))

	// Vertically center: add top padding
	contentLines := strings.Count(content.String(), "\n")
	footerLines := 2
	usedLines := contentLines + footerLines
	emptyLines := m.height - usedLines
	if emptyLines > 0 {
		topPad := emptyLines / 2
		for i := 0; i < topPad; i++ {
			b.WriteString("\n")
		}
	}

	b.WriteString(content.String())
	return b.String()
}

func (m DashboardModel) renderHeader() string {
	var b strings.Builder

	title := dashHeaderStyle.Render("ken")
	tagline := dashTaglineStyle.Render("terminal study harness")
	headerLine := title + "  " + tagline
	b.WriteString(centerBlock(headerLine, m.width))
	b.WriteString("\n")

	weak, dev, strong, total := 0, 0, 0, 0
	for _, s := range m.subjects {
		prog := m.progData[s.Name]
		concepts := m.conceptData[s.Name]
		for _, c := range concepts {
			total++
			conf := 0.5
			if prog != nil {
				if cs, ok := prog.Concepts[c.ID]; ok {
					conf = cs.Confidence
				}
			}
			switch {
			case conf < 0.3:
				weak++
			case conf < 0.7:
				dev++
			default:
				strong++
			}
		}
	}

	if total > 0 {
		// Scale bar to fit terminal width
		// "Weak X NNN · Developing X NNN · Strong X NNN · N concepts" ≈ 80 chars of labels
		availableForBars := m.width - 80
		if availableForBars < 30 {
			availableForBars = 30
		}
		barWidth := availableForBars / 3
		if barWidth < 10 {
			barWidth = 10
		}
		if barWidth > 40 {
			barWidth = 40
		}

		weakPct := weak * barWidth / total
		devPct := dev * barWidth / total
		strongPct := barWidth - weakPct - devPct

		distLine := fmt.Sprintf("Weak %s %d  ·  Developing %s %d  ·  Strong %s %d  ·  %d concepts",
			dashDistWeakStyle.Render(strings.Repeat("█", weakPct)+strings.Repeat("░", barWidth-weakPct)),
			weak,
			dashDistDevStyle.Render(strings.Repeat("█", devPct)+strings.Repeat("░", barWidth-devPct)),
			dev,
			dashDistStrongStyle.Render(strings.Repeat("█", strongPct)+strings.Repeat("░", barWidth-strongPct)),
			strong,
			total)
		b.WriteString(centerBlock(dashStatsRowStyle.Render(distLine), m.width))
	}

	return b.String()
}

func (m DashboardModel) renderGrid(filtered []discovery.SubjectInfo) string {
	cols := m.columnCount()
	colWidth := (m.width - 2*(cols-1)) / cols
	if colWidth < 20 {
		colWidth = 20
	}

	visible := m.visibleRows()
	startRow := m.scrollTop / cols
	if m.scrollTop%cols != 0 {
		startRow++
	}
	startIdx := startRow * cols
	if startIdx > len(filtered) {
		startIdx = len(filtered)
	}
	endIdx := startIdx + visible*cols
	if endIdx > len(filtered) {
		endIdx = len(filtered)
	}

	var rows []string
	for i := startIdx; i < endIdx; i += cols {
		end := i + cols
		if end > len(filtered) {
			end = len(filtered)
		}
		rowCards := filtered[i:end]
		var rendered []string
		for j, s := range rowCards {
			selected := (i+j) == m.selected
			rendered = append(rendered, m.renderSubjectCard(s, selected, colWidth))
		}
		if len(rendered) > 1 {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rendered...))
		} else {
			rows = append(rows, rendered[0])
		}
	}

	totalRows := (len(filtered) + cols - 1) / cols
	if totalRows > visible {
		scrollInfo := fmt.Sprintf("  %d-%d of %d subjects", startIdx+1, endIdx, len(filtered))
		rows = append(rows, dashDetailStyle.Render(scrollInfo))
	}

	return strings.Join(rows, "\n") + "\n"
}

func (m DashboardModel) renderSubjectCard(s discovery.SubjectInfo, selected bool, colWidth int) string {
	prog := m.progData[s.Name]
	concepts := m.conceptData[s.Name]
	conceptCount := len(concepts)
	noteCount := 0
	if prog != nil {
		noteCount = len(prog.Notes)
	}

	due := m.dueCount(s.Name)

	name := s.Name
	if selected {
		name = dashSubjectSelectedStyle.Render(s.Name)
	} else {
		name = dashSubjectStyle.Render(s.Name)
	}

	badge := ""
	if due > 0 {
		badge = "  " + dashBadgeDueStyle.Render(fmt.Sprintf("%d due", due))
	}

	lastStudied := m.lastStudiedText(s.Name)
	ts := ""
	if lastStudied != "" {
		ts = " " + dashDetailStyle.Render(lastStudied)
	}

	line1 := name + badge + ts

	avg := 0.0
	count := 0
	if prog != nil {
		for _, cs := range prog.Concepts {
			avg += cs.Confidence
			count++
		}
	}
	if count > 0 {
		avg = avg / float64(count) * 100
	}
	var confStyle lipgloss.Style
	switch {
	case avg < 30:
		confStyle = dashDistWeakStyle
	case avg < 70:
		confStyle = dashDistDevStyle
	default:
		confStyle = dashDistStrongStyle
	}
	line2 := fmt.Sprintf("  %s  %d·%d·%d",
		confStyle.Render(fmt.Sprintf("%.0f%%", avg)),
		conceptCount,
		m.cardCounts[s.Name],
		m.quizCounts[s.Name])

	var lines []string
	lines = append(lines, line1)
	lines = append(lines, line2)
	if noteCount > 0 {
		lines = append(lines, "  "+dashDetailStyle.Render(fmt.Sprintf("%d notes", noteCount)))
	}

	content := strings.Join(lines, "\n")

	style := dashCardStyle.
		Width(colWidth - 4).
		Padding(0, 1)
	if selected {
		style = dashCardSelectedStyle.
			Width(colWidth - 4).
			Padding(0, 1)
	}

	return style.Render(content)
}

func (m DashboardModel) renderEmptyState() string {
	boxWidth := 56
	leftPad := (m.width - boxWidth) / 2
	if leftPad < 0 {
		leftPad = 0
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(1, 2).
		Width(boxWidth).
		Render(
			dashDetailStyle.Render("No subjects found.")+"\n\n"+
				dashDetailStyle.Render("Add content to:")+"\n"+
				"  ~/Documents/learn/subjects/\n\n"+
				dashHintStyle.Render("Each subject needs concepts/, flashcards/,"),
			dashHintStyle.Render("and quizzes/ folders with .md files."),
		)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", leftPad))
	b.WriteString(box)
	b.WriteString("\n")
	return b.String()
}

func (m DashboardModel) dueCount(subject string) int {
	prog := m.progData[subject]
	if prog == nil {
		return 0
	}
	now := time.Now().Unix()
	count := 0
	for _, cs := range prog.Concepts {
		if cs.Confidence >= 0.7 {
			continue
		}
		if cs.LastReviewedAt == nil || (now-*cs.LastReviewedAt) > 86400 {
			count++
		}
	}
	return count
}

func (m DashboardModel) lastStudiedText(subject string) string {
	prog := m.progData[subject]
	if prog == nil {
		return ""
	}

	var latest *int64
	for _, cs := range prog.Concepts {
		if cs.LastReviewedAt != nil {
			if latest == nil || *cs.LastReviewedAt > *latest {
				latest = cs.LastReviewedAt
			}
		}
	}

	if latest == nil {
		return ""
	}

	now := time.Now().Unix()
	diff := now - *latest

	switch {
	case diff < 60:
		return "just now"
	case diff < 3600:
		mins := diff / 60
		return fmt.Sprintf("%dm ago", mins)
	case diff < 86400:
		hours := diff / 3600
		return fmt.Sprintf("%dh ago", hours)
	case diff < 172800:
		return "yesterday"
	default:
		days := diff / 86400
		return fmt.Sprintf("%dd ago", days)
	}
}

func (m DashboardModel) renderActionRow() string {
	actions := []string{"flashcards", "quiz", "notes", "summaries", "read", "progress"}
	shortcuts := []string{"f", "t", "n", "s", "r", "p"}

	var items []string
	for i, action := range actions {
		label := fmt.Sprintf("[%s]%s", shortcuts[i], action)
		if i == m.actionSelected {
			items = append(items, dashActionItemSelStyle.Render(label))
		} else {
			items = append(items, dashActionItemStyle.Render(label))
		}
	}

	row := dashActionBarStyle.Render("→ ") + strings.Join(items, "  ")
	return centerBlock(row, m.width) + "\n"
}

func (m DashboardModel) renderHelpBar() string {
	if m.state == dashActionRow {
		return helpStyle.Render("j/k select  ·  enter launch  ·  esc back")
	}
	if m.state == dashFiltering {
		return helpStyle.Render("type to filter  ·  enter confirm  ·  esc clear")
	}
	return helpStyle.Render("hjkl navigate  ·  enter actions  ·  f/t/n/s/r/p launch  ·  / filter  ·  q quit")
}

func actionNameForIndex(idx int) string {
	actions := []string{"flashcards", "quiz", "notes", "summaries", "read", "progress"}
	if idx >= 0 && idx < len(actions) {
		return actions[idx]
	}
	return "flashcards"
}

func countFlashcards(subjectsDir, subject string) int {
	flashcardsDir := filepath.Join(subjectsDir, subject, "flashcards")
	entries, err := os.ReadDir(flashcardsDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(flashcardsDir, entry.Name()))
		if err != nil {
			continue
		}
		set, err := parser.ParseFlashcardSet(data)
		if err != nil {
			continue
		}
		count += len(set.Cards)
	}
	return count
}

func countQuizzes(subjectsDir, subject string) int {
	quizzesDir := filepath.Join(subjectsDir, subject, "quizzes")
	entries, err := os.ReadDir(quizzesDir)
	if err != nil {
		return 0
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(quizzesDir, entry.Name()))
		if err != nil {
			continue
		}
		set, err := parser.ParseQuizSet(data)
		if err != nil {
			continue
		}
		count += len(set.Questions)
	}
	return count
}

func centerBlock(text string, width int) string {
	maxW := 0
	for _, line := range strings.Split(text, "\n") {
		w := lipgloss.Width(line)
		if w > maxW {
			maxW = w
		}
	}
	leftPad := (width - maxW) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	pad := strings.Repeat(" ", leftPad)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

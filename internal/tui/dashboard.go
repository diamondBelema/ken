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

	switch msg.String() {
	case "q", "esc", "ctrl+c":
		return m, tea.Quit
	case "j", "down":
		if m.selected < len(filtered)-1 {
			m.selected++
			m.clampScroll()
		}
	case "k", "up":
		if m.selected > 0 {
			m.selected--
			m.clampScroll()
		}
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
	case "t": // quiz uses 't' since 'q' is quit
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
		// Don't consume j/k while filtering — let them go to the text input
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
	visible := m.visibleCount()
	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}
	if m.selected >= m.scrollTop+visible {
		m.scrollTop = m.selected - visible + 1
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m DashboardModel) visibleCount() int {
	if m.height == 0 {
		return 24
	}
	// header ~5 lines, footer ~3 lines, each card ~5 lines + margin
	available := m.height - 8
	if available < 1 {
		available = 1
	}
	return available
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

	// Header
	b.WriteString(m.renderHeader())
	b.WriteString("\n")

	// Filter line
	if m.state == dashFiltering || m.filterText != "" {
		filterLabel := dashFilterStyle.Render(" / ")
		b.WriteString("  " + filterLabel + m.filterInput.View())
		b.WriteString("\n")
	}

	// Subject cards
	filtered := m.filteredSubjects()
	if len(filtered) == 0 {
		if m.filterText != "" {
			b.WriteString("\n  ")
			b.WriteString(dashDetailStyle.Render("No subjects match your filter."))
		} else {
			b.WriteString(m.renderEmptyState())
		}
	} else {
		visible := m.visibleCount()
		end := m.scrollTop + visible
		if end > len(filtered) {
			end = len(filtered)
		}
		for i := m.scrollTop; i < end; i++ {
			b.WriteString(m.renderSubjectCard(filtered[i], i == m.selected))
			b.WriteString("\n")
		}
		// Scroll indicator
		if len(filtered) > visible {
			scrollInfo := fmt.Sprintf("  %d-%d of %d", m.scrollTop+1, end, len(filtered))
			b.WriteString(dashDetailStyle.Render(scrollInfo))
			b.WriteString("\n")
		}
	}

	// Action row (inline below selected card)
	if m.state == dashActionRow && len(filtered) > 0 {
		b.WriteString(m.renderActionRow())
	}

	// Status bar
	b.WriteString(strings.Repeat("─", m.width))
	b.WriteString("\n")
	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	b.WriteString(m.renderHelpBar())

	return b.String()
}

func (m DashboardModel) renderHeader() string {
	var b strings.Builder

	b.WriteString("  ")
	b.WriteString(dashHeaderStyle.Render("ken"))
	b.WriteString("\n")
	b.WriteString("  ")
	b.WriteString(dashTaglineStyle.Render("a terminal study harness that forgets nothing on purpose"))
	b.WriteString("\n")

	// Aggregate stats
	totalConcepts, totalMastered, totalCards, totalQuizzes := 0, 0, 0, 0
	for _, s := range m.subjects {
		prog := m.progData[s.Name]
		concepts := m.conceptData[s.Name]
		totalConcepts += len(concepts)
		totalCards += m.cardCounts[s.Name]
		totalQuizzes += m.quizCounts[s.Name]
		if prog != nil {
			for _, cs := range prog.Concepts {
				if cs.Confidence >= 0.7 {
					totalMastered++
				}
			}
		}
	}

	pct := 0
	if totalConcepts > 0 {
		pct = (totalMastered * 100) / totalConcepts
	}

	statsLine := fmt.Sprintf("  %d concepts  ·  %d mastered (%d%%)  ·  %d cards  ·  %d quizzes  ·  %d subjects",
		totalConcepts, totalMastered, pct, totalCards, totalQuizzes, len(m.subjects))
	b.WriteString(dashStatsRowStyle.Render(statsLine))

	return b.String()
}

func (m DashboardModel) renderEmptyState() string {
	var b strings.Builder
	b.WriteString("\n")
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(colorMuted).
		Padding(1, 2).
		MarginLeft(2).
		Width(56).
		Render(
			dashDetailStyle.Render("No subjects found.")+"\n\n"+
				dashDetailStyle.Render("Add content to:")+"\n"+
				"  ~/Documents/learn/subjects/\n\n"+
				dashHintStyle.Render("Each subject needs concepts/, flashcards/,"),
			dashHintStyle.Render("and quizzes/ folders with .md files."),
		)
	b.WriteString(box)
	b.WriteString("\n\n  ")
	b.WriteString(helpStyle.Render("q quit"))
	return b.String()
}

func (m DashboardModel) renderSubjectCard(s discovery.SubjectInfo, selected bool) string {
	prog := m.progData[s.Name]
	concepts := m.conceptData[s.Name]
	conceptCount := len(concepts)
	mastered := 0
	noteCount := 0
	if prog != nil {
		noteCount = len(prog.Notes)
		for _, cs := range prog.Concepts {
			if cs.Confidence >= 0.7 {
				mastered++
			}
		}
	}

	var lines []string

	// Subject name + due badge
	name := dashSubjectStyle.Render(s.Name)
	if selected {
		name = dashSubjectSelectedStyle.Render(s.Name)
	}

	due := m.dueCount(s.Name)
	badge := ""
	if due > 0 {
		badge = "  " + dashBadgeDueStyle.Render(fmt.Sprintf("%d due", due))
	} else if mastered == conceptCount && conceptCount > 0 {
		badge = "  " + lipgloss.NewStyle().Foreground(colorSuccess).Render("all mastered")
	}

	lastStudied := m.lastStudiedText(s.Name)
	timestamp := ""
	if lastStudied != "" {
		timestamp = "  " + dashDetailStyle.Render(lastStudied)
	}

	lines = append(lines, name+badge+timestamp)

	// Confidence bar
	lines = append(lines, "  "+m.renderGradientBar(prog, conceptCount))

	// Stats line
	stats := fmt.Sprintf("  %d concepts  ·  %d cards  ·  %d quizzes  ·  %d notes",
		conceptCount, m.cardCounts[s.Name], m.quizCounts[s.Name], noteCount)
	lines = append(lines, dashDetailStyle.Render(stats))

	// Hint for unstudied subjects
	if conceptCount > 0 && due == conceptCount && mastered == 0 {
		lines = append(lines, "  "+dashHintStyle.Render("press f to start studying"))
	}

	content := strings.Join(lines, "\n")

	if selected {
		return dashCardSelectedStyle.Render(content)
	}
	return dashCardStyle.Render(content)
}

func (m DashboardModel) renderGradientBar(prog *progress.Progress, total int) string {
	if total == 0 {
		return dashBarEmptyStyle.Render(strings.Repeat("─", 30)) + " " + dashDetailStyle.Render("0%")
	}

	mastered := 0
	if prog != nil {
		for _, cs := range prog.Concepts {
			if cs.LastReviewedAt != nil && cs.Confidence >= 0.7 {
				mastered++
			}
		}
	}

	barWidth := 30
	if m.width > 0 && m.width-30 < barWidth {
		barWidth = m.width - 30
		if barWidth < 10 {
			barWidth = 10
		}
	}

	filled := 0
	if total > 0 {
		filled = (mastered * barWidth) / total
	}

	filledStyle := dashBarFilledStyle
	pct := 0
	if total > 0 {
		pct = (mastered * 100) / total
	}

	switch {
	case pct < 30:
		filledStyle = dashBarRedStyle
	case pct < 60:
		filledStyle = dashBarAmberStyle
	case pct >= 85:
		filledStyle = dashBarTealStyle
	}

	barFilled := strings.Repeat("━", filled)
	barEmpty := strings.Repeat("─", barWidth-filled)

	result := filledStyle.Render(barFilled) + dashBarEmptyStyle.Render(barEmpty)
	result += fmt.Sprintf(" %d%%", pct)

	return result
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
		return fmt.Sprintf("last studied %dm ago", mins)
	case diff < 86400:
		hours := diff / 3600
		return fmt.Sprintf("last studied %dh ago", hours)
	case diff < 172800:
		return "last studied yesterday"
	default:
		days := diff / 86400
		return fmt.Sprintf("last studied %dd ago", days)
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

	row := "  " + dashActionBarStyle.Render("→ ") + strings.Join(items, "  ")
	return row + "\n"
}

func (m DashboardModel) renderStatusBar() string {
	totalConcepts, totalMastered := 0, 0
	for _, s := range m.subjects {
		concepts := m.conceptData[s.Name]
		totalConcepts += len(concepts)
		prog := m.progData[s.Name]
		if prog != nil {
			for _, cs := range prog.Concepts {
				if cs.Confidence >= 0.7 {
					totalMastered++
				}
			}
		}
	}

	statsLine := fmt.Sprintf("  %d concepts  ·  %d mastered  ·  %d subjects",
		totalConcepts, totalMastered, len(m.subjects))
	return statusBarStyle.Render(statsLine)
}

func (m DashboardModel) renderHelpBar() string {
	if m.state == dashActionRow {
		return helpStyle.Render("  j/k select action  ·  enter launch  ·  esc back")
	}
	if m.state == dashFiltering {
		return helpStyle.Render("  type to filter  ·  enter confirm  ·  esc clear")
	}
	return helpStyle.Render("  j/k navigate  ·  enter actions  ·  f/t/n/s/r/p quick launch  ·  / filter  ·  q quit")
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

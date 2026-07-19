package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

const minCardWidth = 28

type DashboardModel struct {
	subjects       []discovery.SubjectInfo
	progData       map[string]*progress.Progress
	conceptData    map[string][]parser.Concept
	cardCounts     map[string]int
	quizCounts     map[string]int
	err            error
	viewWidth      int
	viewHeight     int
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
		m.viewWidth = msg.Width
		m.viewHeight = msg.Height
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
	cols := m.columnCount()
	if m.selected < m.scrollTop {
		m.scrollTop = m.selected
	}
	if m.selected >= m.scrollTop+visible*cols {
		m.scrollTop = m.selected - visible*cols + cols
	}
	if m.scrollTop < 0 {
		m.scrollTop = 0
	}
}

func (m DashboardModel) visibleRows() int {
	if m.viewHeight == 0 {
		return 4
	}
	headerLines := 4
	footerLines := 1
	actionLines := 0
	if m.state == dashActionRow {
		actionLines = 1
	}
	activityMin := 3
	available := m.viewHeight - headerLines - footerLines - actionLines - activityMin
	if available < 1 {
		available = 1
	}
	ch := m.cardHeight()
	rows := available / ch
	if rows < 1 {
		rows = 1
	}
	return rows
}

func (m DashboardModel) cardHeight() int {
	sample := m.renderSubjectCard(discovery.SubjectInfo{Name: "sample"}, false, 60)
	h := lipgloss.Height(sample)
	if h < 3 {
		h = 5
	}
	return h
}

func (m DashboardModel) columnCount() int {
	cols := m.viewWidth / (minCardWidth + 2)
	if cols > 3 {
		cols = 3
	}
	if cols < 1 {
		cols = 1
	}
	return cols
}

func (m DashboardModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n  Error: %v\n\n  Press q to exit.\n", m.err)
	}

	if m.viewWidth == 0 {
		m.viewWidth = 80
	}
	if m.viewHeight == 0 {
		m.viewHeight = 24
	}

	// === BUILD ALL SECTIONS ===

	// Header (no top margin)
	header := m.renderHeader()
	headerH := lipgloss.Height(header)

	// Gap after header
	gap1 := "\n"
	gap1H := 1

	// Filter
	var filter string
	var filterH int
	if m.state == dashFiltering || m.filterText != "" {
		filterLabel := dashFilterStyle.Render(" / ")
		filter = centerBlock(filterLabel+m.filterInput.View(), m.viewWidth)
		filterH = lipgloss.Height(filter)
	}

	// Action row
	var actionRow string
	var actionRowH int
	if m.state == dashActionRow {
		actionRow = m.renderActionRow()
		actionRowH = lipgloss.Height(actionRow)
	}

	// Grid
	filtered := m.filteredSubjects()
	var grid string
	var gridH int
	if len(filtered) == 0 {
		if m.filterText != "" {
			grid = "  " + dashDetailStyle.Render("No subjects match your filter.")
			gridH = 1
		} else {
			grid = m.renderEmptyState()
			gridH = lipgloss.Height(grid)
		}
	} else {
		grid = m.renderGrid(filtered)
		gridH = lipgloss.Height(grid)
	}

	// Calculate used height before activity panel
	usedH := headerH + gap1H + filterH + actionRowH + gridH

	// Footer
	footer := centerBlock(m.renderHelpBar(), m.viewWidth)
	footerH := lipgloss.Height(footer)

	// Activity panel height = whatever is left
	activityH := m.viewHeight - usedH - footerH
	if activityH < 3 {
		activityH = 3
	}

	// Build activity panel
	activity := m.renderActivityPanel(activityH)

	// === ASSEMBLE ===
	var b strings.Builder

	b.WriteString(header)
	b.WriteString(gap1)

	if filter != "" {
		b.WriteString(filter)
		b.WriteString("\n")
	}
	if actionRow != "" {
		b.WriteString(actionRow)
	}

	b.WriteString(grid)

	if activity != "" {
		b.WriteString(activity)
	}

	// Calculate current rendered height
	currentH := usedH
	if activity != "" {
		currentH += lipgloss.Height(activity)
	}

	// Pad to push footer to bottom
	padding := m.viewHeight - currentH - footerH
	for i := 0; i < padding; i++ {
		b.WriteString("\n")
	}

	b.WriteString(footer)

	return b.String()
}

func (m DashboardModel) renderHeader() string {
	var b strings.Builder

	title := dashHeaderStyle.Render("ken")
	tagline := dashTaglineStyle.Render("terminal study harness")
	headerLine := title + "  " + tagline
	b.WriteString(centerBlock(headerLine, m.viewWidth))

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
		labelOnly := fmt.Sprintf("Weak %d  ·  Developing %d  ·  Strong %d  ·  %d concepts",
			weak, dev, strong, total)
		labelWidth := lipgloss.Width(labelOnly)

		minBarWidth := 6
		minTotalWidth := labelWidth + 3*minBarWidth + 12

		if m.viewWidth < minTotalWidth {
			distLine := fmt.Sprintf("Weak %d  ·  Developing %d  ·  Strong %d  ·  %d concepts",
				weak, dev, strong, total)
			b.WriteString("\n")
			b.WriteString(centerBlock(dashStatsRowStyle.Render(distLine), m.viewWidth))
		} else {
			availableForBars := m.viewWidth - labelWidth - 12
			barWidth := availableForBars / 3
			if barWidth < minBarWidth {
				barWidth = minBarWidth
			}
			if barWidth > 18 {
				barWidth = 18
			}

			weakPct := weak * barWidth / total
			if weakPct > barWidth {
				weakPct = barWidth
			}
			devPct := dev * barWidth / total
			if devPct > barWidth {
				devPct = barWidth
			}
			strongPct := barWidth - weakPct - devPct
			if strongPct < 0 {
				strongPct = 0
			}

			weakBar := dashDistWeakStyle.Render(strings.Repeat("█", weakPct) + strings.Repeat("░", barWidth-weakPct))
			devBar := dashDistDevStyle.Render(strings.Repeat("█", devPct) + strings.Repeat("░", barWidth-devPct))
			strongBar := dashDistStrongStyle.Render(strings.Repeat("█", strongPct) + strings.Repeat("░", barWidth-strongPct))

			distLine := fmt.Sprintf("Weak %s %d  ·  Developing %s %d  ·  Strong %s %d  ·  %d concepts",
				weakBar, weak, devBar, dev, strongBar, strong, total)
			b.WriteString("\n")
			b.WriteString(centerBlock(dashStatsRowStyle.Render(distLine), m.viewWidth))
		}
	}

	return b.String()
}

func (m DashboardModel) renderGrid(filtered []discovery.SubjectInfo) string {
	cols := m.columnCount()
	colWidth := m.viewWidth / cols
	if colWidth < minCardWidth {
		colWidth = minCardWidth
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
			selected := (i + j) == m.selected
			rendered = append(rendered, m.renderSubjectCard(s, selected, colWidth))
		}

		if len(rendered) > 1 {
			var withGaps []string
			for k, card := range rendered {
				withGaps = append(withGaps, card)
				if k < len(rendered)-1 {
					withGaps = append(withGaps, " ")
				}
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, withGaps...))
		} else {
			rows = append(rows, rendered[0])
		}
	}

	totalRows := (len(filtered) + cols - 1) / cols
	if totalRows > visible {
		scrollInfo := fmt.Sprintf("  %d-%d of %d subjects", startIdx+1, endIdx, len(filtered))
		rows = append(rows, dashDetailStyle.Render(scrollInfo))
	}

	grid := strings.Join(rows, "\n")
	return centerBlock(grid, m.viewWidth)
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

	innerWidth := colWidth - 6
	if innerWidth < 10 {
		innerWidth = 10
	}

	name := s.Name
	if selected {
		name = dashSubjectSelectedStyle.Render(truncate(s.Name, innerWidth-12))
	} else {
		name = dashSubjectStyle.Render(truncate(s.Name, innerWidth-12))
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
	line2 := fmt.Sprintf("  %s  %dc·%df·%dq",
		confStyle.Render(fmt.Sprintf("%.0f%%", avg)),
		conceptCount,
		m.cardCounts[s.Name],
		m.quizCounts[s.Name])

	var lines []string
	lines = append(lines, line1)
	lines = append(lines, line2)
	lines = append(lines, "  "+dashDetailStyle.Render(fmt.Sprintf("%d notes", noteCount)))

	content := strings.Join(lines, "\n")

	style := dashCardStyle.
		Width(colWidth - 4).
		Padding(0, 1).
		MarginBottom(0)
	if selected {
		style = dashCardSelectedStyle.
			Width(colWidth - 4).
			Padding(0, 1).
			MarginBottom(0)
	}

	return style.Render(content)
}

func (m DashboardModel) renderEmptyState() string {
	boxWidth := 56
	leftPad := (m.viewWidth - boxWidth) / 2
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
				dashHintStyle.Render("Each subject needs:")+"\n"+
				"  concepts/  flashcards/  quizzes/\n\n"+
				dashDetailStyle.Render("See docs for format details."),
		)

	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(strings.Repeat(" ", leftPad))
	b.WriteString(box)
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

type activityEntry struct {
	subject     string
	conceptName string
	confidence  float64
	updatedAt   *int64
}

func formatRelativeTime(ts int64) string {
	now := time.Now().Unix()
	diff := now - ts
	switch {
	case diff < 60:
		return "just now"
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%dh ago", diff/3600)
	case diff < 172800:
		return "yesterday"
	default:
		return fmt.Sprintf("%dd ago", diff/86400)
	}
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
	return formatRelativeTime(*latest)
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
	return centerBlock(row, m.viewWidth)
}

func (m DashboardModel) recentlyStudied() []activityEntry {
	type candidate struct {
		entry activityEntry
		ts    int64
	}
	var candidates []candidate
	for _, s := range m.subjects {
		prog := m.progData[s.Name]
		concepts := m.conceptData[s.Name]
		if prog == nil {
			continue
		}
		for _, c := range concepts {
			cs, ok := prog.Concepts[c.ID]
			if !ok || cs.LastReviewedAt == nil {
				continue
			}
			candidates = append(candidates, candidate{
				entry: activityEntry{
					subject:     s.Name,
					conceptName: c.Name,
					confidence:  cs.Confidence,
					updatedAt:   cs.LastReviewedAt,
				},
				ts: *cs.LastReviewedAt,
			})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].ts > candidates[j].ts
	})
	var result []activityEntry
	for _, c := range candidates {
		result = append(result, c.entry)
	}
	return result
}

func (m DashboardModel) comingUp() []activityEntry {
	type candidate struct {
		entry activityEntry
		ts    int64
	}
	var reviewed []candidate
	var never []activityEntry
	now := time.Now().Unix()
	for _, s := range m.subjects {
		prog := m.progData[s.Name]
		concepts := m.conceptData[s.Name]
		if prog == nil {
			continue
		}
		for _, c := range concepts {
			cs, ok := prog.Concepts[c.ID]
			if !ok {
				continue
			}
			if cs.Confidence >= 0.7 {
				continue
			}
			entry := activityEntry{
				subject:     s.Name,
				conceptName: c.Name,
				confidence:  cs.Confidence,
				updatedAt:   cs.LastReviewedAt,
			}
			if cs.LastReviewedAt == nil {
				never = append(never, entry)
			} else if (now - *cs.LastReviewedAt) > 86400 {
				reviewed = append(reviewed, candidate{entry: entry, ts: *cs.LastReviewedAt})
			}
		}
	}
	sort.Slice(reviewed, func(i, j int) bool {
		return reviewed[i].ts < reviewed[j].ts
	})
	var result []activityEntry
	result = append(result, never...)
	for _, r := range reviewed {
		result = append(result, r.entry)
	}
	return result
}

func (m DashboardModel) renderActivityPanel(maxRows int) string {
	if maxRows < 3 {
		return ""
	}

	recent := m.recentlyStudied()
	upcoming := m.comingUp()

	sepWidth := m.viewWidth - 4
	if sepWidth < 20 {
		sepWidth = 20
	}
	separator := dashSeparatorStyle.Render(strings.Repeat("─", sepWidth))

	sideBySide := m.viewWidth >= minCardWidth*2+4
	var panel string
	if sideBySide {
		panel = m.renderActivitySideBySide(recent, upcoming, maxRows)
	} else {
		panel = m.renderActivityStacked(recent, upcoming, maxRows)
	}

	return "\n" + separator + "\n" + panel
}

func (m DashboardModel) renderActivitySideBySide(recent, upcoming []activityEntry, maxRows int) string {
	halfW := (m.viewWidth - 4) / 2
	innerW := halfW - 2
	if innerW < 12 {
		innerW = 12
	}

	available := maxRows - 1
	if available < 1 {
		available = 1
	}

	var leftLines []string
	leftLines = append(leftLines, dashPanelHeaderStyle.Render("recently studied"))
	if len(recent) == 0 {
		leftLines = append(leftLines, dashPanelEmptyStyle.Render("nothing studied yet"))
	} else {
		for i, e := range recent {
			if i >= available {
				leftLines = append(leftLines, dashPanelTimeStyle.Render(fmt.Sprintf("+%d more", len(recent)-available)))
				break
			}
			overhead := len(e.subject) + 2 + 1
			maxName := innerW - overhead
			if maxName < 8 {
				maxName = 8
			}
			name := truncate(e.conceptName, maxName)
			timeStr := ""
			if e.updatedAt != nil {
				timeStr = " " + dashPanelTimeStyle.Render(formatRelativeTime(*e.updatedAt))
			}
			leftLines = append(leftLines, dashPanelSubjectStyle.Render(e.subject)+"  "+dashPanelItemStyle.Render(name)+timeStr)
		}
	}

	var rightLines []string
	rightLines = append(rightLines, dashPanelHeaderStyle.Render("coming up"))
	if len(upcoming) == 0 {
		rightLines = append(rightLines, dashPanelEmptyStyle.Render("all caught up"))
	} else {
		for i, e := range upcoming {
			if i >= available {
				rightLines = append(rightLines, dashPanelTimeStyle.Render(fmt.Sprintf("+%d more", len(upcoming)-available)))
				break
			}
			overhead := len(e.subject) + 2 + 5
			maxName := innerW - overhead
			if maxName < 8 {
				maxName = 8
			}
			name := truncate(e.conceptName, maxName)
			conf := fmt.Sprintf("%.0f%%", e.confidence*100)
			confStyle := dashDistWeakStyle
			if e.confidence >= 0.3 {
				confStyle = dashDistDevStyle
			}
			rightLines = append(rightLines, dashPanelSubjectStyle.Render(e.subject)+"  "+dashPanelItemStyle.Render(name)+"  "+confStyle.Render(conf))
		}
	}

	maxH := len(leftLines)
	if len(rightLines) > maxH {
		maxH = len(rightLines)
	}
	for len(leftLines) < maxH {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxH {
		rightLines = append(rightLines, "")
	}

	left := strings.Join(leftLines, "\n")
	right := strings.Join(rightLines, "\n")
	result := lipgloss.JoinHorizontal(lipgloss.Top, left, "    ", right)
	return centerBlock(result, m.viewWidth)
}

func (m DashboardModel) renderActivityStacked(recent, upcoming []activityEntry, maxRows int) string {
	innerW := m.viewWidth - 4
	if innerW < 12 {
		innerW = 12
	}

	available := maxRows - 1
	if available < 1 {
		available = 1
	}

	var sections []string

	recentRowsShown := 0
	sections = append(sections, dashPanelHeaderStyle.Render("recently studied"))
	if len(recent) == 0 {
		sections = append(sections, dashPanelEmptyStyle.Render("nothing studied yet"))
	} else {
		for _, e := range recent {
			if recentRowsShown >= available {
				sections = append(sections, dashPanelTimeStyle.Render(fmt.Sprintf("+%d more", len(recent)-available)))
				break
			}
			overhead := len(e.subject) + 2 + 1
			maxName := innerW - overhead
			if maxName < 8 {
				maxName = 8
			}
			name := truncate(e.conceptName, maxName)
			timeStr := ""
			if e.updatedAt != nil {
				timeStr = " " + dashPanelTimeStyle.Render(formatRelativeTime(*e.updatedAt))
			}
			sections = append(sections, dashPanelSubjectStyle.Render(e.subject)+"  "+dashPanelItemStyle.Render(name)+timeStr)
			recentRowsShown++
		}
	}

	remaining := available - recentRowsShown
	if remaining < 1 {
		remaining = 1
	}

	sections = append(sections, dashPanelHeaderStyle.Render("coming up"))
	if len(upcoming) == 0 {
		sections = append(sections, dashPanelEmptyStyle.Render("all caught up"))
	} else {
		shown := 0
		for _, e := range upcoming {
			if shown >= remaining {
				sections = append(sections, dashPanelTimeStyle.Render(fmt.Sprintf("+%d more", len(upcoming)-remaining)))
				break
			}
			overhead := len(e.subject) + 2 + 5
			maxName := innerW - overhead
			if maxName < 8 {
				maxName = 8
			}
			name := truncate(e.conceptName, maxName)
			conf := fmt.Sprintf("%.0f%%", e.confidence*100)
			confStyle := dashDistWeakStyle
			if e.confidence >= 0.3 {
				confStyle = dashDistDevStyle
			}
			sections = append(sections, dashPanelSubjectStyle.Render(e.subject)+"  "+dashPanelItemStyle.Render(name)+"  "+confStyle.Render(conf))
			shown++
		}
	}

	result := strings.Join(sections, "\n")
	return centerBlock(result, m.viewWidth)
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
	text = strings.TrimRight(text, "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 {
		return ""
	}
	maxW := 0
	for _, line := range lines {
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
	for i, line := range lines {
		lines[i] = pad + line
	}
	return strings.Join(lines, "\n")
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return strings.Repeat(".", maxLen)
	}
	return s[:maxLen-3] + "..."
}

package main

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/reflow/ansi"
	"golang.org/x/term"
)

const builtinPagerStatusBarHeight = 1

var (
	builtinPagerStatusBarBg = lipgloss.AdaptiveColor{Light: "#E6E6E6", Dark: "#242424"}

	builtinPagerScrollPosStyle = lipgloss.NewStyle().
					Foreground(lipgloss.AdaptiveColor{Light: "#949494", Dark: "#5A5A5A"}).
					Background(builtinPagerStatusBarBg).
					Render

	builtinPagerNoteStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}).
				Background(builtinPagerStatusBarBg).
				Render

	builtinPagerHelpStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}).
				Background(lipgloss.AdaptiveColor{Light: "#DCDCDC", Dark: "#323232"}).
				Render
)

type builtinPagerModel struct {
	viewport    viewport.Model
	rawViewport viewport.Model
	content     string
	rawMarkdown string
	fileName    string
	quitting    bool
	splitMode   bool
	width       int
	height      int
}

func newBuiltinPagerModel(content, rawMarkdown, fileName string, w, h int) builtinPagerModel {
	vp := viewport.New(w, h-builtinPagerStatusBarHeight)
	vp.SetContent(content)
	vp.HighPerformanceRendering = false

	rawVp := viewport.New(0, 0)
	rawVp.YPosition = 0
	rawVp.HighPerformanceRendering = false

	return builtinPagerModel{
		viewport:    vp,
		rawViewport: rawVp,
		content:     content,
		rawMarkdown: rawMarkdown,
		fileName:    fileName,
		width:       w,
		height:      h,
	}
}

func (m builtinPagerModel) Init() tea.Cmd {
	return nil
}

func (m builtinPagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc":
			m.quitting = true
			return m, tea.Quit
		case "s":
			if m.width < 80 {
				break
			}
			m.splitMode = !m.splitMode
			m.setSize(m.width, m.height)
			if m.splitMode {
				m.rawViewport.SetContent(formatRawMarkdown(m.rawMarkdown, m.rawViewport.Width))
				m.rawViewport.GotoTop()
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.setSize(msg.Width, msg.Height)
		if m.splitMode {
			m.rawViewport.SetContent(formatRawMarkdown(m.rawMarkdown, m.rawViewport.Width))
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)

	// Sync scroll in split mode
	if m.splitMode {
		syncScroll(&m.viewport, &m.rawViewport)
	}

	return m, cmd
}

func (m *builtinPagerModel) setSize(w, h int) {
	viewHeight := h - builtinPagerStatusBarHeight
	if m.splitMode {
		panelWidth := (w - 1) / 2
		m.rawViewport.Width = panelWidth
		m.rawViewport.Height = viewHeight
		m.viewport.Width = panelWidth
		m.viewport.Height = viewHeight
	} else {
		m.viewport.Width = w
		m.viewport.Height = viewHeight
	}
}

func (m builtinPagerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	if m.splitMode {
		left := m.rawViewport.View()
		right := m.viewport.View()
		divStyle := lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
			Render
		divider := divStyle("│")
		body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
		fmt.Fprint(&b, body+"\n")
	} else {
		fmt.Fprint(&b, m.viewport.View())
		fmt.Fprint(&b, "\n")
	}
	m.statusBarView(&b)
	return b.String()
}

func (m builtinPagerModel) statusBarView(b *strings.Builder) {
	percent := math.Max(0.0, math.Min(1.0, m.viewport.ScrollPercent()))
	scrollPercent := fmt.Sprintf(" %3.f%% ", percent*100)
	scrollPercent = builtinPagerScrollPosStyle(scrollPercent)

	helpNote := builtinPagerHelpStyle(" q quit  s split ")

	note := " " + m.fileName + " "
	if m.splitMode {
		note = " [SPLIT] " + m.fileName + " "
	}
	maxNoteWidth := m.width - ansi.PrintableRuneWidth(scrollPercent) - ansi.PrintableRuneWidth(helpNote)
	if maxNoteWidth > 0 && runewidth.StringWidth(note) > maxNoteWidth {
		note = runewidth.Truncate(note, maxNoteWidth, "…")
	}
	note = builtinPagerNoteStyle(note)

	padding := m.width - ansi.PrintableRuneWidth(note) - ansi.PrintableRuneWidth(scrollPercent) - ansi.PrintableRuneWidth(helpNote)
	emptySpace := strings.Repeat(" ", max(0, padding))
	emptySpace = builtinPagerNoteStyle(emptySpace)

	fmt.Fprintf(b, "%s%s%s%s", note, emptySpace, scrollPercent, helpNote)
}

// countLines counts the number of newlines in a string.
func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

// runBuiltinPager starts a Bubble Tea program that displays content in a
// scrollable viewport with vim-style keybindings.
func runBuiltinPager(content, rawMarkdown, fileName string) error {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = 80
		h = 24
	}

	p := tea.NewProgram(
		newBuiltinPagerModel(content, rawMarkdown, fileName, w, h),
		tea.WithAltScreen(),
	)
	_, err = p.Run()
	return err
}

const builtinRawLineNumWidth = 4

// formatRawMarkdown formats raw markdown text with line numbers for the
// split view's left panel.
func formatRawMarkdown(body string, maxWidth int) string {
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	lineNumFg := lipgloss.AdaptiveColor{Light: "#656565", Dark: "#7D7D7D"}
	lineNumStyle := lipgloss.NewStyle().Foreground(lineNumFg).Render

	var b strings.Builder
	for i, line := range lines {
		num := lineNumStyle(fmt.Sprintf("%"+fmt.Sprint(builtinRawLineNumWidth)+"d", i+1))
		b.WriteString(num)
		if maxWidth > 0 {
			trunc := lipgloss.NewStyle().MaxWidth(maxWidth - builtinRawLineNumWidth).Render
			b.WriteString(trunc(line))
		} else {
			b.WriteString(line)
		}
		if i+1 < len(lines) {
			b.WriteRune('\n')
		}
	}
	return b.String()
}

// syncScroll maps the scroll percentage from source viewport to target viewport.
func syncScroll(source, target *viewport.Model) {
	if target.TotalLineCount() <= target.Height {
		target.GotoTop()
		return
	}
	percent := source.ScrollPercent()
	yOffset := int(percent * float64(target.TotalLineCount()-target.Height))
	target.SetYOffset(yOffset)
}

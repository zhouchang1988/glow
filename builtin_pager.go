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
	viewport  viewport.Model
	content   string
	fileName  string
	quitting  bool
	width     int
	height    int
}

func newBuiltinPagerModel(content, fileName string, w, h int) builtinPagerModel {
	vp := viewport.New(w, h-builtinPagerStatusBarHeight)
	vp.SetContent(content)
	vp.HighPerformanceRendering = false

	return builtinPagerModel{
		viewport: vp,
		content:  content,
		fileName: fileName,
		width:    w,
		height:   h,
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - builtinPagerStatusBarHeight
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m builtinPagerModel) View() string {
	if m.quitting {
		return ""
	}

	var b strings.Builder
	fmt.Fprint(&b, m.viewport.View())
	fmt.Fprint(&b, "\n")
	m.statusBarView(&b)
	return b.String()
}

func (m builtinPagerModel) statusBarView(b *strings.Builder) {
	percent := math.Max(0.0, math.Min(1.0, m.viewport.ScrollPercent()))
	scrollPercent := fmt.Sprintf(" %3.f%% ", percent*100)
	scrollPercent = builtinPagerScrollPosStyle(scrollPercent)

	helpNote := builtinPagerHelpStyle(" q quit ")

	note := " " + m.fileName + " "
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
func runBuiltinPager(content, fileName string) error {
	w, h, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		w = 80
		h = 24
	}

	p := tea.NewProgram(
		newBuiltinPagerModel(content, fileName, w, h),
		tea.WithAltScreen(),
	)
	_, err = p.Run()
	return err
}

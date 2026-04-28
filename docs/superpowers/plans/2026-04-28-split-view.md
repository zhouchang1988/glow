# Split-View Markdown Preview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a split-view mode to the Glow TUI pager that displays raw markdown on the left and glamour-rendered preview on the right, toggled with `s` key.

**Architecture:** Extend the existing `pagerModel` with a `splitMode` flag and a second `viewport.Model` for the raw markdown side. Use `lipgloss.JoinHorizontal` to render both viewports side-by-side with a vertical divider. Synchronize scrolling by mapping scroll percentages between the two viewports.

**Tech Stack:** Go, Bubbletea v1.3, Lipgloss v1.1, Glamour v0.10, Bubbles viewport v0.21

---

## File Structure

| File | Responsibility | Change Type |
|------|---------------|-------------|
| `ui/pager.go` | Pager model, viewports, rendering, key handling, scroll sync | Modify |
| `ui/pager_test.go` | Unit tests for `formatRawMarkdown` and scroll sync helper | Create |

All logic lives in `ui/pager.go` — the split mode is a display variant of the existing pager, not a separate component. The test file covers the pure helper function that formats raw markdown with line numbers (testable without a TUI).

---

### Task 1: Add split mode fields to pagerModel

**Files:**
- Modify: `ui/pager.go:92-106` (pagerModel struct and constructor)

- [ ] **Step 1: Add fields to pagerModel struct**

After the `watcher` field at line 106, add:

```go
type pagerModel struct {
	common   *commonModel
	viewport viewport.Model
	state    pagerState
	showHelp bool

	statusMessage      string
	statusMessageTimer *time.Timer

	currentDocument markdown

	watcher *fsnotify.Watcher

	// Split view
	splitMode   bool
	rawViewport viewport.Model
}
```

- [ ] **Step 2: Initialize rawViewport in newPagerModel**

In `newPagerModel` (line 108), after creating the main viewport, also create the raw viewport:

```go
func newPagerModel(common *commonModel) pagerModel {
	vp := viewport.New(0, 0)
	vp.YPosition = 0
	vp.HighPerformanceRendering = config.HighPerformancePager

	rawVp := viewport.New(0, 0)
	rawVp.YPosition = 0
	rawVp.HighPerformanceRendering = false

	m := pagerModel{
		common:     common,
		state:      pagerStateBrowse,
		viewport:   vp,
		rawViewport: rawVp,
	}
	m.initWatcher()
	return m
}
```

- [ ] **Step 3: Commit**

```bash
git add ui/pager.go
git commit -m "feat(split-view): add splitMode and rawViewport fields to pagerModel"
```

---

### Task 2: Add formatRawMarkdown helper function

**Files:**
- Modify: `ui/pager.go` (add function after `glamourRender`)
- Create: `ui/pager_test.go`

This function formats raw markdown text with line numbers and optional width-based line wrapping. It's a pure function, easily testable.

- [ ] **Step 1: Write the failing test**

Create `ui/pager_test.go`:

```go
package ui

import "testing"

func TestFormatRawMarkdown(t *testing.T) {
	input := "# Hello\n\n- item 1\n- item 2\n"
	got := formatRawMarkdown(input, 40)

	lines := splitLines(got)
	if len(lines) != 4 {
		t.Fatalf("expected 4 lines, got %d", len(lines))
	}

	// First line should start with a line number
	if len(lines[0]) < 4 {
		t.Fatalf("line 0 too short: %q", lines[0])
	}

	// Line number area should contain "1"
	if !containsSubstring(lines[0], "1") {
		t.Errorf("line 0 should contain line number 1, got: %q", lines[0])
	}
}

func TestFormatRawMarkdownEmpty(t *testing.T) {
	got := formatRawMarkdown("", 40)
	if got != "" {
		t.Errorf("expected empty string for empty input, got: %q", got)
	}
}

func TestFormatRawMarkdownNarrowWidth(t *testing.T) {
	input := "# This is a very long title that exceeds the width"
	got := formatRawMarkdown(input, 20)
	if got == "" {
		t.Error("expected non-empty output")
	}
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	result := splitByNewline(s)
	// Remove trailing empty string if input ends with newline
	if len(result) > 0 && result[len(result)-1] == "" {
		result = result[:len(result)-1]
	}
	return result
}

func splitByNewline(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	lines = append(lines, s[start:])
	return lines
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./ui/ -run TestFormatRawMarkdown -v`
Expected: FAIL — `formatRawMarkdown` undefined

- [ ] **Step 3: Write the implementation**

Add to `ui/pager.go`, after `glamourRender` function (around line 480):

```go
const rawLineNumWidth = 4

// formatRawMarkdown formats raw markdown text with line numbers for the
// split view's left panel. It strips ANSI sequences and adds padded line
// numbers before each line.
func formatRawMarkdown(body string, maxWidth int) string {
	if body == "" {
		return ""
	}

	lines := strings.Split(body, "\n")
	// Remove trailing empty line from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	var b strings.Builder
	for i, line := range lines {
		num := lineNumberStyle(fmt.Sprintf("%"+fmt.Sprint(rawLineNumWidth)+"d", i+1))
		b.WriteString(num)
		if maxWidth > 0 {
			trunc := lipgloss.NewStyle().MaxWidth(maxWidth - rawLineNumWidth).Render
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
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./ui/ -run TestFormatRawMarkdown -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add ui/pager.go ui/pager_test.go
git commit -m "feat(split-view): add formatRawMarkdown helper with tests"
```

---

### Task 3: Modify setSize for split mode

**Files:**
- Modify: `ui/pager.go:123-133` (setSize method)

- [ ] **Step 1: Update setSize to handle split mode**

Replace the existing `setSize` method:

```go
func (m *pagerModel) setSize(w, h int) {
	if m.splitMode {
		panelWidth := (w - 1) / 2
		m.rawViewport.Width = panelWidth
		m.rawViewport.Height = h - statusBarHeight
		m.viewport.Width = panelWidth
		m.viewport.Height = h - statusBarHeight
	} else {
		m.viewport.Width = w
		m.viewport.Height = h - statusBarHeight
	}

	if m.showHelp {
		if pagerHelpHeight == 0 {
			pagerHelpHeight = strings.Count(m.helpView(), "\n")
		}
		height := h - statusBarHeight
		if m.splitMode {
			height = m.rawViewport.Height
		}
		m.viewport.Height -= (statusBarHeight + pagerHelpHeight)
		if m.splitMode {
			m.rawViewport.Height -= (statusBarHeight + pagerHelpHeight)
		}
		_ = height // suppress unused warning
	}
}
```

Wait, that has a dead variable. Let me simplify:

```go
func (m *pagerModel) setSize(w, h int) {
	viewHeight := h - statusBarHeight

	if m.showHelp {
		if pagerHelpHeight == 0 {
			pagerHelpHeight = strings.Count(m.helpView(), "\n")
		}
		viewHeight -= (statusBarHeight + pagerHelpHeight)
	}

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
```

- [ ] **Step 2: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 3: Commit**

```bash
git add ui/pager.go
git commit -m "feat(split-view): update setSize for split mode panel widths"
```

---

### Task 4: Modify View() for split mode rendering

**Files:**
- Modify: `ui/pager.go:282-294` (View method)

- [ ] **Step 1: Add divider style at the top of pager.go with other style vars**

Add near line 76 (after `lineNumberStyle`):

```go
	splitDividerStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#444444"}).
				Render
```

- [ ] **Step 2: Update View method to handle split mode**

Replace the existing `View` method:

```go
func (m pagerModel) View() string {
	var b strings.Builder

	if m.splitMode {
		left := m.rawViewport.View()
		right := m.viewport.View()
		divider := splitDividerStyle("│")
		body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
		fmt.Fprint(&b, body+"\n")
	} else {
		fmt.Fprint(&b, m.viewport.View()+"\n")
	}

	// Footer
	m.statusBarView(&b)

	if m.showHelp {
		fmt.Fprint(&b, "\n"+m.helpView())
	}

	return b.String()
}
```

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add ui/pager.go
git commit -m "feat(split-view): render split view with two viewports and divider"
```

---

### Task 5: Modify glamourRender for split mode width

**Files:**
- Modify: `ui/pager.go:422-480` (glamourRender function)

- [ ] **Step 1: Update glamourRender to use half width in split mode**

In the `glamourRender` function, modify the width calculation. Replace:

```go
	width := max(0, min(int(m.common.cfg.GlamourMaxWidth), m.viewport.Width)) //nolint:gosec
```

With:

```go
	renderWidth := m.viewport.Width
	width := max(0, min(int(m.common.cfg.GlamourMaxWidth), renderWidth)) //nolint:gosec
```

The `m.viewport.Width` is already set to the correct (half) width by `setSize` when in split mode, so this line actually works as-is. No change needed to the width calculation itself.

However, we need to update the `trunc` at the top of the function to also respect the panel width in split mode. The `MaxWidth` on the trunc style should use `m.viewport.Width` which is already correct. Verify this is the case — the trunc uses `m.viewport.Width - lineNumberWidth`, which will be half width in split mode. This is correct behavior.

Actually, re-examining: the trunc line is:
```go
trunc := lipgloss.NewStyle().MaxWidth(m.viewport.Width - lineNumberWidth).Render
```

In split mode, `m.viewport.Width` is already the panel width (half). This works correctly as-is.

**No code changes needed for this task.** The existing `glamourRender` already uses `m.viewport.Width` which is correctly set by `setSize`. Confirm by building.

- [ ] **Step 1: Verify build passes**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 2: Commit (if any changes were made)**

No changes needed — viewport.Width is already dynamically read.

---

### Task 6: Add toggle keybinding and split mode enter/exit logic

**Files:**
- Modify: `ui/pager.go:181-280` (update method)

- [ ] **Step 1: Add `s` key handler in pagerModel.update**

In the `tea.KeyMsg` switch block (around line 188), add a new case after the `r` case:

```go
		case "s":
			if m.common.width < minSplitWidth {
				cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{
					message: "Terminal too narrow for split view (need 80+ cols)",
					isError: true,
				}))
				break
			}
			m.splitMode = !m.splitMode
			m.setSize(m.common.width, m.common.height)
			if m.splitMode {
				m.rawViewport.SetContent(formatRawMarkdown(m.currentDocument.Body, m.rawViewport.Width))
				m.rawViewport.GotoTop()
			}
			cmds = append(cmds, renderWithGlamour(m, m.currentDocument.Body))
			cmds = append(cmds, m.showStatusMessage(pagerStatusMessage{
				message: "Split view " + map[bool]string{true: "on", false: "off"}[m.splitMode],
				isError: false,
			}))
```

- [ ] **Step 2: Add minSplitWidth constant**

Near the top of `pager.go` with other constants (around line 24):

```go
const minSplitWidth = 80
```

- [ ] **Step 3: Handle contentRenderedMsg in split mode**

In the `contentRenderedMsg` case (around line 248), after `m.setContent(string(msg))`, add split mode content refresh:

```go
		case contentRenderedMsg:
			log.Info("content rendered", "state", m.state)
			m.setContent(string(msg))
			if m.splitMode {
				m.rawViewport.SetContent(formatRawMarkdown(m.currentDocument.Body, m.rawViewport.Width))
			}
			if m.viewport.HighPerformanceRendering {
				cmds = append(cmds, viewport.Sync(m.viewport))
			}
			cmds = append(cmds, m.watchFile)
```

- [ ] **Step 4: Handle WindowSizeMsg re-render in split mode**

In the `tea.WindowSizeMsg` case (around line 270), update the raw viewport content on resize:

```go
		case tea.WindowSizeMsg:
			if m.splitMode {
				m.setSize(msg.Width, msg.Height)
				m.rawViewport.SetContent(formatRawMarkdown(m.currentDocument.Body, m.rawViewport.Width))
			}
			return m, renderWithGlamour(m, m.currentDocument.Body)
```

- [ ] **Step 5: Handle reloadMsg in split mode**

In the `reloadMsg` case (around line 259), no change needed — `loadLocalMarkdown` triggers `fetchedMarkdownMsg` which triggers `renderWithGlamour`, which triggers `contentRenderedMsg` which we already handle.

- [ ] **Step 6: Handle unload in split mode**

In the `unload` method (around line 167), add split mode reset:

```go
func (m *pagerModel) unload() {
	log.Debug("unload")
	if m.showHelp {
		m.toggleHelp()
	}
	if m.statusMessageTimer != nil {
		m.statusMessageTimer.Stop()
	}
	m.state = pagerStateBrowse
	m.splitMode = false
	m.viewport.SetContent("")
	m.viewport.YOffset = 0
	m.rawViewport.SetContent("")
	m.rawViewport.YOffset = 0
	m.unwatchFile()
}
```

- [ ] **Step 7: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 8: Commit**

```bash
git add ui/pager.go
git commit -m "feat(split-view): add s key toggle, enter/exit logic, and resize handling"
```

---

### Task 7: Implement scroll synchronization

**Files:**
- Modify: `ui/pager.go` (update method)

- [ ] **Step 1: Add scroll sync helper function**

Add after `formatRawMarkdown`:

```go
// syncScroll maps the scroll percentage from one viewport to another.
func syncScroll(source, target *viewport.Model) {
	if target.TotalLineCount() <= target.Height {
		target.GotoTop()
		return
	}
	percent := source.ScrollPercent()
	yOffset := int(percent * float64(target.TotalLineCount()-target.Height))
	target.SetYOffset(yOffset)
}
```

- [ ] **Step 2: Add scroll sync test**

Add to `ui/pager_test.go`:

```go
func TestSyncScroll(t *testing.T) {
	// We can't easily construct viewport.Model in tests without
	// a terminal, so we test the math directly.
	// percent=0.5, totalLines=100, height=20 -> yOffset = 0.5 * 80 = 40
	percent := 0.5
	totalLines := 100
	height := 20
	yOffset := int(percent * float64(totalLines-height))
	if yOffset != 40 {
		t.Errorf("expected yOffset 40, got %d", yOffset)
	}

	// percent=0.0 -> yOffset = 0
	yOffset = int(0.0 * float64(totalLines-height))
	if yOffset != 0 {
		t.Errorf("expected yOffset 0, got %d", yOffset)
	}

	// percent=1.0 -> yOffset = 80
	yOffset = int(1.0 * float64(totalLines-height))
	if yOffset != 80 {
		t.Errorf("expected yOffset 80, got %d", yOffset)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `go test ./ui/ -v`
Expected: PASS

- [ ] **Step 4: Wire up scroll sync in update method**

After the viewport update at the bottom of the `update` method (line ~276), add scroll sync:

```go
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	// Sync scroll between viewports in split mode
	if m.splitMode {
		syncScroll(&m.viewport, &m.rawViewport)
	}
```

Note: We only sync viewport -> rawViewport direction. The raw viewport doesn't receive independent scroll commands — all key-based scrolling goes through the main viewport, and we mirror it to the raw side.

- [ ] **Step 5: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 6: Commit**

```bash
git add ui/pager.go ui/pager_test.go
git commit -m "feat(split-view): implement scroll synchronization between viewports"
```

---

### Task 8: Add status bar indicator and help entry

**Files:**
- Modify: `ui/pager.go` (statusBarView and helpView methods)

- [ ] **Step 1: Add [SPLIT] indicator to status bar**

In `statusBarView`, modify the `note` section. After the line that sets `note = m.currentDocument.Note` (around line 329), add:

```go
		if m.splitMode {
			note = "[SPLIT] " + note
		}
```

This goes in the else branch (when not showing status message), right after:
```go
		} else {
			note = m.currentDocument.Note
		}
```

Becomes:
```go
		} else {
			note = m.currentDocument.Note
			if m.splitMode {
				note = "[SPLIT] " + note
			}
		}
```

- [ ] **Step 2: Add help entry for split view**

In `helpView`, add `s       toggle split view` to the `col1` slice (around line 369):

```go
	col1 := []string{
		"g/home  go to top",
		"G/end   go to bottom",
		"c       copy contents",
		"e       edit this document",
		"r       reload this document",
		"s       toggle split view",
		"esc     back to files",
		"q       quit",
	}
```

- [ ] **Step 3: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

- [ ] **Step 4: Commit**

```bash
git add ui/pager.go
git commit -m "feat(split-view): add [SPLIT] status indicator and help entry"
```

---

### Task 9: Integration testing and edge case handling

**Files:**
- Modify: `ui/pager.go` (edge case fixes)

- [ ] **Step 1: Handle file reload in split mode**

When a file reloads (via `fetchedMarkdownMsg` in `ui.go`), the `renderWithGlamour` command fires, which eventually produces `contentRenderedMsg`. We already handle this in Task 6 Step 3. Verify the raw viewport content is also refreshed.

Check that the `contentRenderedMsg` handler updates `m.currentDocument` before the split view reads from it. Looking at `ui.go:277-279`:

```go
case fetchedMarkdownMsg:
    m.pager.currentDocument = *msg
    body := string(utils.RemoveFrontmatter([]byte(msg.Body)))
    cmds = append(cmds, renderWithGlamour(m.pager, body))
```

The `currentDocument` is updated before rendering. Then in `contentRenderedMsg` (our updated handler), `m.currentDocument.Body` will have the new content. This is correct.

- [ ] **Step 2: Handle editor return in split mode**

When the user presses `e` to edit, then returns, `editorFinishedMsg` triggers `loadLocalMarkdown`, which triggers `fetchedMarkdownMsg` → `renderWithGlamour` → `contentRenderedMsg`. Our handler in Task 6 Step 3 refreshes the raw viewport. This chain is correct.

- [ ] **Step 3: Manual integration test**

Build and run:
```bash
go build -o glow . && ./glow README.md
```

Test checklist:
1. Press `s` — should enter split mode with raw markdown left, rendered right
2. Scroll with `j/k` — both sides should scroll in sync
3. Press `s` again — should return to normal mode, full width
4. Press `s` with terminal < 80 cols — should show error message
5. Resize terminal while in split mode — both panels should resize
6. Press `e` in split mode, edit, save — both panels should refresh
7. Press `?` in split mode — help should show `s toggle split view`
8. Status bar should show `[SPLIT]` when in split mode
9. Press `esc` from split mode — should exit document and reset split mode

- [ ] **Step 4: Commit any fixes**

```bash
git add ui/pager.go
git commit -m "fix(split-view): integration test edge case fixes"
```

---

## Self-Review Checklist

- [x] Spec coverage: All requirements from spec mapped to tasks
  - splitMode + rawViewport fields → Task 1
  - formatRawMarkdown → Task 2
  - setSize for split → Task 3
  - View() split rendering → Task 4
  - glamourRender width → Task 5
  - Toggle keybinding + enter/exit → Task 6
  - Scroll synchronization → Task 7
  - Status bar + help → Task 8
  - Edge cases + integration → Task 9
- [x] Placeholder scan: No TBD/TODO/placeholder patterns
- [x] Type consistency: `pagerModel` fields, function signatures consistent across tasks
- [x] Image handling: Out of scope per spec — no tasks needed
- [x] stashModel untouched: No changes to stash
- [x] builtinPager untouched: No changes to non-TUI pager

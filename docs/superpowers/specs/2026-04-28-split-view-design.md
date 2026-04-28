# Split-View Markdown Preview Design

## Overview

Add a split-view mode to the Glow TUI pager that displays raw markdown on the left and glamour-rendered preview on the right, separated by a vertical divider. Toggled via `s` key, with synchronized scrolling by scroll percentage.

## Architecture

### Data Model Changes

Extend `pagerModel` in `ui/pager.go`:

```go
type pagerModel struct {
    // ... existing fields ...
    splitMode   bool              // whether split view is active
    rawViewport viewport.Model    // left panel: raw markdown
}
```

### Layout

```
┌─────────────────────┬─┬─────────────────────┐
│  1  # Title         │ │  # Title             │
│  2                  │ │                      │
│  3  - list item 1   │ │  • list item 1       │
│  4  - list item 2   │ │  • list item 2       │
│  5                  │ │                      │
│  6  > blockquote    │ │  ┃ blockquote         │
│  7                  │ │                      │
│  8  ```python       │ │  ┌──────────────┐    │
│  9  print("hi")     │ │  │ print("hi")  │    │
│  10 ```             │ │  └──────────────┘    │
│  Raw Markdown       ││  Rendered Preview     │
└─────────────────────┴─┴─────────────────────┘
```

- Left panel width: `(terminalWidth - 1) / 2`
- Divider: 1 column, `│` character
- Right panel width: `(terminalWidth - 1) / 2`
- Minimum terminal width for split mode: 80 columns

### Component Details

#### Left Panel (Raw Markdown)

- Displays `currentDocument.Body` with line numbers
- Uses `lineNumberStyle` (already exists) for line number formatting
- Line wraps at panel width

#### Right Panel (Rendered Preview)

- Displays glamour-rendered markdown (existing rendering pipeline)
- `wordWrap` width set to right panel width (half of normal)
- Re-rendered on mode toggle and window resize

#### Divider

- Vertical bar `│` rendered as a separate column
- Styled with a subtle color matching the existing status bar aesthetic

### Key Bindings

| Key | Action |
|-----|--------|
| `s` | Toggle split view on/off |

Added to `pagerModel.update()` key handling and help view.

### Scroll Synchronization

When either viewport scrolls, synchronize the other by percentage:

1. After updating the primary viewport, compute `viewport.ScrollPercent()`
2. Map that percentage to the other viewport: `otherViewport.SetYOffset(int(percent * float64(otherViewport.TotalLineCount() - otherViewport.Height)))`
3. This is approximate (raw vs rendered line counts differ) but sufficient for side-by-side reading

### State Transitions

**Enter split mode (`s` pressed):**
1. Set `splitMode = true`
2. Call `setSize()` to recompute viewport dimensions
3. Populate `rawViewport` with raw markdown + line numbers
4. Trigger `renderWithGlamour` at the new (half) width

**Exit split mode (`s` pressed again):**
1. Set `splitMode = false`
2. Call `setSize()` to restore full-width viewport
3. Trigger `renderWithGlamour` at full width

### View Rendering

**Normal mode:** Unchanged from current behavior.

**Split mode:**
```go
func (m pagerModel) View() string {
    if m.splitMode {
        left := m.rawViewport.View()
        right := m.viewport.View()
        divider := lipgloss.NewStyle().Foreground(...).Render("│")
        body := lipgloss.JoinHorizontal(lipgloss.Top, left, divider, right)
        // ... status bar and help as usual
    }
    // ... normal mode
}
```

### `setSize` Changes

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
    // ... help height adjustment
}
```

### `glamourRender` Changes

When `splitMode` is true, set `wordWrap` width to the right panel width instead of the full viewport width.

### Status Bar

Add a split-mode indicator next to the filename in the status bar, e.g., `[SPLIT]`.

### Help View

Add entry: `s  toggle split view`

### Guards

- If terminal width < 80 columns when `s` is pressed, show a status message: "Terminal too narrow for split view" and do not enter split mode
- Split mode only applies in `stateShowDocument` (pager context), not in stash view

### Files Modified

| File | Change |
|------|--------|
| `ui/pager.go` | Add `splitMode`, `rawViewport`, modify `setSize`, `View`, `update`, `glamourRender` |
| `ui/config.go` | No changes needed |

### Image Handling

Explicitly out of scope for this iteration. Images in markdown will display as the alt text or link on both sides.

### What Stays the Same

- `stashModel` — no changes
- `builtinPager` (non-TUI mode) — no changes
- Existing keybindings — all preserved
- `fsnotify` file watcher — continues to work, re-renders both panels on file change
- Editor integration (`e` key) — still opens the file, on return both panels refresh

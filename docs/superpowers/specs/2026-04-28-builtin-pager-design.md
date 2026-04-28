# Built-in Pager for CLI Mode

## Problem

When running `glow file.md`, rendered markdown is dumped to stdout. If the content exceeds the terminal height, it scrolls past and the user must scroll back. The `--pager` flag exists but delegates to an external program (`less`), which doesn't provide a consistent vim-style experience.

## Solution

Auto-detect when rendered content exceeds terminal height and activate a built-in pager with minimal vim keybindings.

## Design

### New File: `pager.go`

A lightweight Bubble Tea model using `bubbles/viewport` for scrolling:

```go
type builtinPagerModel struct {
    viewport  viewport.Model
    content   string
    quitting  bool
    fileName  string
}
```

**Key bindings (minimal vim-style):**
- `j` / `↓` — scroll down one line
- `k` / `↑` — scroll up one line
- `q` / `ESC` — exit pager

**Status bar:** Bottom line showing file name and scroll percentage, styled consistently with the TUI pager.

### Modified: `executeCLI` in `main.go`

After glamour rendering, before displaying:

1. Count lines in the rendered output
2. Get terminal height via `term.GetSize()`
3. If `lines <= terminalHeight`: print directly to stdout (existing behavior unchanged)
4. If `lines > terminalHeight`: call `runBuiltinPager(out, fileName)` to start the Bubble Tea program

### Edge Cases

- **Piped output / non-TTY**: Skip pager entirely, print to stdout as today
- **Content fits on screen**: No pager, direct output
- `--pager` flag still works as before (external pager)
- Terminal resize handled by Bubble Tea's `WindowSizeMsg`

## Dependencies

No new dependencies. Uses existing `bubbletea` and `bubbles/viewport` packages.

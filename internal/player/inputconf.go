package player

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"tvremote/internal/config"
)

// mpv's built-in key bindings are compiled into the binary (input/input.c's
// builtin_conf); there is no file to edit, so --input-conf is the only way to
// add or override one. TinyPlay spawns mpv itself, so it hands that process a
// generated input.conf instead of touching anything the user's own mpv uses.
//
// The generated file is [the user's input.conf, if any] + [TinyPlay's block],
// in that order: mpv applies the last binding for a key, so TinyPlay's lines
// win on conflicts while every other custom binding the user set keeps
// working.
const mpvInputConfName = "mpv-input.conf"

// tinyplayInputBindings is TinyPlay's own block.
//
// Every window-size step leaves fullscreen first. TinyPlay starts mpv with
// --fullscreen, and set window-scale cannot resize a fullscreen window — it
// only scales the picture inside it, which is exactly what made the menu's own
// Window scale rows look broken. Dropping out of fullscreen first is what makes
// these steps resize the window.
//
// That costs the menu's shortcut hints on those rows: the menu resolves a row's
// shortcut by matching the row's command against mpv's input-bindings, and it
// compares for equality, so a command carrying this prefix no longer matches and
// the row ends up showing nothing. mpv offers no way to keep both without
// replacing the whole menu definition, and the behaviour is worth more than the
// hint.
//
//   - 1..5: window size steps at 0.5 / 1 / 1.5 / 2 / 3, PotPlayer-style. 6 is
//     unbound so it cannot keep the picture control it used to hold, and Alt+0
//     is dropped so it cannot resize without leaving fullscreen first.
//   - Alt+1..Alt+6: the picture controls mpv puts on 1..6 — contrast,
//     brightness and gamma, each as a -/+ pair — moved here now that the number
//     keys resize the window. They keep mpv's own commands.
//   - Always on top: mpv binds `T` (a bare capital means Shift+T) to
//     `cycle ontop` and `add sub-pos +1` to the lowercase `t`, so pressing what
//     the menu advertises as "T" moved the subtitles instead. `t` is the easier
//     key, so it now toggles on-top as well; mpv's own T is left as it is.
//   - Subtitle position: -1 sat on `r` and +1 on `t`, with mpv also binding `R`
//     to +1 as a duplicate of `t`. `r` and `t` move as a pair to Alt+r / Alt+t
//     — moving only the +1 half would strand a -1 on `r` with nothing to undo
//     it — and both bare keys are dropped, so that no stray +1 is left sitting
//     next to a `T` that now toggles on-top.
const tinyplayInputBindings = `
# ---- TinyPlay ----
# Window size steps on the number keys. Leaving fullscreen first is required:
# --fullscreen is how TinyPlay starts mpv, and a fullscreen window cannot be
# resized.
Alt+0 ignore
1 set fullscreen no; set window-scale 0.5
2 set fullscreen no; set window-scale 1
3 set fullscreen no; set window-scale 1.5
4 set fullscreen no; set window-scale 2
5 set fullscreen no; set window-scale 3
6 ignore
# mpv's picture controls used to sit on 1..6; they move to Alt+1..Alt+6 now
# that the number keys resize the window.
Alt+1 add contrast -1
Alt+2 add contrast 1
Alt+3 add brightness -1
Alt+4 add brightness 1
Alt+5 add gamma -1
Alt+6 add gamma 1
# Always on top: the lowercase t is easier to hit than Shift+T, so it toggles
# on-top too. mpv's own T is left alone and keeps working.
t cycle ontop
# Subtitle position: -1 sat on r, +1 on t, and mpv also binds R to +1 (a
# leftover duplicate of t). r and t move as a pair to Alt+r / Alt+t, and both
# bare keys are dropped — a bare capital is Shift+<letter>, so leaving R would
# keep a stray +1 sitting next to a T that toggles on-top.
Alt+r add sub-pos -1
Alt+t add sub-pos +1
r ignore
R ignore
`

// ensureMPVInputConf writes the generated input.conf next to config.json and
// returns its path. It reports false when the file cannot be written, in which
// case the caller should start mpv without --input-conf rather than point it
// at a stale or missing file.
func ensureMPVInputConf() (string, bool) {
	dir := config.DataDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		log.Printf("mpv input.conf: cannot create %s: %v", dir, err)
		return "", false
	}
	var body strings.Builder
	if user := userMPVInputConf(); user != "" {
		body.WriteString(user)
		body.WriteString("\n")
	}
	body.WriteString(tinyplayInputBindings)
	path := filepath.Join(dir, mpvInputConfName)
	if err := os.WriteFile(path, []byte(body.String()), 0o644); err != nil {
		log.Printf("mpv input.conf: cannot write %s: %v", path, err)
		return "", false
	}
	return path, true
}

// userMPVInputConf returns the contents of the user's own mpv input.conf, or
// "" when they have none. mpv looks for it in the per-user config dir
// (%AppData%\mpv on Windows, ~/.config/mpv or $XDG_CONFIG_HOME/mpv elsewhere);
// a portable_config next to mpv.exe is not consulted here because TinyPlay's
// bundled player directory is not the one a hand-installed mpv reads from.
func userMPVInputConf() string {
	base, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(base, "mpv", "input.conf"))
	if err != nil {
		return ""
	}
	return strings.TrimRight(string(data), "\n")
}

package render

import "github.com/gookit/color"

// init forces true-color ANSI rendering on.
//
// curlstreet is an HTTP service: its output is streamed to remote curl
// clients, never to the server process's own stdout. gookit/color otherwise
// auto-detects color support from the local terminal/environment (TTY, TERM,
// NO_COLOR, CI), which would disable color when the binary runs headless —
// e.g. on Fly.io or in CI — and serve plain, colorless text to every client.
// Forcing the level guarantees the documented ANSI output regardless of where
// the binary runs. Clients that can't render ANSI strip it themselves.
func init() {
	// SupportColor() must report true-color, and Enable must be on even when
	// the environment (NO_COLOR, non-TTY) would otherwise switch it off.
	color.ForceSetColorLevel(color.LevelRgb)
	color.Enable = true
}

package tui

import "os"

// openTTY opens /dev/tty for direct terminal access, which allows bubbletea
// to render and receive input even when stdin/stdout are redirected (e.g.
// inside $() command substitution used by the shell integration function).
func openTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

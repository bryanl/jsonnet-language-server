// +build js

package terminal

// IsTerminal returns false because there will be no terminal on JS
func IsTerminal(fd int) bool {
	return false
}

// Package console provides ANSI color constants and cross-platform Unicode detection.
package console

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color escape sequences.
var (
	G = "\033[32m" // green
	Y = "\033[33m" // yellow
	R = "\033[31m" // red
	B = "\033[1m"  // bold
	X = "\033[0m"  // reset
)

// Unicode/ASCII glyphs — set at init based on stdout encoding.
var (
	CHECK = "✓"
	CROSS = "✗"
	WARN  = "⚠"
	ARROW = "→"
	DOT   = "·"
)

func init() {
	if !supportsUnicode() {
		CHECK = "[ok]"
		CROSS = "[X]"
		WARN = "[!]"
		ARROW = "->"
		DOT = "*"
		// Kill ANSI colour too — legacy consoles may not render it.
		G = ""
		Y = ""
		R = ""
		B = ""
		X = ""
	}
}

// supportsUnicode checks whether stdout can handle UTF-8 glyphs.
// On Windows, we check for WT_SESSION (Windows Terminal), ConEmuPID, or
// TERM_PROGRAM, which signal modern terminals. On Unix, we check LANG/LC_ALL.
func supportsUnicode() bool {
	// Windows Terminal always supports Unicode.
	if os.Getenv("WT_SESSION") != "" {
		return true
	}
	if os.Getenv("ConEmuPID") != "" {
		return true
	}
	if tp := os.Getenv("TERM_PROGRAM"); tp != "" {
		return true
	}
	// Check LANG / LC_ALL for UTF-8
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := strings.ToLower(os.Getenv(key))
		if strings.Contains(v, "utf") {
			return true
		}
	}
	// On Unix, if none of the above matched and there's no LANG at all,
	// default to Unicode (most modern terminals handle it).
	// On Windows without a modern terminal indicator, fall back to ASCII.
	if os.Getenv("LANG") == "" && os.Getenv("OS") == "" {
		// Likely Unix with no LANG set — still try Unicode.
		return true
	}
	// Windows with OS=Windows_NT but no modern terminal indicator → ASCII.
	if strings.Contains(strings.ToLower(os.Getenv("OS")), "windows") {
		return false
	}
	return true
}

// PrintOK prints a green success line.
func PrintOK(format string, a ...any) {
	fmt.Printf("%s%s %s%s\n", G, CHECK, fmt.Sprintf(format, a...), X)
}

// PrintErr prints a red error line to stderr.
func PrintErr(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "%s%s %s%s\n", R, CROSS, fmt.Sprintf(format, a...), X)
}

// PrintWarn prints a yellow warning line.
func PrintWarn(format string, a ...any) {
	fmt.Printf("%s%s %s%s\n", Y, WARN, fmt.Sprintf(format, a...), X)
}

// PrintInfo prints a line with an arrow prefix.
func PrintInfo(format string, a ...any) {
	fmt.Printf("  %s %s\n", ARROW, fmt.Sprintf(format, a...))
}

// PrintDot prints a dotted list item.
func PrintDot(format string, a ...any) {
	fmt.Printf("  %s %s\n", DOT, fmt.Sprintf(format, a...))
}

// PrintBold prints a bold line.
func PrintBold(format string, a ...any) {
	fmt.Printf("%s%s%s\n", B, fmt.Sprintf(format, a...), X)
}

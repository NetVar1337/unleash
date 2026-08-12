package bytepatch

import (
	"regexp"
)

// firstBackrefRE matches \1..\9 tokens.
var firstBackrefRE = regexp.MustCompile(`\\([1-9])`)

// HasBackrefs reports whether pattern uses \1..\9 backreferences, which
// RE2 (Go's regexp) cannot compile.
func HasBackrefs(pattern string) bool {
	off, _ := FirstBackref(pattern)
	return off >= 0
}

// FirstBackref locates the first \N backreference (N = 1..9) in pattern,
// ignoring \N sequences preceded by an escaped backslash. Returns the byte
// offset of the backslash and the referenced group number, or (-1, 0).
func FirstBackref(pattern string) (int, int) {
	for _, loc := range firstBackrefRE.FindAllStringSubmatchIndex(pattern, -1) {
		if loc[0] > 0 && pattern[loc[0]-1] == '\\' {
			continue // escaped backslash + digit, not a backref
		}
		return loc[0], int(pattern[loc[2]] - '0')
	}
	return -1, 0
}

// translateReplaceRE matches \1..\9 in replacement templates.
var translateReplaceRE = regexp.MustCompile(`\\([1-9])`)

// TranslateReplace converts \N group references in a replacement template to
// Go's re.Expand $N syntax.
func TranslateReplace(replace string) string {
	return translateReplaceRE.ReplaceAllStringFunc(replace, func(m string) string {
		return "$" + m[1:]
	})
}

// FindSubmatchBackrefs matches pattern (which may contain \N backreferences)
// against data. It returns submatch indices in the same layout as
// regexp.FindSubmatchIndex — group numbering matches the original pattern —
// plus the concrete compiled regexp that produced the match (usable with
// re.Expand). Returns (nil, nil) when no verified match exists.
//
// RE2 cannot express backreferences, so matching is done by concretization:
// match the prefix before the first backref, capture the referenced group's
// text, substitute it literally, and recurse for any remaining backrefs.
// Prefix candidates that fail full-pattern verification are skipped by
// advancing one byte past the candidate start.
func FindSubmatchBackrefs(pattern string, data []byte) ([]int, *regexp.Regexp) {
	off, n := FirstBackref(pattern)
	if off < 0 {
		re, err := regexp.Compile("(?s)" + pattern)
		if err != nil {
			return nil, nil
		}
		return re.FindSubmatchIndex(data), re
	}

	prefix := pattern[:off]
	suffix := pattern[off+2:]
	rePrefix, err := regexp.Compile("(?s)" + prefix)
	if err != nil {
		return nil, nil
	}

	pos := 0
	for pos <= len(data) {
		ploc := rePrefix.FindSubmatchIndex(data[pos:])
		if ploc == nil {
			return nil, nil
		}
		if 2*n+1 < len(ploc) && ploc[2*n] >= 0 {
			txt := data[pos+ploc[2*n] : pos+ploc[2*n+1]]
			concrete := prefix + regexp.QuoteMeta(string(txt)) + suffix
			loc, re := FindSubmatchBackrefs(concrete, data[pos:])
			if loc != nil {
				for i := range loc {
					if loc[i] >= 0 {
						loc[i] += pos
					}
				}
				return loc, re
			}
		}
		pos += ploc[0] + 1
	}
	return nil, nil
}

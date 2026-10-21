// Package suggest finds the name someone probably meant.
package suggest

import "strings"

// Closest returns the one candidate within a small edit distance of name,
// ignoring case, such as resource_class for resouce_class. With none, or more
// than one as close, it returns "". name itself is never suggested.
func Closest(name string, candidates []string) string {
	limit := min(2, len(name)/3)
	best, bestDistance, tied := "", limit+1, false

	for _, candidate := range candidates {
		if candidate == name {
			continue
		}
		distance := editDistance(strings.ToLower(name), strings.ToLower(candidate))
		switch {
		case distance < bestDistance:
			best, bestDistance, tied = candidate, distance, false
		case distance == bestDistance:
			tied = true
		}
	}

	if tied || bestDistance > limit {
		return ""
	}
	return best
}

// editDistance counts the insertions, deletions, substitutions and swaps of
// adjacent characters that turn a into b.
func editDistance(a, b string) int {
	x, y := []rune(a), []rune(b)
	rows := make([][]int, len(x)+1)
	for i := range rows {
		rows[i] = make([]int, len(y)+1)
		rows[i][0] = i
	}
	for j := range rows[0] {
		rows[0][j] = j
	}

	for i := 1; i <= len(x); i++ {
		for j := 1; j <= len(y); j++ {
			cost := 1
			if x[i-1] == y[j-1] {
				cost = 0
			}
			rows[i][j] = min(rows[i-1][j]+1, rows[i][j-1]+1, rows[i-1][j-1]+cost)
			if i > 1 && j > 1 && x[i-1] == y[j-2] && x[i-2] == y[j-1] {
				rows[i][j] = min(rows[i][j], rows[i-2][j-2]+1)
			}
		}
	}

	return rows[len(x)][len(y)]
}

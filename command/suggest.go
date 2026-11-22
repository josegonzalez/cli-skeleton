package command

import (
	"sort"
	"strings"
)

// levenshtein returns the edit distance between a and b.
func levenshtein(a string, b string) int {
	ra, rb := []rune(a), []rune(b)
	if len(ra) == 0 {
		return len(rb)
	}
	if len(rb) == 0 {
		return len(ra)
	}

	prev := make([]int, len(rb)+1)
	curr := make([]int, len(rb)+1)
	for j := range prev {
		prev[j] = j
	}

	for i := 1; i <= len(ra); i++ {
		curr[0] = i
		for j := 1; j <= len(rb); j++ {
			cost := 1
			if ra[i-1] == rb[j-1] {
				cost = 0
			}
			curr[j] = min(prev[j]+1, curr[j-1]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}

	return prev[len(rb)]
}

// suggestionDistance is the largest edit distance considered a typo for a
// word of the given length.
func suggestionDistance(word string) int {
	n := len([]rune(word))
	switch {
	case n <= 3:
		return 1
	case n <= 6:
		return 2
	default:
		return 3
	}
}

// Suggest returns the candidates that look like typos of input, closest
// first. A candidate matches when its edit distance from input is small
// for its length, or when input is a prefix of it.
func Suggest(input string, candidates []string) []string {
	type scored struct {
		value    string
		distance int
	}

	input = strings.ToLower(input)
	maxDistance := suggestionDistance(input)

	seen := map[string]bool{}
	var matches []scored
	for _, candidate := range candidates {
		if candidate == "" || seen[candidate] || candidate == input {
			continue
		}
		seen[candidate] = true

		distance := levenshtein(input, strings.ToLower(candidate))
		if distance <= maxDistance || (len(input) >= 2 && strings.HasPrefix(strings.ToLower(candidate), input)) {
			matches = append(matches, scored{value: candidate, distance: distance})
		}
	}

	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].distance != matches[j].distance {
			return matches[i].distance < matches[j].distance
		}
		return matches[i].value < matches[j].value
	})

	result := make([]string, 0, len(matches))
	for _, match := range matches {
		result = append(result, match.value)
	}
	return result
}

// didYouMean formats suggestions as a question, or returns an empty string
// when there are none. format is applied to each suggestion.
func didYouMean(suggestions []string, format func(string) string) string {
	if len(suggestions) == 0 {
		return ""
	}
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}

	quoted := make([]string, 0, len(suggestions))
	for _, s := range suggestions {
		quoted = append(quoted, format(s))
	}

	switch len(quoted) {
	case 1:
		return "Did you mean " + quoted[0] + "?"
	case 2:
		return "Did you mean " + quoted[0] + " or " + quoted[1] + "?"
	default:
		return "Did you mean " + strings.Join(quoted[:len(quoted)-1], ", ") + " or " + quoted[len(quoted)-1] + "?"
	}
}

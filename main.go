package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Println("Usage: go run main.go <input> <output>")
		os.Exit(1)
	}

	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Println("Error reading:", err)
		os.Exit(1)
	}

	result := processText(string(data))

	err = os.WriteFile(os.Args[2], []byte(result), 0644)
	if err != nil {
		fmt.Println("Error writing:", err)
		os.Exit(1)
	}
	fmt.Println("Success.")
}

// processText uses bufio.Scanner (replacing the old strings.Split + strings.Fields
func processText(input string) string {
	scanner := bufio.NewScanner(strings.NewReader(input))
	var out []string

	for scanner.Scan() {
		line := scanner.Text()
		words := strings.Fields(line)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		words = processModifiers(words)
		// fixPunctuation MUST run before fixQuotes so that a lone "." is already
		// glued to its word before the closing "'" looks for its previous word.
		// Without this order, ". '" becomes "raincoat. '" (trailing space).
		words = fixPunctuation(words)
		words = fixQuotes(words)
		words = fixArticles(words)
		out = append(out, strings.Join(words, " "))
	}
	return strings.Join(out, "\n")
}

// capitalize upper-cases the first rune and lower-cases the rest.
 func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + strings.ToLower(s[1:])
} 

// isOurPunct is the single source of truth for punctuation we handle.
// unicode.IsPunct guards against non-punctuation (fast path); then we confirm
// it is one of our six characters.
func isOurPunct(r rune) bool {
	return unicode.IsPunct(r) && strings.ContainsRune(".,!?:;", r)
}

// isPunctuation reports whether every rune in s is in our punctuation set.
func isPunctuation(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isOurPunct(r) {
			return false
		}
	}
	return true
}

// processModifiers applies (up), (low), (cap), (hex), (bin) and their
func processModifiers(words []string) []string {
	transformations := map[string]func(string) string{
		"(up)":  strings.ToUpper,
		"(low)": strings.ToLower,
		"(cap)": capitalize,
		"(hex)": func(s string) string {
			if v, err := strconv.ParseInt(s, 16, 64); err == nil {
				return strconv.FormatInt(v, 10)
			}
			return s
		},
		"(bin)": func(s string) string {
			if v, err := strconv.ParseInt(s, 2, 64); err == nil {
				return strconv.FormatInt(v, 10)
			}
			return s
		},
	}

	result := []string{}
	for i := 0; i < len(words); i++ {
		word := words[i]

		// Counted modifiers: strings.Fields splits "(up, 2)" into "(up," and "2)".
		if (word == "(up," || word == "(low," || word == "(cap," || word == "(bin," || word == "(hex,") && i+1 < len(words) {
			// Extract digits from "2)" or "2)," etc. using unicode.IsDigit.
			nStr := strings.TrimFunc(words[i+1], 
			func(r rune) bool {
				return !unicode.IsDigit(r)
			})
			if n, err := strconv.Atoi(nStr); err == nil {
				// Reconstruct the simple tag: "(up," → "(up)"
				tag := word[:len(word)-1] + ")"
				if fn, ok := transformations[tag]; ok {
					for j := 1; j <= n; j++ {
						if target := len(result) - j; target >= 0 {
							result[target] = fn(result[target])
						}
					}
				}
			}
			i++ // skip the number token
			continue
		}

		// Strip trailing punctuation so "(up)," is still recognised as a modifier.
		suffix := ""
		clean := word
		for len(clean) > 0 && isOurPunct(rune(clean[len(clean)-1])) {
			suffix = string(clean[len(clean)-1]) + suffix
			clean = clean[:len(clean)-1]
		}

		if fn, ok := transformations[clean]; ok {
			if len(result) > 0 {
				result[len(result)-1] = fn(result[len(result)-1]) + suffix
			}
			continue
		}

		result = append(result, word)
	}
	return result
}

// fixPunctuation glues punctuation tokens onto the preceding word and moves
// leading punctuation (e.g. "!Hello") onto the word before it.
func fixPunctuation(words []string) []string {
	result := []string{}
	for _, word := range words {
		// Entire token is punctuation — glue to previous word.
		if isPunctuation(word) {
			if len(result) > 0 {
				result[len(result)-1] += word
			} else {
				result = append(result, word)
			}
			continue
		}

		// Token starts with punctuation but is not purely punctuation
		// e.g. "!Hello", "...world" — split and glue the leading punctuation.
		if len(word) > 1 && isOurPunct(rune(word[0])) {
			pEnd := 0
			for pEnd < len(word) && isOurPunct(rune(word[pEnd])) {
				pEnd++
			}
			prefix := word[:pEnd]
			if len(result) > 0 {
				result[len(result)-1] += prefix
			} else {
				result = append(result, prefix)
			}
			result = append(result, word[pEnd:])
			continue
		}

		result = append(result, word)
	}
	return result
}

func fixQuotes(words []string) []string {
	result := []string{}
	quoteOpen := false

	for i := 0; i < len(words); i++ {
		word := words[i]

		if word == "'" {
			if !quoteOpen {
				// Opening: prepend "'" to the next word.
				if i+1 < len(words) {
					words[i+1] = "'" + words[i+1]
					quoteOpen = true
				} else {
					result = append(result, word) // dangling quote, keep it
				}
			} else {
				// Closing: append "'" to the previous word.
				if len(result) > 0 {
					result[len(result)-1] += "'"
					quoteOpen = false
				} else {
					result = append(result, word)
				}
			}
			continue
		}

		hasOpen := strings.HasPrefix(word, "'")
		hasClose := strings.HasSuffix(word, "'")
		switch {
		case hasOpen && !hasClose:
			quoteOpen = true
		case hasClose && !hasOpen:
			quoteOpen = false
		// hasOpen && hasClose → self-contained, no state change
		}

		result = append(result, word)
	}
	return result
}

// fixArticles:
func fixArticles(words []string) []string {
	// hVowelSet includes 'h' for words like "hour", "honest", "heir".
	const hVowelSet = "aeiouhAEIOUH"

	for i := 0; i+1 < len(words); i++ {
		runes := []rune(words[i])

		// Find the letter content of this token, preserving decoration.
		start := 0
		for start < len(runes) && !unicode.IsLetter(runes[start]) {
			start++
		}
		end := len(runes)
		for end > start && !unicode.IsLetter(runes[end-1]) {
			end--
		}
		if start >= end {
			continue
		}
		prefix := string(runes[:start])
		article := string(runes[start:end])
		suffix := string(runes[end:])

		lower := strings.ToLower(article)
		if lower != "a" && lower != "an" {
			continue
		}

		// Find the first letter of the next word (skip leading non-letters).
		nextRunes := []rune(words[i+1])
		letIdx := 0
		for letIdx < len(nextRunes) && !unicode.IsLetter(nextRunes[letIdx]) {
			letIdx++
		}
		if letIdx >= len(nextRunes) {
			continue
		}
		firstLetter := nextRunes[letIdx]

		startsVowelSound := strings.ContainsRune(hVowelSet, firstLetter)
		isUpper := unicode.IsUpper(runes[start])

		var correct string
		switch {
		case startsVowelSound && isUpper:
			correct = "An"
		case startsVowelSound:
			correct = "an"
		case isUpper:
			correct = "A"
		default:
			correct = "a"
		}

		if article != correct {
			words[i] = prefix + correct + suffix
		}
	}
	return words
}
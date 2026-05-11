package main

import (
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
	fmt.Println("Done.")
}

// ── pipeline ──────────────────────────────────────────────────────────────────

func processText(input string) string {
	lines := strings.Split(input, "\n")
	for i, line := range lines {
		lines[i] = processLine(line)
	}
	return strings.Join(lines, "\n")
}

func processLine(line string) string {
	words := strings.Fields(line)
	if len(words) == 0 {
		return ""
	}
	words = applyModifiers(words)
	words = fixPunctuation(words)
	words = fixQuotes(words)
	words = fixArticles(words)
	return strings.Join(words, " ")
}

// ── modifiers ─────────────────────────────────────────────────────────────────

var transforms = map[string]func(string) string{
	"(up)":  strings.ToUpper,
	"(low)": strings.ToLower,
	"(cap)": func(w string) string {
		return strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	},
	"(hex)": func(w string) string {
		if v, err := strconv.ParseInt(w, 16, 64); err == nil {
			return strconv.Itoa(int(v))
		}
		return w
	},
	"(bin)": func(w string) string {
		if v, err := strconv.ParseInt(w, 2, 64); err == nil {
			return strconv.Itoa(int(v))
		}
		return w
	},
}

func isWordToken(s string) bool {
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return true
		}
	}
	return false
}

func applyModifiers(words []string) []string {
	result := []string{}
	for i := 0; i < len(words); i++ {
		w := words[i]
		if (w == "(up," || w == "(low," || w == "(cap," || w == "(bin,") && i+1 < len(words) {
			tag := strings.TrimSuffix(w, ",") + ")"
			n, _ := strconv.Atoi(strings.Trim(words[i+1], "(),"))
			if fn, ok := transforms[tag]; ok {
				found := 0
				for j := len(result) - 1; j >= 0 && found < n; j-- {
					if isWordToken(result[j]) {
						result[j] = fn(result[j])
						found++
					}
				}
			}
			i++
			continue
		}
		clean := strings.TrimRight(w, ".,!?:;")
		tail := w[len(clean):]
		if fn, ok := transforms[clean]; ok {
			if len(result) > 0 {
				result[len(result)-1] = fn(result[len(result)-1]) + tail
			}
			continue
		}
		result = append(result, w)
	}
	return result
}

// ── punctuation ───────────────────────────────────────────────────────────────

func fixPunctuation(words []string) []string {
	result := []string{}
	for _, w := range words {
		switch {
		case strings.Trim(w, ".,!?:;") == "":
			if len(result) > 0 {
				result[len(result)-1] += w
			}
		case strings.ContainsRune(".,!?:;", rune(w[0])) && len(w) > 1:
			end := strings.IndexFunc(w, func(r rune) bool {
				return !unicode.IsPunct(r)
			})
			if len(result) > 0 {
				result[len(result)-1] += w[:end]
			}
			result = append(result, w[end:])
		default:
			result = append(result, w)
		}
	}
	return result
}

// ── quotes ────────────────────────────────────────────────────────────────────

func fixQuotes(words []string) []string {
	result := []string{}
	open := false

	for i, w := range words {
		switch {
		case w == "'" && !open && i+1 < len(words):
			words[i+1] = "'" + words[i+1]
			open = true

		case w == "'" && open && len(result) > 0:
			result[len(result)-1] += "'"
			open = false

		case open && strings.HasPrefix(w, "'") && strings.Trim(w[1:], ".,!?:;") == "" && len(result) > 0:
			result[len(result)-1] += w
			open = false

		default:
			result = append(result, w)
		}
	}
	return result
}

// ── articles ──────────────────────────────────────────────────────────────────

var vowels = map[rune]bool{
	'a': true, 'e': true, 'i': true, 'o': true, 'u': true, 'h': true,
}

func isVowelSound(word string) bool {
	clean := strings.TrimLeft(word, "'\".,!?:;")
	if clean == "" {
		return false
	}
	return vowels[unicode.ToLower(rune(clean[0]))]
}

func fixArticles(words []string) []string {
	for i := 0; i+1 < len(words); i++ {
		bare := strings.TrimLeft(words[i], "'\".,!?:;")
		prefix := words[i][:len(words[i])-len(bare)]

		lower := strings.ToLower(bare)
		if lower != "a" && lower != "an" {
			continue
		}

		isUpper := unicode.IsUpper(rune(bare[0]))

		switch {
		case isVowelSound(words[i+1]) && isUpper:
			words[i] = prefix + "An"
		case isVowelSound(words[i+1]) && !isUpper:
			words[i] = prefix + "an"
		case !isVowelSound(words[i+1]) && isUpper:
			words[i] = prefix + "A"
		default:
			words[i] = prefix + "a"
		}
	}
	return words
}
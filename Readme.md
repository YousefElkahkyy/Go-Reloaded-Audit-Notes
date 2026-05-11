# go-reloaded — Complete Guide to main.go

A CLI text processor that reads a file, applies a four-stage transformation pipeline to every line, and writes the result to a new file.

**Usage:** `go run main.go input.txt output.txt`

---

## Table of Contents

1. [Project Overview](#1-project-overview)
2. [Imports](#2-imports)
3. [main()](#3-main)
4. [processText()](#4-processtext)
5. [processLine()](#5-processline)
6. [transforms — the modifier map](#6-transforms--the-modifier-map)
7. [isWordToken()](#7-iswordtoken)
8. [applyModifiers()](#8-applymodifiers)
9. [fixPunctuation()](#9-fixpunctuation)
10. [fixQuotes()](#10-fixquotes)
11. [fixArticles()](#11-fixarticles)
12. [isVowelSound()](#12-isvowelsound)
13. [Pipeline Order — Why It Matters](#13-pipeline-order--why-it-matters)
14. [Edge Cases Handled](#14-edge-cases-handled)
15. [Go Concepts Used](#15-go-concepts-used)

---

## 1. Project Overview

The program transforms text through four passes applied to each line:

```
applyModifiers → fixPunctuation → fixQuotes → fixArticles
```

| Feature | Example input | Example output |
|---------|--------------|----------------|
| Hex conversion | `FF (hex)` | `255` |
| Binary conversion | `101 (bin)` | `5` |
| Uppercase | `hello (up)` | `HELLO` |
| Lowercase | `HELLO (low)` | `hello` |
| Title case | `hello (cap)` | `Hello` |
| Counted modifier | `a b c (up, 2)` | `a B C` |
| Punctuation spacing | `hello , world !` | `hello, world!` |
| Quote gluing | `' hello '` | `'hello'` |
| Article correction | `a apple` | `an apple` |

---

## 2. Imports

```go
import (
    "fmt"
    "os"
    "strconv"
    "strings"
    "unicode"
)
```

| Package | What it does in this program |
|---------|------------------------------|
| `fmt` | `fmt.Println` for CLI messages |
| `os` | `os.Args` for CLI args, `os.ReadFile` / `os.WriteFile` for file I/O, `os.Exit` for error exits |
| `strconv` | `strconv.ParseInt` converts hex/binary strings; `strconv.Atoi` parses the count in `(up, 2)`; `strconv.Itoa` converts integers back to strings |
| `strings` | Almost everything — splitting, joining, trimming, searching |
| `unicode` | `unicode.IsLetter` and `unicode.IsDigit` in `isWordToken`; `unicode.IsPunct` in `fixPunctuation`; `unicode.ToLower` and `unicode.IsUpper` in the articles section |

---

## 3. main()

```go
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
```

`len(os.Args) != 3` — `os.Args[0]` is the program name, `[1]` is the input path, `[2]` is the output path. Exactly 3 elements are required.

`os.Exit(1)` — exits with code 1 (error). Non-zero exit codes signal failure to the shell.

`os.ReadFile` — reads the entire file into memory as `[]byte` in one call. `string(data)` converts it to a string for processing.

`[]byte(result)` — converts the processed string back to bytes for writing.

`0644` — Unix file permission in octal: owner read+write, others read-only.

`err = os.WriteFile(...)` — reuses the `err` variable with `=` (not `:=`) because it was already declared above with `:=`.

---

## 4. processText()

```go
func processText(input string) string {
    lines := strings.Split(input, "\n")
    for i, line := range lines {
        lines[i] = processLine(line)
    }
    return strings.Join(lines, "\n")
}
```

`strings.Split(input, "\n")` — splits on newline characters. An empty line becomes `""` in the slice, which is preserved throughout.

`lines[i] = processLine(line)` — writes back to `lines[i]`, not `line`, because `line` is a copy of the element and assigning to it would not change the slice.

`strings.Join(lines, "\n")` — reassembles lines with newlines. Exact reverse of `Split`.

**Why split into lines first?** `strings.Fields` collapses all whitespace including newlines. Running it on the whole file would erase empty lines. Splitting first keeps the structure intact.

---

## 5. processLine()

```go
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
```

`strings.Fields(line)` — splits on any whitespace and discards empty strings from consecutive spaces. `"a    b"` becomes `["a", "b"]`.

`len(words) == 0` — empty or whitespace-only line. Return `""` immediately rather than running four passes on nothing.

Each of the four function calls takes `[]string` and returns `[]string`, so they chain cleanly.

`strings.Join(words, " ")` — rejoins with single spaces, normalising any original extra spacing.

---

## 6. transforms — the modifier map

```go
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
```

`map[string]func(string) string` — maps a tag name like `"(up)"` to the function that performs its transformation. Both the plain and counted modifier branches in `applyModifiers` use this single map.

`strings.ToUpper` / `strings.ToLower` — assigned directly; their signatures already match `func(string) string`.

`(cap)` — `w[:1]` is the first byte uppercased; `w[1:]` is the rest lowercased. Together they give title case for one word.

`(hex)` and `(bin)` — `strconv.ParseInt(w, base, 64)` parses the word as base-16 or base-2. On failure the original word is returned unchanged.

**Package-level `var`** — initialised once at startup. A map declared inside `applyModifiers` would be rebuilt on every function call.

---

## 7. isWordToken()

```go
func isWordToken(s string) bool {
    for _, r := range s {
        if unicode.IsLetter(r) || unicode.IsDigit(r) {
            return true
        }
    }
    return false
}
```

Reports whether a token contains at least one letter or digit. A bare `'` or a token like `...` returns `false`.

**`unicode.IsLetter` and `unicode.IsDigit`** — the proper standard-library functions for this question. They are Unicode-aware, recognising letters and digits from any script. They replace what used to be a long hardcoded string listing every ASCII letter and digit character by character.

**Why is this needed?** Counted modifiers like `(cap, 2)` walk backwards through the result slice to find N words to transform. Without this check, a lone `'` would be counted as a word slot, wasting it and leaving the wrong word untransformed.

---

## 8. applyModifiers()

```go
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
```

### Counted modifier branch

`strings.Fields` splits `(up, 2)` into two tokens: `"(up,"` and `"2)"`. The `if` detects these half-tokens.

`strings.TrimSuffix(w, ",") + ")"` — rebuilds the clean tag. `"(up,"` → `"(up)"`. Looked up in `transforms`.

`strings.Trim(words[i+1], "(),")` — strips parentheses and commas from `"2)"` leaving `"2"`. `strconv.Atoi` converts that to integer `2`.

The inner loop walks backwards through `result`, uses `isWordToken` to skip lone punctuation tokens, and applies the transformation to the first `n` real words found.

`i++` then `continue` — consume the count token so the main loop does not try to process it as a word.

### Plain modifier branch

`strings.TrimRight(w, ".,!?:;")` — strips trailing punctuation. `"(up),"` → `clean="(up)"`, `tail=","`.

`transforms[clean]` — if found, apply it to the last word in `result` and re-attach the tail.

`len(result) > 0` — guard against a modifier at the very start of a line.

---

## 9. fixPunctuation()

```go
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
```

**Purpose:** Remove the space before punctuation by gluing punctuation tokens onto the preceding word.

### Case 1 — whole token is punctuation

`strings.Trim(w, ".,!?:;") == ""` — removing all six punctuation characters leaves nothing. The entire token is punctuation (e.g. `","`, `"..."`, `"!?"`). Glue it to the last word.

### Case 2 — token starts with punctuation but is not purely punctuation

Example: `"!Hello"`. The `!` belongs to the previous word; `Hello` is its own word.

`strings.IndexFunc(w, func(r rune) bool { return !unicode.IsPunct(r) })` — **`unicode.IsPunct(r)`** asks the `unicode` package whether a rune is a punctuation character. `strings.IndexFunc` returns the index of the first rune where the function returns `true` — in other words, the first non-punctuation character. That is the boundary between the punctuation prefix and the word.

**Why use `unicode.IsPunct` here but the explicit `".,!?:;"` set for the case 1 trigger?** Case 1 tests whether the *whole token* belongs strictly to our handled set — we intentionally exclude `'` and `"` there. Case 2 only uses `unicode.IsPunct` to *locate the boundary* inside a mixed token, which is safe because the case 2 trigger already confirmed the token starts with one of our six characters.

### Default

A normal word. Append as-is.

---

## 10. fixQuotes()

```go
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
```

`open` tracks whether the next standalone `'` is an opening or closing quote.

**Case 1 — opening quote:** prepend `'` to the next word. Token consumed. `open = true`.

**Case 2 — closing quote (standalone):** append `'` to the last word in result. Token consumed. `open = false`.

**Case 3 — closing quote with punctuation attached:** `fixPunctuation` runs before `fixQuotes`, so a closing `'` followed by `!?` arrives as the single token `'!?`. Cases 1 and 2 only match a bare `"'"`, so this case catches it instead. `strings.HasPrefix(w, "'")` and `strings.Trim(w[1:], ".,!?:;") == ""` confirm the token is a `'` followed only by punctuation. Glue the whole token to the previous word and close.

The `Trim` check is critical: it prevents words like `'An` or `'apple` from being mistaken for closing quotes.

**Default:** all other words pass through normally.

---

## 11. fixArticles()

```go
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
```

**Stripping the prefix:** after `fixQuotes`, an article `a` inside a quote becomes `'a`. `TrimLeft` strips the `'`, leaving `bare = "a"` and `prefix = "'"`. The prefix is put back in every `switch` branch.

**`unicode.IsUpper(rune(bare[0]))`** — checks whether the article's first character is uppercase. `rune(bare[0])` reads the first byte directly as a rune. This is simpler than the previous approach of converting the whole string to `[]rune` just to read element zero — that allocated memory for a new slice unnecessarily. For the ASCII letters `a` and `A` this direct byte-to-rune cast is correct.

The `switch` covers all four combinations of vowel-sound (true/false) and case (upper/lower), and modifies `words[i]` in place.

---

## 12. isVowelSound()

```go
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
```

Reports whether a word begins with a vowel sound, which determines whether `a` or `an` is correct.

### vowels map — replacing the const string

The previous version used:
```go
const vowelSounds = "aeiouAEIOUhH"
// checked with: strings.ContainsRune(vowelSounds, rune(clean[0]))
```

This had to list every vowel **twice** — lowercase and uppercase — because `ContainsRune` does a literal character search with no case normalisation.

The current version uses a `map[rune]bool` with **only lowercase keys**, and normalises the input with **`unicode.ToLower`** before lookup:

```go
return vowels[unicode.ToLower(rune(clean[0]))]
```

`unicode.ToLower(r)` converts any uppercase letter to lowercase. `'A'` → `'a'`, `'H'` → `'h'`. The map needs each letter only once. This is shorter, cleaner, and uses the `unicode` package's built-in case knowledge rather than a manually maintained character list.

`'h'` is in the map because words like `hour`, `honest`, and `heir` begin with a silent h and take `an` in English.

`strings.TrimLeft` strips any leading punctuation (e.g. `'hour` inside a quote) so the first actual letter is checked.

---

## 13. Pipeline Order — Why It Matters

```
applyModifiers → fixPunctuation → fixQuotes → fixArticles
```

**Modifiers first** — they change word content on raw tokens. `isWordToken` can then correctly skip lone `'` separators when counting backwards for `(cap, 2)`.

**Punctuation second** — attaches punctuation to words before quotes are processed. A closing `'` followed by `!?` gets merged into `'!?`, which `fixQuotes` case 3 then handles correctly.

**Quotes third** — `fixArticles` needs to see final token shapes. After `fixQuotes`, an article inside a quote is `'a`. The prefix-stripping logic in `fixArticles` depends on this.

**Articles last** — must inspect the final form of the next word after any modifier transformation and quote attachment, to determine the correct vowel sound.

---

## 14. Edge Cases Handled

| Situation | Input | Output | Where handled |
|-----------|-------|--------|---------------|
| Empty line | `""` | `""` | `processLine`: early return |
| Modifier with no preceding word | `(up) hello` | `hello` | `applyModifiers`: `len(result) > 0` guard |
| Count larger than available words | `a b (up, 10)` | `A B` | backwards loop stops at `j >= 0` |
| Lone `'` counted as word slot | `' a b (cap, 2)` | `'A B` | `isWordToken` skips `'` |
| Invalid hex/binary | `ZZ (hex)` | `ZZ` | transform returns `w` on parse error |
| Modifier with trailing punct | `hello (up),` | `HELLO,` | `clean`/`tail` split in `applyModifiers` |
| Lone `'` at end of line | `hello '` | `hello '` | `fixQuotes` default case |
| Closing `'` with punct glued | `' hello '!?` | `'hello'!?` | `fixQuotes` case 3 |
| Article with quote prefix | `' a apple '` | `'an apple'` | `fixArticles` prefix strip |
| `h` words: hour, honest | `a hour` | `an hour` | `'h'` key in `vowels` map |
| Uppercase article | `A apple` | `An apple` | `unicode.IsUpper` in `fixArticles` |

---

## 15. Go Concepts Used

| Concept | Where used | What it does |
|---------|-----------|--------------|
| `map[string]func(string) string` | `transforms` | Maps tag names to transformation functions; both modifier branches share one lookup |
| `map[rune]bool` | `vowels` | Maps lowercase vowel runes to `true`; O(1) lookup, replaces a string scan |
| Package-level `var` | `transforms`, `vowels` | Initialised once at startup, not rebuilt on every call |
| Inline functions (closures) | `(cap)`, `(hex)`, `(bin)` in `transforms` | Encapsulate conversion logic right where it is defined |
| `unicode.IsLetter` / `unicode.IsDigit` | `isWordToken` | Unicode-aware letter and digit detection — replaces a long hardcoded character string |
| `unicode.IsPunct` | `fixPunctuation` case 2 | Finds the boundary between punctuation and letters inside a mixed token |
| `unicode.ToLower` | `isVowelSound` | Normalises any letter to lowercase before the vowel map lookup — eliminates listing both cases |
| `unicode.IsUpper` | `fixArticles` | Checks whether the article's first rune is uppercase — replaces a `[]rune` slice allocation |
| `strings.Fields` | `processLine` | Splits on any whitespace, discards empty tokens, normalises multiple spaces |
| `strings.Split` / `strings.Join` | `processText` | Split file into lines, reassemble after processing |
| `strings.TrimRight` / `strings.TrimLeft` | `applyModifiers`, `fixArticles`, `isVowelSound` | Strip characters from one end only |
| `strings.Trim` | `applyModifiers`, `fixPunctuation`, `fixQuotes` | Strip characters from both ends |
| `strings.TrimSuffix` | `applyModifiers` | Remove a specific suffix: `"(up,"` → `"(up"` |
| `strings.ContainsRune` | `fixPunctuation` case 2 trigger | Check if the first byte is one of our six punctuation characters |
| `strings.HasPrefix` | `fixQuotes` | Check if a token starts with `'` |
| `strings.IndexFunc` | `fixPunctuation` | Find the first character matching a condition (used with `unicode.IsPunct`) |
| `strconv.ParseInt(s, base, bits)` | `(hex)` and `(bin)` transforms | Parse base-16 or base-2 integer |
| `strconv.Atoi` | `applyModifiers` | Parse the count string `"2"` as an integer |
| `strconv.Itoa` | `(hex)` and `(bin)` transforms | Convert an integer back to a decimal string |
| `for i, w := range` + `words[i+1]` | `fixQuotes` | Range copies `w`; direct index modifies the element the next iteration sees |
| In-place slice modification | `fixArticles` | Modifies `words[i]` directly instead of building a new result slice |
| `switch` with no expression | `fixPunctuation`, `fixQuotes` | Each `case` is an independent boolean; cleaner than nested `if/else` |
| `os.ReadFile` / `os.WriteFile` | `main` | Read or write an entire file in one call |
| `os.Exit(1)` | `main` | Signal error to the shell without using panic |
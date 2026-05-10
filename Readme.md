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
)
```

| Package | What it does in this program |
|---------|------------------------------|
| `fmt` | `fmt.Println` for CLI messages |
| `os` | `os.Args` for CLI args, `os.ReadFile` / `os.WriteFile` for file I/O, `os.Exit` for error exits |
| `strconv` | `strconv.ParseInt` to convert hex/binary strings to integers, `strconv.Atoi` to parse the count in `(up, 2)`, `strconv.Itoa` to turn the integer back into a string |
| `strings` | Almost everything — splitting, joining, trimming, searching |

Notice there is no `unicode`, `bufio`, or `regexp`. The program uses only the four packages it actually needs.

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

**Line by line:**

`len(os.Args) != 3` — `os.Args` is a slice where `[0]` is the program name, `[1]` is the input path, `[2]` is the output path. Exactly 3 elements are required.

`os.Exit(1)` — exits with code 1 (error). Non-zero exit codes signal failure to the shell.

`os.ReadFile(os.Args[1])` — reads the entire file into memory as `[]byte`. Returns the bytes and an error. This is simpler than opening a file handle and reading line by line.

`string(data)` — converts `[]byte` to `string`. Safe for UTF-8 text files.

`[]byte(result)` — converts `string` back to `[]byte` for writing.

`0644` — Unix file permission in octal. `6` = owner can read+write, `4` = others can read.

`err = os.WriteFile(...)` — reuses the `err` variable with `=` (not `:=`) because it was already declared above.

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

**Purpose:** Split the whole file into lines, process each line, join them back.

`strings.Split(input, "\n")` — splits on newline characters. An empty line becomes an empty string `""` in the slice, which is preserved through the whole pipeline.

`for i, line := range lines` — iterates with both index and value. We write back to `lines[i]` (not `line`) because `line` is a copy — assigning to it would not change the slice.

`strings.Join(lines, "\n")` — reassembles the lines with newlines between them. This is the exact reverse of `Split`.

**Why split into lines first?** `strings.Fields` (used inside `processLine`) collapses all whitespace including newlines. If we ran `Fields` on the whole file, empty lines would disappear. Splitting first keeps the structure intact.

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

**Purpose:** Turn one line of text into a slice of words, run the four passes, join back with single spaces.

`strings.Fields(line)` — splits on any whitespace (spaces, tabs) and discards the empty strings that would result from consecutive spaces. `"a    b"` → `["a", "b"]`. This naturally normalises multiple spaces.

`len(words) == 0` — an empty or whitespace-only line produces an empty slice. Return `""` immediately rather than running four passes on nothing.

The four function calls are the pipeline. Each one takes `[]string` and returns `[]string`, so they chain cleanly.

`strings.Join(words, " ")` — joins with a single space. After the pipeline the words are in the right order, so one space between each word gives the final line.

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

**Purpose:** A single map from modifier tag name to the function that performs the transformation. Both `applyModifiers` passes (plain and counted) look up from this one map.

`map[string]func(string) string` — the key is the tag string like `"(up)"`, the value is any function that takes one string and returns one string.

`strings.ToUpper` — assigned directly without wrapping because its signature already matches `func(string) string`.

`(cap)` inline function — `w[:1]` is the first byte as a string (safe for ASCII letters), uppercased. `w[1:]` is everything after the first byte, lowercased. Together they produce title case for a single word.

`(hex)` and `(bin)` inline functions — `strconv.ParseInt(w, 16, 64)` parses `w` as a base-16 (or base-2) signed 64-bit integer. If parsing fails (the word is not valid hex/binary), `err != nil` and the word is returned unchanged. `strconv.Itoa` converts the integer back to a base-10 decimal string.

**Why a `var` at package level instead of inside the function?** A `var` is initialised once when the program starts. Putting the map inside `applyModifiers` would rebuild it on every call — once per word in every line.

---

## 7. isWordToken()

```go
func isWordToken(s string) bool {
    for _, r := range s {
        if strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", r) {
            return true
        }
    }
    return false
}
```

**Purpose:** Report whether a token contains at least one letter or digit. A bare quote `'` or a punctuation-only token like `...` returns `false`.

**Why is this needed?** Counted modifiers like `(cap, 2)` count backwards through the result slice to find the last N words to transform. Without this check, a lone `'` sitting between two words would be counted as one of those words, wasting a slot and leaving the wrong word untransformed.

`for _, r := range s` — iterates over runes (Unicode code points). The `_` discards the byte index.

`strings.ContainsRune(chars, r)` — returns `true` if `r` appears anywhere in `chars`. As soon as one letter or digit is found the function returns `true` immediately.

---

## 8. applyModifiers()

```go
func applyModifiers(words []string) []string {
    result := []string{}
    for i := 0; i < len(words); i++ {
        w := words[i]

        // Counted modifier: (up, 2) is split by Fields into "(up," and "2)"
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

        // Plain modifier, possibly with trailing punctuation: "(up)," → clean="(up)" tail=","
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

**Purpose:** Detect modifier tags and apply their transformation to preceding words. Non-modifier words pass through unchanged.

### Counted modifier branch

`strings.Fields` splits `(up, 2)` into two tokens: `"(up,"` and `"2)"`. The `if` checks for these half-tokens.

`strings.TrimSuffix(w, ",") + ")"` — rebuilds the clean tag. `"(up,"` → `"(up)"`. This tag is then looked up in `transforms`.

`strings.Trim(words[i+1], "(),")` — strips parentheses, commas from `"2)"` leaving `"2"`. `strconv.Atoi` converts that to the integer `2`.

The inner loop walks backwards through `result`, skipping any token that fails `isWordToken` (like a lone `'`), and applies the transformation to the first `n` real word tokens found.

`i++` then `continue` — consume the count token `"2)"` so the main loop does not try to process it as a word.

### Plain modifier branch

`strings.TrimRight(w, ".,!?:;")` — strips any trailing punctuation from the token. `"(up),"` → `clean="(up)"`, `tail=","`. This handles the case where a modifier is immediately followed by a comma or other punctuation in the original text.

`transforms[clean]` — looks up the cleaned token. If found, it is a modifier; apply it to the last word in `result` and re-attach the tail. If not found, it is a regular word.

`len(result) > 0` — guard against a modifier appearing at the very start of a line with no preceding word.

`result = append(result, w)` — normal words just get added to the result slice.

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
                return !strings.ContainsRune(".,!?:;", r)
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

The `switch` has no expression — each `case` is a boolean condition. Go evaluates them top to bottom and runs the first one that is `true`.

### Case 1 — whole token is punctuation

`strings.Trim(w, ".,!?:;") == ""` — if removing all punctuation characters leaves an empty string, the entire token is punctuation (e.g. `","`, `"..."`, `"!?"`).

`result[len(result)-1] += w` — append it directly to the last word in `result`, eliminating the space.

### Case 2 — token starts with punctuation but is not purely punctuation

Example: `"!Hello"`. The `!` belongs to the previous word; `Hello` is its own word.

`strings.ContainsRune(".,!?:;", rune(w[0]))` — check the first byte.

`strings.IndexFunc(w, ...)` — find the index of the first non-punctuation character. Everything before that index is the leading punctuation prefix; everything from that index onwards is the actual word.

The prefix is glued to the previous word; the rest is appended as its own token.

### Case 3 — default

A normal word with no leading punctuation. Just append it.

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

**Purpose:** Convert `' hello '` (quote as separate tokens) into `'hello'` (quotes attached to words).

`open` is a state flag — `false` means the next standalone `'` is an opening quote, `true` means it is a closing quote.

### Case 1 — opening quote

`w == "'"` and `open == false`. Prepend `'` to the next word: `words[i+1] = "'" + words[i+1]`. The current `'` token is consumed (not appended to result). `open` becomes `true`.

`i+1 < len(words)` — guard: do not prepend if there is no next word.

### Case 2 — closing quote (standalone)

`w == "'"` and `open == true`. Append `'` to the last word already in result. The `'` token is consumed. `open` becomes `false`.

### Case 3 — closing quote with punctuation attached

`fixPunctuation` runs before `fixQuotes`. If a closing `'` was immediately followed by `!?`, `fixPunctuation` glued them together into `'!?`. The standalone-`'` check in cases 1 and 2 would miss this.

This case catches it: `strings.HasPrefix(w, "'")` and `strings.Trim(w[1:], ".,!?:;") == ""` — the token starts with `'` and everything after the `'` is pure punctuation. Glue the whole token onto the previous word and close the quote.

**Why the `Trim` check is critical:** Without it, words like `'An` or `'apple` (which also start with `'` after the opening quote is prepended) would be mistakenly treated as closing quotes and glued to the previous word.

### Default

All other words — including words that now start with `'` because the opening-quote case prepended it — pass through normally.

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

        isUpper := bare[:1] == strings.ToUpper(bare[:1])

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

**Purpose:** Correct `a`/`an` to match the sound of the following word.

`i+1 < len(words)` — the loop stops one before the end because it always needs to look at `words[i+1]`.

### Stripping the prefix

After `fixQuotes` runs, the opening quote gets glued to the next token. An article like `a` becomes `'a`. A direct `strings.ToLower("'a")` gives `"'a"` which never equals `"a"`, so the article would be skipped.

`strings.TrimLeft(words[i], "'\".,!?:;")` — strips any leading punctuation characters, leaving just the bare article letter(s) in `bare`.

`prefix := words[i][:len(words[i])-len(bare)]` — captures the stripped prefix so it can be put back when writing the corrected article. If `words[i]` was `"'a"`, `bare` is `"a"` and `prefix` is `"'"`.

### Checking the article

`strings.ToLower(bare)` — makes the comparison case-insensitive. Both `"a"` and `"A"` match.

`bare[:1] == strings.ToUpper(bare[:1])` — checks whether the article's first letter is uppercase. `"A"` uppercased is still `"A"`, so they are equal. `"a"` uppercased is `"A"`, so they are not equal.

### Correcting the article

The `switch` covers all four combinations of vowel-sound and case. In every branch the prefix is prepended: `prefix + "An"` etc. This preserves the `'` on `'a` → `'An`.

`words[i] = prefix + "An"` — modifies the slice in place. `fixArticles` does not build a new `result` slice; it modifies `words` directly and returns it.

---

## 12. isVowelSound()

```go
const vowelSounds = "aeiouAEIOUhH"

func isVowelSound(word string) bool {
    clean := strings.TrimLeft(word, "'\".,!?:;")
    if clean == "" {
        return false
    }
    return strings.ContainsRune(vowelSounds, rune(clean[0]))
}
```

**Purpose:** Report whether a word begins with a vowel sound — which determines whether `a` or `an` is correct before it.

`const vowelSounds = "aeiouAEIOUhH"` — the vowels in both cases, plus `h`/`H`. The `h` is included because words like `hour`, `honest`, and `heir` begin with a silent h and are preceded by `an` in English.

`strings.TrimLeft(word, "'\".,!?:;")` — skip any leading punctuation on the next word (for example the next word might be `'hour` if it is inside a quote). The first actual letter is what matters for the sound.

`rune(clean[0])` — take the first byte of the cleaned string as a rune. This is safe for ASCII letters.

`strings.ContainsRune(vowelSounds, ...)` — check if that character is in the vowel set.

---

## 13. Pipeline Order — Why It Matters

```
applyModifiers → fixPunctuation → fixQuotes → fixArticles
```

The order is not arbitrary. Each stage depends on what the previous stage has already done.

**Modifiers must be first** because they change the content of words. If quotes were glued first, the modifier `(cap, 2)` might count a lone `'` as one of its target words and waste a slot. Running modifiers first on the raw token list means the `'` is still a separate token and `isWordToken` can skip it correctly.

**Punctuation must be second** so that punctuation gets attached to words before quotes are processed. If a closing `'` is immediately followed by `!?`, punctuation glues them into `'!?`. The third case in `fixQuotes` then handles this combined token. If punctuation ran after quotes, that case would never be needed — but then there would be a different problem: the closing `'` might be processed before the `!?` is attached.

**Quotes must be third** because article correction needs to see the final token shapes. After `fixQuotes`, an article inside a quote becomes `'a`. The `fixArticles` prefix-stripping logic depends on this being the token's actual form.

**Articles must be last** because they need to inspect the final form of the next word — after any modifier has transformed it and any quote has been glued to it — to determine the correct vowel sound.

---

## 14. Edge Cases Handled

| Situation | Input | Output | Where handled |
|-----------|-------|--------|---------------|
| Empty line | `""` | `""` | `processLine`: early return |
| Modifier with no preceding word | `(up) hello` | `hello` | `applyModifiers`: `len(result) > 0` guard |
| Count larger than available words | `a b (up, 10)` | `A B` | backwards loop stops at `j >= 0` |
| Invalid hex/binary | `ZZ (hex)` | `ZZ` | `transforms["(hex)"]`: returns `w` on parse error |
| Modifier with trailing punct | `(up),` | previous word uppercased + `,` | `clean`/`tail` split |
| Lone `'` at end of line | `hello '` | `hello '` | `fixQuotes` default case |
| Closing `'` with punct glued | `' hello '!?` | `'hello'!?` | `fixQuotes` case 3 |
| Article with quote prefix | `' a apple '` | `'an apple'` | `fixArticles` prefix strip |
| `h` words: hour, honest | `a hour` | `an hour` | `vowelSounds` includes `hH` |
| Uppercase article | `A apple` | `An apple` | `isUpper` check in `fixArticles` |
| Lone quote inside counted range | `' a b (cap, 2)` | `'A B` | `isWordToken` skips `'` |

---

## 15. Go Concepts Used

| Concept | Where used | What it does |
|---------|-----------|--------------|
| `map[string]func(string) string` | `transforms` | Maps tag names to transformation functions; both plain and counted modifiers share one lookup |
| Package-level `var` | `transforms` | Initialised once at startup, not rebuilt on every call |
| Inline functions (closures) | `(cap)`, `(hex)`, `(bin)` in transforms | Encapsulate the conversion logic right where the function is defined |
| `strings.Fields` | `processLine` | Splits on any whitespace and discards empty tokens — handles multiple spaces automatically |
| `strings.Split` / `strings.Join` | `processText` | Split file into lines, reassemble after processing |
| `strings.TrimRight` / `strings.TrimLeft` | `applyModifiers`, `fixArticles`, `isVowelSound` | Strip punctuation from one end of a string without touching the other end |
| `strings.Trim` | `applyModifiers` (count parsing), `fixPunctuation`, `fixQuotes` | Strip characters from both ends |
| `strings.TrimSuffix` | `applyModifiers` | Remove a specific suffix — `"(up,"` → `"(up"` |
| `strings.ContainsRune` | `isWordToken`, `fixPunctuation`, `isVowelSound` | Check if a single character appears in a string |
| `strings.HasPrefix` | `fixQuotes` | Check if a token starts with `'` |
| `strings.IndexFunc` | `fixPunctuation` | Find the first character matching a condition |
| `strconv.ParseInt(s, base, bits)` | `(hex)` and `(bin)` transforms | Parse a string as base-16 or base-2 integer |
| `strconv.Atoi` | `applyModifiers` | Parse the count string `"2"` as an integer |
| `strconv.Itoa` | `(hex)` and `(bin)` transforms | Convert an integer back to a decimal string |
| `for i, w := range words` then `words[i+1]` | `fixQuotes` | Range gives a copy in `w`; direct index `words[i+1]` modifies the slice element for the next iteration to see |
| In-place slice modification | `fixArticles` | Modifies `words[i]` directly instead of building a new slice |
| `switch` with no expression | `fixPunctuation`, `fixQuotes` | Each `case` is an independent boolean condition; cleaner than nested `if/else` |
| Early `return ""` | `processLine`, `isVowelSound` | Exit immediately when there is nothing to do |
| `os.ReadFile` / `os.WriteFile` | `main` | Read or write an entire file in one call — simple and sufficient for text files of reasonable size |
| `os.Exit(1)` | `main` | Signal error to the shell without using panic |
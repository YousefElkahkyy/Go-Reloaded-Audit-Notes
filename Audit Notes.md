# go-reloaded — Audit Notes

Notes taken during the senior code review of the go-reloaded project.
Covers every concept discussed: imports, functions, file I/O, permissions, packages, and more.

---

## Table of Contents

1. [Imports](#1-imports)
2. [main() — The Entry Point](#2-main--the-entry-point)
3. [go build vs go run](#3-go-build-vs-go-run)
4. [os.Exit — Exiting the Program](#4-osexit--exiting-the-program)
5. [os.ReadFile — Reading a File](#5-osreadfile--reading-a-file)
6. [os.WriteFile — Writing a File](#6-oswritefile--writing-a-file)
7. [Linux File Permissions](#7-linux-file-permissions)
8. [bufio — Buffered I/O](#8-bufio--buffered-io)
9. [strings.Fields vs strings.Split](#9-stringsfields-vs-stringssplit)
10. [map of functions — transforms](#10-map-of-functions--transforms)
11. [strconv.ParseInt — Parsing Numbers](#11-strconvparseint--parsing-numbers)
12. [strconv.Itoa — Integer to String](#12-strconvitoa--integer-to-string)
13. [strings.TrimSuffix](#13-stringstrimesuffix)
14. [unicode Package](#14-unicode-package)

---

## 1. Imports

```go
import (
    "bufio"    // Buffered I/O — reads large chunks into memory for faster processing
    "fmt"      // fmt.Println for CLI messages
    "os"       // os.Args, os.ReadFile, os.WriteFile, os.Exit
    "strconv"  // ParseInt, Atoi, Itoa — number/string conversions
    "strings"  // Splitting, joining, trimming, searching strings
    "unicode"  // IsLetter, IsDigit, IsPunct, ToLower, IsUpper
)
```

| Package | Role in the project |
|---------|-------------------|
| `bufio` | Buffered reading — reads data in chunks instead of byte by byte |
| `fmt` | Print messages to the terminal |
| `os` | CLI arguments, file reading/writing, program exit |
| `strconv` | Convert between strings and numbers |
| `strings` | All string manipulation — split, join, trim, search |
| `unicode` | Classify and convert individual characters (letters, digits, punctuation) |

---

## 2. main() — The Entry Point

```go
func main() {
    if len(os.Args) != 3 {
        fmt.Println("Usage: go run main.go <input> <output>")
        os.Exit(1)
    }
    ...
}
```

- `main()` is the **entry point** of every Go executable program.
- It takes **no arguments** and returns **no value**.
- `os.Args` is a slice of strings containing the command-line arguments:
  - `os.Args[0]` — the name of the program itself (e.g. `./main`)
  - `os.Args[1]` — the input file path
  - `os.Args[2]` — the output file path
- We need exactly **3 arguments**, so `len(os.Args) != 3` catches any wrong usage.

---

## 3. go build vs go run

| | `go run main.go` | `go build` |
|--|--|--|
| **What it does** | Compiles to a temporary location, runs it, then deletes it | Compiles and creates a permanent binary file in the current directory |
| **Output file** | None (deleted after running) | A permanent executable (`./main` or `main.exe`) |
| **Best for** | Development and quick testing | Deployment or running without the Go toolchain |

**Tip:** Add the `-x` flag to see everything happening behind the scenes:

```bash
go run -x main.go
```

---

## 4. os.Exit — Exiting the Program

```go
os.Exit(1)   // exit with error
os.Exit(0)   // exit with success
return        // in main() — also exits the program
```

| Method | Exit Code | Runs `defer`? | Typical Use |
|--------|-----------|--------------|-------------|
| `return` in `main()` | 0 (success) | ✅ Yes | Normal program completion |
| `os.Exit(0)` | 0 (success) | ❌ No | Immediate termination, no error |
| `os.Exit(1)` | 1 (failure) | ❌ No | Immediate termination due to an error |

**Key points:**

- Exit code `0` = success. Any non-zero value (1–255) = failure.
- `os.Exit` stops the program **instantly** — any `defer` calls are skipped.
- `return` in `main()` lets all `defer` functions finish before the program closes.
- `os.Exit` can be called from **anywhere** in the program, including nested functions.

**What is `defer`?**
A keyword that schedules a function to run just before the surrounding function returns. Used mainly for cleanup — closing files, releasing connections, etc.

```go
file, _ := os.Open("data.txt")
defer file.Close()  // runs automatically when the function exits
```

---

## 5. os.ReadFile — Reading a File

```go
data, err := os.ReadFile(os.Args[1])
```

**Signature:**
```go
func ReadFile(name string) ([]byte, error)
```

- **Takes:** a file path string.
- **Returns:** the file contents as `[]byte` and an `error`.
- Reads the **entire file into memory** in one call.
- `string(data)` converts the raw bytes into a usable string.

**Common error cases:**

| Error | Cause |
|-------|-------|
| File Not Found | The path does not exist |
| Permission Denied | The program cannot read the file |
| Path is a Directory | The path points to a folder, not a file |
| File Too Large | Not enough RAM to hold the whole file |
| I/O Failure | Hardware or disk-level error |

---

## 6. os.WriteFile — Writing a File

```go
err = os.WriteFile(os.Args[2], []byte(result), 0644)
```

**Signature:**
```go
func WriteFile(name string, data []byte, perm FileMode) error
```

- **Takes:** file path, content as `[]byte`, and file permissions.
- **Returns:** `nil` on success, or an error describing the failure.
- Handles opening the file, writing, and closing — all in one call.
- `[]byte(result)` converts the processed string back to bytes for writing.

**Common error cases:**

| Error | Cause |
|-------|-------|
| Permission Denied | No write access to that path or directory |
| Missing Directories | The parent directory does not exist |
| Disk Full | Not enough space on the storage device |
| Read-Only Filesystem | The target storage cannot be written to |
| Invalid File Path | The filename contains illegal characters |
| Incomplete Write | Power loss mid-write can corrupt the file |

---

## 7. Linux File Permissions

The permission `0644` in `os.WriteFile` is a Unix octal permission code.

**The three permission types:**

| Symbol | Name | Value |
|--------|------|-------|
| `r` | Read | 4 |
| `w` | Write | 2 |
| `x` | Execute | 1 |
| `-` | No permission | 0 |

**The three user classes (one digit each):**

| Position | Applies to |
|----------|-----------|
| 1st digit | Owner (the user who owns the file) |
| 2nd digit | Group (users in the file's assigned group) |
| 3rd digit | Others (everyone else) |

**To get a digit, add the values of permissions granted:**

| Octal | Binary | Symbolic | Meaning |
|-------|--------|----------|---------|
| 0 | 000 | `---` | No permissions |
| 1 | 001 | `--x` | Execute only |
| 2 | 010 | `-w-` | Write only |
| 3 | 011 | `-wx` | Write and Execute |
| 4 | 100 | `r--` | Read only |
| 5 | 101 | `r-x` | Read and Execute |
| 6 | 110 | `rw-` | Read and Write |
| 7 | 111 | `rwx` | Read, Write, and Execute |

**Common permission examples:**

| Code | Symbolic | Meaning |
|------|----------|---------|
| `755` | `rwxr-xr-x` | Owner: full access. Everyone else: read and execute. Common for scripts. |
| `644` | `rw-r--r--` | Owner: read/write. Everyone else: read only. Standard for text files. |
| `600` | `rw-------` | Owner only: read/write. Used for sensitive files like SSH keys. |
| `777` | `rwxrwxrwx` | Everyone: full access. Considered a security risk. |

So `0644` means: the owner can read and write the file, everyone else can only read it.

---

## 8. bufio — Buffered I/O

```go
scanner := bufio.NewScanner(strings.NewReader(input))
```

**What is `bufio`?**

`bufio` is a Go standard library package that implements **buffered I/O**. Instead of reading one byte at a time (which is slow because each read requires a system call), `bufio` reads a large chunk of data into a **buffer** in memory. Your program then reads from the buffer, which is much faster.

**What is `bufio.NewScanner`?**

A function that creates a `Scanner` — a tool for reading data line by line or word by word.

- **Takes:** any object that implements `io.Reader` (files, network connections, strings wrapped with `strings.NewReader`, etc.)
- **Returns:** a `*bufio.Scanner` pointer ready to scan.
- **Default behaviour:** reads one line at a time with `scanner.Scan()`.

**What is `strings.NewReader`?**

A raw string is just data. `strings.NewReader(input)` wraps it into an `io.Reader` — the format that `bufio.NewScanner` requires. By nesting them:

```go
bufio.NewScanner(strings.NewReader(input))
```

You are telling Go: *"Take this raw string, treat it like a file stream, and scan through it line by line."*

**Why the project moved away from `bufio`:**

Using `bufio.Scanner` + `strings.NewReader` for a string that is already in memory adds unnecessary complexity. `strings.Split(input, "\n")` does the same thing in one simple call with no extra types needed.

---

## 9. strings.Fields vs strings.Split

```go
words := strings.Fields(line)
```

**What `strings.Fields` does:**

Splits a string into words using any whitespace (spaces, tabs, newlines) as a delimiter, and ignores leading/trailing whitespace.

- **Takes:** a single string.
- **Returns:** a `[]string` slice of words. Returns an empty slice if the input is only whitespace.

**Comparison:**

| Feature | `strings.Fields(s)` | `strings.Split(s, " ")` |
|---------|--------------------|-----------------------|
| Delimiter | Any whitespace (`space`, `\t`, `\n`) | Only a single space `" "` |
| Multiple spaces | Merges them: `"a   b"` → `["a", "b"]` | Keeps each: `"a   b"` → `["a", "", "", "b"]` |
| Leading/trailing spaces | Ignored | Returns empty strings: `" a "` → `["", "a", ""]` |

**When to use which:**

Use `strings.Split` only if your data is perfectly formatted with exactly one space between every word. Use `strings.Fields` whenever the input might have extra spaces, tabs, or uneven spacing — it is safer and more robust.

---

## 10. map of functions — transforms

```go
var transforms = map[string]func(string) string{
    "(up)":  strings.ToUpper,
    "(cap)": func(word string) string {
        return strings.ToUpper(word[:1]) + strings.ToLower(word[1:])
    },
}
```

In Go, **functions are first-class citizens** — they can be stored in variables, passed as arguments, and used as map values just like any other type.

`map[string]func(string) string` is a map where:
- The **key** is a string (the modifier tag name, e.g. `"(up)"`)
- The **value** is any function that takes one string and returns one string

**How it works:**

```go
// Store a function
transforms["(up)"] = strings.ToUpper

// Look it up and call it
applyFunction, exists := transforms["(up)"]
if exists {
    result := applyFunction("hello")  // "HELLO"
}
```

**Why this is better than if/else or switch:**

| Without map (hard way) | With map (easy way) |
|----------------------|---------------------|
| Long `if/else` or `switch` that grows with every new modifier | One map lookup works for all modifiers |
| Adding a new modifier requires editing the logic | Adding a new modifier means adding one line to the map |
| Logic and data are mixed together | Data (the map) is separate from the logic (the loop) |

**Benefits:**
- Replaces complex branching logic with a single lookup.
- Easy to extend — add a new key/function without changing anything else.
- Makes the code declarative: *what* each tag does is defined in the map, *how* it is applied is in the loop.

---

## 11. strconv.ParseInt — Parsing Numbers

```go
strconv.ParseInt(w, 16, 64)
```

**Signature:**
```go
func ParseInt(s string, base int, bitSize int) (int64, error)
```

**The three arguments:**

| Argument | What it means |
|----------|--------------|
| `s` | The string to parse (e.g. `"FF"`, `"1010"`) |
| `base` | The number system: `16` = hexadecimal, `2` = binary, `10` = decimal, `0` = auto-detect |
| `bitSize` | The size to validate against: `8`, `16`, `32`, `64`. The result must fit in this many bits |

**What it returns:**

- `int64` — the parsed value (always `int64` regardless of `bitSize`).
- `error` — `nil` on success, a `*NumError` if the string is invalid or the value overflows.

**Why does it always return `int64`?**

Go does not support function overloading (multiple functions with the same name but different return types). `int64` is chosen because it is the largest signed integer type — it can hold any value from smaller types without losing data. The `bitSize` argument restricts the *range* without changing the return *type*.

**Bases explained:**

| Base | Number system | Example input |
|------|--------------|--------------|
| 2 | Binary | `"1010"` → 10 |
| 10 | Decimal | `"255"` → 255 |
| 16 | Hexadecimal | `"FF"` → 255 |

**`ParseInt` vs `Atoi`:**

`strconv.Atoi(s)` is a shortcut that calls `ParseInt(s, 10, 0)` and returns a plain `int` instead of `int64`. Use `Atoi` for simple decimal numbers; use `ParseInt` when you need a specific base or bit size.

---

## 12. strconv.Itoa — Integer to String

```go
strconv.Itoa(int(v))
```

- `Itoa` stands for **"Integer to ASCII"**.
- **Takes:** a single `int`.
- **Returns:** the decimal string representation of that integer.
- `int(v)` is needed because `ParseInt` returns `int64`, and `Itoa` only accepts `int`.

**Why `strconv.Itoa` over `fmt.Sprintf`?**

Both produce the same result, but they differ in performance:

| | `strconv.Itoa` | `fmt.Sprintf("%d", v)` |
|--|--|--|
| Speed | Faster — direct conversion | Slower — uses reflection to check the type at runtime |
| Memory | Fewer allocations | More allocations, more work for the garbage collector |
| Readability | Explicit intent | More general-purpose |

For converting a number to a string in a tight loop (like processing every word in a file), `strconv.Itoa` is the better choice.

---

## 13. strings.TrimSuffix

```go
tag := strings.TrimSuffix(w, ",") + ")"
```

**What it does:** removes a specific suffix from the end of a string — but only if the string actually ends with that suffix. If it does not, the original string is returned unchanged.

**Signature:**
```go
func TrimSuffix(s, suffix string) string
```

**Example in the project:**

`strings.Fields` splits `(up, 2)` into two tokens: `"(up,"` and `"2)"`.
To look up the modifier in the `transforms` map, we need the clean tag `"(up)"`.

```go
strings.TrimSuffix("(up,", ",") + ")"
// → "(up" + ")" → "(up)"
```

**Compared to `strings.TrimRight`:**

| | `strings.TrimSuffix(s, suffix)` | `strings.TrimRight(s, cutset)` |
|--|--|--|
| Removes | One specific suffix string | Any characters in the cutset |
| Example | `TrimSuffix("hello,", ",")` → `"hello"` | `TrimRight("hello,!.", ",!.")` → `"hello"` |
| Precision | Exact match only | Removes any matching chars from the right |

---

## 14. unicode Package

```go
if unicode.IsLetter(r) || unicode.IsDigit(r) {
```

The `unicode` package provides functions for testing and converting individual characters (runes). It is Unicode-aware — it works correctly for all scripts and languages, not just ASCII.

**Functions used in the project:**

| Function | What it checks | Example |
|----------|---------------|---------|
| `unicode.IsLetter(r)` | Is `r` a letter (any language)? | `IsLetter('a')` → `true` |
| `unicode.IsDigit(r)` | Is `r` a decimal digit 0–9? | `IsDigit('3')` → `true` |
| `unicode.IsPunct(r)` | Is `r` a punctuation character? | `IsPunct('.')` → `true` |
| `unicode.ToLower(r)` | Convert `r` to lowercase | `ToLower('A')` → `'a'` |
| `unicode.IsUpper(r)` | Is `r` an uppercase letter? | `IsUpper('A')` → `true` |

**Unicode categories referenced:**

| Category | Code | Meaning |
|----------|------|---------|
| Letter | `L` | Any letter in any script |
| Decimal digit | `Nd` | Digits 0–9 in any script |
| Punctuation | `P` | Any punctuation mark |
| Uppercase letter | `Lu` | Any uppercase letter |

**Why `unicode` over hardcoded strings?**

| Hardcoded string (old way) | unicode package (new way) |
|---------------------------|--------------------------|
| `strings.ContainsRune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789", r)` | `unicode.IsLetter(r) \|\| unicode.IsDigit(r)` |
| Must list every character manually | Works for every letter in every language |
| Easy to miss characters | Impossible to miss — the package knows the full Unicode standard |
| Fragile and long | Short, readable, and correct |

**Specific improvements in the project using `unicode`:**

`isWordToken` — replaced a 62-character hardcoded alphabet+digits string with two clean function calls.

`fixPunctuation` — `unicode.IsPunct(r)` finds the boundary between punctuation and letters inside a mixed token like `"!Hello"`.

`isVowelSound` — `unicode.ToLower(r)` normalises the first letter before a map lookup, so the vowels map only needs lowercase keys (`a e i o u h`) instead of listing both cases.

`fixArticles` — `unicode.IsUpper(rune(bare[0]))` checks the article's case directly, replacing a `[]rune` slice allocation that was only needed to read the first character.
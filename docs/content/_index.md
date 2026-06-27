---
title: "hagane"
description: "Go to C transpiler. Write Go, get readable C11."
---

# hagane 鋼

hagane compiles Go source to readable, portable C11.
You write normal Go, run `hagane build`, and get a native binary built by GCC or Clang.
No CGO, no LLVM, no hidden runtime linking.

## Install

**Go install**

```sh
go install github.com/tamnd/hagane/cmd/hagane@latest
```

**Homebrew (macOS / Linux)**

```sh
brew install tamnd/tap/hagane
```

**Scoop (Windows)**

```sh
scoop bucket add tamnd https://github.com/tamnd/scoop-bucket
scoop install hagane
```

**apt (Debian / Ubuntu)**

```sh
curl -fsSL https://pkg.tamnd.com/apt/key.gpg | sudo tee /etc/apt/keyrings/tamnd.gpg >/dev/null
echo "deb [signed-by=/etc/apt/keyrings/tamnd.gpg] https://pkg.tamnd.com/apt stable main" \
  | sudo tee /etc/apt/sources.list.d/tamnd.list
sudo apt update && sudo apt install hagane
```

**dnf (Fedora / RHEL)**

```sh
sudo dnf config-manager --add-repo https://pkg.tamnd.com/rpm/tamnd.repo
sudo dnf install hagane
```

## Quick start

```sh
# 1. write a Go program
cat > hello.go <<'EOF'
package main

import "fmt"

func main() {
    fmt.Println("Hello, World!")
}
EOF

# 2. build it
hagane build .

# 3. run the native binary
./hello
# Hello, World!
```

## Commands

| Command | What it does |
|---------|-------------|
| `hagane build ./pkg` | Transpile to C and compile to a binary |
| `hagane emit ./pkg` | Print the emitted C to stdout |
| `hagane run ./pkg` | Transpile, compile, and run in one step |
| `hagane check ./pkg` | Type-check only, no output files |

## Why hagane instead of CGO

CGO adds overhead on every Go-to-C call: goroutine stack switching, argument marshalling, and a minimum ~100 ns per call.
hagane eliminates the boundary entirely.
The emitted C is plain C11 with a small header (`hagane_rt.h`) for strings and slices.
You can read the output, audit it, or link it into a larger C project without any Go toolchain on the target machine.

## C interop

Declare external C functions with a `//go:extern` directive and call them from Go as normal functions.

```go
//go:extern int printf(const char *fmt, ...);

func main() {
    printf("hello from C\n")
}
```

hagane emits a matching C prototype and wires the call through directly, no wrapper needed.

## Status

M0 is complete: single-package programs with integers, strings, slices, structs, and functions work end to end.
Goroutines, channels, maps, and the standard library are on the roadmap.

See the [GitHub repository](https://github.com/tamnd/hagane) for the full roadmap and issue tracker.

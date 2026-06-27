# hagane 鋼

[![CI](https://github.com/tamnd/hagane/actions/workflows/ci.yml/badge.svg)](https://github.com/tamnd/hagane/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/tamnd/hagane)](https://github.com/tamnd/hagane/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

hagane compiles Go source to readable, portable C11.
You write normal Go, run `hagane build`, and the tool loads your packages via `go/packages`, builds SSA form, walks the SSA, and emits C11 that GCC or Clang can compile into a native binary.
The emitted C uses a small runtime header (`hagane_rt.h`) for strings, slices, and wrapping arithmetic.
No CGO, no LLVM, no intermediate IR files.

## Why hagane

### CGO call overhead

Every Go-to-C call through CGO crosses a runtime boundary: the goroutine stack is switched to a system thread, arguments are marshalled, and the scheduler is notified on return.
On a modern machine that costs roughly 100-200 ns per call.

| Approach | Call overhead | Notes |
|----------|--------------|-------|
| CGO | ~100-200 ns | Stack switch + marshalling per call |
| hagane | ~0 ns | No boundary; Go compiles to the same C translation unit |
| Pure Go | ~0 ns | No C at all |

At low call rates CGO is fine.
When you need tight loops that cross a C library boundary, or you want to ship a binary with no Go runtime attached, CGO overhead becomes the ceiling on your throughput.

### Interop boilerplate

A typical CGO wrapper for a single C function:

```go
// cgo approach
/*
#include <stdio.h>
*/
import "C"
import "unsafe"

func printMessage(s string) {
    cs := C.CString(s)
    defer C.free(unsafe.Pointer(cs))
    C.printf(cs)
}
```

The same thing with hagane:

```go
// hagane approach
//go:extern int printf(const char *fmt, ...);

func printMessage(s string) {
    printf(s)
}
```

No `import "C"`, no `unsafe`, no manual memory management for the crossing.

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

Or download a prebuilt binary from [Releases](https://github.com/tamnd/hagane/releases).

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
```

Output:

```
Hello, World!
```

## Usage

### `hagane build`

Transpile a Go package to C and compile to a binary.

```sh
hagane build ./myprogram
hagane build -o out ./myprogram
hagane build -cc clang ./myprogram
```

### `hagane emit`

Print the emitted C to stdout.
Useful for auditing the translation or feeding the C into another build system.

```sh
hagane emit ./myprogram
```

Example output (trimmed):

```c
#include "hagane_rt.h"

void main_main(void) {
    hagane_string s = hagane_str_lit("Hello, World!\n");
    fmt_Println(s);
}

int main(int argc, char **argv) {
    hagane_runtime_init(argc, argv);
    main_main();
    return 0;
}
```

### `hagane run`

Transpile, compile, and run in one step.

```sh
hagane run ./myprogram
hagane run ./myprogram -- --flag value
```

### `hagane check`

Type-check only.
No C is emitted, no compilation happens.
Exits non-zero if the package has type errors.

```sh
hagane check ./myprogram
```

## C interop

Declare external C functions with a `//go:extern` directive and call them from Go as normal functions.
hagane emits a C prototype and wires the call through directly.

```go
package main

//go:extern int printf(const char *fmt, ...);
//go:extern void exit(int status);

func main() {
    printf("pid: %d\n", getpid())
    exit(0)
}
```

You can also pull in a header:

```go
//go:include <stdio.h>
//go:include <stdlib.h>
```

hagane emits the corresponding `#include` at the top of the C file.

## How it works

1. Load Go packages with `go/packages` (full type information).
2. Build SSA form with `golang.org/x/tools/go/ssa`.
3. Walk the SSA and emit C11 into a single `.c` file per package.
4. Invoke GCC or Clang to compile and link.

The emitted C is meant to be readable.
Variable names are preserved where possible, control flow maps to C `if`/`for`/`goto`, and struct fields keep their Go names.
The runtime header (`hagane_rt.h`) provides `hagane_string`, `hagane_slice`, and a handful of arithmetic helpers.
There is no garbage collector in the emitted binary; memory layout is explicit.

## Status

**M0 complete** (current release): single-package programs with integers, strings, slices, structs, functions, and basic control flow work end to end.

Roadmap:

- M1: multiple packages, init functions, global variables
- M2: interfaces and type assertions
- M3: goroutines and channels (via a small C coroutine library)
- M4: maps
- M5: standard library stubs (fmt, os, io, bufio, strings, strconv)
- M6: generics

See [issues](https://github.com/tamnd/hagane/issues) and [docs](https://hagane.tamnd.com/) for the full roadmap.

## Contributing

Bug reports and test cases are the most useful contributions right now.
If you find a Go program that hagane translates incorrectly, open an issue with a minimal reproducer.

## License

MIT. See [LICENSE](LICENSE).

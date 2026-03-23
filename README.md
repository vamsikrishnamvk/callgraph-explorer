# Call Graph Explorer

An interactive, browser-based function call graph explorer for large codebases.

Point it at any **Go** repository, parse it in minutes, and explore every function's
call chain interactively in the browser — click a function to see what it calls and
what calls it, drill in level by level, and navigate back with a breadcrumb trail.

> Tested on [Teleport](https://github.com/gravitational/teleport) —
> **3,719 files · 68,767 functions** parsed in under 2 minutes.

---

## Demo

```
Search: "NewServer"
  └─ Click result → focus on auth.NewServer
       ├─ [green] validateConfig     ← click to drill in
       ├─ [green] NewLogger
       ├─ [green] setupTLS
       └─ [orange] main.run          ← who calls NewServer
```

Each click re-centres the graph on the selected function.
The breadcrumb at the top lets you retrace your steps.

---

## Supported Languages

| Language | Status | Parser |
|----------|--------|--------|
| **Go** | ✅ Supported | Uses Go's built-in `go/ast` — no extra deps |
| TypeScript / JavaScript | 🔜 Planned | Same JSON output format, viewer works as-is |
| Python | 🔜 Planned | Same JSON output format, viewer works as-is |
| Other | 🤝 Contribute | See [Adding a new language parser](#adding-a-new-language-parser) |

The HTML viewer is **language-agnostic** — it reads a simple JSON format.
Anyone can write a parser for their language and plug it straight in.

---

## How it works

```
┌──────────────────────────────────────────────────┐
│  1. Parser  (Go binary)                          │
│     Walks .go files → go/ast → callgraph.json    │
└─────────────────────┬────────────────────────────┘
                      │  callgraph.json
┌─────────────────────▼────────────────────────────┐
│  2. Viewer  (index.html)                         │
│     React + D3 force graph                       │
│     Served by Python's built-in HTTP server      │
│     No build step, no npm, no Node.js needed     │
└──────────────────────────────────────────────────┘
```

---

## Prerequisites

| Tool | Required for | Install |
|------|-------------|---------|
| **Go 1.18+** | Compiling the parser | https://go.dev/dl |
| **Python 3** | Serving the viewer | Usually pre-installed on Mac/Linux. Windows: https://python.org |
| A modern browser | Viewing the graph | Chrome, Firefox, Safari, Edge |

---

## Quick Start

### macOS / Linux

```bash
# 1. Clone this repo
git clone https://github.com/vamsikrishnamvk/callgraph-explorer.git
cd callgraph-explorer

# 2. Make scripts executable
chmod +x parse.sh serve.sh

# 3. Parse your Go repo
./parse.sh --repo /path/to/your/go/repo --output callgraph.json

# 4. Start the viewer
./serve.sh
# → open http://localhost:8080 in your browser
```

### Windows

```bat
REM 1. Clone this repo
git clone https://github.com/vamsikrishnamvk/callgraph-explorer.git
cd callgraph-explorer

REM 2. Build the parser
cd parser
go build -o ..\parse.exe .
cd ..

REM 3. Parse your Go repo
parse.exe --repo "C:\path\to\your\go\repo" --output callgraph.json

REM 4. Start the viewer (or double-click serve.bat)
python -m http.server 8080
REM → open http://localhost:8080 in your browser
```

---

## Detailed Usage

### Step 1 — Build the parser

**macOS / Linux**
```bash
cd parser
go build -o ../parse .
cd ..
```
Or just use `./parse.sh` — it auto-builds if the binary is missing or the source changed.

**Windows**
```bat
cd parser
go build -o ..\parse.exe .
cd ..
```

---

### Step 2 — Parse a repository

```bash
# macOS / Linux
./parse.sh --repo /path/to/repo --output callgraph.json

# Windows
parse.exe --repo "C:\path\to\repo" --output callgraph.json
```

**All flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--repo` | `.` | Path to the root of the Go repository |
| `--output` | `callgraph.json` | Output JSON file path |
| `--pkg` | _(all)_ | Only parse files whose path contains this string (e.g. `lib/auth`) |
| `--max` | `0` (all) | Cap the number of files parsed — useful for a quick test run |
| `--tests` | `false` | Include `_test.go` files |

**Examples:**

```bash
# Parse the whole repo
./parse.sh --repo ../teleport --output callgraph.json

# Only parse the auth subsystem of a large repo
./parse.sh --repo ../teleport --pkg lib/auth --output auth.json

# Quick smoke-test on first 200 files
./parse.sh --repo ../teleport --max 200 --output sample.json
```

---

### Step 3 — Serve and open

**macOS / Linux**
```bash
./serve.sh
# or manually:
python3 -m http.server 8080
```

**Windows**
```bat
serve.bat
REM or manually:
python -m http.server 8080
```

Open **http://localhost:8080** in your browser.

> **Why a server?** Browsers block `fetch()` from `file://` URLs for security reasons.
> Python's built-in server is the zero-install solution — nothing extra to install.

---

## Using the Viewer

### Search

Type any function or package name in the **search bar** (sidebar, top-left).
Results update as you type. Click any result to focus on that function.

### The Graph

| Visual | Meaning |
|--------|---------|
| Large blue node (centre) | The function currently in focus |
| Green nodes | Functions this function **calls** (callees) |
| Orange nodes | Functions that **call** this function (callers) |
| Coloured outer ring | Each package gets a unique colour |
| Arrow direction | Direction of the call |
| Dashed arrows | Calls to/from external packages |

### Interactions

| Action | Result |
|--------|--------|
| Click a node | That function becomes the new focus |
| Hover a node | Tooltip: package, file path, line number |
| Drag a node | Reposition it on the canvas |
| Scroll / pinch | Zoom in and out |
| Click breadcrumb item | Jump back to that point in history |
| Click **← Back** | One step back |

### Sidebar

Shows the focused function's full details:
- Name, receiver type (for methods), package
- File path and line number
- **Calls** list (green) — every function it calls; click any to navigate into it
- **Called by** list (orange) — every function that calls it; click any to trace upstream

### Loading a different callgraph

Click **"Load different callgraph.json"** at the bottom of the sidebar to open any
`.json` file — useful for switching between repos or subsystems without restarting.

---

## Working with multiple repos

Generate a file per repo and switch between them in the viewer:

```bash
./parse.sh --repo ../teleport      --output teleport.json
./parse.sh --repo ../kubernetes    --output k8s.json
./parse.sh --repo ../my-service    --output my-service.json
```

Use **"Load different callgraph.json"** to switch — no server restart needed.

---

## Refreshing after a `git pull`

Just re-run the parser and refresh the browser:

```bash
cd /your/go/repo && git pull
cd /path/to/callgraph-explorer
./parse.sh --repo /your/go/repo --output callgraph.json
# refresh browser
```

---

## Output JSON Format

```json
{
  "repo": "teleport",
  "totalFiles": 3719,
  "totalFunctions": 68767,
  "functions": {
    "auth.NewServer": {
      "id":      "auth.NewServer",
      "name":    "NewServer",
      "recv":    "",
      "pkg":     "auth",
      "file":    "lib/auth/server.go",
      "line":    123,
      "calls":   ["auth.validateConfig", "utils.NewLogger"],
      "callers": ["main.run"]
    }
  }
}
```

This schema is stable. Any parser that outputs this format works with the viewer.

---

## Adding a New Language Parser

Want to add TypeScript, Python, Rust, Java, etc.?

1. Write a parser (in any language) that outputs the JSON schema above
2. The `functions` map key is `"package.FunctionName"` (or any unique string per function)
3. Populate `calls` and `callers` as arrays of those same keys
4. Drop the output JSON into this folder and open it with the viewer

That's it — the viewer needs no changes.

**Suggested approach per language:**

| Language | Suggested parser tool |
|----------|-----------------------|
| TypeScript / JavaScript | `@typescript-eslint/typescript-estree` or `ts-morph` |
| Python | `ast` module (standard library) |
| Rust | `syn` crate |
| Java | `JavaParser` library |
| C / C++ | `libclang` / `clangd` |
| Any | `tree-sitter` (universal, has bindings for most languages) |

PRs welcome!

---

## Known Limitations

- **Go only** for now — TypeScript/Python parsers are planned.
- **Best-effort call resolution** — uses name matching, not full type checking.
  Calls through interfaces, function variables, or reflection are not captured.
- **External calls filtered out** — only calls between functions within the repo
  appear in the graph (stdlib and third-party calls are excluded).
- **Large repos** (60k+ functions) produce JSON files of 15–20 MB.
  The viewer handles this fine but the initial load takes a couple of seconds.

---

## Project Structure

```
callgraph-explorer/
├── parser/
│   ├── main.go       Go AST parser
│   └── go.mod
├── index.html        React + D3 viewer (zero build-step, CDN only)
├── parse.sh          Build + run script (macOS / Linux)
├── serve.sh          Python HTTP server script (macOS / Linux)
├── parse_teleport.bat  Example parse script (Windows)
├── serve.bat         Python HTTP server script (Windows)
└── README.md
```

---

## Contributing

1. Fork the repo
2. Create a feature branch: `git checkout -b feat/python-parser`
3. Make your changes
4. Open a pull request

Ideas welcome: new language parsers, UI improvements, performance tweaks.

---

## License

MIT

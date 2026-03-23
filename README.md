# Code Explorer

Three interactive, browser-based analysis tools for Go repositories — all in one viewer,
no build step required.

| Tab | What it shows |
|-----|---------------|
| **Call Graph** | Every function and who calls whom. Click to drill in level by level. |
| **Dep Graph** | Every internal package and which packages it imports. Click to explore dependencies. |
| **Hotspots** | A heatmap treemap of your codebase — files sized by lines of code, coloured by git commit churn (green = stable, red = hotspot). |

> Tested on [Teleport](https://github.com/gravitational/teleport) —
> **3,719 files · 68,767 functions · 942 packages**

---

## Demo

**Call Graph**
```
Search: "NewServer"
  └─ Click → focus on auth.NewServer
       ├─ [green] validateConfig    ← click to drill deeper
       ├─ [green] NewLogger
       └─ [orange] main.run         ← who calls this function
```

**Dep Graph**
```
Search: "auth"
  └─ Click → focus on lib/auth package
       ├─ [green] lib/utils         ← packages auth imports
       ├─ [green] lib/tlsca
       └─ [orange] lib/proxy        ← packages that import auth
```

**Hotspots**
```
Treemap — each rectangle is a file
  Size   = lines of code
  Colour = git churn (green → yellow → red)
  Click directory → zoom in
  Click file      → highlight in sortable table
```

Each view has a breadcrumb trail and a Back button to retrace navigation.

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

The parser has three modes. Run whichever analyses you want:

```bash
# macOS / Linux
./parse.sh --mode callgraph --repo /path/to/repo   # → callgraph.json
./parse.sh --mode deps      --repo /path/to/repo   # → deps.json
./parse.sh --mode hotspot   --repo /path/to/repo   # → hotspot.json

# Windows
parse.exe --mode callgraph --repo "C:\path\to\repo"
parse.exe --mode deps      --repo "C:\path\to\repo"
parse.exe --mode hotspot   --repo "C:\path\to\repo"
```

> **Note on hotspot mode:** requires a full git clone with history.
> A `--depth 1` shallow clone will show only 1 commit per file.
> For meaningful churn data, use a full clone or `--depth 500` or deeper.

**All flags:**

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `callgraph` | Analysis mode: `callgraph`, `deps`, or `hotspot` |
| `--repo` | `.` | Path to the root of the Go repository |
| `--output` | `<mode>.json` | Output JSON file path |
| `--pkg` | _(all)_ | Only parse files whose path contains this string — `callgraph` mode only |
| `--max` | `0` (all) | Cap number of files — `callgraph` mode only |
| `--tests` | `false` | Include `_test.go` files — `callgraph` mode only |

**Examples:**

```bash
# Generate all three analyses for a repo
./parse.sh --mode callgraph --repo ../myrepo
./parse.sh --mode deps      --repo ../myrepo
./parse.sh --mode hotspot   --repo ../myrepo

# Only parse the auth subsystem of a large repo (call graph)
./parse.sh --mode callgraph --repo ../teleport --pkg lib/auth --output auth.json

# Quick smoke-test on first 200 files
./parse.sh --mode callgraph --repo ../teleport --max 200 --output sample.json
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

The viewer has three tabs at the top. Each tab lazy-loads its own JSON file the
first time you switch to it, or you can click the **Open** button in the sidebar
to load any `.json` file manually.

---

### Call Graph tab

**Search** — type any function or package name in the sidebar search bar.

| Visual | Meaning |
|--------|---------|
| Large blue node | Function currently in focus |
| Green nodes | Functions this function **calls** |
| Orange nodes | Functions that **call** this function |
| Coloured ring | Package colour (each package gets a unique colour) |

| Action | Result |
|--------|--------|
| Click a node | That function becomes the new focus |
| Hover a node | Tooltip: package, file, line number |
| Drag nodes | Reposition on canvas |
| Scroll / pinch | Zoom |
| Click breadcrumb | Jump back in history |

The sidebar shows the full **Calls** and **Called by** lists — click any item to navigate.

---

### Dep Graph tab

Same interaction model as Call Graph, but nodes are **packages** not functions.

| Visual | Meaning |
|--------|---------|
| Blue node | Package in focus |
| Green nodes | Packages this package **imports** |
| Orange nodes | Packages that **import** this package |

The sidebar shows import count, file count, and full import/imported-by lists.

---

### Hotspots tab

A D3 treemap where every file in the repo is a rectangle.

| Visual | Meaning |
|--------|---------|
| Rectangle size | Lines of code |
| Green colour | Low git churn (stable file) |
| Red colour | High git churn (frequently changed = hotspot) |

| Action | Result |
|--------|--------|
| Click a directory area | Zoom into that directory |
| Click repo name in breadcrumb | Zoom back to root |
| Click a file rectangle | Highlight it in the table |
| Hover | Tooltip: path, commits, authors, LOC |

The **sortable table** in the sidebar lists all files sorted by commit count by default.
Click column headers to re-sort by authors or lines of code. Type to filter by filename.

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

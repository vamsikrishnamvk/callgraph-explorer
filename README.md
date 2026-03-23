# Code Explorer

Three interactive, browser-based analysis tools for any Go repository — all in one
viewer, zero build step, no Node.js required.

| Tab | What it does |
|-----|-------------|
| **Call Graph** | Maps every function and who calls whom. Search any function, click to drill into its call chain level by level. |
| **Dep Graph** | Maps every internal package and what it imports. Find dependency clusters, see what a package pulls in, trace who depends on you. |
| **Hotspots** | Treemap of the whole codebase — rectangles sized by lines of code and coloured by git commit churn. Green = stable. Red = the files your team fights with most. |

> Tested on [Teleport](https://github.com/gravitational/teleport) —
> **3,719 files · 68,767 functions · 942 packages · 13,410 files in hotspot map**

---

## Table of Contents

1. [How it works](#how-it-works)
2. [Prerequisites](#prerequisites)
3. [Quick Start — macOS / Linux](#quick-start--macos--linux)
4. [Quick Start — Windows](#quick-start--windows)
5. [Step-by-step Setup](#step-by-step-setup)
6. [Feature: Call Graph](#feature-call-graph)
7. [Feature: Dependency Graph](#feature-dependency-graph)
8. [Feature: Hotspot Heatmap](#feature-hotspot-heatmap)
9. [Working with Multiple Repos](#working-with-multiple-repos)
10. [Parser Reference](#parser-reference)
11. [JSON Output Schemas](#json-output-schemas)
12. [Adding a New Language Parser](#adding-a-new-language-parser)
13. [Known Limitations](#known-limitations)
14. [Project Structure](#project-structure)
15. [Contributing](#contributing)

---

## How it works

```
┌──────────────────────────────────────────────────────────┐
│  parse  (Go binary — go/ast + git log)                   │
│                                                          │
│  --mode callgraph  →  callgraph.json  (functions)        │
│  --mode deps       →  deps.json       (packages)         │
│  --mode hotspot    →  hotspot.json    (git churn tree)   │
└──────────────────────────┬───────────────────────────────┘
                           │  JSON files
┌──────────────────────────▼───────────────────────────────┐
│  index.html  (React 18 + D3 v7, loaded from CDN)         │
│  Served by Python's built-in HTTP server                 │
│  Three tabs — lazy-loads each JSON on first switch       │
└──────────────────────────────────────────────────────────┘
```

The parser uses only Go's **standard library** (`go/ast`, `go/parser`, `os/exec`).
The viewer uses only **CDN scripts** (React, D3, Babel standalone).
Nothing to install beyond Go and Python.

---

## Prerequisites

| Tool | Version | Why | Install |
|------|---------|-----|---------|
| **Go** | 1.18+ | Compile the parser | https://go.dev/dl |
| **Python** | 3.x | Serve `index.html` over HTTP | Pre-installed on macOS/Linux. Windows: https://python.org |
| **git** | any | Required for hotspot mode only | Pre-installed on most systems |
| A browser | modern | Chrome, Firefox, Safari, Edge | — |

---

## Quick Start — macOS / Linux

```bash
# 1. Clone this tool
git clone https://github.com/vamsikrishnamvk/callgraph-explorer.git
cd callgraph-explorer
chmod +x parse.sh serve.sh

# 2. Generate all three analyses for your repo
./parse.sh --mode callgraph --repo /path/to/your/go/repo
./parse.sh --mode deps      --repo /path/to/your/go/repo
./parse.sh --mode hotspot   --repo /path/to/your/go/repo

# 3. Serve and open
./serve.sh
# open http://localhost:8080
```

---

## Quick Start — Windows

```bat
REM 1. Clone this tool
git clone https://github.com/vamsikrishnamvk/callgraph-explorer.git
cd callgraph-explorer

REM 2. Build the parser
cd parser
go build -o ..\parse.exe .
cd ..

REM 3. Generate all three analyses for your repo
parse.exe --mode callgraph --repo "C:\path\to\your\go\repo"
parse.exe --mode deps      --repo "C:\path\to\your\go\repo"
parse.exe --mode hotspot   --repo "C:\path\to\your\go\repo"

REM 4. Serve and open (or double-click serve.bat)
python -m http.server 8080
REM open http://localhost:8080
```

---

## Step-by-step Setup

### 1. Build the parser

**macOS / Linux**
```bash
cd parser && go build -o ../parse . && cd ..
```
`parse.sh` auto-builds for you if the binary is missing or `main.go` is newer.

**Windows**
```bat
cd parser
go build -o ..\parse.exe .
cd ..
```

---

### 2. Run the analysis modes

You can run one, two, or all three depending on what you need.
Each produces an independent JSON file.

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

Progress is printed to the terminal. Typical times on a large repo (Teleport):

| Mode | Time | Output size |
|------|------|-------------|
| callgraph | ~90 sec | 17 MB |
| deps | ~15 sec | 1.5 MB |
| hotspot | ~2–5 min (depends on git history depth) | 4 MB |

---

### 3. Start the viewer

> **Why a local server?** Browsers block `fetch()` on `file://` URLs. Python's built-in
> server is the zero-dependency solution.

**macOS / Linux**
```bash
./serve.sh          # starts on port 8080
# or: python3 -m http.server 8080
```

**Windows**
```bat
serve.bat           # starts on port 8080
REM or: python -m http.server 8080
```

Open **http://localhost:8080** in your browser.
The viewer auto-loads `callgraph.json` on startup. The Dep Graph and Hotspot tabs
lazy-load their JSON files the first time you click them.

---

## Feature: Call Graph

### What it is

A function-level call graph of your entire Go codebase. Every function and method is
a node. Every call site creates a directed edge. You navigate it like a web — pick any
function and explore outward.

### Generate the data

```bash
# Full repo
./parse.sh --mode callgraph --repo /path/to/repo

# Large repo — focus on one subsystem
./parse.sh --mode callgraph --repo /path/to/repo --pkg lib/auth

# Include test functions
./parse.sh --mode callgraph --repo /path/to/repo --tests

# Quick test on first 200 files
./parse.sh --mode callgraph --repo /path/to/repo --max 200
```

Output: `callgraph.json` (default). Override with `--output myrepo.json`.

### The graph

When you focus on a function, the graph shows three groups of nodes:

```
                    [orange] caller A ──┐
                    [orange] caller B ──┼──► [BLUE] YourFunction ──► [green] callee X
                    [orange] caller C ──┘                        └──► [green] callee Y
                                                                  └──► [green] callee Z
```

| Node colour | Meaning |
|-------------|---------|
| **Blue** (large, centre) | The function you are currently focused on |
| **Green** | Functions this function calls (callees) |
| **Orange** | Functions that call this function (callers) |
| **Coloured outer ring** | Package colour — each package gets a unique persistent colour |

### Interactions

| Action | What happens |
|--------|-------------|
| Type in search bar | Fuzzy-search across all function and package names |
| Click a search result | That function becomes the focus, graph renders around it |
| Click any node | That function becomes the new focus |
| Click a green node | Drill deeper into the call chain |
| Click an orange node | Trace upstream — who calls my caller? |
| Hover any node | Tooltip: full package name, file path, line number |
| Drag a node | Move it anywhere on the canvas |
| Scroll / pinch | Zoom in and out |
| Click a breadcrumb item | Jump back to that point in your navigation history |
| Click **← Back** | One step back |
| **Open** button (sidebar bottom) | Load any `.json` file from disk |

### Sidebar

The sidebar shows the focused function's full details:

- **Package** — with its assigned colour
- **File : line** — exact source location
- **Calls** (green list) — every internal function it calls, click any to navigate into it
- **Called by** (orange list) — every known caller, click any to trace upstream

### Typical workflow

```
1. Search for an entry point: "main", "Run", "Start", "New..."
2. Click a result → see what it calls
3. Click a callee → drill one level deeper
4. Notice an unfamiliar function? Click it immediately
5. Use the breadcrumb to retrace: main > Run > StartServer > handleConn
6. Use the orange "Called by" list to find all the places that use a function
```

### Tips

- **Finding dead code** — functions with 0 callers and 0 calls are likely unused
- **Understanding an unfamiliar codebase** — start from `main.main` or the top-level
  `New...` constructor and drill down
- **Code review** — paste a function name into search to instantly see its full call
  surface before reviewing the PR
- **Refactoring** — check "Called by" count before moving or renaming a function

---

## Feature: Dependency Graph

### What it is

A package-level import graph of your Go codebase. Every internal package is a node.
Every `import` statement between two internal packages creates a directed edge.
Third-party and stdlib imports are excluded so the graph stays focused on your code.

### Generate the data

```bash
./parse.sh --mode deps --repo /path/to/repo
# → deps.json
```

No extra flags needed. The parser reads `go.mod` to determine the module name and
automatically filters out all imports that aren't part of your own module.

### The graph

When you focus on a package, the graph shows:

```
         [orange] lib/proxy ──┐
         [orange] tool/tctl ──┼──► [BLUE] lib/auth ──► [green] lib/utils
                              │                    └──► [green] lib/tlsca
                                                   └──► [green] api/types
```

| Node colour | Meaning |
|-------------|---------|
| **Blue** (large, centre) | The package you are currently focused on |
| **Green** | Packages this package imports |
| **Orange** | Packages that import this package (its dependents) |
| **Coloured ring** | Top-level directory colour (e.g., all `lib/` packages share a hue) |

The tooltip shows: short package name, full directory path, and number of `.go` files.

### Interactions

| Action | What happens |
|--------|-------------|
| Type in search bar | Search by package name or directory path |
| Click a search result | Focus on that package |
| Click a green node | Explore what that imported package itself imports |
| Click an orange node | See who depends on this package |
| Hover | Tooltip: dir path and file count |
| Drag / scroll | Reposition and zoom |
| Breadcrumb / Back | Retrace navigation |

### Sidebar

- **Dir** — repository-relative path (e.g., `lib/auth`)
- **Files** — number of `.go` files in this package
- **Imports** (green list) — packages this one imports; click any to navigate
- **Imported by** (orange list) — packages that import this one; click any to navigate

### Typical workflow

```
1. Search for a package you're about to change: e.g. "utils"
2. Click it → check "Imported by" — how many things depend on it?
3. If "Imported by" is large, your change has wide blast radius
4. Click each dependent to understand the chain

— or —

1. Search for a high-level package: e.g. "proxy", "server"
2. Click it → see all the packages it pulls in (green)
3. Look for unexpected dependencies (e.g. a UI package importing a DB layer)
4. Drill into any green node to understand why it's needed
```

### Tips

- **Circular dependency risk** — if package A imports B and B imports A, Go won't compile.
  Use this graph to spot packages that are trending toward a cycle before they break the build.
- **Blast radius before refactoring** — check "Imported by" count before touching a shared
  package. High importedBy = high risk change.
- **Finding god packages** — packages with 10+ importers are often doing too much. Good
  candidates for splitting.
- **Onboarding** — show a new team member the dep graph to explain the system's layering
  (e.g. `api` → `lib` → `utils`) before they read any code.

---

## Feature: Hotspot Heatmap

### What it is

A treemap where every file in your repo is a rectangle. The **size** of each rectangle
is proportional to its current lines of code. The **colour** is based on how many git
commits have touched that file — green means stable, red means it gets changed
constantly.

Files that are both **large and red** are your highest-risk areas: lots of code, lots
of churn, and lots of history to understand.

### Generate the data

```bash
./parse.sh --mode hotspot --repo /path/to/repo
# → hotspot.json
```

> **Important — shallow clones:** If you cloned with `git clone --depth 1`, the repo has
> only 1 commit in its history. Every file will show `commits: 1` and the heatmap will
> be uniformly green, which is not useful.
>
> To get real churn data, unshallow the clone first:
> ```bash
> cd /path/to/repo
> git fetch --unshallow
> cd /path/to/callgraph-explorer
> ./parse.sh --mode hotspot --repo /path/to/repo
> ```
> On a large repo like Teleport, unshallowing downloads ~500 MB and takes a few minutes
> but the resulting heatmap shows real historical hotspots.

### Reading the heatmap

```
┌─────────────────────────────────────────────────────────────┐
│  lib/                                                        │
│  ┌──────────────────────┐  ┌──────────┐  ┌───────────────┐ │
│  │ auth/                │  │ srv/     │  │ utils/        │ │
│  │ ┌──────────────────┐ │  │ ┌──────┐ │  │ ┌───┐ ┌────┐ │ │
│  │ │ server.go        │ │  │ │srv.go│ │  │ │   │ │    │ │ │
│  │ │  🔴 HIGH CHURN   │ │  │ │  🟠  │ │  │ │🟢 │ │ 🟢 │ │ │
│  │ └──────────────────┘ │  │ └──────┘ │  │ └───┘ └────┘ │ │
│  └──────────────────────┘  └──────────┘  └───────────────┘ │
└─────────────────────────────────────────────────────────────┘
  Size of box = lines of code        Colour = commit churn
```

| Colour | Churn level | What it means |
|--------|-------------|---------------|
| 🟢 Dark green | Very low | File is stable — rarely touched |
| 🟡 Yellow-green | Medium | Moderate change frequency |
| 🟠 Orange | High | Frequently modified |
| 🔴 Red | Very high | Changes almost every sprint — highest risk |

### Interactions

| Action | What happens |
|--------|-------------|
| Hover a file rectangle | Tooltip: file path, commit count, unique authors, lines of code |
| Hover a directory area | Tooltip: dir path, file count, total LOC, peak churn |
| Click a directory | Zoom into that directory — fills the whole canvas |
| Click the repo name in breadcrumb | Zoom back out to the full repo view |
| Click any breadcrumb segment | Jump to that directory level |
| Click a file rectangle | Highlights that file in the sidebar table |

### Sidebar table

The left sidebar shows a sortable table of all files:

| Column | Description |
|--------|-------------|
| **File** | Filename and parent directory |
| **Commits** | Number of commits that touched this file |
| **Authors** | Number of unique contributors |
| **LOC** | Current lines of code |
| Heat bar | Visual churn indicator |

**Sorting:** Click any column header to sort. Click again to reverse.
**Filtering:** Type in the filter box to search by filename or directory.
**Selecting:** Click any row to highlight that file on the treemap.

### Typical workflow

```
1. Open the Hotspots tab
2. Look for the largest red rectangles — these are your highest-risk files
3. Hover them to see commit count and author count
4. Click a red directory to zoom in — which specific files are hottest?
5. Click a file → it highlights in the table
6. Note the author count — high authors + high churn = coordination overhead
7. Use this to prioritise what to refactor, test, or document first

— or, for onboarding —

1. Open Hotspots and look at the overall shape of the codebase
2. The biggest rectangles = most code (good starting points for reading)
3. Green large rectangles = stable, well-understood code
4. Red areas = where the team currently spends most of their time
```

### Tips

- **Pre-sprint planning** — if you're about to touch a red file, budget more time for
  review and testing. It's complex by definition.
- **Test coverage prioritisation** — red + large + few authors = highest value place to
  add tests.
- **Tech debt tracking** — watch if a previously green file starts turning orange across
  sprints.
- **Hiring / onboarding** — show new team members the hotspot map in their first week.
  "Start in the green areas, approach the red areas with a buddy" is a concrete, visual
  piece of advice.
- **Post-incident review** — was the incident in a red file? Probably not a surprise.

---

## Working with Multiple Repos

Generate separate JSON files per repo and switch between them using the
**Open** button at the bottom of the sidebar — no server restart needed.

```bash
# Generate call graphs for two different repos
./parse.sh --mode callgraph --repo ../teleport   --output teleport-cg.json
./parse.sh --mode callgraph --repo ../my-service --output myservice-cg.json

# Generate dep graphs
./parse.sh --mode deps --repo ../teleport   --output teleport-deps.json
./parse.sh --mode deps --repo ../my-service --output myservice-deps.json
```

In the viewer, click **Open callgraph.json** → pick `teleport-cg.json` or
`myservice-cg.json` to switch instantly.

### Refresh after `git pull`

Re-run only the modes you need and refresh the browser:

```bash
cd /your/go/repo && git pull && cd -
./parse.sh --mode callgraph --repo /your/go/repo
./parse.sh --mode hotspot   --repo /your/go/repo   # if history changed
# Ctrl+Shift+R to hard-refresh the browser
```

---

## Parser Reference

### All flags

| Flag | Default | Modes | Description |
|------|---------|-------|-------------|
| `--mode` | `callgraph` | all | `callgraph`, `deps`, or `hotspot` |
| `--repo` | `.` | all | Path to the root of the Go repository |
| `--output` | `<mode>.json` | all | Output JSON file path |
| `--pkg` | _(all)_ | callgraph | Only parse files whose path contains this string |
| `--max` | `0` (all) | callgraph | Cap number of `.go` files parsed |
| `--tests` | `false` | callgraph | Include `_test.go` files |

### Examples

```bash
# Parse a specific subsystem (much faster on large repos)
./parse.sh --mode callgraph --repo ../teleport --pkg lib/auth --output auth.json

# Quick sanity check on the first 200 files
./parse.sh --mode callgraph --repo ../teleport --max 200 --output sample.json

# Include tests to see test helper call chains
./parse.sh --mode callgraph --repo ../myrepo --tests

# Override output path
./parse.sh --mode deps --repo ../myrepo --output myrepo-packages.json

# Run all three in sequence (bash)
for mode in callgraph deps hotspot; do
  ./parse.sh --mode $mode --repo ../myrepo
done
```

---

## JSON Output Schemas

All three schemas are stable. You can write a parser for any other language that
produces the same format and it will work in the viewer with no changes.

### `callgraph.json`

```json
{
  "repo":           "teleport",
  "totalFiles":     3719,
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
    },
    "auth.*Server.handleLogin": {
      "id":      "auth.*Server.handleLogin",
      "name":    "handleLogin",
      "recv":    "*Server",
      "pkg":     "auth",
      "file":    "lib/auth/server.go",
      "line":    287,
      "calls":   ["auth.checkPassword"],
      "callers": ["auth.*Server.ServeHTTP"]
    }
  }
}
```

### `deps.json`

```json
{
  "repo":          "teleport",
  "module":        "github.com/gravitational/teleport",
  "totalPackages": 942,
  "packages": {
    "github.com/gravitational/teleport/lib/auth": {
      "id":         "github.com/gravitational/teleport/lib/auth",
      "name":       "auth",
      "dir":        "lib/auth",
      "files":      24,
      "imports":    [
        "github.com/gravitational/teleport/lib/utils",
        "github.com/gravitational/teleport/lib/tlsca"
      ],
      "importedBy": [
        "github.com/gravitational/teleport/lib/proxy",
        "github.com/gravitational/teleport/tool/tctl"
      ]
    }
  }
}
```

### `hotspot.json`

```json
{
  "repo":      "teleport",
  "generated": "2026-03-22T22:43:00Z",
  "files": [
    {
      "path":       "lib/auth/server.go",
      "dir":        "lib/auth",
      "commits":    342,
      "authors":    28,
      "lines":      2847,
      "churnScore": 1.0
    },
    {
      "path":       "lib/utils/strings.go",
      "dir":        "lib/utils",
      "commits":    12,
      "authors":    4,
      "lines":      198,
      "churnScore": 0.035
    }
  ],
  "tree": {
    "name": "teleport", "path": ".", "isDir": true,
    "lines": 284701, "commits": 342, "churnScore": 1.0,
    "children": [
      {
        "name": "lib", "path": "lib", "isDir": true,
        "lines": 201500, "commits": 342, "churnScore": 1.0,
        "children": [
          {
            "name": "auth", "path": "lib/auth", "isDir": true,
            "lines": 15200, "commits": 342, "churnScore": 1.0,
            "children": [
              {
                "name": "server.go", "path": "lib/auth/server.go",
                "isDir": false, "lines": 2847,
                "commits": 342, "churnScore": 1.0
              }
            ]
          }
        ]
      }
    ]
  }
}
```

`churnScore` is pre-normalised 0.0–1.0 (1.0 = most-committed file in the repo).
Directory nodes use `max(children.commits)` for commits and `sum(children.lines)` for lines.

---

## Adding a New Language Parser

The viewer is language-agnostic. Any parser that outputs one of the three JSON schemas
above will work. Steps:

1. Write a parser in any language
2. Match the JSON schema for whichever tab you're targeting
3. Drop the output JSON into the `callgraph-explorer` folder
4. Open in the viewer — done

**Recommended parsing tools per language:**

| Language | Tool | Notes |
|----------|------|-------|
| TypeScript / JS | `ts-morph` or `@typescript-eslint/typescript-estree` | Full type resolution available |
| Python | `ast` module (stdlib) | No install needed |
| Rust | `syn` crate | Works on the token stream |
| Java | `JavaParser` | Handles generics well |
| C / C++ | `libclang` | Best accuracy for macros |
| Any language | `tree-sitter` | Universal, bindings for 40+ languages |

PRs for new language parsers are very welcome.

---

## Known Limitations

### Call Graph
- **Go only** for now (Python / TypeScript parsers are planned)
- **Best-effort call resolution** — uses name matching, not full type inference.
  Calls through interfaces, function variables, channels, or `reflect` are not captured
- **External calls excluded** — stdlib and third-party calls are filtered out; only calls
  between functions within the repo appear
- **Large repos** (60k+ functions) produce 15–20 MB JSON files; initial browser load
  takes ~2–3 seconds

### Dependency Graph
- **Internal packages only** — third-party and stdlib imports are excluded
- **File-level granularity** — two files in the same package that import different things
  are merged into one package node

### Hotspot Heatmap
- **Requires git history** — shallow clones (`--depth 1`) give meaningless results.
  Run `git fetch --unshallow` before parsing
- **Renamed files** — `git log --name-only` tracks files by their current name in each
  commit. A renamed file accumulates churn under both old and new paths across history
- **Deleted files** — files that existed historically but are now deleted are excluded
  from the treemap (but their commits still count toward total)
- **Non-Go files** — the hotspot parser currently only counts `.go` files

---

## Project Structure

```
callgraph-explorer/
├── parser/
│   ├── main.go       Go parser — callgraph, deps, hotspot modes
│   └── go.mod
├── index.html        React + D3 viewer — three tabs, zero build step
├── parse.sh          Build + run helper (macOS / Linux)
├── serve.sh          Python HTTP server (macOS / Linux)
├── parse_teleport.bat  Example Windows script
├── serve.bat         Python HTTP server (Windows)
└── README.md
```

---

## Contributing

1. Fork the repo
2. Create a branch: `git checkout -b feat/python-parser`
3. Make your changes
4. Open a pull request

**Good first contributions:**
- Python call graph parser (use the `ast` stdlib module)
- TypeScript/JS parser (use `ts-morph`)
- Export graph as PNG / SVG button
- "Find path between two functions" feature

---

## License

MIT

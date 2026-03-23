package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ── Data model ────────────────────────────────────────────────────────────────

type Function struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Recv    string   `json:"recv,omitempty"` // receiver type for methods
	Pkg     string   `json:"pkg"`
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Calls   []string `json:"calls,omitempty"`
	Callers []string `json:"callers,omitempty"`
}

type CallGraph struct {
	Repo          string               `json:"repo"`
	TotalFiles    int                  `json:"totalFiles"`
	TotalFunctions int                 `json:"totalFunctions"`
	Functions     map[string]*Function `json:"functions"`
}

// ── Helpers ───────────────────────────────────────────────────────────────────

var skipDirs = map[string]bool{
	"vendor": true, "testdata": true, ".git": true,
	"node_modules": true, "dist": true, "build": true, ".cache": true,
}

func shouldSkipDir(name string) bool {
	return skipDirs[name] || strings.HasPrefix(name, ".")
}

func receiverTypeName(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr: // generic receiver T[K]
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

// calleeName extracts a best-effort callee identifier from a call expression.
func calleeName(call *ast.CallExpr) string {
	switch fun := call.Fun.(type) {
	case *ast.Ident:
		return fun.Name
	case *ast.SelectorExpr:
		if x, ok := fun.X.(*ast.Ident); ok {
			return x.Name + "." + fun.Sel.Name
		}
		return fun.Sel.Name
	}
	return ""
}

// collectCalls returns all unique callee names within a function/method body.
func collectCalls(body *ast.BlockStmt) []string {
	if body == nil {
		return nil
	}
	seen := make(map[string]bool)
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		if name := calleeName(call); name != "" {
			seen[name] = true
		}
		return true
	})
	result := make([]string, 0, len(seen))
	for k := range seen {
		result = append(result, k)
	}
	sort.Strings(result)
	return result
}

// ── Parser ────────────────────────────────────────────────────────────────────

func parseRepo(repoPath string, includeTests bool, maxFiles int, pkgFilter string) (*CallGraph, error) {
	cg := &CallGraph{
		Repo:      filepath.Base(repoPath),
		Functions: make(map[string]*Function),
	}

	// Pass 1 – collect all function declarations
	fset := token.NewFileSet()

	err := filepath.WalkDir(repoPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		if !includeTests && strings.HasSuffix(path, "_test.go") {
			return nil
		}
		if maxFiles > 0 && cg.TotalFiles >= maxFiles {
			return filepath.SkipAll
		}
		if pkgFilter != "" {
			rel, _ := filepath.Rel(repoPath, path)
			rel = filepath.ToSlash(rel)
			if !strings.Contains(rel, pkgFilter) {
				return nil
			}
		}

		src, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}

		f, parseErr := parser.ParseFile(fset, path, src, 0)
		if parseErr != nil {
			return nil
		}

		cg.TotalFiles++
		pkgName := f.Name.Name
		relPath, _ := filepath.Rel(repoPath, path)
		relPath = filepath.ToSlash(relPath)

		// Print progress every 100 files
		if cg.TotalFiles%100 == 0 {
			fmt.Fprintf(os.Stderr, "  parsed %d files, %d functions...\n", cg.TotalFiles, len(cg.Functions))
		}

		for _, decl := range f.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}

			recv := ""
			if fn.Recv != nil && len(fn.Recv.List) > 0 {
				recv = receiverTypeName(fn.Recv.List[0])
			}

			// Build a stable ID: pkg + recv + name
			var id string
			if recv != "" {
				id = pkgName + "." + recv + "." + fn.Name.Name
			} else {
				id = pkgName + "." + fn.Name.Name
			}

			pos := fset.Position(fn.Pos())
			rawCalls := collectCalls(fn.Body)

			// Resolve calls: try "pkg.Name" first; fall back to "pkg.Name" when
			// callee looks like a plain identifier in the same package.
			resolved := make([]string, 0, len(rawCalls))
			for _, c := range rawCalls {
				if strings.Contains(c, ".") {
					resolved = append(resolved, c) // already qualified
				} else {
					resolved = append(resolved, pkgName+"."+c) // same-package guess
				}
			}

			cg.Functions[id] = &Function{
				ID:    id,
				Name:  fn.Name.Name,
				Recv:  recv,
				Pkg:   pkgName,
				File:  relPath,
				Line:  pos.Line,
				Calls: resolved,
			}
		}
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Pass 2 – build callers index (only for functions we actually know about)
	for callerID, fn := range cg.Functions {
		for _, calleeID := range fn.Calls {
			if callee, exists := cg.Functions[calleeID]; exists {
				callee.Callers = append(callee.Callers, callerID)
			}
		}
	}

	// Filter Calls to only known functions to reduce noise
	for _, fn := range cg.Functions {
		known := fn.Calls[:0]
		for _, c := range fn.Calls {
			if _, ok := cg.Functions[c]; ok {
				known = append(known, c)
			}
		}
		fn.Calls = known
	}

	// Sort callers for determinism
	for _, fn := range cg.Functions {
		sort.Strings(fn.Callers)
	}

	cg.TotalFunctions = len(cg.Functions)
	return cg, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	repo := flag.String("repo", ".", "Path to the Go repository root")
	output := flag.String("output", "callgraph.json", "Output JSON file path")
	maxFiles := flag.Int("max", 0, "Max .go files to parse (0 = all)")
	includeTests := flag.Bool("tests", false, "Include _test.go files")
	pkgFilter := flag.String("pkg", "", "Only parse files whose path contains this string (e.g. lib/auth)")
	flag.Parse()

	abs, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving repo path: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Parsing repo: %s\n", abs)
	if *pkgFilter != "" {
		fmt.Fprintf(os.Stderr, "Package filter: %s\n", *pkgFilter)
	}

	cg, err := parseRepo(abs, *includeTests, *maxFiles, *pkgFilter)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	data, err := json.Marshal(cg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*output, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ Parsed %d files\n", cg.TotalFiles)
	fmt.Printf("✓ Found %d functions\n", cg.TotalFunctions)
	fmt.Printf("✓ Output → %s  (%.1f MB)\n", *output, float64(len(data))/1_000_000)
}

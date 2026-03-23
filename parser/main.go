package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// ── Shared ────────────────────────────────────────────────────────────────────

var skipDirs = map[string]bool{
	"vendor": true, "testdata": true, ".git": true,
	"node_modules": true, "dist": true, "build": true, ".cache": true,
}

func shouldSkipDir(name string) bool {
	return skipDirs[name] || strings.HasPrefix(name, ".")
}

// ── Call Graph ────────────────────────────────────────────────────────────────

type Function struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Recv    string   `json:"recv,omitempty"`
	Pkg     string   `json:"pkg"`
	File    string   `json:"file"`
	Line    int      `json:"line"`
	Calls   []string `json:"calls,omitempty"`
	Callers []string `json:"callers,omitempty"`
}

type CallGraph struct {
	Repo           string               `json:"repo"`
	TotalFiles     int                  `json:"totalFiles"`
	TotalFunctions int                  `json:"totalFunctions"`
	Functions      map[string]*Function `json:"functions"`
}

func receiverTypeName(field *ast.Field) string {
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return "*" + ident.Name
		}
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name
		}
	}
	return ""
}

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
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func parseRepo(repoPath string, includeTests bool, maxFiles int, pkgFilter string) (*CallGraph, error) {
	cg := &CallGraph{Repo: filepath.Base(repoPath), Functions: make(map[string]*Function)}
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
			if !strings.Contains(filepath.ToSlash(rel), pkgFilter) {
				return nil
			}
		}

		src, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		f, err := parser.ParseFile(fset, path, src, 0)
		if err != nil {
			return nil
		}

		cg.TotalFiles++
		pkgName := f.Name.Name
		relPath, _ := filepath.Rel(repoPath, path)
		relPath = filepath.ToSlash(relPath)

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
			id := pkgName + "." + fn.Name.Name
			if recv != "" {
				id = pkgName + "." + recv + "." + fn.Name.Name
			}
			pos := fset.Position(fn.Pos())
			rawCalls := collectCalls(fn.Body)
			resolved := make([]string, 0, len(rawCalls))
			for _, c := range rawCalls {
				if strings.Contains(c, ".") {
					resolved = append(resolved, c)
				} else {
					resolved = append(resolved, pkgName+"."+c)
				}
			}
			cg.Functions[id] = &Function{
				ID: id, Name: fn.Name.Name, Recv: recv,
				Pkg: pkgName, File: relPath, Line: pos.Line, Calls: resolved,
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	for callerID, fn := range cg.Functions {
		for _, calleeID := range fn.Calls {
			if callee, ok := cg.Functions[calleeID]; ok {
				callee.Callers = append(callee.Callers, callerID)
			}
		}
	}
	for _, fn := range cg.Functions {
		known := fn.Calls[:0]
		for _, c := range fn.Calls {
			if _, ok := cg.Functions[c]; ok {
				known = append(known, c)
			}
		}
		fn.Calls = known
		sort.Strings(fn.Callers)
	}
	cg.TotalFunctions = len(cg.Functions)
	return cg, nil
}

// ── Dependency Graph ──────────────────────────────────────────────────────────

type PackageNode struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Dir        string   `json:"dir"`
	Files      int      `json:"files"`
	Imports    []string `json:"imports,omitempty"`
	ImportedBy []string `json:"importedBy,omitempty"`
}

type DepsGraph struct {
	Repo          string                  `json:"repo"`
	Module        string                  `json:"module"`
	TotalPackages int                     `json:"totalPackages"`
	Packages      map[string]*PackageNode `json:"packages"`
}

func readModuleName(repoPath string) string {
	data, err := os.ReadFile(filepath.Join(repoPath, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

func parseDeps(repoPath string) (*DepsGraph, error) {
	module := readModuleName(repoPath)
	if module == "" {
		module = filepath.Base(repoPath)
		fmt.Fprintf(os.Stderr, "Warning: go.mod not found, using %q as module name\n", module)
	}

	pkgs := make(map[string]*PackageNode)
	fset := token.NewFileSet()
	fileCount := 0

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
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return nil
		}
		fileCount++
		if fileCount%200 == 0 {
			fmt.Fprintf(os.Stderr, "  scanned %d files, %d packages...\n", fileCount, len(pkgs))
		}

		relDir, _ := filepath.Rel(repoPath, filepath.Dir(path))
		relDir = filepath.ToSlash(relDir)

		pkgID := module + "/" + relDir
		if relDir == "." {
			pkgID = module
		}

		if _, ok := pkgs[pkgID]; !ok {
			pkgs[pkgID] = &PackageNode{ID: pkgID, Name: f.Name.Name, Dir: relDir}
		}
		pkgs[pkgID].Files++

		seen := make(map[string]bool)
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			internal := importPath == module || strings.HasPrefix(importPath, module+"/")
			if !internal || importPath == pkgID || seen[importPath] {
				continue
			}
			seen[importPath] = true
			pkgs[pkgID].Imports = append(pkgs[pkgID].Imports, importPath)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Ensure all imported packages have nodes; build importedBy
	for pkgID, pkg := range pkgs {
		sort.Strings(pkg.Imports)
		for _, impID := range pkg.Imports {
			if _, ok := pkgs[impID]; !ok {
				dir := strings.TrimPrefix(impID, module+"/")
				pkgs[impID] = &PackageNode{ID: impID, Name: filepath.Base(dir), Dir: dir}
			}
			pkgs[impID].ImportedBy = append(pkgs[impID].ImportedBy, pkgID)
		}
	}
	for _, pkg := range pkgs {
		sort.Strings(pkg.ImportedBy)
	}

	fmt.Fprintf(os.Stderr, "\n✓ Scanned %d files → %d packages\n", fileCount, len(pkgs))
	return &DepsGraph{
		Repo: filepath.Base(repoPath), Module: module,
		TotalPackages: len(pkgs), Packages: pkgs,
	}, nil
}

// ── Hotspot Heatmap ───────────────────────────────────────────────────────────

type HotspotFile struct {
	Path       string  `json:"path"`
	Dir        string  `json:"dir"`
	Commits    int     `json:"commits"`
	Authors    int     `json:"authors"`
	Lines      int     `json:"lines"`
	ChurnScore float64 `json:"churnScore"` // 0.0–1.0 normalised
}

type TreeNode struct {
	Name       string      `json:"name"`
	Path       string      `json:"path"`
	IsDir      bool        `json:"isDir"`
	Lines      int         `json:"lines"`
	Commits    int         `json:"commits"`
	ChurnScore float64     `json:"churnScore"`
	Children   []*TreeNode `json:"children,omitempty"`
}

type HotspotData struct {
	Repo      string         `json:"repo"`
	Generated string         `json:"generated"`
	Files     []*HotspotFile `json:"files"` // flat, sorted by commits desc
	Tree      *TreeNode      `json:"tree"`
}

func countFileLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 64*1024)
	for sc.Scan() {
		n++
	}
	return n
}

func buildTree(repoName string, files []*HotspotFile) *TreeNode {
	root := &TreeNode{Name: repoName, Path: ".", IsDir: true}
	dirs := map[string]*TreeNode{".": root}

	var getDir func(p string) *TreeNode
	getDir = func(p string) *TreeNode {
		if n, ok := dirs[p]; ok {
			return n
		}
		parent := filepath.ToSlash(filepath.Dir(p))
		if parent == p || parent == "" {
			parent = "."
		}
		n := &TreeNode{Name: filepath.Base(p), Path: p, IsDir: true}
		getDir(parent).Children = append(getDir(parent).Children, n)
		dirs[p] = n
		return n
	}

	for _, f := range files {
		if f.Lines == 0 {
			continue
		}
		dir := f.Dir
		if dir == "" {
			dir = "."
		}
		getDir(dir).Children = append(getDir(dir).Children, &TreeNode{
			Name: filepath.Base(f.Path), Path: f.Path,
			Lines: f.Lines, Commits: f.Commits, ChurnScore: f.ChurnScore,
		})
	}

	// Roll up: sum lines, max commits/churnScore
	var rollup func(*TreeNode)
	rollup = func(n *TreeNode) {
		if !n.IsDir {
			return
		}
		for _, c := range n.Children {
			rollup(c)
			n.Lines += c.Lines
			if c.Commits > n.Commits {
				n.Commits = c.Commits
			}
			if c.ChurnScore > n.ChurnScore {
				n.ChurnScore = c.ChurnScore
			}
		}
	}
	rollup(root)
	return root
}

func parseHotspots(repoPath string) (*HotspotData, error) {
	cmd := exec.Command("git", "-C", repoPath, "log",
		"--no-merges", "--format=COMMIT:%ae", "--name-only", "--diff-filter=ACDMR")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("git not found or failed — is git in PATH? (%v)", err)
	}

	type raw struct {
		authors map[string]bool
		commits int
	}
	stats := make(map[string]*raw)
	total, currentAuthor := 0, ""
	seenInCommit := map[string]bool{}

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 4*1024*1024), 4*1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			seenInCommit = map[string]bool{}
			continue
		}
		if strings.HasPrefix(line, "COMMIT:") {
			currentAuthor = strings.TrimPrefix(line, "COMMIT:")
			total++
			if total%1000 == 0 {
				fmt.Fprintf(os.Stderr, "  processed %d commits...\n", total)
			}
			continue
		}
		line = filepath.ToSlash(line)
		if seenInCommit[line] {
			continue
		}
		seenInCommit[line] = true
		if stats[line] == nil {
			stats[line] = &raw{authors: map[string]bool{}}
		}
		stats[line].commits++
		if currentAuthor != "" {
			stats[line].authors[currentAuthor] = true
		}
	}
	_ = cmd.Wait()
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("scanning git output: %v", err)
	}

	maxCommits := 1
	for _, s := range stats {
		if s.commits > maxCommits {
			maxCommits = s.commits
		}
	}

	var files []*HotspotFile
	for path, s := range stats {
		lines := countFileLines(filepath.Join(repoPath, filepath.FromSlash(path)))
		dir := filepath.ToSlash(filepath.Dir(path))
		if dir == "" {
			dir = "."
		}
		files = append(files, &HotspotFile{
			Path: path, Dir: dir,
			Commits: s.commits, Authors: len(s.authors),
			Lines:      lines,
			ChurnScore: float64(s.commits) / float64(maxCommits),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Commits > files[j].Commits })

	fmt.Fprintf(os.Stderr, "\n✓ Total commits: %d\n✓ Files tracked: %d\n", total, len(files))
	return &HotspotData{
		Repo:      filepath.Base(repoPath),
		Generated: time.Now().UTC().Format(time.RFC3339),
		Files:     files,
		Tree:      buildTree(filepath.Base(repoPath), files),
	}, nil
}

// ── Main ──────────────────────────────────────────────────────────────────────

func main() {
	mode         := flag.String("mode",   "callgraph", "Analysis mode: callgraph | deps | hotspot")
	repo         := flag.String("repo",   ".",         "Path to the repository root")
	output       := flag.String("output", "",          "Output JSON file (default: <mode>.json)")
	maxFiles     := flag.Int("max",       0,           "Max .go files (callgraph only)")
	includeTests := flag.Bool("tests",    false,       "Include _test.go files (callgraph only)")
	pkgFilter    := flag.String("pkg",    "",          "Only parse paths containing this string (callgraph only)")
	flag.Parse()

	outFile := *output
	if outFile == "" {
		outFile = *mode + ".json"
	}

	abs, err := filepath.Abs(*repo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving repo path: %v\n", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "Mode: %s | Repo: %s\n", *mode, abs)

	var jsonData []byte

	switch *mode {
	case "callgraph":
		cg, e := parseRepo(abs, *includeTests, *maxFiles, *pkgFilter)
		if e != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
			os.Exit(1)
		}
		jsonData, err = json.Marshal(cg)
		fmt.Printf("\n✓ Parsed %d files | %d functions\n", cg.TotalFiles, cg.TotalFunctions)

	case "deps":
		dg, e := parseDeps(abs)
		if e != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
			os.Exit(1)
		}
		jsonData, err = json.Marshal(dg)
		fmt.Printf("✓ %d packages\n", dg.TotalPackages)

	case "hotspot":
		hs, e := parseHotspots(abs)
		if e != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", e)
			os.Exit(1)
		}
		jsonData, err = json.Marshal(hs)

	default:
		fmt.Fprintf(os.Stderr, "Unknown mode %q — valid: callgraph, deps, hotspot\n", *mode)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "JSON marshal error: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outFile, jsonData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ Output → %s  (%.1f MB)\n", outFile, float64(len(jsonData))/1_000_000)
}

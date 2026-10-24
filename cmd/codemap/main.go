// Command codemap inventories first-party Go functions for documentation checks.
package main

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/scanner"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// symbol identifies a function's source location and its attached documentation.
type symbol struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Line    int    `json:"line"`
	End     int    `json:"end"`
	Comment string `json:"comment"`
	Kind    string `json:"kind"`
}

// receiverName preserves the owning type when indexing a Go method.
func receiverName(expr ast.Expr) string {
	switch node := expr.(type) {
	case *ast.StarExpr:
		return receiverName(node.X)
	case *ast.Ident:
		return node.Name
	case *ast.IndexExpr:
		return receiverName(node.X)
	}
	return ""
}

// precedingComment finds documentation immediately above a named local closure.
func precedingComment(file *ast.File, set *token.FileSet, pos token.Pos) string {
	line := set.Position(pos).Line
	for _, group := range file.Comments {
		if set.Position(group.End()).Line == line-1 {
			return strings.TrimSpace(group.Text())
		}
	}
	return ""
}

// inspectFile extracts declarations and named closures without executing code.
func inspectFile(path string) ([]symbol, error) {
	set := token.NewFileSet()
	file, err := parser.ParseFile(set, path, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	var out []symbol
	// add records a declaration after its owner and documentation are resolved.
	add := func(name, kind string, start, end token.Pos, doc string) {
		out = append(out, symbol{filepath.ToSlash(path), name, set.Position(start).Line,
			set.Position(end).Line, strings.TrimSpace(doc), kind})
	}
	for _, decl := range file.Decls {
		if fn, ok := decl.(*ast.FuncDecl); ok {
			name := fn.Name.Name
			if fn.Recv != nil {
				name = receiverName(fn.Recv.List[0].Type) + "." + name
			}
			doc := ""
			if fn.Doc != nil {
				doc = fn.Doc.Text()
			}
			add(name, "function", fn.Pos(), fn.End(), doc)
		}
	}
	ast.Inspect(file, func(node ast.Node) bool {
		switch item := node.(type) {
		case *ast.AssignStmt:
			for i, expr := range item.Rhs {
				if fn, ok := expr.(*ast.FuncLit); ok && i < len(item.Lhs) {
					if name, ok := item.Lhs[i].(*ast.Ident); ok {
						add(name.Name, "closure", item.Pos(), fn.End(), precedingComment(file, set, item.Pos()))
					}
				}
			}
		case *ast.ValueSpec:
			for i, expr := range item.Values {
				if fn, ok := expr.(*ast.FuncLit); ok && i < len(item.Names) {
					add(item.Names[i].Name, "closure", item.Pos(), fn.End(), precedingComment(file, set, item.Pos()))
				}
			}
		}
		return true
	})
	return out, nil
}

// tokenFingerprint detects executable Go changes while ignoring comments and layout.
func tokenFingerprint(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	set := token.NewFileSet()
	file := set.AddFile(path, -1, len(data))
	var lexer scanner.Scanner
	lexer.Init(file, data, nil, 0)
	hash := sha256.New()
	for {
		_, kind, literal := lexer.Scan()
		fmt.Fprintf(hash, "%d:%q\n", kind, literal)
		if kind == token.EOF {
			break
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// main prints the Go symbol inventory as JSON for scripts/code-docs.cjs.
func main() {
	var all []symbol
	tokens := map[string]string{}
	for _, root := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			functions, err := inspectFile(path)
			if err == nil {
				all = append(all, functions...)
				fingerprint, hashErr := tokenFingerprint(path)
				if hashErr != nil {
					return hashErr
				}
				tokens[filepath.ToSlash(path)] = fingerprint
			}
			return err
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	var result any = all
	if len(os.Args) > 1 && os.Args[1] == "--tokens" {
		result = tokens
	}
	if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

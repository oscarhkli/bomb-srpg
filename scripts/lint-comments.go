//go:build ignore

package main

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

const maxBlockLines = 3

type block struct {
	start, end int
}

func main() {
	bad := false
	for _, root := range os.Args[1:] {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") {
				return err
			}
			msgs, err := checkFile(path)
			for _, m := range msgs {
				fmt.Println(m)
				bad = true
			}
			return err
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	}
	if bad {
		os.Exit(1)
	}
}

func checkFile(path string) ([]string, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, path, src, parser.ParseComments)
	if err != nil {
		return nil, err
	}
	tf := fset.File(f.Pos())

	var msgs []string
	var cur *block
	flush := func() {
		if cur != nil && cur.end-cur.start+1 > maxBlockLines {
			msgs = append(msgs, fmt.Sprintf("%s:%d: comment block of %d lines, max %d", path, cur.start, cur.end-cur.start+1, maxBlockLines))
		}
		cur = nil
	}
	for _, c := range flatten(f.Comments) {
		if strings.HasPrefix(c.Text, "//go:") {
			continue
		}
		start, end := tf.Line(c.Pos()), tf.Line(c.End())
		if hasCodeOnLine(src, tf, c) {
			flush()
			msgs = append(msgs, fmt.Sprintf("%s:%d: trailing comment after code", path, start))
		} else if cur != nil && start == cur.end+1 {
			cur.end = end
		} else {
			flush()
			cur = &block{start, end}
		}
		if strings.HasPrefix(strings.TrimLeft(strings.TrimPrefix(c.Text, "//"), " "), "nolint") {
			msgs = append(msgs, fmt.Sprintf("%s:%d: //nolint is banned", path, start))
		}
	}
	flush()
	return msgs, nil
}

func flatten(groups []*ast.CommentGroup) []*ast.Comment {
	var all []*ast.Comment
	for _, g := range groups {
		all = append(all, g.List...)
	}
	return all
}

func hasCodeOnLine(src []byte, tf *token.File, c *ast.Comment) bool {
	startOff, endOff := tf.Offset(c.Pos()), tf.Offset(c.End())
	before := src[tf.Offset(tf.LineStart(tf.Line(c.Pos()))):startOff]
	rest := src[endOff:]
	if i := bytes.IndexByte(rest, '\n'); i >= 0 {
		rest = rest[:i]
	}
	return len(bytes.TrimSpace(before)) > 0 || len(bytes.TrimSpace(rest)) > 0
}

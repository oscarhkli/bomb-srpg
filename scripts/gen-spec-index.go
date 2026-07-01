//go:build ignore

// gen-spec-index.go generates the spec table in docs/frontend/README.md.
// Usage: go run scripts/gen-spec-index.go
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	docsDir    = "docs/frontend"
	tocFile    = "docs/frontend/toc.yaml"
	readmeFile = "docs/frontend/README.md"
	beginTag   = "<!-- BEGIN SPEC TABLE -->"
	endTag     = "<!-- END SPEC TABLE -->"
)

type specEntry struct {
	file           string
	status         string
	category       string
	completionDate string
}

func main() {
	specs, err := parseTOC(tocFile)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	table, err := buildTable(specs)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	if err := updateReadme(readmeFile, table); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	fmt.Println("✓ Updated", readmeFile)
}

// parseTOC reads toc.yaml and returns spec entries in order.
// It parses the simple YAML structure manually without an external library.
func parseTOC(path string) ([]specEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var specs []specEntry
	var current *specEntry

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		// New list item
		if strings.HasPrefix(trimmed, "- file:") {
			if current != nil {
				specs = append(specs, *current)
			}
			file := strings.TrimSpace(strings.TrimPrefix(trimmed, "- file:"))
			if file == "" {
				return nil, fmt.Errorf("spec entry missing 'file' value")
			}
			current = &specEntry{file: file}
			continue
		}

		if current == nil {
			continue
		}

		key, value, _ := strings.Cut(trimmed, ":")
		value = strings.TrimSpace(value)

		switch key {
		case "status":
			current.status = value
		case "category":
			current.category = value
		case "completion-date":
			current.completionDate = value
		}
	}

	if current != nil {
		specs = append(specs, *current)
	}

	return specs, scanner.Err()
}

// extractTitle reads the YAML frontmatter title from a spec file.
func extractTitle(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	inFrontmatter := false
	firstLine := true

	for scanner.Scan() {
		line := scanner.Text()

		if line == "---" {
			if firstLine {
				inFrontmatter = true
				firstLine = false
				continue
			}
			break // closing ---
		}
		firstLine = false

		if inFrontmatter && strings.HasPrefix(line, "title:") {
			title := strings.TrimSpace(strings.TrimPrefix(line, "title:"))
			return strings.Trim(title, `"`), nil
		}
	}

	return "", scanner.Err()
}

func buildTable(specs []specEntry) (string, error) {
	var sb strings.Builder
	sb.WriteString("| Specs | Category | Status | Completion Date |\n")
	sb.WriteString("| --- | --- | --- | --- |\n")

	for _, s := range specs {
		title, err := extractTitle(filepath.Join(docsDir, s.file))
		if err != nil {
			return "", fmt.Errorf("reading %s: %w", s.file, err)
		}
		if title == "" {
			fmt.Fprintf(os.Stderr, "Warning: %s has no title in frontmatter\n", s.file)
			title = s.file
		}

		date := s.completionDate
		if date == "" {
			date = "—"
		}

		fmt.Fprintf(&sb, "| [%s](%s) | %s | %s | %s |\n", title, s.file, s.category, s.status, date)
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

func updateReadme(path, table string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var out []string
	inTable := false

	for _, line := range lines {
		if line == beginTag {
			out = append(out, line)
			out = append(out, strings.Split(table, "\n")...)
			inTable = true
			continue
		}
		if line == endTag {
			inTable = false
		}
		if !inTable {
			out = append(out, line)
		}
	}

	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

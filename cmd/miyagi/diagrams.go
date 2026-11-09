package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Diagrams
//
// A chapter page can carry a hand-drawn diagram with a fence:
//
//	```diagram folds
//	```
//
// The SVG lives beside the chapter in `diagrams/folds.svg`. It is committed, because
// regenerating it needs Node and the course promises no Node to take the course. The
// generator is in `tools/scribble/` and is a maintainer tool, not a build step.
//
// Two renderers read the same fence and must agree:
//
//   - the site inlines the SVG, so the page palette reaches the drawing, the text stays
//     real text, and the reveal runs on CSS;
//   - the terminal cannot draw it, so `miyagi read` prints the diagram's own description
//     instead. A reader in a terminal loses the picture but not the claim it makes.

// diagramFence matches an opening ```diagram line and the ``` that closes it. The body is
// empty today; it exists so a future option has somewhere to go without a new syntax.
var diagramFence = regexp.MustCompile("(?m)^```diagram[ \t]+([A-Za-z0-9_-]+)[ \t]*\n(?:[^`]*?\n)??```[ \t]*$")

// expandDiagrams replaces each fence with the figure markup for its SVG. `dir` is the
// directory holding the page, so a chapter's diagrams resolve relative to that chapter.
func expandDiagrams(md, dir string) (string, error) {
	var firstErr error

	out := diagramFence.ReplaceAllStringFunc(md, func(match string) string {
		name := diagramFence.FindStringSubmatch(match)[1]
		svg, desc, err := readDiagram(dir, name)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return match
		}
		return figure(name, svg, desc)
	})

	if firstErr != nil {
		return md, firstErr
	}
	return out, nil
}

// figure wraps a diagram so it reads as one unit: a picture with its claim underneath.
//
// The SVG is inlined rather than referenced with <img>, which is what makes the page's
// colours apply to the drawing, keeps the labels as selectable text, and lets the pen
// animation start when the figure is scrolled into view.
func figure(name, svg, desc string) string {
	var b strings.Builder
	b.WriteString(`<figure class="diagram">` + "\n")
	b.WriteString(svg)
	if desc != "" {
		b.WriteString("\n<figcaption>" + escapeHTML(desc) + "</figcaption>")
	}
	b.WriteString("\n</figure>")
	return b.String()
}

// describeDiagrams is the terminal form: the fence becomes the diagram's description, so
// `miyagi read` says what the picture shows instead of dropping it silently.
func describeDiagrams(md, dir string) (string, error) {
	var firstErr error

	out := diagramFence.ReplaceAllStringFunc(md, func(match string) string {
		name := diagramFence.FindStringSubmatch(match)[1]
		_, desc, err := readDiagram(dir, name)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			return "[diagram: " + name + "]"
		}
		if desc == "" {
			return "[diagram: " + name + "]"
		}
		return wrap("  [diagram] "+desc, 76, "  ")
	})

	if firstErr != nil {
		return md, firstErr
	}
	return out, nil
}

// readDiagram loads a diagram and the one-sentence description the generator wrote into
// its <desc>, which is the same sentence the site prints under the picture.
func readDiagram(dir, name string) (svg, desc string, err error) {
	if !validDiagramName(name) {
		return "", "", fmt.Errorf("diagram name %q may contain only letters, digits, dash and underscore", name)
	}
	path := filepath.Join(dir, "diagrams", name+".svg")
	body, err := os.ReadFile(path)
	if err != nil {
		return "", "", fmt.Errorf("no diagram %q beside this page (expected %s)", name, path)
	}
	svg = string(body)
	desc = svgDesc(svg)
	return svg, desc, nil
}

func validDiagramName(name string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
		default:
			return false
		}
	}
	return true
}

var descTag = regexp.MustCompile(`(?s)<desc>(.*?)</desc>`)

// svgDesc pulls the description out of an SVG. The generator writes plain text there, so
// the only entity that needs undoing is the handful escapeHTML produces.
func svgDesc(svg string) string {
	m := descTag.FindStringSubmatch(svg)
	if m == nil {
		return ""
	}
	return collapseSpace(unescapeHTML(m[1]))
}

func unescapeHTML(s string) string {
	return strings.NewReplacer(
		"&lt;", "<", "&gt;", ">", "&quot;", `"`, "&apos;", "'", "&amp;", "&",
	).Replace(s)
}

func collapseSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// wrap folds text to width columns with a hanging indent, so a long description does not
// run off the edge of a terminal at its default 80 columns.
func wrap(s string, width int, indent string) string {
	var b strings.Builder
	line := 0
	for i, word := range strings.Fields(s) {
		if i > 0 && line+1+len(word) > width {
			b.WriteString("\n" + indent)
			line = 0
		} else if i > 0 {
			b.WriteString(" ")
			line++
		}
		b.WriteString(word)
		line += len(word)
	}
	return b.String()
}

func escapeHTML(s string) string {
	return strings.NewReplacer(
		"&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#39;",
	).Replace(s)
}

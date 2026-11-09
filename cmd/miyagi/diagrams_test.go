package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Diagrams
//
// The point of these tests is that a diagram is part of the course, not decoration: every
// chapter carries one, every one resolves, and the site and the terminal agree about what
// it says. A missing diagram is a chapter that lost its picture, which is invisible on a
// page but obvious here.

func TestEveryWrittenChapterHasADiagram(t *testing.T) {
	root := repoRootForTests(t)
	for _, slug := range writtenChapters(t) {
		body, err := os.ReadFile(filepath.Join(root, "chapters", slug, "README.md"))
		if err != nil {
			t.Fatalf("read %s: %v", slug, err)
		}
		if n := strings.Count(string(body), "```diagram"); n != 1 {
			t.Errorf("%s has %d diagram fences, want exactly 1", slug, n)
		}
	}
}

func TestEveryDiagramResolves(t *testing.T) {
	root := repoRootForTests(t)
	for _, slug := range writtenChapters(t) {
		dir := filepath.Join(root, "chapters", slug)
		body, err := os.ReadFile(filepath.Join(dir, "README.md"))
		if err != nil {
			t.Fatalf("read %s: %v", slug, err)
		}
		// expandDiagrams returns an error for an unresolvable name, which is the failure
		// a learner would meet as a raw fence on the page.
		out, err := expandDiagrams(string(body), dir)
		if err != nil {
			t.Errorf("%s: %v", slug, err)
			continue
		}
		if strings.Contains(out, "```diagram") {
			t.Errorf("%s: a diagram fence survived expansion", slug)
		}
		if !strings.Contains(out, `<figure class="diagram">`) {
			t.Errorf("%s: no figure was produced", slug)
		}
		if !strings.Contains(out, "<svg") {
			t.Errorf("%s: the figure has no inline svg", slug)
		}
	}
}

func TestEveryDiagramHasACaption(t *testing.T) {
	root := repoRootForTests(t)
	for _, slug := range writtenChapters(t) {
		dir := filepath.Join(root, "chapters", slug)
		svg, desc, err := readDiagram(dir, "mechanism")
		if err != nil {
			t.Errorf("%s: %v", slug, err)
			continue
		}
		if desc == "" {
			t.Errorf("%s: the diagram has no <desc>, so the terminal would print nothing in its place", slug)
		}
		// A caption is the whole picture for a reader in a terminal, so one short clause
		// is not enough: it has to state the claim.
		if len(desc) < 80 {
			t.Errorf("%s: the caption is %d chars, too short to stand in for the drawing", slug, len(desc))
		}
		if !strings.Contains(svg, `class="scratch"`) {
			t.Errorf("%s: the svg is missing the class the stylesheet hooks", slug)
		}
		// The stroke class and the tone classes are what let the page palette apply. A
		// diagram with literal colours cannot follow the theme.
		if strings.Contains(svg, `stroke="#`) || strings.Contains(svg, `fill="#221b16"`) {
			t.Errorf("%s: the svg carries a literal colour instead of a tone class", slug)
		}
	}
}

// The terminal form has to name the claim, not just announce that a picture was skipped.
func TestTerminalShowsTheCaptionNotTheFence(t *testing.T) {
	root := repoRootForTests(t)
	slug := writtenChapters(t)[0]
	dir := filepath.Join(root, "chapters", slug)
	body, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatalf("read %s: %v", slug, err)
	}

	out, err := describeDiagrams(string(body), dir)
	if err != nil {
		t.Fatalf("%s: %v", slug, err)
	}
	if strings.Contains(out, "```diagram") {
		t.Errorf("%s: the fence reached the terminal", slug)
	}
	if !strings.Contains(out, "[diagram]") {
		t.Errorf("%s: the terminal form does not mark the diagram", slug)
	}
	_, desc, err := readDiagram(dir, "mechanism")
	if err != nil {
		t.Fatalf("%s: %v", slug, err)
	}
	// Wrapped text may break a phrase, so compare a short distinctive prefix.
	head := desc[:40]
	if !strings.Contains(collapseSpace(out), collapseSpace(head)) {
		t.Errorf("%s: the terminal output does not carry the caption\nwant prefix: %q", slug, head)
	}
}

// A bad name must fail loudly rather than resolve to an unexpected path.
func TestDiagramNamesAreRestricted(t *testing.T) {
	for _, name := range []string{"../escape", "a/b", "with space", "", "dots..dots"} {
		if validDiagramName(name) {
			t.Errorf("name %q was accepted", name)
		}
	}
	for _, name := range []string{"mechanism", "retry-loop", "loop_2"} {
		if !validDiagramName(name) {
			t.Errorf("name %q was rejected", name)
		}
	}
}

// The site must ship the font the diagrams are drawn in, or every label falls back and the
// drawings stop looking hand made.
func TestSiteShipsTheDiagramFont(t *testing.T) {
	root := repoRootForTests(t)
	woff := filepath.Join(root, "docs", "assets", "PatrickHand-Regular.woff2")
	if _, err := os.Stat(woff); err != nil {
		t.Fatalf("the diagram font is missing: %v", err)
	}
	css := themeCSS
	for _, want := range []string{
		"@font-face",
		"Patrick Hand",
		"figure.diagram",
		".scratch .s-line",
		".scratch .s-text",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("the stylesheet is missing %q", want)
		}
	}
	// The palette must be a variable per tone, so one committed file works in both themes.
	for _, tone := range []string{"--dg-ink", "--dg-accent", "--dg-good", "--dg-bad", "--dg-violet"} {
		if !strings.Contains(css, tone+":") {
			t.Errorf("the stylesheet defines no %s", tone)
		}
	}
}

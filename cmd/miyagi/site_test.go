package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The site is generated, so these tests cover the generation: which files are copied,
// and whether the sidebar knows about every chapter. Rendering belongs to VitePress and
// is not tested here.

func TestConfigListsEveryChapter(t *testing.T) {
	root := repoRootForTests(t)
	config := buildConfig("Miyagi", root)

	for _, ch := range allChapters(root) {
		if !ch.written() {
			continue
		}
		if !strings.Contains(config, `/chapters/`+ch.Slug+`/`) {
			t.Errorf("sidebar is missing chapter %s (%s)", ch.ID, ch.Slug)
		}
		if !strings.Contains(config, `"/chapters/`+ch.Slug+`/example"`) {
			t.Errorf("sidebar is missing the worked example for chapter %s", ch.ID)
		}
	}
}

func TestConfigNamesTheParts(t *testing.T) {
	root := repoRootForTests(t)
	config := buildConfig("Miyagi", root)
	for _, part := range []string{"I. Wire", "II. Act", "III. Remember", "IV. Extend", "V. Compose"} {
		if !strings.Contains(config, `"`+part+`"`) {
			t.Errorf("sidebar is missing part %q", part)
		}
	}
}

func TestConfigIsValidShape(t *testing.T) {
	config := buildConfig("Miyagi", repoRootForTests(t))
	for _, want := range []string{
		"withMermaid(defineConfig({",
		"search: { provider: 'local' }",
		"mermaid: {},",
		"export default",
	} {
		if !strings.Contains(config, want) {
			t.Errorf("config is missing %q", want)
		}
	}
	// Unbalanced braces would produce a file VitePress cannot parse.
	if strings.Count(config, "{") != strings.Count(config, "}") {
		t.Errorf("config braces are unbalanced")
	}
}

// readTestSource reads a file from the repository, for tests that assert on source text
// rather than behaviour. Used only where a rule is about the list a developer maintains.
func readTestSource(t *testing.T, rel string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(repoRootForTests(t), rel))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(body)
}

// The repository keeps working notes that are not part of the course. `docs/adr/` holds
// decision records written for whoever maintains this repository, addressed to
// contributors rather than to a learner, and they carry no teaching value for someone
// building the agent. Publishing them puts internal bookkeeping in front of readers and
// invites them to treat a past trade-off as part of the subject.
func TestSitePublishesNoDecisionRecords(t *testing.T) {
	root := repoRootForTests(t)
	config := buildConfig("Miyagi", root)

	if strings.Contains(config, "adr") {
		t.Errorf("the sidebar links an ADR; decision records are internal")
	}
	// prepareContent is the only path a file can take onto the site, so its allowlist is
	// what decides. An ADR added to the list would be published with no other change.
	source := readTestSource(t, "cmd/miyagi/site.go")
	start := strings.Index(source, "for _, doc := range []string{")
	if start < 0 {
		t.Fatal("could not find the published-documents list in site.go")
	}
	end := strings.Index(source[start:], "}")
	if end < 0 {
		t.Fatal("could not find the end of the published-documents list")
	}
	list := source[start : start+end]
	if strings.Contains(list, "docs/adr") {
		t.Errorf("docs/adr is in the published-documents list; decision records are internal")
	}
}

// The site's own pages are embedded, so they ship inside the binary. A page renamed or
// dropped here would 404 in production while every other test stayed green, and the
// install command is the one thing a reader needs to be exactly right.
func TestSitePagesAreEmbeddedAndCorrect(t *testing.T) {
	entries, err := sitePages.ReadDir("pages")
	if err != nil {
		t.Fatalf("read embedded pages: %v", err)
	}
	have := map[string]bool{}
	for _, e := range entries {
		have[e.Name()] = true
	}
	for _, want := range []string{"index.md", "install.md", "how-it-works.md"} {
		if !have[want] {
			t.Errorf("embedded pages are missing %s", want)
		}
	}

	install, err := sitePages.ReadFile("pages/install.md")
	if err != nil {
		t.Fatalf("read install page: %v", err)
	}
	// The install command must name a real tag. A reader who copies a stale version gets
	// the old broken module path, which is the exact defect this page exists to close.
	if !strings.Contains(string(install), "go install github.com/theakhandpatel/miyagi/cmd/miyagi@v") {
		t.Error("install page does not show a tagged go install command")
	}
	if strings.Contains(string(install), "@latest") {
		t.Error("install page should pin a tag, not @latest: the proxy can serve a stale commit")
	}

	home, err := sitePages.ReadFile("pages/index.md")
	if err != nil {
		t.Fatalf("read home page: %v", err)
	}
	if !strings.Contains(string(home), "layout: home") {
		t.Error("home page should use the VitePress home layout rather than dumping the README")
	}
}

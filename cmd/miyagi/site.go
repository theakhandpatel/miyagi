package main

import (
	"embed"
	"fmt"
	"os"
	"io/fs"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// cmdSite renders the course as a static website with VitePress.
//
// The generator does one thing: copy the course markdown into a VitePress project,
// write the sidebar from the chapter directories, and call VitePress. Rendering,
// search, anchors, syntax highlighting, prev/next navigation and the theme all belong
// to VitePress, so this file has no Markdown parser and no CSS.
//
// The project it writes is disposable. `site/src` is regenerated on every run, and
// only `site/dist` is kept for deployment.
func cmdSite(args []string) error {
	root, err := findRoot()
	if err != nil {
		return err
	}

	out := filepath.Join(root, "site", "dist")
	title := "Miyagi"
	skipInstall := false
	v2 := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--out", "-o":
			if i+1 < len(args) {
				out = args[i+1]
				i++
			}
		case "--title", "-t":
			if i+1 < len(args) {
				title = args[i+1]
				i++
			}
		case "--no-install":
			skipInstall = true
		case "--v2":
			// The experimental build: three dimensional figures and a denser visual
			// language. It writes to site/dist-v2 by default so building one never
			// replaces the other, and it sets the /v2 base so the output can also be
			// nested inside the stable site at site/dist/v2.
			v2 = true
		}
	}
	if v2 && out == filepath.Join(root, "site", "dist") {
		out = filepath.Join(root, "site", "dist-v2")
	}
	if !filepath.IsAbs(out) {
		out = filepath.Join(root, out)
	}

	siteDir := filepath.Join(root, "site")
	srcDir := filepath.Join(siteDir, "src")

	if err := os.RemoveAll(srcDir); err != nil {
		return err
	}
	if err := os.MkdirAll(srcDir, 0o755); err != nil {
		return err
	}

	pages, err := prepareContent(root, srcDir, v2)
	if err != nil {
		return err
	}
	if v2 {
		n, err := writeScenes(srcDir)
		if err != nil {
			return err
		}
		fmt.Printf("\n  %d scenes\n", n)
	}
	if pages == 0 {
		return fmt.Errorf("nothing to render")
	}

	if err := writeProject(srcDir, title, root, v2); err != nil {
		return err
	}

	if !skipInstall {
		if err := ensureDependencies(siteDir); err != nil {
			return err
		}
	}

	if err := build(siteDir); err != nil {
		return err
	}

	built := filepath.Join(srcDir, ".vitepress", "dist")
	if err := os.RemoveAll(out); err != nil {
		return err
	}
	if err := copyTree(built, out); err != nil {
		return fmt.Errorf("copy build output: %w", err)
	}

	fmt.Printf("\n  wrote %d pages to %s\n", pages, out)
	fmt.Printf("  %s\n\n", filepath.Join(out, "index.html"))
	return nil
}

// prepareContent copies the course markdown into the VitePress source tree.
//
// Paths are preserved, so every relative link written for `miyagi read` and GitHub also
// resolves here. Only `README.md` is renamed, because VitePress wants `index.md` for a
// directory's landing page.
func prepareContent(root, srcDir string, v2 bool) (int, error) {
	pages := 0

	copyPage := func(source, target string) error {
		body, err := os.ReadFile(filepath.Join(root, source))
		if err != nil {
			return fmt.Errorf("read %s: %w", source, err)
		}
		// Diagrams are inlined here, while we still know which directory the page came
		// from: a chapter's fence names a file beside that chapter, and the generated
		// site keeps the same layout, so resolving at copy time is what lets the two
		// renderers agree without the page needing to know where it will be served.
		page, err := expandDiagrams(string(body), filepath.Join(root, filepath.Dir(source)), v2)
		if err != nil {
			return err
		}
		body = withDescription([]byte(page), source)
		dest := filepath.Join(srcDir, target)
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, body, 0o644); err != nil {
			return err
		}
		pages++
		return nil
	}

	// The home, install and how-it-works pages come from the binary, not the repository.
	// See writeSitePages for why the home page is not the README.
	if err := writeSitePages(srcDir); err != nil {
		return 0, err
	}
	pages += 3

	for _, ch := range allChapters(root) {
		if !ch.written() {
			continue
		}
		dir := "chapters/" + ch.Slug
		for _, doc := range []struct{ source, target string }{
			{dir + "/README.md", dir + "/index.md"},
			{dir + "/example.md", dir + "/example.md"},
		} {
			if _, err := os.Stat(filepath.Join(root, doc.source)); err != nil {
				continue
			}
			if err := copyPage(doc.source, doc.target); err != nil {
				return pages, err
			}
		}
	}

	for _, doc := range []string{
		"docs/architecture.md", "docs/chapter-format.md",
		"docs/writing-style.md", "docs/designs.md", "docs/design-audit.md",
		"chapters/where-next.md",
	} {
		if _, err := os.Stat(filepath.Join(root, doc)); err != nil {
			continue
		}
		if err := copyPage(doc, doc); err != nil {
			return pages, err
		}
	}

	// Images under docs/assets are copied so the same relative paths resolve on the
	// site as they do on GitHub. Chapter pages and the README reference them by that
	// path, so the two renderers must agree about where they live.
	if err := copyAssets(root, srcDir); err != nil {
		return pages, err
	}

	return pages, nil
}

// copyAssets mirrors docs/assets into the VitePress public directory.
//
// Files under `public` are served verbatim at the site root and are never passed through
// Rollup, which matters here: a relative image path in markdown is otherwise treated as a
// module import and fails to resolve. After this copy, `/docs/assets/logo.png` works on
// the site, and the same file is reached relatively on GitHub.
func copyAssets(root, srcDir string) error {
	src := filepath.Join(root, "docs", "assets")
	if _, err := os.Stat(src); err != nil {
		return nil
	}
	publicDir := filepath.Join(srcDir, "public", "docs", "assets")
	if err := os.MkdirAll(publicDir, 0o755); err != nil {
		return err
	}
	if err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		dest := filepath.Join(publicDir, d.Name())
		return os.WriteFile(dest, body, 0o644)
	}); err != nil {
		return err
	}

	// robots.txt points crawlers at the sitemap VitePress writes, so a new chapter is
	// discoverable without anyone submitting a URL by hand.
	robots := "User-agent: *\nAllow: /\n\nSitemap: https://miyagi.akhand.cc/sitemap.xml\n"
	return os.WriteFile(filepath.Join(srcDir, "public", "robots.txt"), []byte(robots), 0o644)
}

func writeProject(srcDir, title, root string, v2 bool) error {
	pkg := `{
  "name": "miyagi-site",
  "private": true,
  "type": "module",
  "devDependencies": {
    "mermaid": "^11.14.0",
    "vitepress": "^1.6.4",
    "vitepress-plugin-mermaid": "^2.0.17"
  }
}
`
	if err := os.WriteFile(filepath.Join(filepath.Dir(srcDir), "package.json"), []byte(pkg), 0o644); err != nil {
		return err
	}

	config := buildConfig(title, root, v2)
	dir := filepath.Join(srcDir, ".vitepress")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, "config.mts"), []byte(config), 0o644); err != nil {
		return err
	}

	// The theme directory holds the stylesheet and the logo. VitePress picks up
	// theme/index.ts by convention when a theme directory exists.
	themeDir := filepath.Join(dir, "theme")
	if err := os.MkdirAll(themeDir, 0o755); err != nil {
		return err
	}
	entry := themeEntry
	if v2 {
		entry = themeEntryV2
	}
	if err := os.WriteFile(filepath.Join(themeDir, "index.ts"), []byte(entry), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(themeDir, "custom.css"), []byte(themeCSS), 0o644); err != nil {
		return err
	}
	if !v2 {
		return nil
	}

	// The experimental build adds one stylesheet and one script. Both live in the theme
	// directory so VitePress bundles them and nothing is fetched from a CDN.
	if err := os.WriteFile(filepath.Join(themeDir, "v2.css"), []byte(v2CSS), 0o644); err != nil {
		return err
	}
	js, err := sitePages.ReadFile("pages/scene3d.js")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(themeDir, "scene3d.js"), js, 0o644)
}

// themeEntry pulls in the default theme and then our stylesheet, which is the whole
// extent of the customisation: colours, the logo lockup, and a wider content column for
// pages that carry wide tables.
//
// It also installs a scroll trigger for diagrams. That is the only script on the site,
// and it decides nothing about a diagram's content: it adds a class when a figure comes
// into view so the CSS reveal starts there instead of on page load.
const themeEntry = `import DefaultTheme from 'vitepress/theme'
import './custom.css'

export default {
  extends: DefaultTheme,
  setup() {
    if (typeof window === 'undefined') return

    const play = (fig) => {
      const svg = fig.querySelector('svg.scratch')
      if (!svg || svg.dataset.played === '1') return
      svg.dataset.played = '1'
      // Clear, reflow, re-add: an animation cannot restart without a style flush.
      svg.classList.remove('play')
      void svg.getBoundingClientRect()
      svg.classList.add('play')
    }

    const start = () => {
      const figs = [...document.querySelectorAll('figure.diagram')]
      if (!figs.length) return

      if (!('IntersectionObserver' in window)) {
        figs.forEach(play)
        return
      }
      // threshold 0.4 keeps a diagram still until it is properly in view, so a reader
      // does not catch the tail of a drawing that started off-screen.
      const io = new IntersectionObserver((entries) => {
        for (const e of entries) if (e.isIntersecting) play(e.target)
      }, { threshold: 0.4 })
      figs.forEach((f) => io.observe(f))
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', start)
    } else {
      start()
    }
    // VitePress swaps page content without a full load.
    if (window.__vp_router) window.__vp_router.onAfterRouteChanged = start
  }
}
`

// themeCSS is deliberately small. It sets a warm palette drawn from the logo, tightens
// the hero, and stops the prose column from stretching to an unreadable width on a
// large monitor. Everything else is VitePress defaults, because a course site should
// look like documentation, not like a landing page.
const themeCSS = `/**
 * Miyagi site theme.
 *
 * One accent colour, taken from the logo, plus a readable measure. The default
 * VitePress theme is already good documentation design, so this file changes as
 * little as possible.
 */

:root {
  --vp-c-brand-1: #9a4b28;
  --vp-c-brand-2: #b85c33;
  --vp-c-brand-3: #d4754a;
  --vp-c-brand-soft: rgba(154, 75, 40, 0.14);

  /* A warm off-white rather than pure white, which is easier to read for long prose. */
  --vp-c-bg: #fdfbf9;
  --vp-c-bg-alt: #f6f1ec;
  --vp-c-bg-soft: #f3ece5;

  --vp-font-family-base: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI',
    Roboto, 'Helvetica Neue', sans-serif;
  --vp-font-family-mono: 'JetBrains Mono', ui-monospace, 'SF Mono', Menlo, monospace;
}

.dark {
  --vp-c-brand-1: #e08a5c;
  --vp-c-brand-2: #d4754a;
  --vp-c-brand-3: #c2603a;
  --vp-c-brand-soft: rgba(224, 138, 92, 0.16);

  --vp-c-bg: #16120f;
  --vp-c-bg-alt: #1d1815;
  --vp-c-bg-soft: #241d18;

  /* The diagram pen lifts to stay legible on the dark ground. */
  --dg-ink: #f0e8e1;
  --dg-muted: #a2938a;
  --dg-accent: #e08a5c;
  --dg-good: #7fbf94;
  --dg-bad: #e0705a;
  --dg-violet: #b09ad6;
}

/* Hand-drawn diagrams.
 *
 * A diagram carries no colours of its own. Every stroke and label is tagged with a tone
 * class and this stylesheet decides what that tone is, which is why one committed SVG
 * works in both themes and cannot drift from the palette.
 */
:root {
  --dg-ink: #221b16;
  --dg-muted: #6f6259;
  --dg-accent: #9a4b28;
  --dg-good: #2f6b46;
  --dg-bad: #a8321f;
  --dg-violet: #6c5b9e;
}

@font-face {
  font-family: 'Patrick Hand';
  src: url('/docs/assets/PatrickHand-Regular.woff2') format('woff2');
  font-display: swap;
}

figure.diagram {
  margin: 1.9rem 0;
}

figure.diagram svg.scratch {
  display: block;
  width: 100%;
  height: auto;
  overflow: visible;
}

figure.diagram figcaption {
  margin-top: 0.55rem;
  font-size: 0.82rem;
  line-height: 1.5;
  color: var(--vp-c-text-2);
  border-top: 1px solid var(--vp-c-divider);
  padding-top: 0.45rem;
}

.scratch .s-line {
  fill: none;
  stroke: var(--dg-ink);
  stroke-linecap: round;
  stroke-linejoin: round;
  stroke-dasharray: 1;
  stroke-dashoffset: 1;
  animation-duration: var(--t1);
  animation-delay: var(--t0);
  animation-timing-function: linear;
  animation-fill-mode: both;
}
.scratch .s-line.t-accent { stroke: var(--dg-accent); }
.scratch .s-line.t-good { stroke: var(--dg-good); }
.scratch .s-line.t-bad { stroke: var(--dg-bad); }
.scratch .s-line.t-violet { stroke: var(--dg-violet); }
.scratch .s-line.t-muted { stroke: var(--dg-muted); }

.scratch .s-text {
  fill: var(--dg-ink);
  font-family: 'Patrick Hand', ui-rounded, 'Comic Sans MS', cursive;
  animation-duration: var(--t1);
  animation-delay: var(--t0);
  animation-timing-function: linear;
  animation-fill-mode: both;
}
.scratch .s-text.t-accent { fill: var(--dg-accent); }
.scratch .s-text.t-good { fill: var(--dg-good); }
.scratch .s-text.t-bad { fill: var(--dg-bad); }
.scratch .s-text.t-violet { fill: var(--dg-violet); }
.scratch .s-text.t-muted { fill: var(--dg-muted); }

.scratch .s-fill {
  fill: var(--dg-ink);
  fill-opacity: 0.1;
  opacity: 0;
  animation-fill-mode: both;
  animation-timing-function: ease;
}
.scratch .s-fill.t-accent { fill: var(--dg-accent); }
.scratch .s-fill.t-good { fill: var(--dg-good); }
.scratch .s-fill.t-bad { fill: var(--dg-bad); }
.scratch .s-fill.t-violet { fill: var(--dg-violet); }

.scratch .s-clip {
  animation-duration: var(--t1);
  animation-delay: var(--t0);
  animation-timing-function: linear;
  animation-fill-mode: both;
}

/* Before the figure is in view the drawing is complete, not blank. A reader who never
 * scrolls it into view, has no JavaScript, or prefers reduced motion still gets the whole
 * diagram; only the reveal is optional. */
.scratch:not(.play) .s-line { stroke-dashoffset: 0; }
.scratch:not(.play) .s-fill { opacity: 1; }
.scratch:not(.play) .s-clip { width: 100% !important; animation: none; }

@media (prefers-reduced-motion: reduce) {
  .scratch .s-line { stroke-dashoffset: 0 !important; animation: none !important; }
  .scratch .s-text { animation: none !important; }
  .scratch .s-fill { opacity: 1 !important; animation: none !important; }
  .scratch .s-clip { width: 100% !important; animation: none !important; }
}

/* The hero: logo and title side by side, tagline underneath. */
.VPHero .name,
.VPHero .text {
  letter-spacing: -0.02em;
}

.VPHero .tagline {
  max-width: 34rem;
  margin-inline: auto;
}

/* Long-form pages carry wide tables and code; give them room, but stop before the
   line becomes hard to track. */
@media (min-width: 960px) {
  .VPDoc .content-container {
    max-width: 46rem;
  }
}

/* Chapter pages open with a metadata line in bold, which reads as a caption rather
   than a heading. */
.vp-doc > p:first-of-type strong:first-child {
  font-weight: 600;
}

/* Inline code should not shout. */
.vp-doc :not(pre) > code {
  font-size: 0.86em;
}

/* The course leans on tables; give them a little more breathing room. */
.vp-doc table {
  display: table;
  width: 100%;
}

.vp-doc tr:nth-child(2n) {
  background: var(--vp-c-bg-soft);
}
`

// buildConfig writes the VitePress configuration, including the sidebar.
//
// The sidebar is the only part of the site that knows about the course, and it is
// derived from the chapter directories, so adding a chapter never means editing a
// config by hand.
func buildConfig(title, root string, v2 bool) string {
	base := "/"
	if v2 {
		// The experimental site is served under /v2, so VitePress must prefix every
		// asset and internal link with that base or the whole build 404s.
		base = "/v2/"
	}
	var b strings.Builder

	b.WriteString("import { defineConfig } from 'vitepress'\n")
	b.WriteString("import { withMermaid } from 'vitepress-plugin-mermaid'\n\n")
	b.WriteString("export default withMermaid(defineConfig({\n")
	fmt.Fprintf(&b, "  title: %s,\n", strconv.Quote(title))
	b.WriteString("  description: 'Build a coding-agent harness in Go, graded offline by a local verifier.',\n")
	b.WriteString("  lang: 'en',\n")
	fmt.Fprintf(&b, "  base: %s,\n", strconv.Quote(base))
	// The logo and favicon are copied into the site source from docs/assets, so the
	// repository and the published site use one source of art.
	//
	// `cleanUrls` gives every page a path without `.html`, which is what search engines
	// and shared links should see.
	b.WriteString("  cleanUrls: true,\n")
	b.WriteString("  sitemap: { hostname: 'https://miyagi.akhand.cc' },\n")
	b.WriteString("  head: [\n")
	b.WriteString("    ['link', { rel: 'icon', href: '/docs/assets/favicon.ico', sizes: 'any' }],\n")
	b.WriteString("    ['link', { rel: 'apple-touch-icon', href: '/docs/assets/logo-128.png' }],\n")
	b.WriteString("    ['meta', { name: 'theme-color', content: '#9a4b28' }],\n")
	b.WriteString("    ['meta', { name: 'author', content: 'Akhand Patel' }],\n")
	b.WriteString("    ['meta', { name: 'robots', content: 'index, follow' }],\n")
	// The share card. Absolute URLs are required by every crawler.
	b.WriteString("    ['meta', { property: 'og:type', content: 'website' }],\n")
	b.WriteString("    ['meta', { property: 'og:site_name', content: 'Miyagi' }],\n")
	b.WriteString("    ['meta', { property: 'og:title', content: 'Miyagi: build a coding-agent harness in Go' }],\n")
	b.WriteString("    ['meta', { property: 'og:description', content: 'Twenty-one graded chapters that take you from one HTTP request to a working agent. Offline, deterministic, no API key.' }],\n")
	b.WriteString("    ['meta', { property: 'og:image', content: 'https://miyagi.akhand.cc/docs/assets/logo.png' }],\n")
	b.WriteString("    ['meta', { property: 'og:url', content: 'https://miyagi.akhand.cc/' }],\n")
	b.WriteString("    ['meta', { name: 'twitter:card', content: 'summary_large_image' }],\n")
	b.WriteString("    ['meta', { name: 'twitter:title', content: 'Miyagi: build a coding-agent harness in Go' }],\n")
	b.WriteString("    ['meta', { name: 'twitter:description', content: 'Twenty-one graded chapters that take you from one HTTP request to a working agent. Offline, deterministic, no API key.' }],\n")
	b.WriteString("    ['meta', { name: 'twitter:image', content: 'https://miyagi.akhand.cc/docs/assets/logo.png' }],\n")
	b.WriteString("  ],\n")
	b.WriteString("  themeConfig: {\n")
	b.WriteString("    logo: { src: '/docs/assets/logo-128.png', alt: 'Miyagi', width: 32, height: 32 },\n")
	b.WriteString("    search: { provider: 'local' },\n")
	b.WriteString("    outline: { level: [2, 3] },\n")
	b.WriteString("    nav: [\n")
	b.WriteString("      { text: 'Home', link: '/' },\n")
	if first := firstChapterLink(root); first != "" {
		fmt.Fprintf(&b, "      { text: 'Chapters', link: %s },\n", strconv.Quote(first))
	}
	if v2 {
		b.WriteString("      { text: 'v2 experimental', link: '/v2' },\n")
	}
	b.WriteString("      { text: 'Install', link: '/install' },\n")
	b.WriteString("      { text: 'How it works', link: '/how-it-works' },\n")
	b.WriteString("      { text: 'Designs', link: '/docs/designs' },\n")
	b.WriteString("      { text: 'GitHub', link: 'https://github.com/theakhandpatel/miyagi' },\n")
	b.WriteString("    ],\n")
	b.WriteString("    sidebar: [\n")
	b.WriteString("      { text: 'Start', items: [\n")
	b.WriteString("        { text: 'What Miyagi is', link: '/' },\n")
	b.WriteString("        { text: 'Install', link: '/install' },\n")
	b.WriteString("        { text: 'How it works', link: '/how-it-works' },\n")
	b.WriteString("      ] },\n")

	for _, part := range partTitles() {
		var items []string
		for _, ch := range allChapters(root) {
			if !ch.written() || partOf(ch.ID) != part {
				continue
			}
			base := "/chapters/" + ch.Slug + "/"
			items = append(items, fmt.Sprintf(
				"        { text: %s, link: %s, collapsed: false, items: [\n"+
					"          { text: 'The chapter', link: %s },\n"+
					"          { text: 'Worked example', link: %s },\n"+
					"        ] },",
				strconv.Quote(ch.ID+" "+ch.Title), strconv.Quote(base),
				strconv.Quote(base), strconv.Quote(base+"example")))
		}
		if len(items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "      { text: %s, collapsed: false, items: [\n%s\n      ] },\n",
			strconv.Quote(part), strings.Join(items, "\n"))
	}

	b.WriteString(`      { text: 'Reference', collapsed: false, items: [
        { text: 'Architecture', link: '/docs/architecture' },
        { text: 'Chapter format', link: '/docs/chapter-format' },
        { text: 'Writing style', link: '/docs/writing-style' },
        { text: 'Designs', link: '/docs/designs' },
        { text: 'Design audit', link: '/docs/design-audit' },
      ] },
`)
	b.WriteString("      { text: 'After the course', items: [{ text: 'Where next', link: '/chapters/where-next' }] },\n")
	b.WriteString("    ],\n")
	b.WriteString("    // No editLink: the generated site renames README.md to index.md, so a link\n")
	b.WriteString("    // derived from the page path points at a file the repository does not have.\n")
	b.WriteString("    socialLinks: [{ icon: 'github', link: 'https://github.com/theakhandpatel/miyagi' }],\n")
	b.WriteString("  },\n")
	b.WriteString("  mermaid: {},\n")
	b.WriteString("}))\n")

	return b.String()
}

// firstChapterLink points the top navigation at whichever chapter is first on disk.
// partTitles returns the part headings in order, from the same table the chapter list
// uses, so a new part appears on the site without editing this file.
func partTitles() []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		out = append(out, p.Title)
	}
	return out
}

func firstChapterLink(root string) string {
	all := allChapters(root)
	if len(all) == 0 {
		return ""
	}
	return "/chapters/" + all[0].Slug + "/"
}

func ensureDependencies(siteDir string) error {
	if _, err := os.Stat(filepath.Join(siteDir, "node_modules", "vitepress")); err == nil {
		return nil
	}
	fmt.Println("  installing site dependencies (first run only)")
	cmd := exec.Command("npm", "install", "--no-audit", "--no-fund")
	cmd.Dir = siteDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install: %w", err)
	}
	return nil
}

func build(siteDir string) error {
	cmd := exec.Command("npx", "vitepress", "build", "src")
	cmd.Dir = siteDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("vitepress build: %w", err)
	}
	return nil
}

// withDescription gives every page its own meta description.
//
// VitePress uses one site-wide description for the `description` tag unless a page
// declares its own front matter. A course with 21 similarly named chapters then presents
// search engines with 21 identical snippets, which is the difference between "22 results
// that all look the same" and a result a reader can choose between. The chapter's own
// opening sentence is already written for a human, so it is used verbatim, trimmed to a
// length crawlers display.
//
// A page that already has front matter is left alone: the author said what they wanted.
func withDescription(body []byte, source string) []byte {
	text := string(body)
	if strings.HasPrefix(text, "---\n") {
		return body
	}

	desc := firstSentence(text)
	if desc == "" {
		return body
	}

	var b strings.Builder
	b.WriteString("---\n")
	fmt.Fprintf(&b, "description: %s\n", strconv.Quote(desc))
	b.WriteString("---\n\n")
	b.WriteString(text)
	return []byte(b.String())
}

// firstSentence finds the first real paragraph that is not a heading, code fence, table,
// image or HTML block, and returns its first sentence, capped at about 155 characters
// because that is all a search result shows.
func firstSentence(text string) string {
	lines := strings.Split(text, "\n")
	inFence := false
	var para []string

	flush := func() string {
		if len(para) == 0 {
			return ""
		}
		joined := strings.Join(para, " ")
		para = nil
		// Cut at the first sentence end.
		for i, r := range joined {
			if (r == '.' || r == '!' || r == '?') && i+1 < len(joined) {
				if joined[i+1] == ' ' {
					return trimToSentence(joined[:i+1])
				}
			}
		}
		return trimToSentence(joined)
	}

	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		switch {
		case line == "":
			if s := flush(); s != "" {
				return s
			}
			continue
		case strings.HasPrefix(line, "#"),
			strings.HasPrefix(line, "|"),
			strings.HasPrefix(line, ">"),
			strings.HasPrefix(line, "<"),
			strings.HasPrefix(line, "!["),
			strings.HasPrefix(line, "-"),
			strings.HasPrefix(line, "*"),
			strings.HasPrefix(line, "1."),
			strings.HasPrefix(line, "**Part"):
			continue
		}
		// Strip inline markup a crawler would otherwise show literally.
		line = strings.ReplaceAll(line, "`", "")
		line = strings.ReplaceAll(line, "**", "")
		para = append(para, line)
	}
	return flush()
}

// trimToSentence caps a description without cutting mid-word.
func trimToSentence(s string) string {
	const max = 155
	if len(s) <= max {
		return s
	}
	cut := s[:max]
	if i := strings.LastIndexByte(cut, ' '); i > 0 {
		cut = cut[:i]
	}
	return strings.TrimRight(cut, " ,;:") + "."
}


// The site's own pages, embedded so the binary can generate a site with no repository
// pages beside it. Only the build machine needs the source; `miyagi` carries it.
//
//go:embed pages/*.md pages/*.js
var sitePages embed.FS

// writeSitePages copies the embedded pages into the generated project.
//
// The home page is deliberately not the repository README: a README is for someone already
// looking at the repository, while the site's front page is for someone who is not, so it
// leads with what the thing does and the one command that starts it.
func writeSitePages(srcDir string) error {
	// Only the .md pages become site pages. scene3d.js is embedded too, but it is copied
	// into the theme directory by writeProject rather than published as a page.
	entries, err := fs.Glob(sitePages, "pages/*.md")
	if err != nil {
		return err
	}
	for _, name := range entries {
		body, err := sitePages.ReadFile(name)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(srcDir, filepath.Base(name)), body, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// themeEntryV2 is the experimental theme entry.
//
// It adds the scene stylesheet and the renderer on top of the default theme, and keeps
// everything the shipped theme already does. The renderer is loaded as a module so it is
// bundled and tree-shaken with the rest of the site rather than fetched at runtime.
const themeEntryV2 = `import DefaultTheme from 'vitepress/theme'
import './custom.css'
import './v2.css'
import './scene3d.js'

export default {
  extends: DefaultTheme,
  setup() {
    if (typeof window === 'undefined') return

    const play = (fig) => {
      const svg = fig.querySelector('svg.scratch')
      if (!svg || svg.dataset.played === '1') return
      svg.dataset.played = '1'
      svg.classList.remove('play')
      void svg.getBoundingClientRect()
      svg.classList.add('play')
    }

    // In the experimental build the flat figure is the fallback, not the subject, so it is
    // only revealed when the renderer could not take over.
    const sync = () => {
      const state = document.documentElement.getAttribute('data-scene3d')
      document.querySelectorAll('figure.diagram').forEach((fig) => {
        if (state === 'unsupported' || state === 'failed') fig.classList.remove('is-fallback')
        else fig.classList.add('is-fallback')
      })
    }

    const start = () => {
      const figs = [...document.querySelectorAll('figure.diagram')]
      if (!figs.length) return
      sync()
      if (!('IntersectionObserver' in window)) {
        figs.forEach(play)
        return
      }
      const io = new IntersectionObserver((entries) => {
        for (const e of entries) if (e.isIntersecting) play(e.target)
      }, { threshold: 0.4 })
      figs.forEach((f) => io.observe(f))
      // The renderer sets the attribute asynchronously, so watch for it.
      new MutationObserver(sync).observe(document.documentElement, {
        attributes: true, attributeFilter: ['data-scene3d'],
      })
    }

    if (document.readyState === 'loading') {
      document.addEventListener('DOMContentLoaded', start)
    } else {
      start()
    }
  },
}
`

// v2CSS is the experimental visual language.
//
// The shipped site is deliberately quiet. This build is brighter and denser, which is a
// different bet: that a reader learns a mechanism faster from a thing they can turn over
// than from a figure they can only look at. It is behind a flag because that bet is not
// obviously right, and the quieter version is the one that shipped.
const v2CSS = `/**
 * Miyagi v2, experimental.
 *
 * A denser, more visual treatment: a stage for the scene, a gradient ground, and figures
 * sized to be the thing you look at rather than an illustration beside the text.
 */

:root {
  --scene-height: 380px;
  --scene-radius: 14px;
  --scene-edge: var(--vp-c-divider);
}

.scene3d-wrap {
  position: relative;
  margin: 26px 0 10px;
  border-radius: var(--scene-radius);
  border: 1px solid var(--scene-edge);
  background:
    radial-gradient(120% 90% at 50% 0%, var(--vp-c-bg-soft) 0%, var(--vp-c-bg) 62%);
  overflow: hidden;
}

canvas.scene3d {
  display: block;
  width: 100%;
  height: var(--scene-height);
  touch-action: pan-y;
  outline: none;
}

canvas.scene3d:focus-visible {
  box-shadow: inset 0 0 0 2px var(--vp-c-brand-1);
}

/* While the renderer is running, the flat figure is hidden: it is the fallback, and
   showing both would say the same thing twice. */
figure.diagram.is-fallback svg.scratch {
  display: none;
}

/* A ghost object must be visibly an object: enough opacity to see its edges, clearly less
   than a solid, so "discarded" reads as discarded rather than as missing. */
.scene3d-label.is-ghost {
  opacity: 0.85;
  text-decoration: line-through;
  text-decoration-thickness: 1px;
}

/* If WebGL is missing the canvas is removed by the renderer and the figure returns. */
figure.diagram:not(.is-fallback) svg.scratch {
  display: none;
}

/* The legend beneath a scene. Deterministic placement, so it can never cover the figure,
   and it carries the hue swatch so a reader can map colour to meaning without a caption. */
.scene3d-legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 16px;
  margin: 10px 0 0;
  padding: 12px 2px 2px;
  list-style: none;
  border-top: 1px dashed var(--vp-c-divider);
}

.scene3d-legend-item {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  font: 500 12px/1.4 var(--vp-font-family-mono, ui-monospace, monospace);
  color: var(--vp-c-text-2);
}

.scene3d-legend-item.is-ghost {
  color: var(--vp-c-text-3);
  text-decoration: line-through;
  text-decoration-thickness: 1px;
}

.scene3d-legend-item.is-ghost .scene3d-swatch {
  background: transparent;
  border-style: dashed;
}

.scene3d-swatch {
  width: 13px;
  height: 13px;
  border-radius: 3px;
  border: 1px solid var(--vp-c-divider);
  flex: none;
}

.scene3d-swatch[data-hue="brand"] { background: var(--vp-c-brand-1); }
.scene3d-swatch[data-hue="good"]  { background: var(--vp-c-green-1); }
.scene3d-swatch[data-hue="bad"]   { background: var(--vp-c-danger-1); }
.scene3d-swatch[data-hue="line"]  { background: var(--vp-c-text-2); }
.scene3d-swatch[data-hue="muted"] { background: var(--vp-c-text-3); }

.markdown-only {
  position: absolute;
  width: 1px; height: 1px;
  padding: 0; margin: -1px;
  overflow: hidden; clip: rect(0 0 0 0);
  white-space: nowrap; border: 0;
}

/* The hero carries the scene's colour language: a soft wash behind the logo, so the front
   page is recognisably the same product as the chapters. */
.VPHero .image-bg {
  background: radial-gradient(closest-side, var(--vp-c-brand-soft), transparent 72%);
}

@media (max-width: 640px) {
  :root { --scene-height: 300px; }
}

@media (prefers-reduced-motion: reduce) {
  /* The renderer already draws the finished frame; this covers the CSS reveal. */
  svg.scratch * { animation: none !important; }
}
`

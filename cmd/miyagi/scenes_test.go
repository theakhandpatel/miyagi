package main

import (
	"strings"
	"testing"
)

// The experimental scenes are authored data, and authored data drifts. These tests pin the
// properties the renderer and the prose both depend on.

func TestEveryChapterHasAScene(t *testing.T) {
	for _, ch := range allChapters("../../") {
		if !ch.written() {
			continue
		}
		if _, ok := scenes[ch.ID]; !ok {
			t.Errorf("chapter %s has a page but no scene", ch.ID)
		}
	}
}

// A group named by a move but used by no object animates nothing, and an object in a group
// no move names appears fully formed with no reveal. Both are silent, so both are errors.
func TestSceneGroupsAreConsistent(t *testing.T) {
	for id, sc := range scenes {
		used, moved := map[string]bool{}, map[string]bool{}
		for _, o := range sc.objects {
			used[o.group] = true
			if o.group == "" {
				t.Errorf("scene %s has an object with no group (%q)", id, o.label)
			}
		}
		for _, m := range sc.moves {
			moved[m.group] = true
			if !used[m.group] {
				t.Errorf("scene %s moves group %q, which no object belongs to", id, m.group)
			}
		}
		for g := range used {
			if !moved[g] {
				t.Errorf("scene %s has group %q that no move animates", id, g)
			}
		}
	}
}

// Every scene needs a caption, because the terminal renderer prints the note in place of
// the figure and the site uses it as the figure's alt text.
func TestEverySceneExplainsItself(t *testing.T) {
	for id, sc := range scenes {
		if strings.TrimSpace(sc.note) == "" {
			t.Errorf("scene %s has no note", id)
		}
		if len(sc.note) > 320 {
			t.Errorf("scene %s note is %d chars; it is read as a caption, keep it under 320",
				id, len(sc.note))
		}
		if strings.TrimSpace(sc.title) == "" {
			t.Errorf("scene %s has no title", id)
		}
	}
}

// The palette is deliberately small and shared. A hue outside it renders as the fallback
// colour, which turns a meaningful distinction into a duplicate.
func TestSceneHuesAreFromThePalette(t *testing.T) {
	allowed := map[string]bool{
		"brand": true, "good": true, "bad": true, "line": true, "muted": true, "ink": true,
	}
	for id, sc := range scenes {
		for _, o := range sc.objects {
			if !allowed[o.hue] {
				t.Errorf("scene %s uses hue %q, which the renderer does not define", id, o.hue)
			}
		}
	}
}

// Sizes and positions must be finite-ish numbers that produce a non-degenerate bounding
// box: the camera fit divides by the extent, so a zero extent divides by zero.
func TestScenesHaveNonDegenerateExtents(t *testing.T) {
	for id, sc := range scenes {
		if len(sc.objects) == 0 {
			t.Errorf("scene %s has no objects", id)
			continue
		}
		loX, loY, hiX, hiY := 1e9, 1e9, -1e9, -1e9
		for _, o := range sc.objects {
			if o.size.X <= 0 || o.size.Y <= 0 || o.size.Z <= 0 {
				t.Errorf("scene %s object %q has a non-positive size %v", id, o.label, o.size)
			}
			loX = min(loX, o.at.X-o.size.X/2)
			hiX = max(hiX, o.at.X+o.size.X/2)
			loY = min(loY, o.at.Y-o.size.Y/2)
			hiY = max(hiY, o.at.Y+o.size.Y/2)
		}
		if w, h := hiX-loX, hiY-loY; w < 0.5 || h < 0.5 {
			t.Errorf("scene %s extent is %.2f x %.2f, too small to frame", id, w, h)
		}
	}
}

// Labels become legend entries. A scene whose every object is unlabelled has nothing to
// explain it, and a duplicate label is two entries that say the same thing.
func TestScenesCarryEnoughLabels(t *testing.T) {
	for id, sc := range scenes {
		seen := map[string]bool{}
		labelled := 0
		for _, o := range sc.objects {
			if o.label == "" {
				continue
			}
			labelled++
			if seen[o.label] {
				t.Errorf("scene %s repeats the label %q; a legend entry must be unique", id, o.label)
			}
			seen[o.label] = true
		}
		if labelled < 2 {
			t.Errorf("scene %s has %d labelled objects; a figure needs at least two to mean anything", id, labelled)
		}
	}
}

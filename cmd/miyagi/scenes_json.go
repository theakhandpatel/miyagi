package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// The experimental site renders each chapter's figure as a three dimensional scene.
//
// The scenes are authored in Go (see scenes.go) rather than in JavaScript, for the same
// reason the rest of the course is: they are then reviewable next to the prose that makes
// the same claim, and a scene that contradicts its chapter is caught in review rather than
// discovered by a reader. The renderer is deliberately dumb. It receives a list of objects
// and a list of moves and plays them.
//
// Kept behind `miyagi site --v2` while the flat figures remain the default. A version flag
// rather than a content flag, because the distinction being tested is the whole visual
// language, not one page.

// sceneJSON is the shape the browser receives. Field names are short because there are
// several hundred objects across the site and the payload is inlined per page.
type sceneJSON struct {
	Title   string      `json:"title"`
	Note    string      `json:"note"`
	Yaw     float64     `json:"yaw"`
	Pitch   float64     `json:"pitch"`
	Dist    float64     `json:"dist"`
	Objects []objectJSON `json:"objects"`
	Moves   []moveJSON   `json:"moves"`
}

type objectJSON struct {
	Form  string     `json:"f"`
	At    [3]float64 `json:"p"`
	Size  [3]float64 `json:"s"`
	Hue   string     `json:"c"`
	Label string     `json:"l,omitempty"`
	Ghost bool       `json:"g,omitempty"`
	Group string     `json:"k,omitempty"`
}

type moveJSON struct {
	Group string     `json:"k"`
	Kind  string     `json:"m"`
	To    [3]float64 `json:"t"`
	Delay float64    `json:"d"`
	Dur   float64    `json:"u"`
}

// sceneFor converts the authored scene into the wire shape.
func sceneFor(id string) (sceneJSON, bool) {
	s, ok := scenes[id]
	if !ok {
		return sceneJSON{}, false
	}
	out := sceneJSON{
		Title: s.title, Note: s.note,
		Yaw: s.yaw, Pitch: s.pitch, Dist: s.dist,
	}
	for _, o := range s.objects {
		out.Objects = append(out.Objects, objectJSON{
			Form: o.form,
			At:   [3]float64{o.at.X, o.at.Y, o.at.Z},
			Size: [3]float64{o.size.X, o.size.Y, o.size.Z},
			Hue:  o.hue,
			Label: o.label,
			Ghost: o.ghost,
			Group: o.group,
		})
	}
	for _, m := range s.moves {
		out.Moves = append(out.Moves, moveJSON{
			Group: m.group,
			Kind:  m.kind,
			To:    [3]float64{m.to.X, m.to.Y, m.to.Z},
			Delay: m.delay,
			Dur:   m.dur,
		})
	}
	return out, true
}

// writeScenes emits one JSON file per chapter into the generated site.
//
// One file per chapter rather than one blob, so a page only downloads the scene it shows.
// The renderer fetches its own scene by chapter id.
func writeScenes(srcDir string) (int, error) {
	dir := filepath.Join(srcDir, "public", "scenes")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return 0, err
	}

	ids := make([]string, 0, len(scenes))
	for id := range scenes {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	for _, id := range ids {
		sc, ok := sceneFor(id)
		if !ok {
			continue
		}
		body, err := json.Marshal(sc)
		if err != nil {
			return 0, fmt.Errorf("marshal scene %s: %w", id, err)
		}
		if err := os.WriteFile(filepath.Join(dir, id+".json"), body, 0o644); err != nil {
			return 0, err
		}
	}
	return len(ids), nil
}

package main

// Scenes for the experimental three dimensional diagrams.
//
// Each chapter's mechanism is a thing with a shape, and the shape is chosen from what the
// chapter actually teaches rather than from what looks good in a render. A scene that
// could illustrate any chapter is a scene that illustrates none of them, so every entry
// below names the specific quantity or ordering its chapter is about.
//
// The vocabulary is deliberately small. A scene is objects and moves: an object has a
// form, a position, a colour and a size, and a move is either a reveal, a travel, or a
// pulse. Anything more expressive would need a real 3D authoring tool, and then the
// figures would stop being reviewable against the prose.

// vec3 is a position in scene space. Units are arbitrary and normalised by the renderer.
type vec3 struct{ X, Y, Z float64 }

// obj is one thing in a scene.
type obj struct {
	// form is one of: frame, sheet, band, column, cube, orb, ring, stack, wedge.
	form string
	at   vec3
	// size is width, height, depth. The renderer clamps these.
	size vec3
	// hue is an index into the palette, so a theme change recolours every scene at once.
	hue   string
	label string
	// ghost objects are drawn as edges only, for the thing that is absent or dropped.
	ghost bool
	// group ties objects that move together.
	group string
}

// move is one animation applied to a group.
type move struct {
	group string
	// kind is reveal, travel, pulse, or orbit.
	kind string
	// to is the travel destination. Ignored by reveal and pulse.
	to vec3
	// delay and dur are in seconds, relative to the scene starting.
	delay float64
	dur   float64
}

// scene is one chapter's figure.
type scene struct {
	// title is the chapter's own words for the mechanism.
	title string
	// note is what the reader should come away with. It is the caption, so it must be a
	// claim the chapter already makes.
	note string
	// camera is the initial orbit: yaw and pitch in degrees, and distance.
	yaw     float64
	pitch   float64
	dist    float64
	objects []obj
	moves   []move
}

// scenes is the whole set, keyed by chapter number.
//
// The rule for inclusion: a scene earns its place only if the third dimension carries
// information the flat figure cannot. Where it does not, the scene is still built, but it
// leans on motion rather than depth, and says so in its note.
var scenes = map[string]scene{

	"01": {
		title: "Frames arriving and being appended",
		note:  "Text arrives as frames. The box on the right grows only if each delta is appended to what is already there.",
		yaw:   -28, pitch: 14, dist: 11,
		objects: []obj{
			{form: "frame", at: vec3{-3.2, 1.6, 0}, size: vec3{1.5, 0.9, 0.12}, hue: "line", label: "delta 1", group: "f1"},
			{form: "frame", at: vec3{-3.2, 0.4, 0}, size: vec3{1.5, 0.9, 0.12}, hue: "line", label: "delta 2", group: "f2"},
			{form: "frame", at: vec3{-3.2, -0.8, 0}, size: vec3{1.5, 0.9, 0.12}, hue: "line", label: "delta 3", group: "f3"},
			{form: "stack", at: vec3{1.9, 0, 0}, size: vec3{2.6, 3.0, 0.5}, hue: "brand", label: "text so far", group: "acc"},
		},
		moves: []move{
			{group: "f1", kind: "reveal", delay: 0.2, dur: 0.5},
			{group: "f1", kind: "travel", to: vec3{1.0, 0.8, 0}, delay: 0.8, dur: 0.9},
			{group: "f2", kind: "reveal", delay: 1.0, dur: 0.5},
			{group: "f2", kind: "travel", to: vec3{1.0, 0.1, 0}, delay: 1.6, dur: 0.9},
			{group: "f3", kind: "reveal", delay: 1.8, dur: 0.5},
			{group: "f3", kind: "travel", to: vec3{1.0, -0.6, 0}, delay: 2.4, dur: 0.9},
			{group: "acc", kind: "pulse", delay: 3.3, dur: 0.8},
		},
	},

	"02": {
		title: "The same fragments, folded two ways",
		note:  "Three fragments arrive and only the first carries an id. Keying on the index folds all three into one call, three deep. Keying on the id keeps fragment one alone, and the other two are simply not there.",
		yaw:   -4, pitch: 8, dist: 12,
		objects: []obj{
			// Arrivals. Every fragment is the same neutral colour: they are inputs, not
			// outcomes, and colouring them would overload the palette that the outcomes use.
			{form: "band", at: vec3{-3.6, 2.6, 0}, size: vec3{1.7, 0.42, 0.2}, hue: "line", label: "arrives: frag 1, carries the id", group: "a"},
			{form: "band", at: vec3{-3.6, 1.95, 0}, size: vec3{1.7, 0.42, 0.2}, hue: "line", label: "arrives: frag 2, args", group: "b"},
			{form: "band", at: vec3{-3.6, 1.3, 0}, size: vec3{1.7, 0.42, 0.2}, hue: "line", label: "arrives: frag 3, args", group: "c"},

			// The two outcomes share one baseline, so their heights compare directly.
			// Green is kept, red is what you are left holding. One hue, one meaning.
			{form: "band", at: vec3{0.6, 2.30, 0}, size: vec3{1.7, 0.42, 1.4}, hue: "good", label: "by index: three folded", group: "i1"},
			{form: "band", at: vec3{0.6, 1.86, 0}, size: vec3{1.7, 0.42, 1.4}, hue: "good", label: "", group: "i2"},
			{form: "band", at: vec3{0.6, 1.42, 0}, size: vec3{1.7, 0.42, 1.4}, hue: "good", label: "", group: "i3"},

			// One band, sitting at the same baseline as the bottom of the tall stack, so the
			// missing two are read as absent height rather than as a different object.
			{form: "band", at: vec3{3.4, 1.42, 0}, size: vec3{1.7, 0.42, 1.4}, hue: "bad", label: "by id: one kept", group: "d1"},
		},
		moves: []move{
			{group: "a", kind: "reveal", delay: 0.2, dur: 0.35},
			{group: "b", kind: "reveal", delay: 0.6, dur: 0.35},
			{group: "c", kind: "reveal", delay: 1.0, dur: 0.35},
			{group: "i1", kind: "reveal", delay: 1.6, dur: 0.3},
			{group: "i2", kind: "reveal", delay: 1.85, dur: 0.3},
			{group: "i3", kind: "reveal", delay: 2.1, dur: 0.3},
			{group: "d1", kind: "reveal", delay: 2.6, dur: 0.5},
			{group: "d1", kind: "pulse", delay: 3.3, dur: 0.9},
		},
	},

	"03": {
		title: "Two requests make one answer",
		note:  "The model asks for a call, the program runs it, and the result goes back under the id that asked for it.",
		yaw:   -40, pitch: 18, dist: 13,
		objects: []obj{
			{form: "orb", at: vec3{-2.6, 1.4, 0}, size: vec3{1.1, 1.1, 1.1}, hue: "brand", label: "model", group: "m"},
			{form: "column", at: vec3{0, 1.4, 0}, size: vec3{0.7, 2.0, 0.7}, hue: "line", label: "program", group: "p"},
			{form: "cube", at: vec3{2.6, 1.4, 0}, size: vec3{1.1, 1.1, 1.1}, hue: "good", label: "tool", group: "t"},
			{form: "band", at: vec3{0, -1.4, 0}, size: vec3{6.4, 0.28, 0.28}, hue: "brand", label: "request 1 → call → run → request 2", group: "path"},
		},
		moves: []move{
			{group: "m", kind: "reveal", delay: 0.1, dur: 0.4},
			{group: "p", kind: "reveal", delay: 0.4, dur: 0.4},
			{group: "t", kind: "reveal", delay: 0.7, dur: 0.4},
			{group: "path", kind: "reveal", delay: 1.0, dur: 0.6},
			{group: "m", kind: "pulse", delay: 1.8, dur: 0.6},
			{group: "t", kind: "pulse", delay: 2.4, dur: 0.6},
			{group: "m", kind: "pulse", delay: 3.0, dur: 0.6},
		},
	},

	"04": {
		title: "Three walls around one tool",
		note:  "A tool that touches the world needs three limits: a path that cannot escape, output that cannot flood, and a deadline.",
		yaw:   -36, pitch: 20, dist: 12,
		objects: []obj{
			{form: "cube", at: vec3{0, 0, 0}, size: vec3{2.2, 2.2, 2.2}, hue: "brand", label: "tool", group: "tool"},
			{form: "sheet", at: vec3{-2.4, 0, 0}, size: vec3{0.08, 3.4, 3.4}, hue: "line", label: "path", group: "w1"},
			{form: "sheet", at: vec3{2.4, 0, 0}, size: vec3{0.08, 3.4, 3.4}, hue: "line", label: "size", group: "w2"},
			{form: "sheet", at: vec3{0, -2.4, 0}, size: vec3{3.4, 0.08, 3.4}, hue: "line", label: "deadline", group: "w3"},
		},
		moves: []move{
			{group: "tool", kind: "reveal", delay: 0.2, dur: 0.5},
			{group: "w1", kind: "reveal", delay: 0.8, dur: 0.5},
			{group: "w2", kind: "reveal", delay: 1.2, dur: 0.5},
			{group: "w3", kind: "reveal", delay: 1.6, dur: 0.5},
			{group: "tool", kind: "pulse", delay: 2.3, dur: 1.0},
		},
	},

	"05": {
		title: "A loop that stops asking",
		note:  "Each turn asks, runs what it was asked for, and answers. The loop continues while calls come back; a turn with none ends it.",
		yaw:   -46, pitch: 22, dist: 14,
		objects: []obj{
			{form: "ring", at: vec3{0, 0, 0}, size: vec3{4.2, 4.2, 0.5}, hue: "line", label: "turn", group: "ring"},
			{form: "cube", at: vec3{-1.5, 0, 1.0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "ask", group: "t1"},
			{form: "cube", at: vec3{0, 1.5, 1.0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "run", group: "t2"},
			{form: "cube", at: vec3{1.5, 0, 1.0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "answer", group: "t3"},
			{form: "orb", at: vec3{0, -1.6, 1.0}, size: vec3{0.7, 0.7, 0.7}, hue: "good", label: "no calls: stop", group: "stop"},
		},
		moves: []move{
			{group: "ring", kind: "reveal", delay: 0.1, dur: 0.5},
			{group: "t1", kind: "reveal", delay: 0.5, dur: 0.4},
			{group: "t2", kind: "reveal", delay: 1.1, dur: 0.4},
			{group: "t3", kind: "reveal", delay: 1.7, dur: 0.4},
			{group: "ring", kind: "orbit", delay: 2.2, dur: 6.0},
			{group: "stop", kind: "reveal", delay: 3.2, dur: 0.6},
			{group: "stop", kind: "pulse", delay: 3.8, dur: 0.9},
		},
	},

	"06": {
		title: "The system message is assembled, not stored",
		note:  "A base prompt plus every contributor is rebuilt on each run, so a change to AGENTS.md takes effect without touching the session.",
		yaw:   -30, pitch: 18, dist: 12,
		objects: []obj{
			{form: "sheet", at: vec3{-2.8, 1.3, 0}, size: vec3{0.12, 1.5, 2.0}, hue: "brand", label: "base", group: "c0"},
			{form: "sheet", at: vec3{-2.8, -0.4, 0}, size: vec3{0.12, 1.5, 2.0}, hue: "line", label: "AGENTS.md", group: "c1"},
			{form: "sheet", at: vec3{-2.8, -2.1, 0}, size: vec3{0.12, 1.5, 2.0}, hue: "line", label: "skills", group: "c2"},
			{form: "band", at: vec3{1.2, 0, 0}, size: vec3{3.6, 3.4, 0.3}, hue: "good", label: "system message", group: "sys"},
		},
		moves: []move{
			{group: "c0", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "c1", kind: "reveal", delay: 0.5, dur: 0.4},
			{group: "c2", kind: "reveal", delay: 0.8, dur: 0.4},
			{group: "c0", kind: "travel", to: vec3{0.2, 0, 0}, delay: 1.3, dur: 0.7},
			{group: "c1", kind: "travel", to: vec3{0.5, 0, 0}, delay: 1.6, dur: 0.7},
			{group: "c2", kind: "travel", to: vec3{0.8, 0, 0}, delay: 1.9, dur: 0.7},
			{group: "sys", kind: "pulse", delay: 2.8, dur: 0.9},
		},
	},

	"07": {
		title: "An append-only log, and what is not in it",
		note:  "Each entry is appended and never rewritten. The system message is the ghost: rebuilt from the current prompt rather than stored.",
		yaw:   -38, pitch: 14, dist: 13,
		objects: []obj{
			{form: "band", at: vec3{-2.6, 0, 0}, size: vec3{0.2, 0.8, 2.2}, hue: "brand", label: "1 user", group: "e1"},
			{form: "band", at: vec3{-1.3, 0, 0}, size: vec3{0.2, 0.8, 2.2}, hue: "line", label: "2 call", group: "e2"},
			{form: "band", at: vec3{0, 0, 0}, size: vec3{0.2, 0.8, 2.2}, hue: "line", label: "3 result", group: "e3"},
			{form: "band", at: vec3{1.3, 0, 0}, size: vec3{0.2, 0.8, 2.2}, hue: "line", label: "4 answer", group: "e4"},
			{form: "band", at: vec3{3.2, 1.7, 0}, size: vec3{0.2, 0.8, 2.2}, hue: "muted", label: "system, rebuilt", group: "sys", ghost: true},
		},
		moves: []move{
			{group: "e1", kind: "reveal", delay: 0.2, dur: 0.35},
			{group: "e2", kind: "reveal", delay: 0.7, dur: 0.35},
			{group: "e3", kind: "reveal", delay: 1.2, dur: 0.35},
			{group: "e4", kind: "reveal", delay: 1.7, dur: 0.35},
			{group: "sys", kind: "reveal", delay: 2.3, dur: 0.6},
			{group: "e4", kind: "pulse", delay: 2.9, dur: 0.7},
		},
	},

	"08": {
		title: "A budget with a floor you cannot prune",
		note:  "System message and schemas are the floor. Conversation is trimmed against what is left, and a pruned pair is removed whole so no call is left orphaned.",
		yaw:   -34, pitch: 16, dist: 12,
		objects: []obj{
			{form: "stack", at: vec3{-2.0, 0, 0}, size: vec3{1.6, 1.2, 1.6}, hue: "line", label: "system", group: "sys"},
			{form: "stack", at: vec3{-2.0, 1.5, 0}, size: vec3{1.6, 0.9, 1.6}, hue: "line", label: "schemas", group: "sch"},
			{form: "stack", at: vec3{1.6, 0, 0}, size: vec3{3.2, 2.4, 1.6}, hue: "brand", label: "conversation", group: "conv"},
			{form: "sheet", at: vec3{0, 2.6, 0}, size: vec3{7.0, 0.12, 3.4}, hue: "good", label: "ceiling", group: "ceil"},
		},
		moves: []move{
			{group: "sys", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "sch", kind: "reveal", delay: 0.5, dur: 0.4},
			{group: "ceil", kind: "reveal", delay: 0.9, dur: 0.5},
			{group: "conv", kind: "reveal", delay: 1.2, dur: 0.5},
			{group: "conv", kind: "travel", to: vec3{1.6, -0.6, 0}, delay: 2.0, dur: 0.9},
			{group: "conv", kind: "pulse", delay: 2.9, dur: 0.7},
		},
	},

	"09": {
		title: "An abort keeps what arrived",
		note:  "Text that already streamed is kept and the partial message is still written. A queued message waits for the turn boundary rather than cutting in.",
		yaw:   -32, pitch: 15, dist: 12,
		objects: []obj{
			{form: "column", at: vec3{-2.4, 0, 0}, size: vec3{1.0, 2.8, 1.0}, hue: "good", label: "kept", group: "kept"},
			{form: "column", at: vec3{-0.6, 0, 0}, size: vec3{1.0, 2.8, 1.0}, hue: "bad", label: "never sent", group: "gone", ghost: true},
			{form: "wedge", at: vec3{2.2, 0, 0}, size: vec3{2.2, 1.2, 1.6}, hue: "brand", label: "queued", group: "q"},
			{form: "sheet", at: vec3{0, -2.2, 0}, size: vec3{6.4, 0.1, 2.0}, hue: "line", label: "turn boundary", group: "tb"},
		},
		moves: []move{
			{group: "kept", kind: "reveal", delay: 0.2, dur: 0.5},
			{group: "gone", kind: "reveal", delay: 0.7, dur: 0.4},
			{group: "tb", kind: "reveal", delay: 1.0, dur: 0.4},
			{group: "q", kind: "reveal", delay: 1.3, dur: 0.4},
			{group: "q", kind: "travel", to: vec3{0.6, -1.6, 0}, delay: 2.1, dur: 0.9},
			{group: "kept", kind: "pulse", delay: 3.1, dur: 0.8},
		},
	},

	"10": {
		title: "Advertised by description, loaded on request",
		note:  "Only the name and description enter the prompt. The body is the tall column behind each card, and it stays out until the model asks for it.",
		yaw:   -30, pitch: 17, dist: 12,
		objects: []obj{
			{form: "frame", at: vec3{-2.6, 1.2, 0}, size: vec3{2.0, 0.8, 0.1}, hue: "brand", label: "deploy", group: "a1"},
			{form: "frame", at: vec3{-2.6, 0.0, 0}, size: vec3{2.0, 0.8, 0.1}, hue: "brand", label: "review", group: "a2"},
			{form: "frame", at: vec3{-2.6, -1.2, 0}, size: vec3{2.0, 0.8, 0.1}, hue: "brand", label: "release", group: "a3"},
			{form: "column", at: vec3{1.4, -0.6, -0.8}, size: vec3{0.9, 2.4, 0.9}, hue: "muted", label: "deploy body, not loaded", group: "b1", ghost: true},
			{form: "column", at: vec3{2.6, -0.6, -0.8}, size: vec3{0.9, 2.4, 0.9}, hue: "muted", label: "review body, not loaded", group: "b2", ghost: true},
		},
		moves: []move{
			{group: "a1", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "a2", kind: "reveal", delay: 0.5, dur: 0.4},
			{group: "a3", kind: "reveal", delay: 0.8, dur: 0.4},
			{group: "b1", kind: "reveal", delay: 1.4, dur: 0.6},
			{group: "b2", kind: "reveal", delay: 1.8, dur: 0.6},
			{group: "a2", kind: "pulse", delay: 2.6, dur: 0.7},
			{group: "b1", kind: "pulse", delay: 3.2, dur: 0.8},
		},
	},

	"11": {
		title: "Substitution happens before the request",
		note:  "An invocation is text, not a tool call. The body is spliced in first, then the request is sent, so the model never sees the slash.",
		yaw:   -34, pitch: 16, dist: 12,
		objects: []obj{
			{form: "frame", at: vec3{-2.8, 0, 0}, size: vec3{2.2, 0.9, 0.1}, hue: "line", label: "/deploy prod", group: "typed"},
			{form: "column", at: vec3{-0.2, -0.4, 0}, size: vec3{1.2, 2.2, 1.2}, hue: "brand", label: "skill body", group: "body"},
			{form: "band", at: vec3{2.6, 0, 0}, size: vec3{2.4, 2.4, 0.2}, hue: "good", label: "request", group: "req"},
		},
		moves: []move{
			{group: "typed", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "body", kind: "reveal", delay: 0.8, dur: 0.5},
			{group: "body", kind: "travel", to: vec3{1.4, 0, 0}, delay: 1.5, dur: 0.9},
			{group: "req", kind: "reveal", delay: 2.5, dur: 0.5},
			{group: "req", kind: "pulse", delay: 3.1, dur: 0.7},
		},
	},

	"12": {
		title: "A child gets a fresh context, and discards it",
		note:  "The child starts from the system message and the task, not the parent's history. Its scratch work is spent and then thrown away, and one result returns.",
		yaw:   -40, pitch: 20, dist: 14,
		objects: []obj{
			{form: "cube", at: vec3{-2.8, 0, 0}, size: vec3{1.8, 1.8, 1.8}, hue: "brand", label: "parent", group: "parent"},
			{form: "column", at: vec3{0, 0.4, 0}, size: vec3{1.0, 2.6, 1.0}, hue: "muted", label: "child context", group: "child", ghost: true},
			{form: "orb", at: vec3{2.8, 0, 0}, size: vec3{1.0, 1.0, 1.0}, hue: "good", label: "one result", group: "result"},
		},
		moves: []move{
			{group: "parent", kind: "reveal", delay: 0.2, dur: 0.5},
			{group: "child", kind: "reveal", delay: 0.9, dur: 0.6},
			{group: "child", kind: "pulse", delay: 1.7, dur: 0.8},
			{group: "result", kind: "reveal", delay: 2.6, dur: 0.5},
			{group: "result", kind: "travel", to: vec3{-1.6, 0, 0}, delay: 3.2, dur: 0.9},
		},
	},

	"13": {
		title: "Tools in another process, namespaced",
		note:  "A server is reached over stdin and stdout, one JSON message per line. Its tool names are prefixed, because a flat namespace collides.",
		yaw:   -38, pitch: 18, dist: 13,
		objects: []obj{
			{form: "cube", at: vec3{-2.8, 0, 0}, size: vec3{1.6, 1.6, 1.6}, hue: "brand", label: "agent", group: "agent"},
			{form: "band", at: vec3{0, 0, 0}, size: vec3{2.6, 0.18, 0.18}, hue: "line", label: "stdin", group: "in"},
			{form: "band", at: vec3{0, -0.6, 0}, size: vec3{2.6, 0.18, 0.18}, hue: "line", label: "stdout", group: "out"},
			{form: "column", at: vec3{2.8, 0, 0}, size: vec3{1.2, 2.4, 1.2}, hue: "good", label: "github.*", group: "server"},
		},
		moves: []move{
			{group: "agent", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "server", kind: "reveal", delay: 0.6, dur: 0.5},
			{group: "in", kind: "reveal", delay: 1.1, dur: 0.4},
			{group: "out", kind: "reveal", delay: 1.4, dur: 0.4},
			{group: "in", kind: "pulse", delay: 2.0, dur: 0.6},
			{group: "out", kind: "pulse", delay: 2.7, dur: 0.6},
		},
	},

	"14": {
		title: "A 429 is worth another attempt, a 400 is not",
		note:  "The wait doubles to a ceiling, and Retry-After wins when the server supplies one. A client error never retries.",
		yaw:   -34, pitch: 16, dist: 13,
		objects: []obj{
			{form: "column", at: vec3{-3.0, -1.0, 0}, size: vec3{0.5, 0.6, 0.5}, hue: "line", label: "1", group: "a1"},
			{form: "column", at: vec3{-1.8, -0.8, 0}, size: vec3{0.5, 1.0, 0.5}, hue: "line", label: "2", group: "a2"},
			{form: "column", at: vec3{-0.6, -0.4, 0}, size: vec3{0.5, 1.8, 0.5}, hue: "line", label: "4", group: "a3"},
			{form: "column", at: vec3{0.6, 0, 0}, size: vec3{0.5, 2.6, 0.5}, hue: "line", label: "8", group: "a4"},
			{form: "orb", at: vec3{2.6, 0.8, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "good", label: "success", group: "ok"},
			{form: "cube", at: vec3{2.6, -1.4, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "bad", label: "400: stop", group: "bad"},
		},
		moves: []move{
			{group: "a1", kind: "reveal", delay: 0.2, dur: 0.3},
			{group: "a2", kind: "reveal", delay: 0.7, dur: 0.3},
			{group: "a3", kind: "reveal", delay: 1.4, dur: 0.3},
			{group: "a4", kind: "reveal", delay: 2.3, dur: 0.3},
			{group: "ok", kind: "reveal", delay: 3.2, dur: 0.5},
			{group: "bad", kind: "reveal", delay: 3.6, dur: 0.5},
		},
	},

	"15": {
		title: "Cut, but keep the handle",
		note:  "An oversized result is replaced by a notice plus a content-addressed file holding the whole output, so the model can follow up instead of losing it.",
		yaw:   -36, pitch: 18, dist: 13,
		objects: []obj{
			{form: "column", at: vec3{-2.8, 0, 0}, size: vec3{1.4, 3.0, 1.4}, hue: "bad", label: "50 KB", group: "big"},
			{form: "sheet", at: vec3{0.2, 0.6, 0}, size: vec3{2.2, 0.9, 0.1}, hue: "good", label: "notice", group: "notice"},
			{form: "band", at: vec3{0.2, -1.0, 0}, size: vec3{2.2, 0.5, 1.4}, hue: "brand", label: "sha256 file", group: "file"},
		},
		moves: []move{
			{group: "big", kind: "reveal", delay: 0.2, dur: 0.5},
			{group: "notice", kind: "reveal", delay: 1.1, dur: 0.5},
			{group: "file", kind: "reveal", delay: 1.5, dur: 0.5},
			{group: "notice", kind: "pulse", delay: 2.3, dur: 0.8},
		},
	},

	"16": {
		title: "Every schema is paid on every request",
		note:  "Mounted tools cost tokens whether or not they are called. Beyond a cap the harness advertises a subset and mounts the rest on demand.",
		yaw:   -32, pitch: 20, dist: 13,
		objects: []obj{
			{form: "cube", at: vec3{-2.4, 0.8, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "read", group: "s1"},
			{form: "cube", at: vec3{-1.2, 0.8, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "write", group: "s2"},
			{form: "cube", at: vec3{0, 0.8, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "bash", group: "s3"},
			{form: "cube", at: vec3{1.2, 0.8, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "brand", label: "task", group: "s4"},
			{form: "stack", at: vec3{2.6, -0.6, 0}, size: vec3{0.9, 1.6, 0.9}, hue: "muted", label: "20 more, unmounted", group: "hidden", ghost: true},
			{form: "sheet", at: vec3{-0.6, -1.8, 0}, size: vec3{6.6, 0.1, 2.2}, hue: "line", label: "cap", group: "cap"},
		},
		moves: []move{
			{group: "s1", kind: "reveal", delay: 0.2, dur: 0.3},
			{group: "s2", kind: "reveal", delay: 0.5, dur: 0.3},
			{group: "s3", kind: "reveal", delay: 0.8, dur: 0.3},
			{group: "s4", kind: "reveal", delay: 1.1, dur: 0.3},
			{group: "cap", kind: "reveal", delay: 1.5, dur: 0.4},
			{group: "hidden", kind: "reveal", delay: 2.0, dur: 0.6},
		},
	},

	"17": {
		title: "Project beats user, in the catalogue and at invocation",
		note:  "Two scopes hold skills of the same name. The project wins in both places, so what is advertised and what runs are never different skills.",
		yaw:   -30, pitch: 16, dist: 12,
		objects: []obj{
			{form: "band", at: vec3{-2.4, 1.0, 0}, size: vec3{0.2, 1.2, 2.6}, hue: "muted", label: "user: deploy", group: "u"},
			{form: "band", at: vec3{0.4, 1.0, 0}, size: vec3{0.2, 1.2, 2.6}, hue: "brand", label: "project: deploy", group: "p"},
			{form: "orb", at: vec3{2.8, 0, 0}, size: vec3{1.1, 1.1, 1.1}, hue: "good", label: "wins", group: "win"},
		},
		moves: []move{
			{group: "u", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "p", kind: "reveal", delay: 0.6, dur: 0.4},
			{group: "win", kind: "reveal", delay: 1.2, dur: 0.5},
			{group: "p", kind: "travel", to: vec3{1.6, 0, 0}, delay: 1.9, dur: 0.8},
			{group: "win", kind: "pulse", delay: 2.8, dur: 0.8},
		},
	},

	"18": {
		title: "Three always, two optional, one required even at zero",
		note:  "The body is a shape with rules: present fields, omitted fields, and one field that must appear even when its value is zero.",
		yaw:   -32, pitch: 18, dist: 12,
		objects: []obj{
			{form: "band", at: vec3{-2.4, 0.9, 0}, size: vec3{1.6, 0.4, 1.6}, hue: "brand", label: "model", group: "f1"},
			{form: "band", at: vec3{-2.4, 0.1, 0}, size: vec3{1.6, 0.4, 1.6}, hue: "brand", label: "messages", group: "f2"},
			{form: "band", at: vec3{-2.4, -0.7, 0}, size: vec3{1.6, 0.4, 1.6}, hue: "brand", label: "stream", group: "f3"},
			{form: "band", at: vec3{0.6, 0.9, 0}, size: vec3{1.6, 0.4, 1.6}, hue: "muted", label: "tools: if set", group: "f4", ghost: true},
			{form: "band", at: vec3{0.6, 0.1, 0}, size: vec3{1.6, 0.4, 1.6}, hue: "muted", label: "max_tokens: if set", group: "f5", ghost: true},
			{form: "orb", at: vec3{0.6, -1.5, 0}, size: vec3{0.8, 0.8, 0.8}, hue: "bad", label: "temperature: 0", group: "f6"},
		},
		moves: []move{
			{group: "f1", kind: "reveal", delay: 0.2, dur: 0.3},
			{group: "f2", kind: "reveal", delay: 0.5, dur: 0.3},
			{group: "f3", kind: "reveal", delay: 0.8, dur: 0.3},
			{group: "f4", kind: "reveal", delay: 1.2, dur: 0.4},
			{group: "f5", kind: "reveal", delay: 1.5, dur: 0.4},
			{group: "f6", kind: "reveal", delay: 1.9, dur: 0.4},
			{group: "f6", kind: "pulse", delay: 2.5, dur: 0.9},
		},
	},

	"19": {
		title: "Rows in a database, ordered inside a transaction",
		note:  "The next ordinal is computed in the same transaction as the insert, so two writers cannot be handed the same position.",
		yaw:   -34, pitch: 20, dist: 13,
		objects: []obj{
			{form: "stack", at: vec3{-2.6, 0, 0}, size: vec3{2.2, 2.8, 2.2}, hue: "line", label: "sessions", group: "db"},
			{form: "cube", at: vec3{0.4, 2.0, 0}, size: vec3{0.7, 0.7, 0.7}, hue: "brand", label: "1", group: "r1"},
			{form: "cube", at: vec3{0.4, 0.9, 0}, size: vec3{0.7, 0.7, 0.7}, hue: "brand", label: "2", group: "r2"},
			{form: "cube", at: vec3{0.4, -0.2, 0}, size: vec3{0.7, 0.7, 0.7}, hue: "brand", label: "3", group: "r3"},
			{form: "ring", at: vec3{0.4, -1.5, 0}, size: vec3{1.8, 1.8, 0.2}, hue: "good", label: "transaction", group: "tx"},
		},
		moves: []move{
			{group: "db", kind: "reveal", delay: 0.2, dur: 0.5},
			{group: "tx", kind: "reveal", delay: 0.7, dur: 0.5},
			{group: "r1", kind: "reveal", delay: 1.2, dur: 0.3},
			{group: "r2", kind: "reveal", delay: 1.7, dur: 0.3},
			{group: "r3", kind: "reveal", delay: 2.2, dur: 0.3},
			{group: "tx", kind: "orbit", delay: 2.8, dur: 5.0},
		},
	},

	"20": {
		title: "Usage arrives once, at the end",
		note:  "Tokens and cost are reported on the final message. A stream that never finishes reports nothing, which is why the number is stored rather than recomputed.",
		yaw:   -34, pitch: 16, dist: 12,
		objects: []obj{
			{form: "band", at: vec3{-2.8, 0, 0}, size: vec3{0.2, 0.8, 2.0}, hue: "line", label: "delta, no usage yet", group: "b1"},
			{form: "band", at: vec3{-1.6, 0, 0}, size: vec3{0.2, 0.8, 2.0}, hue: "line", label: "delta, still none", group: "b2"},
			{form: "band", at: vec3{-0.4, 0, 0}, size: vec3{0.2, 0.8, 2.0}, hue: "line", label: "last delta, carries it", group: "b3"},
			{form: "orb", at: vec3{2.0, 0, 0}, size: vec3{1.3, 1.3, 1.3}, hue: "good", label: "usage", group: "usage"},
			{form: "stack", at: vec3{3.4, -1.8, 0}, size: vec3{1.0, 1.2, 1.0}, hue: "muted", label: "stored", group: "stored", ghost: true},
		},
		moves: []move{
			{group: "b1", kind: "reveal", delay: 0.2, dur: 0.3},
			{group: "b2", kind: "reveal", delay: 0.7, dur: 0.3},
			{group: "b3", kind: "reveal", delay: 1.2, dur: 0.3},
			{group: "usage", kind: "reveal", delay: 1.9, dur: 0.6},
			{group: "usage", kind: "pulse", delay: 2.6, dur: 0.9},
			{group: "stored", kind: "reveal", delay: 3.2, dur: 0.5},
		},
	},

	"21": {
		title: "Four sources, first match wins",
		note:  "User file, project file, environment, then flags. Later wins, and the resolved value is what the program sees, with its source recorded.",
		yaw:   -36, pitch: 22, dist: 13,
		objects: []obj{
			{form: "sheet", at: vec3{0, 2.6, 0}, size: vec3{5.0, 0.12, 2.6}, hue: "muted", label: "user file", group: "s1"},
			{form: "sheet", at: vec3{0, 1.3, 0}, size: vec3{4.4, 0.12, 2.4}, hue: "line", label: "project file", group: "s2"},
			{form: "sheet", at: vec3{0, 0.0, 0}, size: vec3{3.8, 0.12, 2.2}, hue: "line", label: "environment", group: "s3"},
			{form: "sheet", at: vec3{0, -1.3, 0}, size: vec3{3.2, 0.12, 2.0}, hue: "brand", label: "flags", group: "s4"},
			{form: "orb", at: vec3{0, -2.8, 0}, size: vec3{0.9, 0.9, 0.9}, hue: "good", label: "resolved", group: "out"},
		},
		moves: []move{
			{group: "s1", kind: "reveal", delay: 0.2, dur: 0.4},
			{group: "s2", kind: "reveal", delay: 0.6, dur: 0.4},
			{group: "s3", kind: "reveal", delay: 1.0, dur: 0.4},
			{group: "s4", kind: "reveal", delay: 1.4, dur: 0.4},
			{group: "out", kind: "reveal", delay: 2.1, dur: 0.6},
			{group: "out", kind: "pulse", delay: 2.8, dur: 0.9},
		},
	},
}

# Maintainer scripts

Not part of the course. Nothing here is needed to take it.

## `sweep-scenes.js`

Measures every chapter's experimental scene in a real browser: it counts the draw calls each
figure issues, compares that against the number of objects the scene defines, and reports
the legend size. It exists because a scene can fail silently. A wrong hue, a group no move
animates, or a missing scene file produces a figure that renders as nothing, and no Go test
can see that.

Chrome is already on this machine, so the script drives it over the DevTools protocol rather
than pulling in Playwright. It needs a Chrome binary and Node 22 or later, and it needs the
site served locally:

```sh
go run ./cmd/miyagi site --out site/dist
go run ./cmd/miyagi site --v2 --out site/dist/v2
cd site/dist && python3 -m http.server 8097 &

node ../../scripts/sweep-scenes.js http://127.0.0.1:8097 ""     # the stable site
node ../../scripts/sweep-scenes.js http://127.0.0.1:8097 /v2   # the nested experimental one
```

It exits non-zero if any scene fails to draw, so it is safe to use as a gate.

Headless Chrome has no GPU. The script passes `--enable-unsafe-swiftshader` so WebGL runs on
a software rasteriser; without that flag the renderer correctly falls back to the flat figure
and every measurement reads empty.

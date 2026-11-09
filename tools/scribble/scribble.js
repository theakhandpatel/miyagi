#!/usr/bin/env node
/**
 * scribble.js — hand-drawn explainer diagrams for the Miyagi chapters.
 *
 * Target: a web page. Not GitHub, not a committed-file reader, not a video.
 *
 * That single decision removes the constraints that were shaping the code:
 *
 *   - Text is real <text>, not converted to outlines. Outlines were 83% of a file,
 *     because a committed SVG served as <img> cannot load a font. Inlined into a page,
 *     the font is inherited from the page and costs nothing per diagram. Text stays
 *     selectable, searchable, translatable and readable by a screen reader.
 *   - Colours come from CSS custom properties, so dark mode works and a diagram cannot
 *     drift from the site palette.
 *   - Playback is controlled by the page: a diagram draws when it scrolls into view,
 *     not on page load, so a reader arriving late still sees it being drawn.
 *
 * The animation itself is still declarative CSS inside the SVG, so a diagram is legible
 * with JavaScript disabled; the script only decides when to start it.
 */

const fs = require('fs');
const path = require('path');
const rough = require('roughjs');
const opentype = require('opentype.js');

const FONT_PATH = process.env.SCRIBBLE_FONT || path.join(__dirname, 'fonts', 'PatrickHand-Regular.ttf');

// Pen speed in canvas units per second. A diagram is read, not scrolled past, so this is
// deliberately unhurried: the drawing should take long enough to follow the pen.
const PEN_SPEED = 650;
const MIN_STROKE = 0.18;
const STROKE_GAP = 0.09;
const WRITE_SPEED = 26; // characters per second

// ---------------------------------------------------------------------------
// font: used for measurement and layout only; the SVG emits real text
// ---------------------------------------------------------------------------

let font = null;
function loadFont() {
  if (font) return font;
  const buf = fs.readFileSync(FONT_PATH);
  font = opentype.parse(buf.buffer.slice(buf.byteOffset, buf.byteOffset + buf.byteLength));
  return font;
}

/**
 * Measure a string and return its ink box.
 *
 * opentype emits non-finite control points for a few glyphs in these handwriting fonts.
 * Nothing is drawn from these paths any more, but a bad glyph would still corrupt the
 * bounding box, so non-finite glyphs are skipped for measurement the same way they used
 * to be skipped for drawing.
 */
function measure(str, x, y, size) {
  const f = loadFont();
  const scale = size / f.unitsPerEm;
  let cursor = x;
  let x1 = Infinity, x2 = -Infinity, y1 = Infinity, y2 = -Infinity;
  let prev = null;

  for (const ch of str) {
    const glyph = f.charToGlyph(ch);
    const cmds = glyph.getPath(cursor, y, size).commands;
    const finite = cmds.every((c) =>
      ['x', 'y', 'x1', 'y1', 'x2', 'y2'].every((k) => c[k] === undefined || Number.isFinite(c[k]))
    );
    if (finite) {
      for (const c of cmds) {
        for (const k of ['x', 'x1', 'x2']) if (Number.isFinite(c[k])) { x1 = Math.min(x1, c[k]); x2 = Math.max(x2, c[k]); }
        for (const k of ['y', 'y1', 'y2']) if (Number.isFinite(c[k])) { y1 = Math.min(y1, c[k]); y2 = Math.max(y2, c[k]); }
      }
    }
    let adv = (glyph.advanceWidth || 0) * scale;
    if (prev) {
      const k = f.getKerningValue(prev, glyph);
      if (Number.isFinite(k)) adv += k * scale;
    }
    cursor += adv;
    prev = glyph;
  }
  if (x1 === Infinity) { x1 = x; x2 = x + size * 0.4; y1 = y - size; y2 = y; }
  return { x1, y1, x2, y2, width: x2 - x1 };
}

// ---------------------------------------------------------------------------
// rough geometry
// ---------------------------------------------------------------------------

function opsToD(ops) {
  let s = '';
  for (const op of ops) {
    if (op.op === 'move') s += `M${op.data[0].toFixed(1)} ${op.data[1].toFixed(1)}`;
    else if (op.op === 'lineTo') s += `L${op.data[0].toFixed(1)} ${op.data[1].toFixed(1)}`;
    else if (op.op === 'bcurveTo') s += `C${op.data.map((v) => v.toFixed(1)).join(' ')}`;
  }
  return s;
}

/** Approximate arc length, so every stroke is drawn at one pen speed. */
function opsLength(ops) {
  let len = 0, cx = 0, cy = 0;
  for (const op of ops) {
    if (op.op === 'move') { cx = op.data[0]; cy = op.data[1]; }
    else if (op.op === 'lineTo') { len += Math.hypot(op.data[0] - cx, op.data[1] - cy); cx = op.data[0]; cy = op.data[1]; }
    else if (op.op === 'bcurveTo') {
      const [x1, y1, x2, y2, x, y] = op.data;
      let px = cx, py = cy;
      for (let i = 1; i <= 10; i++) {
        const t = i / 10, u = 1 - t;
        const bx = u * u * u * cx + 3 * u * u * t * x1 + 3 * u * t * t * x2 + t * t * t * x;
        const by = u * u * u * cy + 3 * u * u * t * y1 + 3 * u * t * t * y2 + t * t * t * y;
        len += Math.hypot(bx - px, by - py);
        px = bx; py = by;
      }
      cx = x; cy = y;
    }
  }
  return len;
}

function shape(kind, args, opts) {
  const o = Object.assign({ roughness: 1.5, bowing: 1.3, strokeWidth: 2.2 }, opts || {});
  const gen = rough.generator();
  let dr;
  if (kind === 'rect') dr = gen.rectangle(args[0], args[1], args[2], args[3], o);
  else if (kind === 'ellipse') dr = gen.ellipse(args[0], args[1], args[2], args[3], o);
  else if (kind === 'line') dr = gen.line(args[0], args[1], args[2], args[3], o);
  else if (kind === 'curve') dr = gen.curve(args[0], o);
  else throw new Error('unknown shape ' + kind);

  const out = [];
  if (o.fill) {
    const d = (dr.sets || []).filter((s) => s.type === 'fillPath').map((s) => opsToD(s.ops)).join(' ');
    if (d) out.push({ role: 'fill', d, tone: o.fill });
  }
  for (const set of dr.sets) {
    if (set.type === 'fillPath' || set.type === 'fillSketch') continue;
    out.push({ role: 'stroke', d: opsToD(set.ops), len: opsLength(set.ops), tone: o.tone || 'ink', width: o.strokeWidth || 2.2 });
  }
  return out;
}

function arrow(x1, y1, x2, y2, opts) {
  const o = Object.assign({ roughness: 1.3, bowing: 2.0, strokeWidth: 2.2 }, opts || {});
  const parts = shape('line', [x1, y1, x2, y2], o);
  const ang = Math.atan2(y2 - y1, x2 - x1);
  for (const spread of [0.44, -0.44]) {
    const a = ang + Math.PI + spread;
    parts.push(...shape('line', [x2, y2, x2 + Math.cos(a) * 12, y2 + Math.sin(a) * 12], o));
  }
  return parts;
}

// ---------------------------------------------------------------------------
// scene
// ---------------------------------------------------------------------------

class Scene {
  constructor(opts) {
    this.o = Object.assign({
      width: 720, height: 268,
      tone: 'ink',
      title: 'Hand-drawn diagram', desc: '',
      label: 'diagram'
    }, opts || {});
    this.items = [];
    this.clock = 0;
  }

  /** Queue strokes in the order a hand would draw them. */
  strokes(parts, extra) {
    let t = this.clock;
    for (const p of parts) {
      if (p.role === 'fill') {
        this.items.push(Object.assign({}, p, { start: t, dur: 0.22 }));
        continue;
      }
      const dur = Math.max(MIN_STROKE, p.len / PEN_SPEED);
      this.items.push(Object.assign({}, p, { start: t, dur }));
      t += dur + STROKE_GAP;
    }
    this.clock = t + (extra || 0);
  }

  box(x, y, w, h, label, opts) {
    const o = opts || {};
    if (o.fill) this.strokes([{ role: 'fill', d: `M${x} ${y}L${x + w} ${y}L${x + w} ${y + h}L${x} ${y + h}Z`, tone: o.fill }]);
    this.strokes(shape('rect', [x, y, w, h], {
      tone: o.tone || 'ink', strokeWidth: o.strokeWidth || 2.2,
      seed: o.seed, roughness: o.roughness || 1.5
    }), o.after != null ? o.after : 0.06);

    if (label) {
      const size = o.textSize || 20;
      const m = measure(label, 0, 0, size);
      const tx = x + (w - m.width) / 2 - m.x1;
      const ty = y + h / 2 + size * 0.34;
      const outline = Math.max(MIN_STROKE, (2 * (w + h)) / PEN_SPEED);
      this.write(label, tx, ty, { size, tone: o.tone || 'ink', join: true, maxDur: outline });
    }
    return { x, y, w, h };
  }

  write(str, x, y, opts) {
    const o = opts || {};
    const size = o.size || 22;
    const m = measure(str, x, y, size);
    const start = o.at != null ? o.at : this.clock;
    const raw = Math.max(0.3, str.length / WRITE_SPEED);
    const dur = o.join && o.maxDur != null ? Math.min(raw, o.maxDur) : raw;
    this.items.push({ role: 'text', text: str, x, y, size, tone: o.tone || 'ink', start, dur, bb: m });
    if (!o.join) this.clock = Math.max(this.clock, start + dur) + 0.08;
    return m;
  }

  arrow(x1, y1, x2, y2, opts) {
    const o = opts || {};
    this.strokes(arrow(x1, y1, x2, y2, { tone: o.tone || 'ink', strokeWidth: o.strokeWidth }), o.after != null ? o.after : 0.08);
  }

  underline(x, y, w, opts) {
    const o = opts || {};
    this.strokes(shape('curve', [[[x, y], [x + w * 0.3, y - 4], [x + w * 0.7, y + 3], [x + w, y - 2]]], {
      tone: o.tone || 'accent', strokeWidth: o.strokeWidth || 2.6, roughness: 1.1
    }), 0.12);
  }

  wait(s) { this.clock += s; }

  total() {
    let m = 0;
    for (const it of this.items) m = Math.max(m, it.start + it.dur);
    return m;
  }

  render() {
    const o = this.o;
    const total = this.total();
    const kf = [];
    const out = [];
    let i = 0;

    for (const it of this.items) {
      const start = it.start;
      const end = it.start + it.dur;

      if (it.role === 'fill') {
        const id = `f${i++}`;
        kf.push(`@keyframes ${id}{from{opacity:0}to{opacity:1}}`);
        out.push(`<path class="s-fill t-${it.tone || 'ink'}" d="${it.d}" style="--t0:${start.toFixed(2)}s;--t1:${it.dur.toFixed(2)}s;animation-name:${id}"/>`);
        continue;
      }

      if (it.role === 'stroke') {
        const id = `s${i++}`;
        // pathLength="1" makes the dash reveal independent of the stroke's real size:
        // without it a short tick finishes in a fraction of its slot and a long sweep
        // crawls, which reads as the drawing appearing rather than being made.
        kf.push(`@keyframes ${id}{from{stroke-dashoffset:1}to{stroke-dashoffset:0}}`);
        out.push(
          `<path class="s-line t-${it.tone}" d="${it.d}" pathLength="1" ` +
          `style="--t0:${start.toFixed(2)}s;--t1:${it.dur.toFixed(2)}s;animation-name:${id};stroke-width:${it.width}"/>`
        );
        continue;
      }

      // Text is written by growing a clip rectangle left to right. The rectangle is
      // widened past the ink box on both sides: `measure` reports the ink, and a clip at
      // exactly that width shaves the final glyph, which reads as a typo.
      const cid = `c${i++}`;
      const { x1, y1, x2, y2 } = it.bb;
      const padL = it.size * 0.25;
      const padR = it.size * 0.35;
      const w = Math.max(1, x2 - x1) + padL + padR;
      const h = Math.max(1, y2 - y1) + it.size * 0.6;
      kf.push(`@keyframes ${cid}{from{width:0}to{width:${w.toFixed(1)}px}}`);
      out.push(
        `<clipPath id="${cid}-clip"><rect class="s-clip" x="${(x1 - padL).toFixed(1)}" y="${(y1 - it.size * 0.4).toFixed(1)}" ` +
        `width="0" height="${h.toFixed(1)}" style="--t0:${start.toFixed(2)}s;--t1:${it.dur.toFixed(2)}s;animation-name:${cid}"/></clipPath>`
      );
      out.push(
        `<text class="s-text t-${it.tone}" x="${it.x.toFixed(1)}" y="${it.y.toFixed(1)}" font-size="${it.size}" ` +
        `clip-path="url(#${cid}-clip)">${esc(it.text)}</text>`
      );
    }

    return `<svg class="scratch" viewBox="0 0 ${o.width} ${o.height}" role="img" ` +
      `aria-label="${esc(o.title)}" data-draw="${(total + 0.4).toFixed(2)}">\n` +
      `<title>${esc(o.title)}</title>\n<desc>${esc(o.desc)}</desc>\n` +
      `<style>\n${kf.join('\n')}\n</style>\n` +
      `<g class="scene">\n${out.join('\n')}\n</g>\n</svg>\n`;
  }
}

function esc(s) {
  return String(s).replace(/[<>&'"]/g, (c) => ({ '<': '&lt;', '>': '&gt;', '&': '&amp;', "'": '&apos;', '"': '&quot;' }[c]));
}

module.exports = { Scene, shape, arrow, measure };

// ---------------------------------------------------------------------------
// scenes: one idea each, short enough to sit inline in prose
// ---------------------------------------------------------------------------

if (require.main === module) {
  const outDir = process.argv[2] || path.join(__dirname, 'drawn3');
  fs.mkdirSync(outDir, { recursive: true });

  const scenes = {
    folds: () => {
      const s = new Scene({
        width: 720, height: 268,
        title: 'One call arrives as four pieces',
        desc: 'Four streamed frames. Keyed on index the arguments reassemble; keyed on id three frames are dropped.'
      });
      s.write('one call, four pieces', 30, 40, { size: 25 });
      s.underline(30, 50, 236);
      s.wait(0.08);

      ['id + name', 'arguments', 'arguments', 'arguments'].forEach((label, i) => {
        s.box(30, 68 + i * 38, 210, 28, label, { textSize: 15, seed: 40 + i, roughness: 1.1, after: 0.03 });
      });

      s.wait(0.05);
      s.arrow(252, 140, 298, 140);
      s.box(304, 92, 386, 34, 'read({"path":"a.txt"})', {
        textSize: 16, tone: 'good', fill: 'good', seed: 5
      });
      s.write('by index: all four', 304, 156, { size: 15, tone: 'good' });

      s.wait(0.06);
      s.box(304, 176, 386, 34, 'read()', { textSize: 16, tone: 'bad', fill: 'bad', seed: 6 });
      s.write('by id: three dropped', 304, 240, { size: 15, tone: 'bad' });
      return s;
    },

    loop: () => {
      const s = new Scene({
        width: 720, height: 268,
        title: 'The reply decides when to stop',
        desc: 'A turn asks, runs tools and answers. A turn with no tool call ends the run; a model that never stops hits the step cap.'
      });
      s.write('one turn', 30, 40, { size: 25 });
      s.wait(0.1);

      const y = 76;
      s.box(30, y, 132, 48, 'ask', { textSize: 20, seed: 11 });
      s.box(210, y, 132, 48, 'run tools', { textSize: 20, seed: 12 });
      s.box(390, y, 132, 48, 'answer', { textSize: 20, seed: 13 });
      s.arrow(166, y + 24, 204, y + 24);
      s.arrow(346, y + 24, 384, y + 24);

      s.wait(0.1);
      s.arrow(456, y + 66, 96, y + 66);
      s.write('still asking?', 196, y + 92, { size: 16, tone: 'accent' });

      s.wait(0.2);
      s.write('no tool call ends it', 30, 232, { size: 17, tone: 'good' });
      s.write('the step cap ends it', 380, 232, { size: 17, tone: 'bad' });
      return s;
    },

    ceiling: () => {
      const s = new Scene({
        width: 720, height: 300,
        title: 'Pruning cannot rescue every request',
        desc: 'System and tool schemas are a floor. Once they reach the ceiling, trimming the conversation does not fit the request.'
      });
      s.write('what a request carries', 30, 40, { size: 25 });
      s.underline(30, 50, 300);
      s.wait(0.1);

      s.box(70, 210, 190, 34, 'system', { textSize: 17, fill: 'accent', seed: 21 });
      s.box(70, 156, 190, 50, 'conversation', { textSize: 17, fill: 'violet', seed: 22 });
      s.box(70, 102, 190, 50, 'tool schemas', { textSize: 17, fill: 'good', seed: 23 });

      s.wait(0.1);
      s.strokes(shape('line', [300, 96, 690, 96], { tone: 'bad', strokeWidth: 2.4, roughness: 1.0 }), 0.05);
      s.write('the window', 566, 84, { size: 15, tone: 'bad' });

      s.wait(0.12);
      s.write('pruning reaches only', 320, 262, { size: 16 });
      s.write('the middle block', 320, 286, { size: 16, tone: 'accent' });
      s.arrow(452, 250, 372, 186, { tone: 'accent' });
      return s;
    }
  };

  for (const [name, build] of Object.entries(scenes)) {
    const scene = build();
    const svg = scene.render();

    if (/NaN|Infinity|undefined/.test(svg)) {
      const bad = svg.match(/[^ ]{0,40}(NaN|Infinity|undefined)[^ ]{0,40}/)[0];
      throw new Error(`${name}: non-finite value near: ${bad}`);
    }
    const kfDefs = new Set([...svg.matchAll(/@keyframes ([\w-]+)/g)].map((m) => m[1]));
    const kfRefs = [...svg.matchAll(/animation-name:([\w-]+)/g)].map((m) => m[1]);
    const missing = kfRefs.filter((r) => !kfDefs.has(r));
    if (missing.length) throw new Error(`${name}: missing keyframes ${[...new Set(missing)].join(', ')}`);
    if (!svg.includes('pathLength="1"')) throw new Error(`${name}: strokes are not length-normalised`);

    // Text that leaves the canvas is silently cut and the caption reads wrong. Cheap
    // stand-in for looking at every rendered frame.
    const { width: W, height: H } = scene.o;
    for (const it of scene.items) {
      if (it.role !== 'text') continue;
      const { x1, x2, y1, y2 } = it.bb;
      if (x1 < 4 || x2 > W - 4 || y1 < 2 || y2 > H - 2) {
        throw new Error(
          `${name}: text runs outside the ${W}x${H} canvas ` +
          `(x ${x1.toFixed(0)}..${x2.toFixed(0)}, y ${y1.toFixed(0)}..${y2.toFixed(0)}): ${JSON.stringify(it.text)}`
        );
      }
    }

    const file = path.join(outDir, name + '.svg');
    fs.writeFileSync(file, svg);
    console.log(`${file}  ${(svg.length / 1024).toFixed(1)} KB  ${scene.total().toFixed(1)}s`);
  }
}

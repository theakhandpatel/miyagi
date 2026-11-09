#!/usr/bin/env node
/**
 * The course diagrams.
 *
 * One scene per chapter, each drawn from that chapter's own mechanism and checked against
 * it: the labels use the chapter's vocabulary and the claim in the <desc> is the claim the
 * chapter makes. A diagram that says something the prose does not is a bug, so these are
 * reviewed against the pages rather than invented here.
 *
 * Write to `chapters/NN-slug/diagrams/<name>.svg` with:
 *
 *   node tools/scribble/build.js
 *
 * The output is committed. Regenerating needs Node; taking the course does not.
 */

const fs = require('fs');
const path = require('path');
const { Scene, shape } = require('./scribble.js');

const ROOT = path.resolve(__dirname, '..', '..');

// ---------------------------------------------------------------------------
// helpers shared by scenes
// ---------------------------------------------------------------------------

/** A vertical or horizontal stack of boxes with a caption under each, as one idea. */
function stack(s, x, y, w, rows, opts) {
  const o = opts || {};
  const h = o.h || 34;
  const gap = o.gap != null ? o.gap : 34;
  let cy = y;
  for (const row of rows) {
    s.box(x, cy, w, h, row.label, {
      textSize: o.textSize || 16,
      tone: row.tone || 'ink',
      fill: row.fill,
      seed: row.seed,
      roughness: 1.1,
      after: 0.04
    });
    cy += h + gap - h + (o.step || 4);
  }
  return cy;
}

const scenes = {
  // 01 — one HTTP request, one streamed reply, reassembled by the parser.
  '01-chat-with-the-model': (s) => {
    s.write('one request, one reply', 30, 42, { size: 25 });
    s.underline(30, 52, 232);

    s.wait(0.08);
    s.box(30, 76, 210, 40, 'POST chat/completions', { textSize: 15, seed: 101 });
    s.box(30, 132, 210, 40, 'stream: true', { textSize: 15, seed: 102 });

    s.arrow(248, 96, 300, 96);
    s.write('the reply arrives in frames', 306, 92, { size: 16 });
    s.write('not in one piece', 306, 116, { size: 16, tone: 'muted' });

    s.wait(0.1);
    const frames = ['data: {"choices":[{"delta":{"content":"The "}}]}',
                    'data: {"choices":[{"delta":{"content":"quick "}}]}',
                    'data: [DONE]'];
    frames.forEach((f, i) => {
      s.box(30, 190 + i * 34, 470, 26, f, { textSize: 12, seed: 110 + i, roughness: 0.9, after: 0.02 });
    });

    s.wait(0.12);
    s.box(520, 190, 170, 60, 'joined text', { textSize: 16, tone: 'good', fill: 'good', seed: 120 });
    s.write('append, do not assign', 520, 268, { size: 15, tone: 'good' });
    return s;
  },

  // 02 — fragments folded by index; the id is absent after the first frame.
  '02-tool-calling': (s) => {
    s.write('arguments arrive in pieces', 30, 42, { size: 25 });
    s.underline(30, 52, 286);

    s.wait(0.08);
    const rows = ['1  id + name', '2  arguments', '3  arguments', '4  arguments'];
    rows.forEach((label, i) => {
      s.box(30, 76 + i * 38, 230, 28, label, { textSize: 15, seed: 201 + i, roughness: 1.1, after: 0.03 });
    });

    s.wait(0.05);
    s.arrow(268, 150, 306, 150);
    s.box(312, 100, 378, 34, 'read({"path":"a.txt"})', { textSize: 16, tone: 'good', fill: 'good', seed: 210 });
    s.write('folded by index', 312, 164, { size: 15, tone: 'good' });

    s.wait(0.06);
    s.box(312, 184, 378, 34, 'read()', { textSize: 16, tone: 'bad', fill: 'bad', seed: 211 });
    s.write('folded by id: three dropped', 312, 248, { size: 15, tone: 'bad' });
    return s;
  },

  // 03 — the round trip: call, run, result, ask again.
  '03-running-a-tool': (s) => {
    s.write('two requests, one answer', 30, 42, { size: 25 });
    s.underline(30, 52, 268);

    const y = 84;
    s.box(30, y, 140, 46, 'ask', { textSize: 17, seed: 301 });
    s.box(212, y, 140, 46, 'run read', { textSize: 17, seed: 302 });
    s.box(394, y, 140, 46, 'send result', { textSize: 17, seed: 303 });
    s.box(30, y + 78, 140, 46, 'ask again', { textSize: 17, tone: 'good', fill: 'good', seed: 304 });

    s.arrow(174, y + 23, 206, y + 23);
    s.arrow(356, y + 23, 388, y + 23);
    s.arrow(464, y + 50, 464, y + 110, { tone: 'good' });
    s.arrow(464, y + 120, 176, y + 120, { tone: 'good' });

    s.wait(0.15);
    s.write('the tool runs in the program, not in the model', 30, y + 168, { size: 16, tone: 'muted' });
    return s;
  },

  // 04 — three limits on every tool that touches the world.
  '04-tools-that-touch-disk': (s) => {
    s.write('every tool has three limits', 30, 42, { size: 25 });
    s.underline(30, 52, 292);

    s.wait(0.08);
    s.write('who wrote the arguments', 30, 92, { size: 17 });
    s.box(30, 104, 220, 34, 'resolve inside root', { textSize: 15, tone: 'good', fill: 'good', seed: 401 });
    s.write('a model cannot be trusted with a path', 268, 126, { size: 15, tone: 'muted' });

    s.wait(0.08);
    s.write('how much comes back', 30, 168, { size: 17 });
    s.box(30, 180, 220, 34, 'cap and mark', { textSize: 15, tone: 'good', fill: 'good', seed: 402 });
    s.write('an unbounded read fills the window', 268, 202, { size: 15, tone: 'muted' });

    s.wait(0.08);
    s.write('how long it may run', 30, 244, { size: 17 });
    s.box(30, 256, 220, 34, 'a deadline', { textSize: 15, tone: 'good', fill: 'good', seed: 403 });
    s.write('a hung command wedges the program', 268, 278, { size: 15, tone: 'muted' });
    return s;
  },

  // 05 — the loop, and the two ways it ends.
  '05-the-agent-loop': (s) => {
    s.write('one turn', 30, 42, { size: 25 });
    s.wait(0.08);

    const y = 78;
    s.box(30, y, 132, 48, 'ask', { textSize: 20, seed: 501 });
    s.box(210, y, 132, 48, 'run tools', { textSize: 20, seed: 502 });
    s.box(390, y, 132, 48, 'answer', { textSize: 20, seed: 503 });
    s.arrow(166, y + 24, 204, y + 24);
    s.arrow(346, y + 24, 384, y + 24);

    s.wait(0.08);
    s.arrow(456, y + 70, 96, y + 70);
    s.write('still asking?', 196, y + 96, { size: 16, tone: 'accent' });

    s.wait(0.15);
    s.write('no tool call ends it', 30, 246, { size: 17, tone: 'good' });
    s.write('the step cap ends it', 380, 246, { size: 17, tone: 'bad' });
    return s;
  },

  // 06 — the system message is assembled, and AGENTS.md is one contributor.
  '06-prompt-assembly': (s) => {
    s.write('the system message is built', 30, 42, { size: 25 });
    s.underline(30, 52, 300);

    s.wait(0.08);
    s.box(30, 82, 200, 34, 'BasePrompt', { textSize: 15, seed: 601 });
    s.box(30, 126, 200, 34, 'AGENTS.md', { textSize: 15, tone: 'accent', fill: 'accent', seed: 602 });
    s.box(30, 170, 200, 34, 'skills later', { textSize: 15, tone: 'muted', seed: 603 });

    s.arrow(238, 99, 296, 99, { tone: 'muted' });
    s.arrow(238, 143, 296, 143, { tone: 'accent' });
    s.arrow(238, 187, 296, 187, { tone: 'muted' });

    s.box(302, 82, 208, 122, 'system message', { textSize: 17, tone: 'good', fill: 'good', seed: 610 });

    s.wait(0.1);
    s.arrow(514, 143, 560, 143, { tone: 'good' });
    s.write('first in every request', 566, 132, { size: 16, tone: 'good' });
    s.write('and rebuilt, not stored', 566, 156, { size: 16, tone: 'good' });

    s.wait(0.1);
    s.write('outermost file wins', 30, 240, { size: 16, tone: 'muted' });
    s.write('a Contributor adds a block', 30, 264, { size: 16, tone: 'muted' });
    return s;
  },

  // 07 — append-only log; the system message is rebuilt on resume.
  '07-sessions': (s) => {
    s.write('a log, not a snapshot', 30, 42, { size: 25 });
    s.underline(30, 52, 268);

    s.wait(0.08);
    const lines = ['{"type":"header","version":1}',
                   '{"role":"user","content":"..."}',
                   '{"role":"assistant","tool_calls":[...]}',
                   '{"role":"tool","content":"alpha"}'];
    lines.forEach((l, i) => {
      s.box(30, 80 + i * 34, 350, 26, l, { textSize: 11, seed: 701 + i, roughness: 0.9, after: 0.02 });
    });

    s.wait(0.08);
    s.arrow(392, 120, 440, 120);
    s.box(446, 80, 244, 120, 'replay', { textSize: 18, tone: 'good', fill: 'good', seed: 710 });
    s.write('append only, never rewrite', 446, 222, { size: 15, tone: 'good' });

    s.wait(0.1);
    s.write('the system message is not in the file.', 30, 250, { size: 16 });
    s.write('it is rebuilt, so a prompt fix reaches old sessions.', 30, 274, { size: 16, tone: 'muted' });
    return s;
  },

  // 08 — a ceiling, not a threshold; pruning cannot rescue the schemas.
  '08-context-budget': (s) => {
    s.write('what a request carries', 30, 42, { size: 25 });
    s.underline(30, 52, 292);

    s.wait(0.08);
    s.box(70, 216, 190, 34, 'system', { textSize: 17, fill: 'accent', seed: 801 });
    s.box(70, 162, 190, 50, 'conversation', { textSize: 17, fill: 'violet', seed: 802 });
    s.box(70, 108, 190, 50, 'tool schemas', { textSize: 17, fill: 'good', seed: 803 });

    s.wait(0.08);
    s.strokes(shape('line', [300, 102, 690, 102], { tone: 'bad', strokeWidth: 2.4, roughness: 1.0 }), 0.05);
    s.write('the window', 566, 90, { size: 15, tone: 'bad' });

    s.wait(0.1);
    s.write('pruning reaches only', 320, 268, { size: 16 });
    s.write('the middle block', 320, 292, { size: 16, tone: 'accent' });
    s.arrow(452, 256, 372, 192, { tone: 'accent' });

    s.wait(0.08);
    s.write('fit before sending, never after', 30, 288, { size: 15, tone: 'muted' });
    return s;
  },

  // 09 — abort keeps the partial text; steering waits for the boundary.
  '09-steer-and-abort': (s) => {
    s.write('two ways to interrupt', 30, 42, { size: 25 });
    s.underline(30, 52, 284);

    s.wait(0.08);
    // the stream, with a cut in it
    s.strokes(shape('line', [30, 100, 330, 100], { tone: 'ink', strokeWidth: 2.2, roughness: 0.8 }), 0.04);
    s.strokes(shape('line', [380, 100, 560, 100], { tone: 'muted', strokeWidth: 2.2, roughness: 0.8, }), 0.02);
    s.strokes(shape('line', [330, 84, 340, 116], { tone: 'bad', strokeWidth: 2.6 }), 0.02);
    s.strokes(shape('line', [380, 84, 370, 116], { tone: 'bad', strokeWidth: 2.6 }), 0.02);

    s.write('what arrived is kept', 30, 148, { size: 17, tone: 'good' });
    s.write('exit 130, and still write the session', 30, 172, { size: 16, tone: 'muted' });

    s.wait(0.12);
    s.box(30, 200, 200, 42, 'queued message', { textSize: 16, tone: 'accent', fill: 'accent', seed: 901 });
    s.arrow(238, 221, 300, 221, { tone: 'accent' });
    s.write('enters at the turn boundary', 306, 214, { size: 16, tone: 'accent' });
    s.write('not in the middle of a turn', 306, 238, { size: 16, tone: 'muted' });
    return s;
  },

  // 10 — the catalogue: names and descriptions, bodies not loaded.
  '10-skills': (s) => {
    s.write('the model sees a catalogue', 30, 42, { size: 25 });
    s.underline(30, 52, 306);

    s.wait(0.08);
    s.box(30, 84, 210, 34, 'SKILL.md', { textSize: 15, seed: 1001 });
    s.box(30, 128, 210, 34, '--- name: deploy', { textSize: 13, seed: 1002, roughness: 0.9 });
    s.box(30, 172, 210, 34, 'description: ...', { textSize: 13, seed: 1003, roughness: 0.9 });
    s.box(30, 216, 210, 34, 'body', { textSize: 15, tone: 'muted', seed: 1004 });

    s.wait(0.06);
    s.arrow(248, 145, 300, 145, { tone: 'good' });
    s.box(306, 110, 250, 70, 'catalogue', { textSize: 18, tone: 'good', fill: 'good', seed: 1010 });
    s.write('name + description only', 306, 200, { size: 15, tone: 'good' });

    s.wait(0.08);
    s.write('the body is not in the prompt', 306, 236, { size: 16 });
    s.write('the model asks for it by name', 306, 260, { size: 16, tone: 'muted' });
    return s;
  },

  // 11 — invocation is text substitution, and stacking is left to right.
  '11-skill-invocation': (s) => {
    s.write('you invoke, the model does not', 30, 42, { size: 25 });
    s.underline(30, 52, 330);

    s.wait(0.08);
    s.box(30, 84, 400, 34, '/deploy staging', { textSize: 15, tone: 'accent', fill: 'accent', seed: 1101 });

    s.wait(0.06);
    s.arrow(224, 128, 224, 164, { tone: 'accent' });
    s.box(30, 170, 400, 70, 'the skill body, arguments filled', { textSize: 15, seed: 1102 });

    s.wait(0.1);
    s.write('two skills in one prompt stack:', 30, 268, { size: 16 });
    s.write('/a then /b, left to right', 30, 292, { size: 16, tone: 'muted' });

    s.wait(0.06);
    s.box(470, 84, 220, 34, 'unknown command', { textSize: 15, tone: 'bad', fill: 'bad', seed: 1110 });
    s.write('becomes text, not a crash', 470, 140, { size: 15, tone: 'bad' });
    return s;
  },

  // 12 — a child with an empty context, and no way to recurse.
  '12-subagent': (s) => {
    s.write('the child starts empty', 30, 42, { size: 25 });
    s.underline(30, 52, 272);

    s.wait(0.08);
    s.box(30, 84, 200, 40, 'parent', { textSize: 17, seed: 1201 });
    s.write('20 file reads', 30, 146, { size: 15, tone: 'muted' });
    s.write('all in the child', 30, 168, { size: 15, tone: 'muted' });

    s.wait(0.06);
    s.arrow(238, 104, 292, 104);
    s.write('one result comes back', 300, 96, { size: 16, tone: 'good' });

    s.box(300, 120, 200, 60, 'child', { textSize: 17, tone: 'good', fill: 'good', seed: 1210 });
    s.write('[ system, task ]', 300, 200, { size: 15, tone: 'good' });
    s.write('not the parent history', 300, 222, { size: 15, tone: 'muted' });

    s.wait(0.1);
    s.box(540, 84, 150, 34, 'read', { textSize: 15, seed: 1220 });
    s.box(540, 128, 150, 34, 'write', { textSize: 15, seed: 1221 });
    s.box(540, 172, 150, 34, 'bash', { textSize: 15, seed: 1222 });
    s.write('no task tool:', 540, 226, { size: 15, tone: 'bad' });
    s.write('recursion impossible', 540, 248, { size: 15, tone: 'bad' });
    return s;
  },

  // 13 — stdio JSON-RPC; names are namespaced by the server.
  '13-mcp': (s) => {
    s.write('tools in another process', 30, 42, { size: 25 });
    s.underline(30, 52, 280);

    s.wait(0.08);
    s.box(30, 88, 170, 46, 'your agent', { textSize: 16, seed: 1301 });
    s.box(470, 88, 170, 46, 'MCP server', { textSize: 16, seed: 1302 });

    s.wait(0.06);
    const msgs = ['{"jsonrpc":"2.0","method":"initialize"}',
                  '{"jsonrpc":"2.0","method":"tools/list"}',
                  '{"jsonrpc":"2.0","method":"tools/call"}'];
    msgs.forEach((m, i) => {
      s.box(30, 168 + i * 36, 260, 28, m, { textSize: 11, seed: 1310 + i, roughness: 0.9, after: 0.02 });
    });

    s.arrow(238, 111, 462, 111);
    s.write('stdin / stdout, one JSON per line', 268, 148, { size: 15, tone: 'muted' });

    s.wait(0.1);
    s.write('names come back namespaced:', 340, 200, { size: 16 });
    s.write('files.read, not read', 340, 224, { size: 16, tone: 'accent' });

    s.wait(0.08);
    s.write('a server that will not start is a warning', 30, 300, { size: 15, tone: 'bad' });
    return s;
  },

  // 14 — which failures are worth another attempt.
  '14-retries-and-rate-limits': (s) => {
    s.write('not every error deserves a retry', 30, 42, { size: 25 });
    s.underline(30, 52, 350);

    s.wait(0.08);
    s.box(30, 88, 150, 40, '429', { textSize: 17, tone: 'accent', fill: 'accent', seed: 1401 });
    s.write('wait and try again', 196, 112, { size: 16, tone: 'accent' });

    s.box(30, 148, 150, 40, '5xx', { textSize: 17, tone: 'accent', fill: 'accent', seed: 1402 });
    s.write('wait and try again', 196, 172, { size: 16, tone: 'accent' });

    s.box(30, 208, 150, 40, '400', { textSize: 17, tone: 'bad', fill: 'bad', seed: 1403 });
    s.write('the request is wrong: fail now', 196, 232, { size: 16, tone: 'bad' });

    s.wait(0.1);
    s.write('the wait doubles', 430, 112, { size: 16 });
    s.write('1s, 2s, 4s, capped', 430, 136, { size: 16, tone: 'muted' });
    s.write('Retry-After wins when given', 430, 186, { size: 16, tone: 'good' });
    s.write('a cancelled context stops at once', 430, 236, { size: 16, tone: 'bad' });
    return s;
  },

  // 15 — cut the body, keep a handle that can fetch the whole thing.
  '15-bounding-a-tool-result': (s) => {
    s.write('a big result, with a way back', 30, 42, { size: 25 });
    s.underline(30, 52, 320);

    s.wait(0.08);
    s.box(30, 88, 200, 120, '40 KB result', { textSize: 16, seed: 1501 });

    s.wait(0.06);
    s.arrow(238, 148, 290, 148);
    s.box(296, 96, 210, 44, 'first 20 KB', { textSize: 16, tone: 'good', fill: 'good', seed: 1510 });
    s.write('what the model sees', 296, 164, { size: 15, tone: 'good' });

    s.wait(0.06);
    s.box(296, 190, 210, 44, 'a handle', { textSize: 16, tone: 'accent', fill: 'accent', seed: 1511 });
    s.write('content addressed, so it is reused', 296, 258, { size: 15, tone: 'accent' });

    s.wait(0.08);
    s.write('the notice says: how much, where, and how to ask for the rest', 30, 300, { size: 15, tone: 'muted' });
    return s;
  },

  // 16 — a schema budget, and mounting is permanent.
  '16-the-tool-schema-budget': (s) => {
    s.write('schemas have a budget too', 30, 42, { size: 25 });
    s.underline(30, 52, 300);

    s.wait(0.08);
    s.box(30, 84, 170, 34, 'read', { textSize: 15, seed: 1601 });
    s.box(30, 126, 170, 34, 'write', { textSize: 15, seed: 1602 });
    s.box(30, 168, 170, 34, 'bash', { textSize: 15, seed: 1603 });
    s.write('always advertised', 30, 226, { size: 15, tone: 'muted' });

    s.wait(0.06);
    s.arrow(208, 145, 260, 145);
    s.box(266, 96, 190, 96, 'schemas', { textSize: 17, seed: 1610 });
    s.write('alphabetical, capped', 266, 214, { size: 15, tone: 'good' });

    s.wait(0.08);
    s.box(490, 84, 200, 34, 'mcp__files__read', { textSize: 13, tone: 'accent', fill: 'accent', seed: 1620, roughness: 0.9 });
    s.write('mounted on demand', 490, 140, { size: 15, tone: 'accent' });
    s.write('and stays mounted', 490, 164, { size: 15, tone: 'accent' });

    s.wait(0.08);
    s.strokes(shape('line', [490, 190, 690, 190], { tone: 'bad', strokeWidth: 2.2, roughness: 0.8 }), 0.03);
    s.write('a tool that disappears mid-run', 490, 216, { size: 15, tone: 'bad' });
    s.write('breaks the conversation', 490, 238, { size: 15, tone: 'bad' });
    return s;
  },

  // 17 — two scopes, project wins, and the winner must be what runs.
  '17-skill-scopes-and-precedence': (s) => {
    s.write('two scopes, one winner', 30, 42, { size: 25 });
    s.underline(30, 52, 276);

    s.wait(0.08);
    s.box(30, 84, 220, 34, '~/.config/agent', { textSize: 15, tone: 'muted', seed: 1701 });
    s.write('user scope', 30, 140, { size: 15, tone: 'muted' });
    s.box(30, 162, 220, 34, '.agent/skills', { textSize: 15, tone: 'good', fill: 'good', seed: 1702 });
    s.write('project scope', 30, 218, { size: 15, tone: 'good' });

    s.wait(0.06);
    s.arrow(258, 179, 306, 179, { tone: 'good' });
    s.box(312, 100, 180, 60, 'deploy', { textSize: 17, seed: 1710 });
    s.write('one entry, project wins', 312, 184, { size: 15, tone: 'good' });
    s.write('an absent directory adds nothing', 312, 208, { size: 15, tone: 'muted' });

    s.wait(0.1);
    s.write('the catalogue said project,', 520, 116, { size: 15 });
    s.write('so invocation must run project', 520, 140, { size: 15, tone: 'accent' });
    s.write('two code paths, one answer', 520, 186, { size: 15, tone: 'bad' });
    return s;
  },

  // 18 — three body fields, two conditional, one always present.
  '18-building-the-request-body': (s) => {
    s.write('what the body always has', 30, 42, { size: 25 });
    s.underline(30, 52, 290);

    s.wait(0.08);
    s.box(30, 84, 200, 34, 'model', { textSize: 15, tone: 'good', fill: 'good', seed: 1801 });
    s.box(30, 126, 200, 34, 'messages', { textSize: 15, tone: 'good', fill: 'good', seed: 1802 });
    s.box(30, 168, 200, 34, 'stream', { textSize: 15, tone: 'good', fill: 'good', seed: 1803 });
    s.write('always', 30, 226, { size: 15, tone: 'good' });

    s.wait(0.06);
    s.box(300, 84, 210, 34, 'tools', { textSize: 15, tone: 'accent', fill: 'accent', seed: 1810 });
    s.box(300, 126, 210, 34, 'temperature', { textSize: 15, tone: 'accent', fill: 'accent', seed: 1811 });
    s.write('only when set', 300, 184, { size: 15, tone: 'accent' });

    s.wait(0.06);
    s.box(300, 216, 210, 34, 'max_tokens', { textSize: 15, tone: 'bad', fill: 'bad', seed: 1812 });
    s.write('present even when zero', 300, 268, { size: 15, tone: 'bad' });

    s.wait(0.08);
    s.write('a zero-valued optional field', 540, 110, { size: 15 });
    s.write('is a different request', 540, 134, { size: 15, tone: 'muted' });
    return s;
  },

  // 19 — a database, and ordinals computed inside the transaction.
  '19-session-storage': (s) => {
    s.write('one conversation, many rows', 30, 42, { size: 25 });
    s.underline(30, 52, 306);

    s.wait(0.08);
    s.box(30, 84, 210, 34, 'sessions', { textSize: 15, seed: 1901 });
    s.box(30, 126, 210, 34, 'messages', { textSize: 15, seed: 1902 });
    s.write('one file per conversation', 30, 184, { size: 15, tone: 'bad' });
    s.write('stops working here', 30, 206, { size: 15, tone: 'bad' });

    s.wait(0.08);
    s.arrow(248, 145, 300, 145);
    s.box(306, 92, 210, 96, 'sqlite', { textSize: 18, tone: 'good', fill: 'good', seed: 1910 });

    s.wait(0.08);
    s.write('the next ordinal is read inside', 306, 218, { size: 15, tone: 'accent' });
    s.write('the same transaction as the insert', 306, 242, { size: 15, tone: 'accent' });
    s.write('two writers cannot pick the same number', 306, 272, { size: 15, tone: 'muted' });
    return s;
  },

  // 20 — usage arrives once; two records; money in integers.
  '20-what-a-conversation-costs': (s) => {
    s.write('usage arrives once, at the end', 30, 42, { size: 25 });
    s.underline(30, 52, 330);

    s.wait(0.08);
    s.box(30, 84, 190, 34, 'streaming', { textSize: 15, tone: 'muted', seed: 2001 });
    s.box(30, 126, 190, 34, 'final message', { textSize: 15, seed: 2002 });
    s.box(30, 168, 190, 34, 'usage', { textSize: 15, tone: 'accent', fill: 'accent', seed: 2003 });

    s.wait(0.06);
    s.arrow(228, 185, 282, 185);
    s.box(288, 96, 200, 40, 'tokens', { textSize: 16, seed: 2010 });
    s.box(288, 146, 200, 40, 'cost', { textSize: 16, seed: 2011 });
    s.write('two records, not one', 288, 208, { size: 15, tone: 'good' });

    s.wait(0.1);
    s.write('money in integer micros,', 520, 116, { size: 15 });
    s.write('never a float', 520, 140, { size: 15, tone: 'bad' });
    s.write('a stream that never ends', 520, 186, { size: 15 });
    s.write('reports nothing', 520, 210, { size: 15, tone: 'muted' });
    return s;
  },

  // 21 — four sources, and the key lives in a file.
  '21-configuring-a-provider': (s) => {
    s.write('where a setting comes from', 30, 42, { size: 25 });
    s.underline(30, 52, 318);

    s.wait(0.08);
    const rows = ['flag', 'environment', 'project config', 'user config'];
    rows.forEach((label, i) => {
      s.box(30, 84 + i * 38, 190, 28, label, { textSize: 15, seed: 2101 + i, roughness: 1.1, after: 0.03 });
    });
    s.write('first match wins', 30, 250, { size: 15, tone: 'good' });

    s.wait(0.06);
    s.arrow(228, 160, 282, 160);
    s.box(288, 108, 180, 100, 'resolved', { textSize: 17, tone: 'good', fill: 'good', seed: 2110 });

    s.wait(0.08);
    s.write('the key is a file, not a variable:', 500, 112, { size: 16 });
    s.write('api_key_file', 500, 138, { size: 16, tone: 'accent' });
    s.write('an environment variable is', 500, 180, { size: 15, tone: 'muted' });
    s.write('readable in a process listing', 500, 202, { size: 15, tone: 'muted' });

    s.wait(0.06);
    s.write('and the tool can say what it resolved', 500, 250, { size: 15, tone: 'good' });
    return s;
  }
};

// ---------------------------------------------------------------------------

// The one sentence under each drawing. This is the only text the terminal prints in place
// of the picture, so each one has to carry the claim on its own.
const CAPTIONS = {
  '01-chat-with-the-model':
    'The reply streams as frames. Each delta is appended to the text so far; assigning instead of appending keeps only the last one.',
  '02-tool-calling':
    'Arguments arrive split across frames, and only the first carries an id. Keying the accumulator on the index reassembles the call; keying it on the id drops everything after the first piece.',
  '03-running-a-tool':
    'Two requests make one answer. The model asks for a call, the program runs it, and the result goes back under the id that asked for it.',
  '04-tools-that-touch-disk':
    'Every tool that touches the world needs three limits: a path that cannot escape the project, output that cannot fill the window, and a deadline that cannot wedge the program.',
  '05-the-agent-loop':
    'A turn asks, runs the tools it was asked for, and answers. The loop continues while the model asks for tools; a turn with no call ends it, and a positive step cap ends a model that never stops.',
  '06-prompt-assembly':
    'The system message is assembled from a base prompt and every contributor, and is rebuilt on each run rather than stored, so a change to AGENTS.md reaches a session that already exists.',
  '07-sessions':
    'A session is an append-only log. The system message is deliberately not in the file: it is rebuilt from the current prompt and contributors, so a session resumed next week follows today’s instructions.',
  '08-context-budget':
    'A request carries the system message, the conversation and the tool schemas. Schemas are a floor under the cost, so pruning the conversation cannot rescue a request whose schemas already fill the window.',
  '09-steer-and-abort':
    'An abort keeps the text that already arrived and still writes the session. A queued message is not injected mid-stream: it enters at the turn boundary, so the history stays valid.',
  '10-skills':
    'A skill is advertised by its name and description only. The body stays out of the prompt until the model asks for it, which is what lets a repository carry more skills than fit in the budget.',
  '11-skill-invocation':
    'A skill invocation is text substitution, not a tool call: you type it, and the body is expanded before the request is sent. Stacked skills expand left to right, and an unknown command becomes text rather than a crash.',
  '12-subagent':
    'The child starts from a system message and the task, not the parent’s history, so it spends context that gets thrown away. It cannot recurse because the task tool is absent from its registry.',
  '13-mcp':
    'An MCP server is tools in another process, reached over stdin and stdout with one JSON message per line. Its tool names are namespaced by the server, so a server’s read cannot collide with yours.',
  '14-retries-and-rate-limits':
    'A 429 or a 5xx is worth another attempt; a 400 is not. The wait doubles up to a ceiling, Retry-After wins when the server supplies it, and a cancelled context stops the run immediately.',
  '15-bounding-a-tool-result':
    'A result too large for the budget is cut, but cut with a handle: a content-addressed file holding the whole output, plus a notice saying how much was withheld and how to ask for the rest.',
  '16-the-tool-schema-budget':
    'Schemas cost tokens on every request, so a harness that advertises everything eventually cannot afford to speak. A tool mounted on demand stays mounted, because withdrawing one mid-conversation leaves a call the next request cannot answer.',
  '17-skill-scopes-and-precedence':
    'Skills are read from a user scope and a project scope, and the project wins. Precedence in the catalogue and precedence at invocation are separate code paths, so the copy that was advertised has to be the copy that runs.',
  '18-building-the-request-body':
    'Three fields are always present, two appear only when set, and one must be present even when it is zero. Sending a zero-valued optional field is a different request, not the same one.',
  '19-session-storage':
    'A conversation is rows in a database, not a file you name. The next ordinal is computed inside the same transaction as the insert, so two writers cannot claim the same position.',
  '20-what-a-conversation-costs':
    'Usage arrives once, on the final message of the stream, so a stream that never ends reports nothing. Tokens and cost are stored as two records, and money is kept in integer micros rather than a float.',
  '21-configuring-a-provider':
    'Four sources resolve in order, and the first match wins. The key lives in a file named by api_key_file rather than an environment variable, because the environment is readable in a process listing.'
};

// Each chapter's canvas height, because the scenes differ in how much they carry.
const HEIGHT = {
  '01-chat-with-the-model': 300,
  '03-running-a-tool': 300,
  '04-tools-that-touch-disk': 320,
  '05-the-agent-loop': 288,
  '06-prompt-assembly': 300,
  '07-sessions': 300,
  '08-context-budget': 320,
  '09-steer-and-abort': 288,
  '10-skills': 300,
  '11-skill-invocation': 320,
  '12-subagent': 300,
  '13-mcp': 320,
  '14-retries-and-rate-limits': 288,
  '15-bounding-a-tool-result': 320,
  '16-the-tool-schema-budget': 288,
  '17-skill-scopes-and-precedence': 288,
  '18-building-the-request-body': 300,
  '19-session-storage': 300,
  '20-what-a-conversation-costs': 300,
  '21-configuring-a-provider': 300
};

let written = 0;
let failed = 0;

for (const [slug, build] of Object.entries(scenes)) {
  const dir = path.join(ROOT, 'chapters', slug);
  if (!fs.existsSync(dir)) {
    console.error(`${slug}: no such chapter directory`);
    failed++;
    continue;
  }

  const s = new Scene({
    width: 720,
    height: HEIGHT[slug] || 300,
    title: titleFor(slug),
    desc: descFor(slug)
  });
  build(s);
  const svg = s.render();

  try {
    validate(slug, s, svg);
  } catch (err) {
    console.error(String(err.message));
    failed++;
    continue;
  }

  const out = path.join(dir, 'diagrams');
  fs.mkdirSync(out, { recursive: true });
  const file = path.join(out, 'mechanism.svg');
  fs.writeFileSync(file, svg);
  written++;
  console.log(`  ${slug}/diagrams/mechanism.svg  ${(svg.length / 1024).toFixed(1)} KB  ${s.total().toFixed(1)}s`);
}

console.log(`\n${written} written, ${failed} failed`);
if (failed) process.exit(1);

// ---------------------------------------------------------------------------

function titleFor(slug) {
  const body = fs.readFileSync(path.join(ROOT, 'chapters', slug, 'README.md'), 'utf8');
  const m = body.match(/^# (\d+) (.+)$/m);
  return m ? `${m[1]} ${m[2]}` : slug;
}

/** The one sentence under the drawing, and the line the terminal prints in its place. */
function descFor(slug) {
  const caption = CAPTIONS[slug];
  if (!caption) throw new Error(`${slug}: no caption in CAPTIONS`);
  return caption;
}

/**
 * The checks. Each one caught a real defect in this build, which is why they are here
 * rather than in a comment.
 */
function validate(slug, scene, svg) {
  if (/NaN|Infinity|undefined/.test(svg)) {
    const bad = svg.match(/[^ ]{0,40}(NaN|Infinity|undefined)[^ ]{0,40}/)[0];
    throw new Error(`${slug}: non-finite value near: ${bad}`);
  }
  const kfDefs = new Set([...svg.matchAll(/@keyframes ([\w-]+)/g)].map((m) => m[1]));
  const kfRefs = [...svg.matchAll(/animation-name:([\w-]+)/g)].map((m) => m[1]);
  const missing = kfRefs.filter((r) => !kfDefs.has(r));
  if (missing.length) {
    throw new Error(`${slug}: animation names undefined keyframes: ${[...new Set(missing)].join(', ')}`);
  }
  if (!svg.includes('pathLength="1"')) {
    throw new Error(`${slug}: strokes are not length-normalised; the reveal will not read as drawing`);
  }
  // Text outside the canvas is silently clipped, and the label reads as a typo.
  const { width: W, height: H } = scene.o;
  for (const it of scene.items) {
    if (it.role !== 'text') continue;
    const { x1, x2, y1, y2 } = it.bb;
    if (x1 < 4 || x2 > W - 4 || y1 < 2 || y2 > H - 2) {
      throw new Error(
        `${slug}: text leaves the ${W}x${H} canvas ` +
        `(x ${x1.toFixed(0)}..${x2.toFixed(0)}, y ${y1.toFixed(0)}..${y2.toFixed(0)}): ${JSON.stringify(it.text)}`
      );
    }
  }
  // A diagram longer than this is a slide, not an illustration, and the reader scrolls
  // past before it finishes.
  if (scene.total() > 22) {
    throw new Error(`${slug}: the drawing takes ${scene.total().toFixed(1)}s, which is too long inline`);
  }
  if (scene.total() < 4) {
    throw new Error(`${slug}: the drawing takes only ${scene.total().toFixed(1)}s, which will not read as drawing`);
  }
}

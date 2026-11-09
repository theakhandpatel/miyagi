// The experimental 3D figure renderer.
//
// Written by hand against WebGL 1 rather than pulled in as a dependency, because the site's
// only dependency is VitePress and a plugin, and the course promises that reading it needs
// no network. A library here would be a second bundle, a second licence and a second thing
// to audit, for a renderer that draws boxes and bands.
//
// The renderer is intentionally dumb. It reads a scene description and plays it. It decides
// nothing about what a figure means, so a wrong figure is a wrong scene in scenes.go, not a
// rendering bug.
//
// Three fallbacks, in order of preference:
//   1. WebGL with the full scene, orbitable, animated on scroll.
//   2. No WebGL: the existing inline SVG figure stays visible and this code never runs.
//   3. Reduced motion: the scene renders in its final state with no animation.
(function () {
  'use strict';

  // VitePress renders every page on the server first, where there is no document. The
  // renderer is a browser concern only, so it exits immediately under SSR rather than
  // forcing the whole site into client-only rendering.
  if (typeof window === 'undefined' || typeof document === 'undefined') return;

  var canvas = document.querySelector('canvas.scene3d');
  if (!canvas) return;

  // --- palette ------------------------------------------------------------
  // Read from CSS custom properties so the theme stays the single source of truth and a
  // palette change recolours every scene without touching this file.
  function cssVar(name, fallback) {
    var v = getComputedStyle(document.documentElement).getPropertyValue(name).trim();
    return v || fallback;
  }

  function palette() {
    // The figures sit on the page background, so they use the site's own accents.
    return {
      brand: hex(cssVar('--vp-c-brand-1', '#9a4b28')),
      // Not --vp-c-tip-1: that variable aliases to the brand colour in the default theme,
      // so a "kept" object rendered brown and the whole figure lost its one distinction.
      good: hex(cssVar('--vp-c-green-1', '#2f6b46')),
      bad: hex(cssVar('--vp-c-danger-1', '#a8321f')),
      line: hex(cssVar('--vp-c-text-2', '#6f6259')),
      muted: hex(cssVar('--vp-c-text-3', '#9a8d82')),
      ink: hex(cssVar('--vp-c-text-1', '#221b16')),
    };
  }

  function hex(s) {
    s = (s || '').trim().replace('#', '');
    if (s.length === 3) s = s[0] + s[0] + s[1] + s[1] + s[2] + s[2];
    var n = parseInt(s, 16);
    if (isNaN(n)) return [0.5, 0.4, 0.35];
    return [((n >> 16) & 255) / 255, ((n >> 8) & 255) / 255, (n & 255) / 255];
  }

  // --- tiny matrix library ------------------------------------------------
  function mat4() { return new Float32Array([1,0,0,0, 0,1,0,0, 0,0,1,0, 0,0,0,1]); }

  function perspective(out, fovy, aspect, near, far) {
    if (!(aspect > 0)) aspect = 1;
    if (!(fovy > 0)) fovy = 0.9;
    var f = 1 / Math.tan(fovy / 2), nf = 1 / (near - far);
    out[0] = f / aspect; out[1] = 0; out[2] = 0; out[3] = 0;
    out[4] = 0; out[5] = f; out[6] = 0; out[7] = 0;
    out[8] = 0; out[9] = 0; out[10] = (far + near) * nf; out[11] = -1;
    out[12] = 0; out[13] = 0; out[14] = 2 * far * near * nf; out[15] = 0;
    return out;
  }

  function lookAt(out, eye, center, up) {
    var z0 = eye[0] - center[0], z1 = eye[1] - center[1], z2 = eye[2] - center[2];
    var len = Math.hypot(z0, z1, z2) || 1;
    z0 /= len; z1 /= len; z2 /= len;
    var x0 = up[1] * z2 - up[2] * z1, x1 = up[2] * z0 - up[0] * z2, x2 = up[0] * z1 - up[1] * z0;
    len = Math.hypot(x0, x1, x2);
    if (!(len > 1e-6)) {
      // The up vector is parallel to the view direction, which happens when pitch is +/-90.
      // Any perpendicular will do; pick one deterministically rather than producing NaN.
      x0 = 1; x1 = 0; x2 = 0;
      len = 1;
    }
    x0 /= len; x1 /= len; x2 /= len;
    var y0 = z1 * x2 - z2 * x1, y1 = z2 * x0 - z0 * x2, y2 = z0 * x1 - z1 * x0;
    out[0] = x0; out[1] = y0; out[2] = z0; out[3] = 0;
    out[4] = x1; out[5] = y1; out[6] = z1; out[7] = 0;
    out[8] = x2; out[9] = y2; out[10] = z2; out[11] = 0;
    out[12] = -(x0 * eye[0] + x1 * eye[1] + x2 * eye[2]);
    out[13] = -(y0 * eye[0] + y1 * eye[1] + y2 * eye[2]);
    out[14] = -(z0 * eye[0] + z1 * eye[1] + z2 * eye[2]);
    out[15] = 1;
    return out;
  }

  // multiply writes into a scratch array before copying to `out`, because every caller
  // passes `out` as its own first argument (multiply(m, m, x)). Writing in place while
  // still reading the same buffer corrupts the rows that have not been computed yet, which
  // projects every vertex to nonsense and leaves an empty canvas with no WebGL error.
  // Column-major multiply, matching the storage WebGL reads: out = a * b.
  // A scratch array is used because every caller passes `out` as its own first argument,
  // and writing in place while still reading the buffer corrupts the rows not yet computed.
  var _mulTmp = mat4();
  function multiply(out, a, b) {
    for (var c = 0; c < 4; c++) {
      for (var r = 0; r < 4; r++) {
        var s = 0;
        for (var k = 0; k < 4; k++) s += a[k * 4 + r] * b[c * 4 + k];
        _mulTmp[c * 4 + r] = s;
      }
    }
    for (var n = 0; n < 16; n++) out[n] = _mulTmp[n];
    return out;
  }

  function translate(out, x, y, z) {
    var m = mat4(); m[12] = x; m[13] = y; m[14] = z;
    return multiply(out, out, m);
  }

  function scale(out, x, y, z) {
    var m = mat4(); m[0] = x; m[5] = y; m[10] = z;
    return multiply(out, out, m);
  }

  // The orbit rotation below writes an already column-major matrix.

  // --- geometry -----------------------------------------------------------
  // One unit cube, reused for every form. A band is a flattened cube, a column is a tall
  // one, an orb is drawn as a low-poly cube smoothed by the shader's lighting. This keeps
  // the mesh count at one and the vertex shader trivial.
  function cubeGeometry() {
    var f = [
      // +Z
      [-.5,-.5, .5],[ .5,-.5, .5],[ .5, .5, .5],[-.5,-.5, .5],[ .5, .5, .5],[-.5, .5, .5],
      // -Z
      [ .5,-.5,-.5],[-.5,-.5,-.5],[-.5, .5,-.5],[ .5,-.5,-.5],[-.5, .5,-.5],[ .5, .5,-.5],
      // +X
      [ .5,-.5, .5],[ .5,-.5,-.5],[ .5, .5,-.5],[ .5,-.5, .5],[ .5, .5,-.5],[ .5, .5, .5],
      // -X
      [-.5,-.5,-.5],[-.5,-.5, .5],[-.5, .5, .5],[-.5,-.5,-.5],[-.5, .5, .5],[-.5, .5,-.5],
      // +Y
      [-.5, .5, .5],[ .5, .5, .5],[ .5, .5,-.5],[-.5, .5, .5],[ .5, .5,-.5],[-.5, .5,-.5],
      // -Y
      [-.5,-.5,-.5],[ .5,-.5,-.5],[ .5,-.5, .5],[-.5,-.5,-.5],[ .5,-.5, .5],[-.5,-.5, .5]
    ];
    var pos = [], nrm = [];
    for (var i = 0; i < f.length; i += 3) {
      // Face normal from the first triangle's winding.
      var a = f[i], b = f[i+1], c = f[i+2];
      var u = [b[0]-a[0], b[1]-a[1], b[2]-a[2]];
      var v = [c[0]-a[0], c[1]-a[1], c[2]-a[2]];
      var n = [u[1]*v[2]-u[2]*v[1], u[2]*v[0]-u[0]*v[2], u[0]*v[1]-u[1]*v[0]];
      var l = Math.hypot(n[0], n[1], n[2]) || 1;
      n = [n[0]/l, n[1]/l, n[2]/l];
      for (var k = 0; k < 3; k++) {
        pos.push(f[i+k][0], f[i+k][1], f[i+k][2]);
        nrm.push(n[0], n[1], n[2]);
      }
    }
    return { pos: new Float32Array(pos), nrm: new Float32Array(nrm), count: pos.length / 3 };
  }

  // --- shaders ------------------------------------------------------------
  var VS = [
    'attribute vec3 aPos;',
    'attribute vec3 aNrm;',
    'uniform mat4 uProj;',
    'uniform mat4 uView;',
    'uniform mat4 uModel;',
    'uniform mat3 uNrmMat;',
    'varying vec3 vNrm;',
    'varying vec3 vView;',
    'void main() {',
    '  vec4 world = uModel * vec4(aPos, 1.0);',
    '  vNrm = normalize(uNrmMat * aNrm);',
    '  vView = normalize(vec3(uView * world));',
    '  gl_Position = uProj * uView * world;',
    '}'
  ].join('\n');

  // Two lights and a rim term. The rim is what makes a flat-coloured box read as a solid
  // on both a light and a dark page: it outlines the silhouette against the background
  // rather than relying on the fill contrasting with it.
  var FS = [
    'precision mediump float;',
    'varying vec3 vNrm;',
    'varying vec3 vView;',
    'uniform vec3 uColor;',
    'uniform float uAlpha;',
    'uniform float uWire;',
    'void main() {',
    '  vec3 n = normalize(vNrm);',
    '  vec3 toEye = normalize(-vView);',
    '  vec3 key = normalize(vec3(0.45, 0.8, 0.55));',
    '  vec3 fill = normalize(vec3(-0.6, -0.2, 0.4));',
    '  float d = max(dot(n, key), 0.0) * 0.75 + max(dot(n, fill), 0.0) * 0.25;',
    '  float amb = 0.50;',
    '  float rim = pow(1.0 - max(dot(n, toEye), 0.0), 2.2);',
    '  // Tone the rim by how much of the surface faces away, so a face pointing at the',
    '  // camera stays its own colour and only grazing edges catch light. Without this,',
    '  // a neutral grey saturates to one flat value and reads as an unshaded rectangle.',
    '  vec3 base = uColor * (amb + d * 0.95);',
    '  base += uColor * rim * 0.28;',
    '  base = clamp(base, 0.0, 1.0);',
    '  if (uWire > 0.5) {',
    '    // A ghost is drawn as a solid outline with a faint wash, so it is unmistakably an',
    '    // object that was removed rather than a shape that failed to render.',
    '    gl_FragColor = vec4(uColor, uAlpha * 0.55);',
    '  } else {',
    '    gl_FragColor = vec4(base, uAlpha);',
    '  }',
    '}'
  ].join('\n');

  function compile(gl, type, src) {
    var s = gl.createShader(type);
    gl.shaderSource(s, src);
    gl.compileShader(s);
    if (!gl.getShaderParameter(s, gl.COMPILE_STATUS)) {
      throw new Error(gl.getShaderInfoLog(s) || 'shader compile failed');
    }
    return s;
  }

  // --- renderer -----------------------------------------------------------
  var gl = canvas.getContext('webgl', { antialias: true, alpha: true, premultipliedAlpha: false })
        || canvas.getContext('experimental-webgl', { antialias: true, alpha: true });
  if (!gl) {
    // No WebGL: leave the inline SVG figure in place, remove the empty canvas.
    canvas.remove();
    document.documentElement.setAttribute('data-scene3d', 'unsupported');
    return;
  }

  var prog = gl.createProgram();
  gl.attachShader(prog, compile(gl, gl.VERTEX_SHADER, VS));
  gl.attachShader(prog, compile(gl, gl.FRAGMENT_SHADER, FS));
  gl.linkProgram(prog);
  if (!gl.getProgramParameter(prog, gl.LINK_STATUS)) {
    canvas.remove();
    document.documentElement.setAttribute('data-scene3d', 'unsupported');
    return;
  }
  gl.useProgram(prog);

  var loc = {
    aPos: gl.getAttribLocation(prog, 'aPos'),
    aNrm: gl.getAttribLocation(prog, 'aNrm'),
    uProj: gl.getUniformLocation(prog, 'uProj'),
    uView: gl.getUniformLocation(prog, 'uView'),
    uModel: gl.getUniformLocation(prog, 'uModel'),
    uNrmMat: gl.getUniformLocation(prog, 'uNrmMat'),
    uColor: gl.getUniformLocation(prog, 'uColor'),
    uAlpha: gl.getUniformLocation(prog, 'uAlpha'),
    uWire: gl.getUniformLocation(prog, 'uWire')
  };

  var geo = cubeGeometry();
  var vboPos = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, vboPos);
  gl.bufferData(gl.ARRAY_BUFFER, geo.pos, gl.STATIC_DRAW);
  var vboNrm = gl.createBuffer();
  gl.bindBuffer(gl.ARRAY_BUFFER, vboNrm);
  gl.bufferData(gl.ARRAY_BUFFER, geo.nrm, gl.STATIC_DRAW);

  gl.enableVertexAttribArray(loc.aPos);
  gl.bindBuffer(gl.ARRAY_BUFFER, vboPos);
  gl.vertexAttribPointer(loc.aPos, 3, gl.FLOAT, false, 0, 0);
  gl.enableVertexAttribArray(loc.aNrm);
  gl.bindBuffer(gl.ARRAY_BUFFER, vboNrm);
  gl.vertexAttribPointer(loc.aNrm, 3, gl.FLOAT, false, 0, 0);

  gl.enable(gl.DEPTH_TEST);
  gl.enable(gl.CULL_FACE);
  gl.cullFace(gl.BACK);
  gl.frontFace(gl.CCW);
  gl.enable(gl.BLEND);
  gl.blendFuncSeparate(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.ONE, gl.ONE_MINUS_SRC_ALPHA);

  var reduced = window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  var state = {
    scene: null,
    yaw: 0,
    pitch: 0,
    dist: 12,
    fit: 0,
    targetYaw: 0,
    targetPitch: 0,
    drag: false,
    lastX: 0,
    lastY: 0,
    start: 0,
    playing: false,
    raf: 0
  };

  function resize() {
    var dpr = Math.min(window.devicePixelRatio || 1, 2);
    var r = canvas.getBoundingClientRect();
    var w = Math.max(1, Math.round(r.width * dpr));
    var h = Math.max(1, Math.round(r.height * dpr));
    if (canvas.width !== w || canvas.height !== h) {
      canvas.width = w;
      canvas.height = h;
    }
    gl.viewport(0, 0, w, h);
  }

  // progressFor returns 0..1 for a move at time t.
  function progressFor(m, t) {
    if (t <= m.d) return 0;
    if (m.u <= 0) return 1;
    var p = (t - m.d) / m.u;
    return p > 1 ? 1 : p;
  }

  function ease(p) {
    // Ease-out cubic. Fast arrival, settled finish, which reads as deliberate rather than
    // springy on a figure a reader is trying to study.
    return 1 - Math.pow(1 - p, 3);
  }

  function draw(t) {
    resize();
    var pal = palette();
    gl.clearColor(0, 0, 0, 0);
    gl.clear(gl.COLOR_BUFFER_BIT | gl.DEPTH_BUFFER_BIT);

    var sc = state.scene;
    if (!sc) return;

    var aspect = canvas.width / Math.max(1, canvas.height);
    var proj = perspective(mat4(), 0.9, aspect, 0.1, 100);

    // Frame the scene from its real bounding box, in scene units, then scale the whole
    // thing so the box fills the viewport. Previously the camera distance and the object
    // scale were chosen independently, which left a small scene as two specks pinned to
    // the corners of a large empty panel.
    var fit = state.fit;
    if (!fit) {
      var lo = [Infinity, Infinity, Infinity], hi = [-Infinity, -Infinity, -Infinity];
      for (var q = 0; q < sc.objects.length; q++) {
        var oo = sc.objects[q];
        for (var ax = 0; ax < 3; ax++) {
          var half = oo.s[ax] * 0.5;
          lo[ax] = Math.min(lo[ax], oo.p[ax] - half);
          hi[ax] = Math.max(hi[ax], oo.p[ax] + half);
        }
      }
      if (!isFinite(lo[0])) { lo = [-1,-1,-1]; hi = [1,1,1]; }
      var centre = [(lo[0]+hi[0])/2, (lo[1]+hi[1])/2, (lo[2]+hi[2])/2];
      var extent = Math.max(hi[0]-lo[0], hi[1]-lo[1], hi[2]-lo[2], 1);
      fit = { centre: centre, extent: extent, extentW: hi[0]-lo[0], extentH: hi[1]-lo[1] };
      state.fit = fit;
    }

    // Normalise so the longest axis of the content is a fixed number of world units, and
    // pull the camera back far enough for that unit to fill the frame at this aspect.
    var span = 4.0;
    var scaleFactor = span / fit.extent;


    // Camera orbit around the origin, driven by drag and by the scene's own yaw/pitch.
    var yaw = (state.yaw * Math.PI) / 180;
    var pitch = (state.pitch * Math.PI) / 180;
    // Distance that puts a sphere of the scene's span just inside the frustum, with
    // headroom so nothing touches the panel edge. The fov is 0.9 rad vertical, so the
    // half-angle is 0.45.
    // A scene taller than it is wide needs more distance than a wide flat one, because the
    // vertical field of view is the constraint. Scale the distance by the content's own
    // aspect so every chapter lands inside its panel with the same margin.
    var halfFov = 0.45;
    var contentAspect = (fit.extentW || fit.extent) / Math.max(fit.extentH || fit.extent, 1e-6);
    var fitBoost = contentAspect < 1 ? (1 / Math.max(contentAspect, 0.45)) : 1;
    var d = (span * 0.58 * fitBoost) / Math.tan(halfFov) * (parseFloat(canvas.dataset.zoom || "1"));
    if (state.zoom) d *= state.zoom;
    if (!isFinite(d) || d < 0.5) d = 8;
    var eye = [
      d * Math.cos(pitch) * Math.sin(yaw),
      d * Math.sin(pitch),
      d * Math.cos(pitch) * Math.cos(yaw)
    ];
    var view = lookAt(mat4(), eye, [0, 0, 0], [0, 1, 0]);

    gl.uniformMatrix4fv(loc.uProj, false, proj);
    gl.uniformMatrix4fv(loc.uView, false, view);

    // Per-group state from the moves list.
    var groups = {};
    for (var i = 0; i < sc.moves.length; i++) {
      var m = sc.moves[i];
      var p = state.playing ? progressFor(m, t) : 1;
      var g = groups[m.k] || (groups[m.k] = { alpha: 1, offset: [0,0,0], lift: 0, spin: 0 });
      if (m.m === 'reveal') {
        if (p <= 0) g.alpha = 0;
        else g.alpha = Math.min(g.alpha, ease(p));
      } else if (m.m === 'travel') {
        var e = ease(p);
        g.offset = [m.t[0] * e, m.t[1] * e, m.t[2] * e];
      } else if (m.m === 'pulse') {
        // A pulse is a brief lift toward the camera, never a flash of colour, because a
        // colour flash cannot be read by anyone who cannot distinguish the two colours.
        var s = p > 0 && p < 1 ? Math.sin(p * Math.PI) : 0;
        g.lift = s * 0.35;
      } else if (m.m === 'orbit') {
        g.spin = (state.playing ? t * 0.35 : 0);
      }
    }



    for (var k = 0; k < sc.objects.length; k++) {
      var o = sc.objects[k];
      var gs = groups[o.k];
      var alpha = gs ? gs.alpha : 1;
      if (alpha <= 0.01) continue;

      var off = gs ? gs.offset : [0, 0, 0];
      var lift = gs ? gs.lift : 0;

      var model = mat4();
      translate(model,
        ((o.p[0] + off[0]) - fit.centre[0]) * scaleFactor,
        ((o.p[1] + off[1]) - fit.centre[1]) * scaleFactor,
        ((o.p[2] + off[2]) - fit.centre[2]) * scaleFactor);
      if (gs && gs.spin) {
        // Orbiting groups turn about the scene's own Y axis, which is what a ring of turns
        // should do. Rotating about the model's centre keeps it in place.
        var a = gs.spin;
        var rot = mat4();
        rot[0] = Math.cos(a); rot[2] = -Math.sin(a);
        rot[8] = Math.sin(a); rot[10] = Math.cos(a);
        model = multiply(model, model, rot);
      }
      scale(model, o.s[0] * scaleFactor * (1 + lift),
                   o.s[1] * scaleFactor * (1 + lift),
                   o.s[2] * scaleFactor * (1 + lift));

      // Normals need the inverse-transpose; for a uniform scale the rotation part is
      // enough, and every form here is uniformly scaled.
      var nrmMat = new Float32Array([
        model[0], model[1], model[2],
        model[4], model[5], model[6],
        model[8], model[9], model[10]
      ]);

      gl.uniformMatrix4fv(loc.uModel, false, model);
      gl.uniformMatrix3fv(loc.uNrmMat, false, nrmMat);

      var c = pal[o.c] || pal.brand;
      gl.uniform3fv(loc.uColor, c);
      gl.uniform1f(loc.uAlpha, alpha * (o.g ? 0.72 : 0.94));
      gl.uniform1f(loc.uWire, o.g ? 1.0 : 0.0);

      gl.drawArrays(gl.TRIANGLES, 0, geo.count);
    }

  }

  // --- legend -------------------------------------------------------------
  // Labels are a legend beneath the scene, not floating pills over it. A pill big enough to
  // read is big enough to cover the shape it names, and offsetting cannot fix that for
  // twenty-one different layouts. A legend is deterministic, never overlaps, stays legible
  // at any viewport, and survives with WebGL disabled.
  var legendEl = document.createElement('ul');
  legendEl.className = 'scene3d-legend';
  var wrap = canvas.parentNode;
  wrap.parentNode.insertBefore(legendEl, wrap.nextSibling);

  function buildLegend(sc) {
    legendEl.textContent = '';
    for (var i = 0; i < sc.objects.length; i++) {
      var o = sc.objects[i];
      if (!o.l) continue;
      var li = document.createElement('li');
      li.className = 'scene3d-legend-item' + (o.g ? ' is-ghost' : '');
      var swatch = document.createElement('span');
      swatch.className = 'scene3d-swatch';
      swatch.setAttribute('data-hue', o.c || 'brand');
      li.appendChild(swatch);
      li.appendChild(document.createTextNode(o.l));
      legendEl.appendChild(li);
    }
    if (!legendEl.children.length) legendEl.remove();
  }

  function tick(now) {
    if (!state.start) state.start = now;
    var t = (now - state.start) / 1000;

    // Ease the orbit toward its target so a drag feels weighted rather than snapped.
    state.yaw += (state.targetYaw - state.yaw) * 0.12;
    state.pitch += (state.targetPitch - state.pitch) * 0.12;

    draw(t);
    state.raf = requestAnimationFrame(tick);
  }

  function start() {
    if (state.raf) return;
    state.raf = requestAnimationFrame(tick);
  }

  function stop() {
    if (state.raf) cancelAnimationFrame(state.raf);
    state.raf = 0;
  }

  // Only animate while the figure is on screen. A canvas that keeps drawing off-screen
  // costs battery on a page with 21 chapters behind it.
  if ('IntersectionObserver' in window) {
    var io = new IntersectionObserver(function (entries) {
      for (var i = 0; i < entries.length; i++) {
        if (entries[i].isIntersecting) start(); else stop();
      }
    }, { threshold: 0.15 });
    io.observe(canvas);
  } else {
    start();
  }

  // --- interaction --------------------------------------------------------
  canvas.addEventListener('pointerdown', function (e) {
    state.drag = true;
    state.lastX = e.clientX;
    state.lastY = e.clientY;
    canvas.setPointerCapture(e.pointerId);
    canvas.style.cursor = 'grabbing';
  });

  canvas.addEventListener('pointermove', function (e) {
    if (!state.drag) return;
    state.targetYaw += (e.clientX - state.lastX) * 0.4;
    state.targetPitch += (e.clientY - state.lastY) * 0.3;
    // Clamp pitch so the camera never rolls past the pole, where the up vector degenerates.
    state.targetPitch = Math.max(-80, Math.min(80, state.targetPitch));
    state.lastX = e.clientX;
    state.lastY = e.clientY;
  });

  canvas.addEventListener('pointerup', function (e) {
    state.drag = false;
    canvas.style.cursor = 'grab';
    try { canvas.releasePointerCapture(e.pointerId); } catch (_) {}
  });

  // Keyboard, so the figure is reachable without a pointer.
  canvas.setAttribute('tabindex', '0');
  canvas.addEventListener('keydown', function (e) {
    var step = e.shiftKey ? 20 : 8;
    if (e.key === 'ArrowLeft') state.targetYaw -= step;
    else if (e.key === 'ArrowRight') state.targetYaw += step;
    else if (e.key === 'ArrowUp') state.targetPitch = Math.max(-80, state.targetPitch - step);
    else if (e.key === 'ArrowDown') state.targetPitch = Math.min(80, state.targetPitch + step);
    else return;
    e.preventDefault();
  });

  // --- load ---------------------------------------------------------------
  var chapter = canvas.getAttribute('data-chapter');
  // Derive the site base from a page asset rather than hardcoding "/", so the same bundle
  // works whether the site is served at the root or under /v2. Any <script> the build
  // emitted starts with the base, so its path is the reliable source of truth.
  function siteBase() {
    var tag = document.querySelector('script[src*="/assets/"]');
    if (tag) {
      var src = tag.getAttribute('src') || '';
      var i = src.indexOf('/assets/');
      if (i >= 0) return src.slice(0, i + 1);
    }
    // Fall back to the directory of the page, which holds for root deployments.
    return '/';
  }

  fetch(siteBase() + 'scenes/' + chapter + '.json')
    .then(function (r) {
      if (!r.ok) throw new Error('scene ' + chapter + ' missing');
      return r.json();
    })
    .then(function (sc) {
      state.scene = sc;
      window.__miyagiScene = sc;
      state.fit = 0;
      buildLegend(sc);
      state.yaw = state.targetYaw = sc.yaw || -30;
      state.pitch = state.targetPitch = sc.pitch || 16;
      state.dist = sc.dist || 12;
      canvas.style.cursor = 'grab';
      document.documentElement.setAttribute('data-scene3d', 'on');
      // Reduced motion gets the finished frame with no animation, which is the same
      // information without the movement.
      state.playing = !reduced;
      start();
    })
    .catch(function () {
      // A missing scene should degrade to the flat figure, never to an empty box.
      canvas.remove();
      document.documentElement.setAttribute('data-scene3d', 'failed');
    });
})();

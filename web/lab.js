// The lab's page. The model lives in Go, in the screen's frame, behind
// window.kinescope there:
//
//   view()          the whole view as JSON: the setup, its Go code, the errors
//   do(command)     runs a command (JSON) and returns the view
//   live()          the values now, modulation on: {params: {key: v}, sources: {name: v}}
//   state()         the setup as a string for the address
//   load(state)     takes such a string, or the address's j=<JSON>, and returns the view
//
// This file only draws the view and turns clicks into commands; the view's
// fields are the Go types' in cmd/kinescope-lab/internal/lab/view.go.
"use strict";

const $ = (id) => document.getElementById(id);
const frame = $("frame");

let bridge = null;
let view = null;
let selected = "";
let flashed = "";
let menu = "";
let tab = "go";
let full = false;

// Talking to the model

function run(command) {
  view = JSON.parse(bridge.do(JSON.stringify(command)));
  if (view.notice) toast(view.notice);
}

// load takes the address's setup: what the link held that the lab could not
// take is told.
function load(address) {
  view = JSON.parse(bridge.load(address));
  if (view.notice) toast(view.notice);
}

function connect() {
  const found = frame.contentWindow && frame.contentWindow.kinescope;
  if (!found) {
    setTimeout(connect, 50);
    return;
  }
  bridge = found;
  if (location.hash) load(location.hash);
  else view = JSON.parse(bridge.view());
  history.reset(bridge.state());
  render();
  requestAnimationFrame(liveLoop);
}

// A link opened in the lab's own tab changes only the address: take it as
// a change. The lab's own address writes do not fire it.
window.addEventListener("hashchange", () => {
  if (!bridge || !location.hash) return;
  load(location.hash);
  history.push(bridge.state());
  render();
});

// change runs a command that changes the setup: it goes to the history and
// the address.
function change(command) {
  run(command);
  history.push(bridge.state());
  render();
}

// History: undo and redo over the setup's states

const history = {
  states: [],
  at: -1,
  reset(state) {
    this.states = [state];
    this.at = 0;
    saveAddress(state);
  },
  push(state) {
    if (state === this.states[this.at]) return;
    this.states.splice(this.at + 1);
    this.states.push(state);
    if (this.states.length > 100) this.states.shift();
    this.at = this.states.length - 1;
    saveAddress(state);
  },
  move(by) {
    const at = this.at + by;
    if (at < 0 || at >= this.states.length) return;
    this.at = at;
    view = JSON.parse(bridge.load(this.states[at]));
    saveAddress(this.states[at]);
    render();
  },
};

let addressTimer = 0;
function saveAddress(state) {
  clearTimeout(addressTimer);
  addressTimer = setTimeout(() => window.history.replaceState(null, "", "#s=" + state), 200);
}

// Drawing

function render() {
  renderHeader();
  renderSetup();
  renderGame();
  renderCode();
}

function renderHeader() {
  $("version").textContent = view.version;
  $("preset").innerHTML = '<option value="">custom</option>' +
    view.presets.map((p) => `<option${p === view.preset ? " selected" : ""}>${esc(p)}</option>`).join("");
  $("seed").value = view.seed;
  $("source").value = view.picture.source;
  $("scale").value = view.picture.scale;
  $("bypass").checked = view.picture.bypass;
  $("drop-hint").hidden = view.dropped;
  $("undo").disabled = history.at <= 0;
  $("redo").disabled = history.at >= history.states.length - 1;
}

function renderSetup() {
  const setup = $("setup");
  let html = "";
  if (view.error) html += `<div class="error">${esc(view.error)}</div>`;
  html += section("effects", "Effects", view.effects.length, effectsHTML(),
    "The look of the set, in the order light goes through it.");
  html += section("sources", "Sources", view.sources.length, view.sources.map(sourceHTML).join(""),
    "Values in time: signals the game sets, drifts and waves.");
  html += section("drives", "Drives", view.drives.length, view.drives.map(driveHTML).join(""),
    "A source moves a param: weight × the source's value is added to its base.");
  html += section("schedules", "Schedules", view.schedules.length, view.schedules.map(scheduleHTML).join(""),
    "Episodes now and then; drive rate.<name> to make them thicker.");
  setup.innerHTML = html;
}

function section(kind, title, count, body, hint) {
  return `<div class="section">
    <div class="section-head">
      <h2>${title}</h2><span class="count">${count}</span>
      <button class="ghost add" data-menu="${kind}">+ add</button>
      ${menu === kind ? menuHTML(kind) : ""}
    </div>
    ${count ? "" : `<div class="section-hint">${hint}</div>`}
    ${body}
  </div>`;
}

function menuHTML(kind) {
  if (kind === "effects") {
    const have = new Set(view.effects.map((e) => e.name));
    const taken = new Set(view.effects.filter((e) => single(e.stage)).map((e) => e.stage));
    let html = "";
    let stage = "";
    for (const e of view.catalog.effects) {
      if (e.stage !== stage) {
        stage = e.stage;
        html += `<div class="stage">${esc(stage)}${single(stage) ? " · one per TV" : ""}</div>`;
      }
      const why = have.has(e.name) ? "already on" : taken.has(e.stage) ? "another " + e.stage + " effect is on" : "";
      html += `<button data-add-effect="${esc(e.name)}"${why ? " disabled" : ""} title="${esc(e.doc)}">
        ${esc(e.name)}<small>${esc(why || firstSentence(e.doc))}</small></button>`;
    }
    return `<div class="menu">${html}</div>`;
  }
  if (kind === "sources") {
    return `<div class="menu">${view.catalog.kinds.map((k) =>
      `<button data-add-source="${k.name}">${k.name}<small>${esc(k.doc)}</small></button>`).join("")}</div>`;
  }
  return "";
}

function single(stage) {
  return view.catalog.single.includes(stage);
}

function effectsHTML() {
  let html = "";
  let stage = "";
  for (const e of view.effects) {
    if (e.stage !== stage) {
      stage = e.stage;
      html += `<div class="stage">${esc(stage)}</div>`;
    }
    const changed = e.params.some((p) => p.value !== p.default);
    const head = `<span class="grip" draggable="true" data-drag="${esc(e.name)}" title="Drag to reorder within ${esc(e.stage)}">⋮⋮</span>
      <span class="name">${esc(e.name)}</span>${changed ? '<span class="dot" title="Changed from the defaults"></span>' : ""}`;
    const body = `<div class="doc">${esc(e.doc)}</div>${e.params.map(paramHTML).join("")}`;
    html += itemHTML(e.id, head, body, { remove: { op: "removeEffect", name: e.name }, stage: e.stage, name: e.name });
  }
  return html;
}

function paramHTML(p) {
  const step = (p.max - p.min) / 200;
  const changed = p.value !== p.default;
  return `<div class="field">
    <label class="${changed ? "changed" : ""}" title="${esc(p.key + " — " + p.doc)}">${esc(p.field)}</label>
    <div class="slider"><input type="range" min="${p.min}" max="${p.max}" step="${step}" value="${p.value}" data-param="${p.key}">
      <i class="live" data-live-param="${p.key}"></i></div>
    <input type="number" min="${p.min}" max="${p.max}" step="${step}" value="${fmt(p.value, p)}" data-param-number="${p.key}">
    <button class="ghost reset${changed ? " shown" : ""}" data-reset="${p.key}" title="Back to ${fmt(p.default, p)}">↺</button>
  </div>`;
}

function sourceHTML(s) {
  const head = `<span class="name">${esc(s.name)}</span><span class="meta">${s.kind}${s.kind === "signal" ? "" : " · " + s.period + "s"}</span>
    <span class="meter"><i data-live-source="${esc(s.name)}"></i></span>`;
  const kinds = view.catalog.kinds.map((k) => `<option${k.name === s.kind ? " selected" : ""}>${k.name}</option>`).join("");
  let body = `<div class="row"><label>name</label><input type="text" value="${esc(s.name)}" data-rename-source="${esc(s.name)}"></div>
    <div class="row"><label>kind</label><select data-source-kind="${esc(s.name)}">${kinds}</select></div>`;
  if (s.kind === "signal") {
    body += `<div class="doc">The game sets it: tv.Signal(${quote(s.name)}).Set(level). Try it under the picture.</div>`;
  } else {
    body += `<div class="row"><label>period</label><input type="number" min="0.1" step="0.5" value="${s.period}" data-source-period="${esc(s.name)}"> s</div>`;
  }
  return itemHTML(s.id, head, body, { remove: { op: "removeSource", name: s.name } });
}

function driveHTML(d, i) {
  const head = `<span class="name">${esc(d.from)} → ${esc(d.to)}</span><span class="meta">× ${d.weight}</span>`;
  const from = view.sources.map((s) => `<option${s.name === d.from ? " selected" : ""}>${esc(s.name)}</option>`).join("");
  const to = view.targets.map((k) => `<option${k === d.to ? " selected" : ""}>${esc(k)}</option>`).join("");
  let body = `<div class="row"><label>from</label><select data-drive="${i}" data-field="from">${from}</select></div>
    <div class="row"><label>to</label><select data-drive="${i}" data-field="to">${to}</select></div>
    <div class="row"><label>weight</label><input type="range" min="-2" max="2" step="0.01" value="${d.weight}" data-drive="${i}" data-field="weight" style="flex:1">
      <input type="number" step="0.01" value="${d.weight}" data-drive="${i}" data-field="weight"></div>`;
  if (view.warnings[d.id]) body += `<div class="note">${esc(view.warnings[d.id])}</div>`;
  return itemHTML(d.id, head, body, { remove: { op: "removeDrive", index: i } });
}

function scheduleHTML(s) {
  const head = `<span class="name">${esc(s.name)}</span><span class="meta">every ${s.mean}±${s.spread}s · ${s.episodes.length} episodes</span>`;
  const episodes = view.catalog.episodes.map((e) =>
    `<label title="${esc(e.doc)}"><input type="checkbox" data-episode="${e.name}" data-schedule="${esc(s.name)}"${s.episodes.includes(e.name) ? " checked" : ""}>${e.name}</label>`).join("");
  let body = `<div class="row"><label>name</label><input type="text" value="${esc(s.name)}" data-rename-schedule="${esc(s.name)}"></div>
    <div class="row"><label>mean</label><input type="number" min="0.1" step="1" value="${s.mean}" data-schedule-field="mean" data-schedule="${esc(s.name)}"> s</div>
    <div class="row"><label>spread</label><input type="number" min="0" step="1" value="${s.spread}" data-schedule-field="spread" data-schedule="${esc(s.name)}"> s</div>
    <div class="row"><label>rate</label><input type="number" min="0" max="10" step="0.1" value="${s.rate}" data-schedule-field="rate" data-schedule="${esc(s.name)}">
      <span class="doc">1 keeps the pace; a drive to rate.${esc(s.name)} moves it</span></div>
    <div class="checks">${episodes}</div>`;
  if (view.warnings[s.id]) body += `<div class="note">${esc(view.warnings[s.id])}</div>`;
  return itemHTML(s.id, head, body, { remove: { op: "removeSchedule", name: s.name } });
}

function itemHTML(id, head, body, opts) {
  const open = id === selected;
  const cls = ["item", open && "selected", view.errorItems.includes(id) && "bad", view.warnings[id] && "warned"].filter(Boolean).join(" ");
  const drag = opts.stage ? ` data-stage="${esc(opts.stage)}" data-name="${esc(opts.name)}"` : "";
  return `<div class="${cls}" data-item="${esc(id)}"${drag}>
    <div class="item-head" data-select="${esc(id)}">${head}
      <span class="right"><button class="ghost" data-remove='${esc(JSON.stringify(opts.remove))}' title="Remove">×</button></span>
    </div>
    ${open ? `<div class="item-body">${body}</div>` : ""}
  </div>`;
}

function renderGame() {
  const signals = view.sources.filter((s) => s.kind === "signal");
  let html = `<h2>In the game — what it tells the TV at any moment</h2>`;
  html += signals.length
    ? signals.map((s) => `<span class="signal">${esc(s.name)}
        <input type="range" min="0" max="1" step="0.01" value="${view.levels[s.name] || 0}" data-signal="${esc(s.name)}"></span>`).join("")
    : `<span class="empty">Add a signal source to set its level here.</span>`;
  html += `<span class="buttons">${view.catalog.episodes.map((e) =>
    `<button data-play="${e.name}" title="${esc(e.doc)}">${e.name}</button>`).join("")}</span>`;
  html += `<span class="buttons">
    <button data-game="power">${view.dark ? "Power on" : "Power off"}</button>
    <button data-game="hold" class="${view.held ? "on" : ""}">Hold</button>
    <button data-game="reset">Reset</button></span>`;
  $("game").innerHTML = html;
}

function renderCode() {
  $("tab-go").classList.toggle("on", tab === "go");
  $("tab-kage").classList.toggle("on", tab === "kage");
  $("mode").style.visibility = tab === "go" ? "visible" : "hidden";
  $("mode").textContent = full ? "Setup only" : "Full example";
  const listing = $("listing");
  if (tab === "kage") {
    listing.innerHTML = `<span>${highlight(view.kage)}</span>`;
    return;
  }
  const chunks = full ? view.code.full : view.code.setup;
  listing.innerHTML = chunks.map((c) => {
    const cls = c.id && c.id === selected ? "hl" : c.id && c.id === flashed ? "flash" : "";
    return `<span${c.id ? ` data-id="${esc(c.id)}"` : ""}${cls ? ` class="${cls}"` : ""}>${highlight(c.text)}</span>`;
  }).join("");
  const mark = listing.querySelector(".hl, .flash");
  if (mark) {
    const top = mark.offsetTop - listing.offsetTop;
    if (top < listing.scrollTop || top > listing.scrollTop + listing.clientHeight - 40) {
      listing.scrollTop = top - 40;
    }
  }
}

// Live values: the modulation on top of the bases

function liveLoop() {
  const live = JSON.parse(bridge.live());
  for (const el of document.querySelectorAll("[data-live-param]")) {
    const p = paramByKey(el.dataset.liveParam);
    const v = live.params[el.dataset.liveParam];
    if (p && v !== undefined) el.style.width = (100 * (v - p.min) / (p.max - p.min || 1)) + "%";
  }
  for (const el of document.querySelectorAll("[data-live-source]")) {
    el.style.width = (100 * (live.sources[el.dataset.liveSource] || 0)) + "%";
  }
  requestAnimationFrame(liveLoop);
}

function paramByKey(key) {
  for (const e of view.effects) {
    for (const p of e.params) if (p.key === key) return p;
  }
  return null;
}

// Events

document.addEventListener("click", (ev) => {
  const t = ev.target.closest("button, [data-select], [data-id]");
  if (!t || !bridge) {
    if (menu) { menu = ""; renderSetup(); }
    return;
  }
  const d = t.dataset;
  if (d.menu) {
    menu = menu === d.menu ? "" : d.menu;
    if (d.menu === "drives") { menu = ""; addDrive(); return; }
    if (d.menu === "schedules") { menu = ""; addSchedule(); return; }
    renderSetup();
    return;
  }
  menu = "";
  if (d.addEffect) {
    change({ op: "addEffect", name: d.addEffect });
    select("effect:" + d.addEffect);
  } else if (d.addSource) {
    change({ op: "addSource", kind: d.addSource });
    select(view.sources[view.sources.length - 1].id);
  } else if (d.remove) {
    change(JSON.parse(d.remove));
  } else if (d.reset) {
    change({ op: "resetParam", key: d.reset });
  } else if (d.play) {
    run({ op: "play", episode: d.play });
    flash("game:play:" + d.play);
  } else if (d.game) {
    run({ op: d.game });
    renderGame();
    flash("game:" + d.game);
  } else if (d.select) {
    select(selected === d.select ? "" : d.select);
  } else if (d.id) {
    if (!d.id.startsWith("game:")) select(d.id);
  }
});

function select(id) {
  selected = id;
  renderSetup();
  renderCode();
  const el = document.querySelector(`[data-item="${CSS.escape(id)}"]`);
  if (el) el.scrollIntoView({ block: "nearest" });
}

function flash(id) {
  flashed = id;
  renderCode();
}

function addDrive() {
  if (!view.sources.length) {
    toast("Add a source first: a drive goes from a source to a param.");
    return;
  }
  change({ op: "addDrive" });
  select(view.drives[view.drives.length - 1].id);
}

function addSchedule() {
  change({ op: "addSchedule" });
  select(view.schedules[view.schedules.length - 1].id);
}

// input moves values while a slider is dragged; change commits them
document.addEventListener("input", (ev) => {
  const t = ev.target;
  const d = t.dataset;
  if (d.param || d.paramNumber) {
    const key = d.param || d.paramNumber;
    const p = paramByKey(key);
    run({ op: "setParam", key, value: fmt(Number(t.value), p) });
    const other = document.querySelector(d.param ? `[data-param-number="${key}"]` : `[data-param="${key}"]`);
    if (other && p) other.value = d.param ? fmt(p.value, p) : p.value;
    renderCode();
  } else if (d.drive && d.field === "weight") {
    run({ op: "setDrive", index: Number(d.drive), weight: Number(t.value) });
    for (const el of document.querySelectorAll(`[data-drive="${d.drive}"][data-field="weight"]`)) {
      if (el !== t) el.value = t.value;
    }
    renderCode();
  } else if (d.signal) {
    run({ op: "signal", name: d.signal, level: Number(t.value) });
    flash("game:signal:" + d.signal);
  }
});

document.addEventListener("change", (ev) => {
  const t = ev.target;
  const d = t.dataset;
  if (!bridge) return;
  if (d.param || d.paramNumber) {
    const key = d.param || d.paramNumber;
    change({ op: "setParam", key, value: fmt(Number(t.value), paramByKey(key)) });
  } else if (d.drive) {
    const value = d.field === "weight" ? Number(t.value) : t.value;
    change({ op: "setDrive", index: Number(d.drive), [d.field]: value });
  } else if (d.renameSource) {
    change({ op: "renameSource", name: d.renameSource, to: t.value });
    if (!view.error) select("source:" + t.value);
  } else if (d.sourceKind) {
    change({ op: "setSource", name: d.sourceKind, kind: t.value });
  } else if (d.sourcePeriod) {
    change({ op: "setSource", name: d.sourcePeriod, period: Number(t.value) });
  } else if (d.renameSchedule) {
    change({ op: "renameSchedule", name: d.renameSchedule, to: t.value });
    if (!view.error) select("schedule:" + t.value);
  } else if (d.scheduleField) {
    change({ op: "setSchedule", name: d.schedule, [d.scheduleField]: Number(t.value) });
  } else if (d.episode) {
    const s = view.schedules.find((x) => x.name === d.schedule);
    const episodes = s.episodes.filter((e) => e !== d.episode);
    if (t.checked) episodes.push(d.episode);
    change({ op: "setSchedule", name: d.schedule, episodes });
  } else if (t.id === "preset") {
    if (t.value) { selected = ""; change({ op: "preset", name: t.value }); }
  } else if (t.id === "seed") {
    change({ op: "seed", seed: Number(t.value) });
  } else if (t.id === "source" || t.id === "scale" || t.id === "bypass") {
    const picture = { source: $("source").value, scale: Number($("scale").value), bypass: $("bypass").checked };
    change({ op: "picture", picture });
  }
});

// Reordering effects within their stage

let dragged = null;
document.addEventListener("dragstart", (ev) => {
  const grip = ev.target.closest("[data-drag]");
  if (!grip) return;
  dragged = grip.closest(".item");
  dragged.classList.add("dragging");
  ev.dataTransfer.effectAllowed = "move";
});
document.addEventListener("dragover", (ev) => {
  const over = ev.target.closest && ev.target.closest(".item[data-stage]");
  for (const el of document.querySelectorAll(".drop-before")) el.classList.remove("drop-before");
  if (!dragged || !over || over === dragged || over.dataset.stage !== dragged.dataset.stage) return;
  ev.preventDefault();
  over.classList.add("drop-before");
});
document.addEventListener("drop", (ev) => {
  const over = ev.target.closest && ev.target.closest(".item[data-stage]");
  if (!dragged || !over || over.dataset.stage !== dragged.dataset.stage) return;
  ev.preventDefault();
  change({ op: "moveEffect", name: dragged.dataset.name, before: over.dataset.name });
});
document.addEventListener("dragend", () => {
  if (dragged) dragged.classList.remove("dragging");
  dragged = null;
});

// Header and code panel

$("undo").onclick = () => history.move(-1);
$("redo").onclick = () => history.move(1);
document.addEventListener("keydown", (ev) => {
  if (!(ev.ctrlKey || ev.metaKey) || ev.key.toLowerCase() !== "z") return;
  if (ev.target.matches("input[type=text], input[type=number]")) return;
  ev.preventDefault();
  history.move(ev.shiftKey ? 1 : -1);
});
$("copy-link").onclick = () => copy(location.origin + location.pathname + "#s=" + bridge.state(), "Link copied");
// go get takes a release tag or a commit, not git describe's mix of both
$("go-get").onclick = () => {
  const commit = view.version.match(/-g([0-9a-f]+)$/);
  const version = commit ? commit[1] : view.version === "dev" ? "latest" : view.version;
  copy("go get github.com/shpaker/kinescope@" + version, "go get line copied");
};
$("tab-go").onclick = () => { tab = "go"; renderCode(); };
$("tab-kage").onclick = () => { tab = "kage"; renderCode(); };
$("mode").onclick = () => { full = !full; renderCode(); };
$("copy-code").onclick = () => {
  const text = tab === "kage" ? view.kage : (full ? view.code.full : view.code.setup).map((c) => c.text).join("\n") + "\n";
  copy(text, tab === "kage" ? "Shader copied" : "Go copied");
};

function copy(text, done) {
  navigator.clipboard.writeText(text).then(() => toast(done), () => toast("Your browser didn't let the page copy"));
}

let toastTimer = 0;
function toast(text) {
  const t = $("toast");
  t.textContent = text;
  t.classList.add("shown");
  clearTimeout(toastTimer);
  toastTimer = setTimeout(() => t.classList.remove("shown"), 1600 + 40 * text.length);
}

// Helpers

function esc(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]);
}

function quote(s) {
  return JSON.stringify(s);
}

function fmt(v, p) {
  const digits = Math.max(0, 3 - Math.floor(Math.log10(p.max - p.min || 1)));
  return Number(v.toFixed(digits));
}

function firstSentence(s) {
  const i = s.indexOf(". ");
  return i < 0 ? s : s.slice(0, i + 1);
}

const goToken = /(\/\/.*$)|("(?:[^"\\]|\\.)*"|`[^`]*`)|\b(func|return|package|import|var|const|type|struct|map|if|err|nil|for|range|go|defer)\b|\b(\d+(?:\.\d+)?)\b|\b(kinescope|ebitengine|ebiten)\.([A-Z]\w*)/gm;

function highlight(text) {
  let out = "";
  let last = 0;
  for (const m of text.matchAll(goToken)) {
    out += esc(text.slice(last, m.index));
    if (m[1]) out += `<i class="c">${esc(m[1])}</i>`;
    else if (m[2]) out += `<i class="s">${esc(m[2])}</i>`;
    else if (m[3]) out += `<i class="k">${m[3]}</i>`;
    else if (m[4]) out += `<i class="n">${m[4]}</i>`;
    else out += `${m[5]}.<i class="t">${m[6]}</i>`;
    last = m.index + m[0].length;
  }
  return out + esc(text.slice(last));
}

connect();

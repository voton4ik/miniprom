package api

const indexHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>miniprom</title>
<style>
  :root {
    color-scheme: dark;
    --bg: #000000; --panel: #111318; --line: #23262e;
    --text: #e4e8f0; --dim: #8b95a7; --up: #3fb950; --down: #b65f6c;
  }
  * { box-sizing: border-box; }
  body {
    margin: 0; background: var(--bg); color: var(--text);
    font: 15px/1.55 system-ui, -apple-system, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  }
  header {
    border-bottom: 1px solid var(--line); padding: 16px 24px;
    display: flex; align-items: center; justify-content: space-between;
  }
  .brand { font-size: 18px; font-weight: 700; }
  .status { color: var(--dim); font-size: 13px; }
  .status b { color: var(--text); font-weight: 600; }
  main { max-width: 940px; margin: 0 auto; padding: 8px 24px 48px; }
  .label { color: var(--dim); font-size: 13px; margin: 28px 0 10px; }
  .targets { border: 1px solid var(--line); overflow: hidden; }
  .row { display: flex; align-items: center; gap: 14px; padding: 13px 16px; border-top: 1px solid var(--line); }
  .row:first-child { border-top: none; }
  .dot { width: 9px; height: 9px; border-radius: 50%; flex: none; background: var(--dim); }
  .up { background: var(--up); }
  .down { background: var(--down); }
  .info { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
  .info .job { font-weight: 600; }
  .info .url { color: var(--dim); font-size: 13px; word-break: break-all; }
  .meta { margin-left: auto; text-align: right; color: var(--dim); font-size: 13px; flex: none; }
  .controls { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
  select, input, button { font: inherit; font-size: 14px; background: var(--panel); color: var(--text); border: 1px solid var(--line); padding: 9px 12px; }
  input { min-width: 240px; }
  input::placeholder { color: #56607280; }
  select:focus, input:focus { outline: none; border-color: #586074; }
  button { cursor: pointer; background: transparent; color: var(--text); border-color: var(--text); font-weight: 600; }
  button:hover { background: var(--text); color: #000; }
  table { width: 100%; border-collapse: collapse; border: 1px solid var(--line); overflow: hidden; }
  th, td { text-align: left; padding: 10px 14px; font-size: 13px; border-top: 1px solid var(--line); }
  thead th { border-top: none; background: var(--panel); color: var(--dim); font-weight: 500; }
  td.val { text-align: right; font-variant-numeric: tabular-nums; font-weight: 600; }
  .labels { color: var(--dim); }
  .empty { color: var(--dim); padding: 14px; border: 1px dashed var(--line); }
</style>
</head>
<body>
<header>
  <div class="brand">miniprom</div>
  <div id="status" class="status">—</div>
</header>
<main>
  <div class="label">Цели</div>
  <div id="targets" class="targets"></div>

  <div class="label">Запрос</div>
  <div class="controls">
    <select id="metric"></select>
    <input id="labels" placeholder="job=demo, code=200">
    <button id="run">Показать</button>
  </div>
  <div id="result"></div>
</main>
<script>
const fmt = n => Number.isInteger(n) ? n : n.toFixed(4);
const ago = ts => {
  const s = Math.round((Date.now() - new Date(ts)) / 1000);
  if (s < 60) return s + " с назад";
  return Math.round(s / 60) + " мин назад";
};

async function loadTargets() {
  const data = await (await fetch("/api/targets")).json();
  const box = document.getElementById("targets");
  box.innerHTML = "";
  let up = 0;
  for (const t of data) {
    if (t.up) up++;
    const el = document.createElement("div");
    el.className = "row";
    const meta = t.up
      ? (t.samples + " метрик · " + Math.round(t.duration_ms / 1e6) + " мс · " + ago(t.last_scrape))
      : (t.last_error || "нет данных");
    el.innerHTML =
      '<span class="dot ' + (t.up ? "up" : "down") + '"></span>' +
      '<div class="info"><span class="job">' + t.job + '</span>' +
      '<span class="url">' + t.url + '</span></div>' +
      '<span class="meta">' + meta + '</span>';
    box.appendChild(el);
  }
  document.getElementById("status").innerHTML =
    '<b>' + up + '</b>/' + data.length + ' online';
}

async function loadMetricList() {
  const names = await (await fetch("/api/metrics")).json();
  const sel = document.getElementById("metric");
  const current = sel.value;
  sel.innerHTML = "";
  for (const n of names || []) {
    const opt = document.createElement("option");
    opt.value = n; opt.textContent = n;
    sel.appendChild(opt);
  }
  if (current) sel.value = current;
}

async function runQuery() {
  const metric = document.getElementById("metric").value;
  const labels = document.getElementById("labels").value.trim();
  if (!metric) return;
  const url = "/api/query?metric=" + encodeURIComponent(metric) +
    (labels ? "&labels=" + encodeURIComponent(labels) : "");
  const data = await (await fetch(url)).json();
  const box = document.getElementById("result");
  if (!data || !data.length) {
    box.innerHTML = '<div class="empty">Нет данных по этой метрике.</div>';
    return;
  }
  let html = '<table><thead><tr><th>Метка</th><th>Значение</th></tr></thead><tbody>';
  for (const s of data) {
    const labelStr = Object.entries(s.labels || {})
      .map(([k, v]) => k + '="' + v + '"').join(", ");
    const v = s.samples[s.samples.length - 1].v;
    html += '<tr><td><span class="labels">{' + labelStr + '}</span></td>' +
      '<td class="val">' + fmt(v) + '</td></tr>';
  }
  html += '</tbody></table>';
  box.innerHTML = html;
}

document.getElementById("run").addEventListener("click", runQuery);
loadTargets();
loadMetricList();
setInterval(loadTargets, 5000);
setInterval(loadMetricList, 10000);
</script>
</body>
</html>`

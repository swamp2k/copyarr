package api

// uiHTML is the whole dashboard: one self-contained page with no build step and
// no external requests, so the binary stays the only thing that has to ship.
//
// This is a raw string literal, so it must never contain a backtick. That means
// string concatenation in the JavaScript below instead of template literals.
const uiHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Copyarr</title>
<style>
/* Palette: dark by default, with a light variant for prefers-color-scheme. */
:root{
  --bg:#0f1117;--panel:#1a1d27;--panel-2:#161923;--inset:#0f1117;
  --line:#334155;--line-strong:#64748b;--field-line:#475569;
  --text:#ffffff;--muted:#94a3b8;--dim:#64748b;--faint:#4b5769;
  --accent:#7c3aed;--accent-hover:#8b5cf6;--accent-text:#c4b5fd;
  --good:#4ade80;--warn:#facc15;--bad:#f87171;--info:#22d3ee;
  --shadow:0 24px 60px rgba(0,0,0,.5);
  color-scheme:dark;
}
@media(prefers-color-scheme:light){
  :root{
    --bg:#f1f5f9;--panel:#ffffff;--panel-2:#f8fafc;--inset:#f8fafc;
    --line:#e2e8f0;--line-strong:#94a3b8;--field-line:#cbd5e1;
    --text:#0f172a;--muted:#475569;--dim:#64748b;--faint:#94a3b8;
    --accent:#7c3aed;--accent-hover:#6d28d9;--accent-text:#6d28d9;
    --good:#16a34a;--warn:#ca8a04;--bad:#dc2626;--info:#0891b2;
    --shadow:0 20px 50px rgba(15,23,42,.14);
    color-scheme:light;
  }
}

*,*::before,*::after{box-sizing:border-box}
[hidden]{display:none!important}
body{margin:0;background:var(--bg);color:var(--text);font:14px/1.5 system-ui,"Segoe UI",Inter,sans-serif;-webkit-font-smoothing:antialiased}
.shell{max-width:1180px;margin:0 auto;padding:22px 20px 72px}
h1,h2,h3{margin:0;font-weight:600}
h1{font-size:20px}
h2{font-size:15px}
p{margin:0}
.dim{color:var(--dim);font-size:12px}
.num{font-variant-numeric:tabular-nums}

/* Header ------------------------------------------------------------------ */
.topbar{display:flex;flex-wrap:wrap;gap:14px;align-items:center;justify-content:space-between;margin-bottom:18px}
.brand{display:flex;align-items:center;gap:12px}
.logo{width:38px;height:38px;border-radius:11px;flex:none;display:grid;place-items:center;color:#fff;font-weight:700;font-size:17px;background:linear-gradient(140deg,var(--accent),#4338ca)}
.topbar-actions{display:flex;align-items:center;gap:10px}
.live{display:inline-flex;align-items:center;gap:7px;font-size:12px;color:var(--muted);padding:6px 11px;border:1px solid var(--line);border-radius:999px;background:var(--panel)}
.dot{width:8px;height:8px;border-radius:50%;background:var(--good);flex:none}
.dot.scan{background:var(--warn)}
.dot.off{background:var(--bad)}

/* Tabs -------------------------------------------------------------------- */
.tabs{display:flex;gap:4px;padding:4px;background:var(--panel);border:1px solid var(--line);border-radius:11px;width:max-content;max-width:100%;overflow:auto;margin-bottom:18px}
.tab{border:0;background:transparent;color:var(--muted);border-radius:8px;padding:7px 14px;font:inherit;font-size:13px;white-space:nowrap;cursor:pointer;transition:color .15s,background .15s}
.tab:hover{color:var(--text)}
.tab.active{background:color-mix(in srgb,var(--accent) 18%,transparent);color:var(--accent-text)}

/* Buttons ----------------------------------------------------------------- */
.btn{display:inline-flex;align-items:center;gap:6px;border:1px solid var(--line);background:transparent;color:var(--muted);border-radius:8px;padding:7px 12px;font:inherit;font-size:12px;cursor:pointer;transition:color .15s,border-color .15s,background .15s}
.btn:hover:not(:disabled){color:var(--text);border-color:var(--line-strong)}
.btn:disabled{opacity:.4;cursor:not-allowed}
.btn-primary{background:var(--accent);border-color:var(--accent);color:#fff}
.btn-primary:hover:not(:disabled){background:var(--accent-hover);border-color:var(--accent-hover);color:#fff}
.btn-danger{color:var(--bad);border-color:transparent}
.btn-danger:hover:not(:disabled){color:var(--bad);border-color:var(--bad)}
.btn-danger-solid{background:var(--bad);border-color:var(--bad);color:#fff}
.btn-danger-solid:hover:not(:disabled){filter:brightness(1.08);color:#fff}
.btn-sm{padding:4px 9px;font-size:11px}

/* Layout blocks ----------------------------------------------------------- */
.page-head{display:flex;flex-wrap:wrap;gap:12px;align-items:flex-start;justify-content:space-between;margin-bottom:14px}
.card{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:18px}
.card-head{display:flex;flex-wrap:wrap;gap:12px;align-items:center;justify-content:space-between;margin-bottom:14px}

/* Stat tiles -------------------------------------------------------------- */
.stats{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:14px}
.stat{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:14px 16px}
.stat-label{font-size:10.5px;color:var(--dim);text-transform:uppercase;letter-spacing:.06em;font-weight:600}
.stat-value{font-size:26px;font-weight:600;line-height:1.15;margin-top:5px;font-variant-numeric:tabular-nums}
.stat-value.sm{font-size:17px;line-height:1.5}
.stat-sub{font-size:11px;color:var(--faint);margin-top:3px}
.v-good{color:var(--good)}.v-warn{color:var(--warn)}.v-bad{color:var(--bad)}.v-info{color:var(--info)}.v-idle{color:var(--dim)}

/* Active transfer --------------------------------------------------------- */
.transfer-empty{min-height:96px;display:flex;flex-direction:column;gap:4px;align-items:center;justify-content:center;border:1px dashed var(--line);border-radius:12px;background:var(--panel-2);color:var(--dim);text-align:center;padding:18px}
.transfer-empty b{color:var(--text);font-weight:600;font-size:13px}
.transfer-name{font-size:15px;font-weight:600;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.transfer-meta{display:flex;flex-wrap:wrap;gap:6px 14px;align-items:center;margin-top:8px;color:var(--muted);font-size:12px}
.progress{height:10px;border-radius:999px;background:var(--inset);border:1px solid var(--line);overflow:hidden;margin:14px 0 9px}
.progress>div{height:100%;border-radius:999px;background:linear-gradient(90deg,var(--accent),var(--info));transition:width .4s ease}

/* Tables ------------------------------------------------------------------ */
.table-wrap{overflow-x:auto;margin:0 -18px -18px;padding:0 18px}
table{width:100%;border-collapse:collapse;font-size:12.5px}
th{text-align:left;padding:9px 10px;color:var(--dim);font-weight:600;font-size:10.5px;text-transform:uppercase;letter-spacing:.05em;border-bottom:1px solid var(--line);white-space:nowrap}
td{padding:10px;border-bottom:1px solid var(--line);vertical-align:middle}
tbody tr:last-child td{border-bottom:0}
tbody tr:hover{background:var(--panel-2)}
.cell-name{font-weight:500;max-width:420px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.cell-err{max-width:300px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;color:var(--muted)}
.row-actions{display:flex;gap:5px;flex-wrap:wrap}
.empty-row{text-align:center;color:var(--dim);padding:28px 10px}

/* Badges and chips -------------------------------------------------------- */
.badge{display:inline-flex;align-items:center;border-radius:999px;padding:3px 9px;font-size:11px;font-weight:600;text-transform:capitalize;white-space:nowrap}
.badge.done{color:var(--good);background:color-mix(in srgb,var(--good) 16%,transparent)}
.badge.copying,.badge.queued{color:var(--accent-text);background:color-mix(in srgb,var(--accent) 22%,transparent)}
.badge.retry_wait{color:var(--warn);background:color-mix(in srgb,var(--warn) 16%,transparent)}
.badge.failed,.badge.cancelled{color:var(--bad);background:color-mix(in srgb,var(--bad) 16%,transparent)}
.badge.paused{color:var(--info);background:color-mix(in srgb,var(--info) 16%,transparent)}
.badge.other{color:var(--muted);background:var(--panel-2)}
.chip{display:inline-flex;align-items:center;gap:5px;border:1px solid var(--line);border-radius:999px;padding:2px 9px;font-size:11px;color:var(--muted);white-space:nowrap}
.chip.src{border-color:color-mix(in srgb,var(--info) 45%,transparent);color:var(--info)}
.chip.src-edited{border-color:color-mix(in srgb,var(--warn) 55%,transparent);color:var(--warn)}
.usage-list{margin:10px 0 0;padding:0;list-style:none;display:grid;gap:6px}
.usage-list li{background:var(--inset);border:1px solid var(--line);border-radius:8px;padding:8px 11px;font-size:12px}
.usage-list b{font-weight:600}
.usage-list span{color:var(--dim)}
.chips{display:flex;flex-wrap:wrap;gap:6px;margin-top:12px}
.filters{display:flex;gap:4px;flex-wrap:wrap}
.filter{padding:5px 10px;font-size:11.5px;border-radius:7px}
.filter.active{background:color-mix(in srgb,var(--accent) 18%,transparent);color:var(--accent-text);border-color:transparent}

/* Entity cards (jobs and remotes) ----------------------------------------- */
.card-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(330px,1fr));gap:12px}
.entity{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:16px;transition:border-color .15s}
.entity:hover{border-color:var(--line-strong)}
.entity-top{display:flex;align-items:flex-start;justify-content:space-between;gap:10px}
.entity-title{font-weight:600;font-size:14px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.entity-foot{display:flex;gap:6px;justify-content:flex-end;margin-top:14px;flex-wrap:wrap}
.remote-grid{grid-template-columns:repeat(auto-fill,minmax(300px,440px))}
.remote-entity{padding:12px 14px}
.remote-main{display:flex;align-items:center;gap:12px;min-width:0}
.remote-id{min-width:0;flex:1}
.remote-meta{display:flex;align-items:center;gap:8px;margin-top:2px}
.remote-status{display:inline-flex;align-items:center;gap:6px;padding:4px 8px;border-radius:999px;border:1px solid var(--line);font-size:11px;line-height:1.2;white-space:nowrap}
.remote-status.ok{color:var(--good);border-color:color-mix(in srgb,var(--good) 45%,transparent);background:color-mix(in srgb,var(--good) 10%,transparent)}
.remote-status.err{color:var(--bad);border-color:color-mix(in srgb,var(--bad) 45%,transparent);background:color-mix(in srgb,var(--bad) 10%,transparent)}
.remote-status.busy{color:var(--accent-text);border-color:color-mix(in srgb,var(--accent) 45%,transparent);background:color-mix(in srgb,var(--accent) 10%,transparent)}
.remote-actions{display:flex;gap:5px;flex:none}
.remote-actions .btn{padding:5px 8px}
.route{display:grid;grid-template-columns:1fr auto 1fr;gap:10px;align-items:center;margin-top:12px;background:var(--inset);border:1px solid var(--line);border-radius:10px;padding:10px 12px}
.route-end{min-width:0}
.route-label{font-size:9.5px;color:var(--dim);text-transform:uppercase;letter-spacing:.06em;font-weight:600}
.route-value{font-size:12px;margin-top:2px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace}
.route-arrow{color:var(--accent-text);font-size:15px}
.empty-state{border:1px dashed var(--line);border-radius:14px;background:var(--panel-2);padding:44px 20px;text-align:center;color:var(--dim)}
.empty-state b{display:block;color:var(--text);font-size:14px;margin-bottom:5px;font-weight:600}

/* Forms ------------------------------------------------------------------- */
input,select,textarea{width:100%;background:var(--inset);border:1px solid var(--field-line);border-radius:8px;padding:8px 10px;font:inherit;font-size:13px;color:var(--text);outline:none;transition:border-color .15s}
input:focus,select:focus,textarea:focus{border-color:var(--accent-hover)}
input::placeholder,textarea::placeholder{color:var(--faint)}
textarea{min-height:76px;resize:vertical;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:12px}
.field{display:block;min-width:0}
.field-label{display:block;font-size:11px;color:var(--dim);margin-bottom:5px}
.field-req{color:var(--accent-text)}
.field-help{font-size:11px;color:var(--faint);margin-top:5px;line-height:1.45}
.field-row{display:flex;align-items:center;gap:9px;font-size:12.5px}
.field-row input[type=checkbox]{width:16px;height:16px;flex:none;accent-color:var(--accent)}
.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:13px}
.form-grid .wide{grid-column:1/-1}
.form-section{margin-bottom:22px}
.form-section:last-child{margin-bottom:0}
.form-section-title{font-size:10.5px;text-transform:uppercase;letter-spacing:.06em;color:var(--dim);font-weight:600;margin-bottom:12px;padding-bottom:8px;border-bottom:1px solid var(--line)}
details.adv{border:1px solid var(--line);border-radius:11px;padding:2px 14px;background:var(--panel-2)}
details.adv summary{cursor:pointer;font-size:12px;color:var(--muted);list-style:none;display:flex;align-items:center;gap:8px;padding:10px 0}
details.adv summary::-webkit-details-marker{display:none}
details.adv summary::before{content:"+";display:grid;place-items:center;width:17px;height:17px;border:1px solid var(--line);border-radius:5px;font-size:11px;line-height:1;flex:none}
details.adv[open] summary::before{content:"\2212"}
details.adv summary:hover{color:var(--text)}
details.adv>div{padding:6px 0 16px}
.param-row{display:grid;grid-template-columns:minmax(0,1fr) minmax(0,1.4fr) auto;gap:8px;align-items:center;margin-bottom:8px}

/* Modals ------------------------------------------------------------------ */
.overlay{position:fixed;inset:0;background:rgba(2,6,23,.66);display:flex;align-items:center;justify-content:center;padding:16px;z-index:60}
.modal{background:var(--panel);border:1px solid var(--line);border-radius:16px;box-shadow:var(--shadow);width:100%;max-height:calc(100vh - 32px);display:flex;flex-direction:column}
.modal.w-md{max-width:560px}
.modal.w-lg{max-width:760px}
.modal-head{display:flex;align-items:flex-start;justify-content:space-between;gap:16px;padding:18px 20px;border-bottom:1px solid var(--line)}
.modal-body{padding:20px;overflow-y:auto}
.modal-foot{display:flex;align-items:center;gap:10px;padding:14px 20px;border-top:1px solid var(--line);flex-wrap:wrap}
.modal-foot .spacer{flex:1}
.icon-btn{border:0;background:transparent;color:var(--dim);font-size:22px;line-height:1;cursor:pointer;padding:0 2px;font-family:inherit}
.icon-btn:hover{color:var(--text)}
.steps{display:flex;align-items:center;gap:8px;font-size:11px;color:var(--dim);margin-top:6px}
.steps b{font-weight:600}
.steps .on{color:var(--accent-text)}

/* Provider picker --------------------------------------------------------- */
.provider-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(158px,1fr));gap:10px}
.provider{display:block;text-align:left;background:var(--inset);border:1px solid var(--line);border-radius:11px;padding:12px;cursor:pointer;color:var(--text);font:inherit;transition:border-color .15s,background .15s}
.provider:hover{border-color:var(--accent);background:color-mix(in srgb,var(--accent) 8%,var(--inset))}
.provider-name{font-size:13px;font-weight:600}
.provider-desc{font-size:11px;color:var(--dim);overflow:hidden;text-overflow:ellipsis;white-space:nowrap;margin-top:2px}
.provider-list{max-height:260px;overflow-y:auto;border:1px solid var(--line);border-radius:11px;background:var(--inset)}
.provider-row{display:flex;align-items:baseline;gap:10px;width:100%;text-align:left;background:transparent;border:0;border-bottom:1px solid var(--line);padding:9px 12px;cursor:pointer;color:var(--text);font:inherit;font-size:12.5px}
.provider-row:last-child{border-bottom:0}
.provider-row:hover{background:color-mix(in srgb,var(--accent) 12%,transparent)}
.provider-row b{font-weight:600;flex:none}
.provider-row span{color:var(--dim);font-size:11px;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}

/* Notices ----------------------------------------------------------------- */
.notice{border:1px solid var(--line);border-radius:10px;padding:11px 13px;font-size:12px;color:var(--muted);background:var(--panel-2);line-height:1.5}
.notice.ok{border-color:color-mix(in srgb,var(--good) 45%,transparent);background:color-mix(in srgb,var(--good) 12%,transparent);color:var(--good)}
.notice.err{border-color:color-mix(in srgb,var(--bad) 45%,transparent);background:color-mix(in srgb,var(--bad) 12%,transparent);color:var(--bad)}
.notice.busy{border-color:color-mix(in srgb,var(--accent) 45%,transparent);background:color-mix(in srgb,var(--accent) 12%,transparent);color:var(--accent-text)}
.stack-gap{display:grid;gap:12px}

.clickable{cursor:pointer}.clickable:hover{text-decoration:underline}
.log-console{background:#090d12;color:#d8e0ea;border:1px solid var(--line);border-radius:11px;padding:12px;min-height:220px;max-height:58vh;overflow:auto;font-family:ui-monospace,SFMono-Regular,Menlo,Consolas,monospace;font-size:11.5px;line-height:1.55;white-space:pre-wrap;word-break:break-word}
.log-line{display:grid;grid-template-columns:155px 58px 70px 1fr;gap:8px;padding:2px 0}
.log-line .ts{color:#77859a}.log-line .lvl{font-weight:700}.log-line .src{color:#9f8cff}.log-line .msg{color:#d8e0ea}
.log-line.ERROR .lvl{color:#ff8585}.log-line.WARN .lvl{color:#f0b04f}.log-line.INFO .lvl{color:#72c9ff}
.detail-grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:10px;margin-bottom:16px}
.detail-stat{background:var(--inset);border:1px solid var(--line);border-radius:10px;padding:10px}
.detail-stat b{display:block;font-size:16px;margin-top:3px}
.detail-tabs{display:flex;gap:5px;margin-bottom:10px}
.file-list{max-height:240px;overflow:auto;border:1px solid var(--line);border-radius:10px}
.file-row{display:flex;justify-content:space-between;gap:14px;padding:8px 10px;border-bottom:1px solid var(--line);font-size:11.5px}
.file-row:last-child{border-bottom:0}.file-row span:first-child{overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.settings-row{display:flex;align-items:center;justify-content:space-between;gap:18px;padding:15px 0;border-bottom:1px solid var(--line)}
.settings-row:last-child{border-bottom:0}
.switch{position:relative;width:44px;height:24px;flex:none}.switch input{opacity:0;width:0;height:0}.switch span{position:absolute;inset:0;background:var(--line-strong);border-radius:999px;cursor:pointer}.switch span:before{content:"";position:absolute;width:18px;height:18px;left:3px;top:3px;background:white;border-radius:50%;transition:.18s}.switch input:checked+span{background:var(--accent)}.switch input:checked+span:before{transform:translateX(20px)}
@media(max-width:700px){.detail-grid{grid-template-columns:repeat(2,1fr)}.log-line{grid-template-columns:1fr}.log-line .ts,.log-line .lvl,.log-line .src{display:inline}}
#toast{position:fixed;left:50%;bottom:26px;transform:translateX(-50%);padding:10px 16px;border-radius:10px;background:var(--panel);border:1px solid var(--line-strong);color:var(--text);box-shadow:var(--shadow);font-size:13px;z-index:80;max-width:min(560px,calc(100vw - 32px))}
#toast.err{border-color:var(--bad);color:var(--bad)}

@media(max-width:900px){.stats{grid-template-columns:repeat(2,minmax(0,1fr))}}
@media(max-width:620px){
  .shell{padding:16px 14px 56px}
  .remote-grid{grid-template-columns:1fr}
  .remote-main{align-items:flex-start;flex-wrap:wrap}
  .remote-actions{width:100%;justify-content:flex-end}
  .remote-status{max-width:100%;white-space:normal}
  .form-grid{grid-template-columns:1fr}
  .form-grid .wide{grid-column:auto}
  .param-row{grid-template-columns:1fr}
  .hide-sm{display:none}
  .cell-name{max-width:170px}
  .topbar-actions{width:100%;justify-content:space-between}
}
</style>
</head>
<body>
<div class="shell">

<div class="topbar">
  <div class="brand">
    <div class="logo">C</div>
    <div>
      <h1>Copyarr</h1>
      <div class="dim" id="build">loading...</div>
    </div>
  </div>
  <div class="topbar-actions">
    <span class="live"><span class="dot" id="liveDot"></span><span id="liveText">Connecting</span></span>
    <button class="btn btn-primary" data-act="scan">Scan now</button>
  </div>
</div>

<nav class="tabs">
  <button class="tab active" data-act="page" data-val="dashboard">Dashboard</button>
  <button class="tab" data-act="page" data-val="jobs">Jobs</button>
  <button class="tab" data-act="page" data-val="remotes">Remotes</button>
</nav>

<!-- Dashboard --------------------------------------------------------------->
<section id="page-dashboard" class="page">
  <div class="stats">
    <div class="stat">
      <div class="stat-label">Queued</div>
      <div class="stat-value num v-info" id="statQueued">0</div>
      <div class="stat-sub">waiting to transfer</div>
    </div>
    <div class="stat">
      <div class="stat-label">Completed</div>
      <div class="stat-value num v-good" id="statDone">0</div>
      <div class="stat-sub">transfers finished</div>
    </div>
    <div class="stat">
      <div class="stat-label">Needs attention</div>
      <div class="stat-value num v-idle" id="statBad">0</div>
      <div class="stat-sub">failed, retrying or cancelled</div>
    </div>
    <div class="stat">
      <div class="stat-label">Last scan</div>
      <div class="stat-value sm" id="statScan">never</div>
      <div class="stat-sub" id="statScanSub">across all enabled jobs</div>
    </div>
  </div>

  <section class="card" style="margin-bottom:14px">
    <div class="card-head"><h2>Active transfer</h2><span class="dim" id="transferHint">Idle</span></div>
    <div id="active"></div>
  </section>

  <section class="card">
    <div class="card-head">
      <div><h2>Recent executions</h2><div class="dim" id="jobCount">loading...</div></div>
      <div class="filters">
        <button class="btn filter active" data-act="filter" data-val="all">All</button>
        <button class="btn filter" data-act="filter" data-val="active">Active</button>
        <button class="btn filter" data-act="filter" data-val="attention">Attention</button>
        <button class="btn filter" data-act="filter" data-val="done">Done</button>
      </div>
    </div>
    <div class="table-wrap">
      <table>
        <thead><tr>
          <th>ID</th><th>Name</th><th>State</th><th>Attempt</th>
          <th class="hide-sm">Size</th><th class="hide-sm">Next retry / error</th><th></th>
        </tr></thead>
        <tbody id="jobRows"></tbody>
      </table>
    </div>
  </section>
</section>

<!-- Jobs -------------------------------------------------------------------->
<section id="page-jobs" class="page" hidden>
  <div class="page-head">
    <div>
      <h2>Jobs</h2>
      <div class="dim">Transfer definitions Copyarr evaluates on every scan.</div>
    </div>
    <button class="btn btn-primary" data-act="job-new">+ New job</button>
  </div>
  <div id="jobDefs"></div>
</section>

<!-- Remotes ----------------------------------------------------------------->
<section id="page-remotes" class="page" hidden>
  <div class="page-head">
    <div>
      <h2>rclone remotes</h2>
      <div class="dim">Stored in Copyarr's own rclone config inside the writable data directory.</div>
    </div>
    <button class="btn btn-primary" data-act="remote-new">+ Add remote</button>
  </div>
  <div id="remoteCards"></div>
</section>

</div>

<!-- Job editor -------------------------------------------------------------->
<div class="overlay" id="jobModal" hidden>
  <div class="modal w-lg" role="dialog" aria-modal="true" aria-labelledby="jobModalTitle">
    <div class="modal-head">
      <div>
        <h2 id="jobModalTitle">New job</h2>
        <div class="dim" id="jobModalSub">Each job carries its own transfer and retry policy.</div>
      </div>
      <button class="icon-btn" data-act="job-close" aria-label="Close">&times;</button>
    </div>
    <div class="modal-body">

      <div class="notice" id="jobProvenance" style="margin-bottom:18px" hidden></div>

      <div class="form-section">
        <div class="form-section-title">Identity</div>
        <div class="form-grid">
          <label class="field">
            <span class="field-label">Job ID <span class="field-req">*</span></span>
            <input id="defId" placeholder="photos-to-nas" autocomplete="off" spellcheck="false">
          </label>
          <label class="field">
            <span class="field-label">Display name</span>
            <input id="defName" placeholder="Photos to NAS" autocomplete="off">
          </label>
          <div class="field wide">
            <label class="field-row"><input id="defEnabled" type="checkbox" checked><span>Enabled - include this job in every scan</span></label>
          </div>
        </div>
      </div>

      <div class="form-section">
        <div class="form-section-title">Route</div>
        <div class="form-grid">
          <label class="field">
            <span class="field-label">Source remote</span>
            <select id="defSrcRemote"></select>
          </label>
          <label class="field">
            <span class="field-label">Source path</span>
            <input id="defSrcPath" placeholder="/incoming" autocomplete="off" spellcheck="false">
          </label>
          <label class="field">
            <span class="field-label">Destination remote</span>
            <select id="defDstRemote"></select>
          </label>
          <label class="field">
            <span class="field-label">Destination path</span>
            <input id="defDstPath" placeholder="/downloads" autocomplete="off" spellcheck="false">
          </label>
        </div>
      </div>

      <div class="form-section">
        <div class="form-section-title">Transfer</div>
        <div class="form-grid">
          <label class="field">
            <span class="field-label">Mode</span>
            <select id="defMode"><option value="copy">Copy</option><option value="move">Move</option></select>
          </label>
          <label class="field">
            <span class="field-label">Verification</span>
            <select id="defVerify"><option value="size">Size</option><option value="none">None</option></select>
          </label>
          <label class="field">
            <span class="field-label">Stability wait (seconds)</span>
            <input id="defStability" type="number" min="0" value="600">
            <span class="field-help">How long a file must stop changing before it is queued.</span>
          </label>
          <label class="field">
            <span class="field-label">Cleanup after (days)</span>
            <input id="defCleanup" type="number" min="0" value="0">
            <span class="field-help">0 keeps transferred sources indefinitely.</span>
          </label>
          <label class="field">
            <span class="field-label">Multi-thread streams</span>
            <input id="defStreams" type="number" min="1" value="4">
          </label>
          <label class="field">
            <span class="field-label">Multi-thread cutoff</span>
            <input id="defCutoff" value="256M" autocomplete="off" spellcheck="false">
          </label>
        </div>
      </div>

      <div class="form-section">
        <div class="form-section-title">Retries</div>
        <div class="form-grid">
          <label class="field">
            <span class="field-label">Automatic retries</span>
            <input id="defRetries" type="number" min="0" value="3">
          </label>
          <label class="field">
            <span class="field-label">Retry wait (seconds)</span>
            <input id="defRetryWait" type="number" min="0" value="300">
          </label>
        </div>
      </div>

      <details class="adv">
        <summary>Advanced</summary>
        <div>
          <div class="form-grid">
            <label class="field">
              <span class="field-label">Initial behavior</span>
              <select id="defInitial">
                <option value="ignore_existing">Ignore files already present on first scan</option>
                <option value="transfer_existing">Transfer everything found on first scan</option>
              </select>
            </label>
            <label class="field">
              <span class="field-label">Extra rclone arguments</span>
              <textarea id="defArgs" placeholder="--transfers&#10;4" spellcheck="false"></textarea>
              <span class="field-help">One argument per line, exactly as rclone expects them.</span>
            </label>
          </div>

          <div style="margin-top:14px">
            <label class="field-row"><input id="defRtEnabled" type="checkbox"><span>Gate this job on rTorrent completion</span></label>
          </div>
          <div id="rtFields" class="form-grid" style="margin-top:12px" hidden>
            <label class="field">
              <span class="field-label">rTorrent URL</span>
              <input id="defRtUrl" placeholder="http://seedbox:8080/RPC2" autocomplete="off" spellcheck="false">
            </label>
            <label class="field">
              <span class="field-label">View</span>
              <input id="defRtView" placeholder="main" autocomplete="off">
            </label>
            <label class="field">
              <span class="field-label">Username</span>
              <input id="defRtUser" autocomplete="off">
            </label>
            <label class="field">
              <span class="field-label">Password</span>
              <input id="defRtPass" type="password" placeholder="unchanged" autocomplete="new-password">
            </label>
            <label class="field wide">
              <span class="field-label">Source base path</span>
              <input id="defRtBase" placeholder="/downloads" autocomplete="off" spellcheck="false">
            </label>
            <div class="field wide">
              <label class="field-row"><input id="defRtRequired" type="checkbox"><span>Fail the scan when rTorrent is unreachable</span></label>
            </div>
          </div>
        </div>
      </details>

    </div>
    <div class="modal-foot">
      <button class="btn btn-danger" id="jobDelete" data-act="job-del" hidden>Delete job</button>
      <button class="btn" id="jobReset" data-act="job-reset" hidden>Reset to config.json</button>
      <span class="spacer"></span>
      <button class="btn" data-act="job-close">Cancel</button>
      <button class="btn btn-primary" data-act="job-save">Save job</button>
    </div>
  </div>
</div>

<!-- Confirmation for destructive actions ------------------------------------>
<div class="overlay" id="confirmModal" hidden>
  <div class="modal w-md" role="alertdialog" aria-modal="true" aria-labelledby="confirmTitle">
    <div class="modal-head">
      <div>
        <h2 id="confirmTitle">Are you sure?</h2>
        <div class="dim" id="confirmSub"></div>
      </div>
    </div>
    <div class="modal-body" id="confirmBody"></div>
    <div class="modal-foot">
      <span class="spacer"></span>
      <button class="btn" data-act="confirm-no">Cancel</button>
      <button class="btn btn-primary" id="confirmYes" data-act="confirm-yes">Confirm</button>
    </div>
  </div>
</div>

<!-- Remote wizard ----------------------------------------------------------->
<div class="overlay" id="remoteModal" hidden>
  <div class="modal w-lg" role="dialog" aria-modal="true" aria-labelledby="remoteModalTitle">
    <div class="modal-head">
      <div>
        <h2 id="remoteModalTitle">Add remote</h2>
        <div class="steps" id="remoteSteps"></div>
      </div>
      <button class="icon-btn" data-act="remote-close" aria-label="Close">&times;</button>
    </div>
    <div class="modal-body" id="remoteBody"></div>
    <div class="modal-foot" id="remoteFoot"></div>
  </div>
</div>

<div id="toast" hidden></div>

<script>
"use strict";

/* Helpers ------------------------------------------------------------------ */
var $ = function(id){ return document.getElementById(id); };

function esc(s){
  return String(s == null ? "" : s).replace(/[&<>"']/g, function(m){
    return { "&":"&amp;", "<":"&lt;", ">":"&gt;", '"':"&quot;", "'":"&#39;" }[m];
  });
}
function bytes(n){
  if(!n) return "0 B";
  var u = ["B","KB","MB","GB","TB"], i = 0;
  while(n >= 1000 && i < u.length - 1){ n /= 1000; i++; }
  return n.toFixed(i ? 1 : 0) + " " + u[i];
}
function dur(s){
  if(s == null) return "unknown";
  s = Math.max(0, Math.round(s));
  if(s < 60) return s + "s";
  if(s < 3600) return Math.floor(s / 60) + "m " + (s % 60) + "s";
  return Math.floor(s / 3600) + "h " + Math.floor((s % 3600) / 60) + "m";
}
function rel(t){
  if(!t) return "never";
  var s = Math.round(Math.max(0, Date.now() - new Date(t).getTime()) / 1000);
  if(s < 60) return s + "s ago";
  if(s < 3600) return Math.floor(s / 60) + "m ago";
  if(s < 86400) return Math.floor(s / 3600) + "h ago";
  return Math.floor(s / 86400) + "d ago";
}
function titleCase(s){
  return String(s || "").replace(/_/g, " ").replace(/^./, function(c){ return c.toUpperCase(); });
}

var toastTimer = null;
function toast(msg, isError){
  var t = $("toast");
  t.textContent = msg;
  t.className = isError ? "err" : "";
  t.hidden = false;
  clearTimeout(toastTimer);
  toastTimer = setTimeout(function(){ t.hidden = true; }, 4000);
}

async function api(url, opt){
  var r = await fetch(url, opt);
  if(!r.ok) throw new Error((await r.text()).trim() || r.statusText);
  var ct = r.headers.get("content-type") || "";
  return ct.indexOf("json") >= 0 ? r.json() : r.text();
}
function jsonBody(v){
  return { method: "POST", headers: { "Content-Type": "application/json" }, body: JSON.stringify(v) };
}

// askConfirm replaces window.confirm so a destructive action can spell out what
// it is about to break, rather than asking to approve a single line of text.
var confirmResolve = null;
function askConfirm(opts){
  return new Promise(function(resolve){
    confirmResolve = resolve;
    $("confirmTitle").textContent = opts.title;
    $("confirmSub").textContent = opts.subtitle || "";
    $("confirmBody").innerHTML = opts.body || "";
    var yes = $("confirmYes");
    yes.textContent = opts.confirmLabel || "Confirm";
    yes.className = "btn " + (opts.danger ? "btn-danger-solid" : "btn-primary");
    $("confirmModal").hidden = false;
    yes.focus();
  });
}
function closeConfirm(answer){
  $("confirmModal").hidden = true;
  if(confirmResolve){
    var resolve = confirmResolve;
    confirmResolve = null;
    resolve(answer);
  }
}

/* State -------------------------------------------------------------------- */
var jobs = [];
var jobFilter = "all";
var defs = [];
var remotes = [];
var remoteTests = {};
var providers = null;
var providersError = "";
var editingDef = null;
var wiz = null;

/* Dashboard ---------------------------------------------------------------- */
function badge(state){
  var known = ["done","copying","queued","retry_wait","failed","cancelled","paused"];
  var cls = known.indexOf(state) >= 0 ? state : "other";
  return '<span class="badge ' + cls + '">' + esc(String(state).replace("_", " ")) + "</span>";
}

function controls(j){
  var acts = [];
  if(j.state === "failed" || j.state === "retry_wait") acts.push("retry");
  if(j.state === "paused") acts.push("resume", "cancel");
  if(j.state === "queued" || j.state === "copying") acts.push("pause", "cancel");
  return acts.map(function(a){
    return '<button class="btn btn-sm" data-act="job-ctl" data-id="' + j.id + '" data-ctl="' + a + '">' + a + "</button>";
  }).join("");
}

function visibleJob(j){
  if(jobFilter === "done") return j.state === "done";
  if(jobFilter === "attention") return ["failed","retry_wait","cancelled"].indexOf(j.state) >= 0;
  if(jobFilter === "active") return ["copying","queued","paused"].indexOf(j.state) >= 0;
  return true;
}

function renderJobs(){
  var shown = jobs.filter(visibleJob);
  $("jobCount").textContent = shown.length + " shown of " + jobs.length + " loaded";
  if(!shown.length){
    $("jobRows").innerHTML = '<tr><td colspan="7" class="empty-row">No executions match this filter.</td></tr>';
    return;
  }
  $("jobRows").innerHTML = shown.map(function(j){
    var attempt = (j.attempt_number || j.attempts || 0) + "/" + (j.max_attempts || "?");
    var trailing = j.next_retry_at
      ? "Retry at " + new Date(j.next_retry_at).toLocaleTimeString()
      : (j.last_error || "-");
    return "<tr>" +
      '<td class="dim num">' + j.id + "</td>" +
      '<td><div class="cell-name" title="' + esc(j.display_name) + '">' + esc(j.display_name) + "</div></td>" +
      "<td>" + badge(j.state) + "</td>" +
      '<td class="num">' + esc(attempt) + "</td>" +
      '<td class="hide-sm num">' + bytes(j.total_bytes) + "</td>" +
      '<td class="hide-sm cell-err" title="' + esc(j.last_error || "") + '">' + esc(trailing) + "</td>" +
      '<td><div class="row-actions">' + controls(j) + "</div></td>" +
      "</tr>";
  }).join("");
}

function renderActive(s){
  var box = $("active");
  if(!s || !s.active){
    $("transferHint").textContent = "Idle";
    box.innerHTML = '<div class="transfer-empty"><b>Nothing transferring</b>' +
      "<div>Queued work starts automatically after the next scan.</div></div>";
    return;
  }
  var a = s.active;
  var pct = Math.max(0, Math.min(100, a.progress_percent || 0));
  $("transferHint").textContent = a.phase || "transferring";
  box.innerHTML =
    '<div class="transfer-name" title="' + esc(a.name) + '">' + esc(a.name) + "</div>" +
    '<div class="transfer-meta">' +
      badge(a.phase || "copying") +
      "<span>Attempt " + (a.attempt_number || "?") + "/" + (a.max_attempts || "?") + "</span>" +
      "<span>" + bytes(a.transferred_bytes) + " of " + bytes(a.total_bytes) + "</span>" +
    "</div>" +
    '<div class="progress"><div style="width:' + pct + '%"></div></div>' +
    '<div class="transfer-meta">' +
      '<strong class="num" style="color:var(--text);font-size:14px">' + pct.toFixed(1) + "%</strong>" +
      '<span class="num">' + bytes(a.speed_bps) + "/s</span>" +
      "<span>ETA " + dur(a.eta_seconds) + "</span>" +
      '<span class="chip">' + (a.multi_thread ? "Multi-thread" : "Single-thread") + "</span>" +
    "</div>";
}

async function refreshStatus(){
  try{
    var s = await api("/api/status");
    $("build").textContent = (s.version || "dev") + " - " + String(s.revision || "unknown").slice(0, 12);

    var queued = (s.queue && s.queue.jobs) || 0;
    var done = (s.jobs && s.jobs.done) || 0;
    var bad = ((s.jobs && s.jobs.failed) || 0) + ((s.jobs && s.jobs.retry_wait) || 0) + ((s.jobs && s.jobs.cancelled) || 0);
    $("statQueued").textContent = queued;
    $("statDone").textContent = done;
    $("statBad").textContent = bad;
    $("statBad").className = "stat-value num " + (bad ? "v-bad" : "v-idle");

    var scans = Object.keys(s.rule_scans || {}).map(function(k){ return s.rule_scans[k]; });
    var last = scans.map(function(x){ return x.last_scan_completed_at; }).filter(Boolean).sort().pop();
    $("statScan").textContent = rel(last);
    var failing = scans.filter(function(x){ return x.last_error; }).length;
    $("statScanSub").textContent = failing
      ? failing + " job" + (failing === 1 ? "" : "s") + " reported a scan error"
      : "across all enabled jobs";

    $("liveText").textContent = s.scanning ? "Scanning" : "Online";
    $("liveDot").className = "dot" + (s.scanning ? " scan" : "");
    renderActive(s);
  }catch(e){
    $("build").textContent = "API unreachable";
    $("liveText").textContent = "Offline";
    $("liveDot").className = "dot off";
  }
}

async function refreshJobs(){
  try{
    jobs = await api("/api/jobs?limit=60");
    renderJobs();
  }catch(e){ /* the live pill already reports the outage */ }
}

async function refresh(){
  await Promise.all([refreshStatus(), refreshJobs()]);
}

/* Job definitions ---------------------------------------------------------- */
function endpointText(ep){
  var remote = ep && ep.remote ? ep.remote + ":" : "";
  var p = (ep && ep.path) || "/";
  return remote ? remote + p : p;
}

// Where a job came from. config.json is read-only to Copyarr, so a UI edit of a
// config job is stored as an override that masks the file - say so plainly
// rather than letting the file look like it stopped taking effect.
function provenanceChip(d){
  if(d.origin !== "config"){
    return '<span class="chip src">Created in the UI</span>';
  }
  if(d.has_override){
    return '<span class="chip src-edited" title="A UI edit is masking the definition in config.json">' +
      "config.json &middot; edited here</span>";
  }
  return '<span class="chip src">From config.json</span>';
}

function renderDefs(){
  var host = $("jobDefs");
  if(!defs.length){
    host.innerHTML = '<div class="empty-state"><b>No jobs yet</b>' +
      "<div>A job tells Copyarr what to watch and where to put it.</div>" +
      '<button class="btn btn-primary" style="margin-top:16px" data-act="job-new">+ Create your first job</button></div>';
    return;
  }
  host.innerHTML = '<div class="card-grid">' + defs.map(function(d){
    var chips = [
      titleCase(d.mode || "copy"),
      "Verify: " + (d.verification || "size"),
      "Stable after " + (d.stability_seconds || 600) + "s",
      (d.retry_count == null ? 3 : d.retry_count) + " retries"
    ];
    if(d.cleanup_days) chips.push("Cleanup after " + d.cleanup_days + "d");
    if(d.rtorrent) chips.push("rTorrent gated");
    return '<div class="entity">' +
      '<div class="entity-top">' +
        '<div style="min-width:0">' +
          '<div class="entity-title" title="' + esc(d.name || d.id) + '">' + esc(d.name || d.id) + "</div>" +
          '<div class="dim">' + esc(d.id) + "</div>" +
        "</div>" +
        '<span class="chip"><span class="dot' + (d.enabled ? "" : " off") + '"></span>' + (d.enabled ? "Enabled" : "Disabled") + "</span>" +
      "</div>" +
      '<div style="margin-top:10px">' + provenanceChip(d) + "</div>" +
      '<div class="route">' +
        '<div class="route-end"><div class="route-label">Source</div>' +
          '<div class="route-value" title="' + esc(endpointText(d.source)) + '">' + esc(endpointText(d.source)) + "</div></div>" +
        '<div class="route-arrow">&rarr;</div>' +
        '<div class="route-end"><div class="route-label">Destination</div>' +
          '<div class="route-value" title="' + esc(endpointText(d.destination)) + '">' + esc(endpointText(d.destination)) + "</div></div>" +
      "</div>" +
      '<div class="chips">' + chips.map(function(c){ return '<span class="chip">' + esc(c) + "</span>"; }).join("") + "</div>" +
      '<div class="entity-foot">' +
        '<button class="btn btn-sm" data-act="job-edit" data-id="' + esc(d.id) + '">Edit</button>' +
      "</div>" +
    "</div>";
  }).join("") + "</div>";
}

function fillRemoteSelects(){
  var opts = ['<option value="">Local filesystem</option>'].concat(remotes.map(function(r){
    return '<option value="' + esc(r.name) + '">' + esc(r.name) + (r.type ? " (" + esc(r.type) + ")" : "") + "</option>";
  })).join("");
  var src = $("defSrcRemote"), dst = $("defDstRemote");
  var keepSrc = src.value, keepDst = dst.value;
  src.innerHTML = opts; dst.innerHTML = opts;
  src.value = keepSrc; dst.value = keepDst;
}

async function refreshDefs(){
  try{
    defs = await api("/api/job-definitions");
    renderDefs();
  }catch(e){ toast(e.message, true); }
}

/* Job editor --------------------------------------------------------------- */
// A job can point at a remote that has since been deleted; keep that value
// selectable so editing an unrelated field cannot silently reroute the job.
function setSelect(el, value){
  el.value = value || "";
  if(el.value === (value || "")) return;
  var opt = document.createElement("option");
  opt.value = value;
  opt.textContent = value + " (missing)";
  el.appendChild(opt);
  el.value = value;
}

function setJobForm(d){
  $("defId").value = d.id || "";
  $("defId").disabled = !!editingDef;
  $("defName").value = d.name || "";
  $("defEnabled").checked = d.enabled !== false;
  setSelect($("defSrcRemote"), (d.source && d.source.remote) || "");
  $("defSrcPath").value = (d.source && d.source.path) || "";
  setSelect($("defDstRemote"), (d.destination && d.destination.remote) || "");
  $("defDstPath").value = (d.destination && d.destination.path) || "";
  $("defMode").value = d.mode || "copy";
  $("defVerify").value = d.verification || "size";
  $("defStability").value = d.stability_seconds || 600;
  $("defCleanup").value = d.cleanup_days || 0;
  $("defStreams").value = d.multi_thread_streams || 4;
  $("defCutoff").value = d.multi_thread_cutoff || "256M";
  $("defRetries").value = d.retry_count == null ? 3 : d.retry_count;
  $("defRetryWait").value = d.retry_wait_seconds == null ? 300 : d.retry_wait_seconds;
  $("defInitial").value = d.initial_behavior || "ignore_existing";
  $("defArgs").value = (d.rclone_args || []).join("\n");

  var rt = d.rtorrent || null;
  $("defRtEnabled").checked = !!rt;
  $("rtFields").hidden = !rt;
  $("defRtUrl").value = (rt && rt.url) || "";
  $("defRtView").value = (rt && rt.view) || "main";
  $("defRtUser").value = (rt && rt.username) || "";
  $("defRtPass").value = "";
  $("defRtBase").value = (rt && rt.source_base_path) || "";
  $("defRtRequired").checked = !!(rt && rt.required);
}

function openJobModal(id){
  editingDef = id ? defs.filter(function(d){ return d.id === id; })[0] : null;
  if(id && !editingDef) return;
  fillRemoteSelects();
  setJobForm(editingDef ? editingDef : { enabled: true, destination: { path: "/downloads" } });
  $("jobModalTitle").textContent = editingDef ? "Edit job" : "New job";
  $("jobModalSub").textContent = editingDef
    ? "Job ID cannot change once it exists."
    : "Each job carries its own transfer and retry policy.";

  var fromConfig = !!editingDef && editingDef.origin === "config";
  // A config.json job can only be deleted by editing that file; the UI offers
  // to drop its override instead, which is the way back to the file's version.
  $("jobDelete").hidden = !editingDef || fromConfig;
  $("jobReset").hidden = !fromConfig || !editingDef.has_override;

  var notice = $("jobProvenance");
  if(fromConfig){
    notice.hidden = false;
    notice.className = "notice" + (editingDef.has_override ? " busy" : "");
    notice.innerHTML = editingDef.has_override
      ? "This job comes from <strong>config.json</strong> and has been edited here. " +
        "Copyarr never rewrites that file, so the version below is stored separately and masks it. " +
        "Use <strong>Reset to config.json</strong> to discard these changes and follow the file again."
      : "This job comes from <strong>config.json</strong>. Copyarr never rewrites that file, so saving " +
        "here stores an override that masks it until you reset it.";
  }else{
    notice.hidden = true;
  }

  $("jobModal").hidden = false;
  setTimeout(function(){ (editingDef ? $("defName") : $("defId")).focus(); }, 0);
}

function closeJobModal(){
  $("jobModal").hidden = true;
  editingDef = null;
}

async function saveJob(){
  try{
    var id = $("defId").value.trim();
    if(!id) throw new Error("Job ID is required");

    var d = editingDef ? JSON.parse(JSON.stringify(editingDef)) : {};
    d.id = id;
    d.name = $("defName").value.trim() || id;
    d.enabled = $("defEnabled").checked;
    d.source = { remote: $("defSrcRemote").value, path: $("defSrcPath").value.trim() };
    d.destination = { remote: $("defDstRemote").value, path: $("defDstPath").value.trim() };
    d.mode = $("defMode").value;
    d.verification = $("defVerify").value;
    d.stability_seconds = Number($("defStability").value);
    d.cleanup_days = Number($("defCleanup").value);
    d.multi_thread_streams = Number($("defStreams").value);
    d.multi_thread_cutoff = $("defCutoff").value.trim() || "256M";
    d.retry_count = Number($("defRetries").value);
    d.retry_wait_seconds = Number($("defRetryWait").value);
    d.initial_behavior = $("defInitial").value;
    d.rclone_args = $("defArgs").value.split("\n").map(function(x){ return x.trim(); }).filter(Boolean);

    if($("defRtEnabled").checked){
      d.rtorrent = {
        url: $("defRtUrl").value.trim(),
        username: $("defRtUser").value.trim(),
        password: $("defRtPass").value,
        view: $("defRtView").value.trim() || "main",
        source_base_path: $("defRtBase").value.trim(),
        required: $("defRtRequired").checked
      };
    }else{
      d.rtorrent = null;
    }

    var url = editingDef ? "/api/job-definitions/" + encodeURIComponent(id) : "/api/job-definitions";
    await api(url, {
      method: editingDef ? "PUT" : "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(d)
    });
    toast("Job saved");
    closeJobModal();
    await refreshDefs();
  }catch(e){ toast(e.message, true); }
}

async function deleteJob(){
  if(!editingDef) return;
  var id = editingDef.id;
  var ok = await askConfirm({
    title: "Delete this job?",
    subtitle: editingDef.name || id,
    body: '<div class="notice">The definition is removed and nothing new will be queued for it. ' +
      "Executions already recorded stay in the history.</div>",
    confirmLabel: "Delete job",
    danger: true
  });
  if(!ok) return;
  try{
    await api("/api/job-definitions/" + encodeURIComponent(id), { method: "DELETE" });
    toast("Job deleted");
    closeJobModal();
    await refreshDefs();
  }catch(e){ toast(e.message, true); }
}

async function resetJob(){
  if(!editingDef) return;
  var id = editingDef.id;
  var ok = await askConfirm({
    title: "Reset to config.json?",
    subtitle: editingDef.name || id,
    body: '<div class="notice">The changes made here are discarded and this job follows ' +
      "<strong>config.json</strong> again. The file itself is not touched.</div>",
    confirmLabel: "Reset to config.json",
    danger: true
  });
  if(!ok) return;
  try{
    await api("/api/job-definitions/" + encodeURIComponent(id) + "/reset", { method: "POST" });
    toast("Job reset to config.json");
    closeJobModal();
    await refreshDefs();
  }catch(e){ toast(e.message, true); }
}

/* Remotes ------------------------------------------------------------------ */
function renderRemotes(){
  var host = $("remoteCards");
  if(!remotes.length){
    host.innerHTML = '<div class="empty-state"><b>No remotes configured</b>' +
      "<div>Add an rclone remote to copy to or from anything that is not local storage.</div>" +
      '<button class="btn btn-primary" style="margin-top:16px" data-act="remote-new">+ Add your first remote</button></div>';
    return;
  }
  host.innerHTML = '<div class="card-grid remote-grid">' + remotes.map(function(r){
    var t = remoteTests[r.name];
    var status = "";
    if(t){
      var cls = t.state === "busy" ? "busy" : (t.state === "ok" ? "ok" : "err");
      status = '<span class="remote-status ' + cls + '">' + esc(t.message) + "</span>";
    }
    return '<div class="entity remote-entity">' +
      '<div class="remote-main">' +
        '<div class="remote-id">' +
          '<div class="entity-title" title="' + esc(r.name) + '">' + esc(r.name) + "</div>" +
          '<div class="remote-meta"><span class="dim">' + esc(r.type || "unknown type") + "</span>" + status + "</div>" +
        "</div>" +
        '<div class="remote-actions">' +
          '<button class="btn btn-sm" data-act="remote-test" data-name="' + esc(r.name) + '"' + (t && t.state === "busy" ? " disabled" : "") + ">Test</button>" +
          '<button class="btn btn-sm" data-act="remote-edit" data-name="' + esc(r.name) + '">Edit</button>' +
          '<button class="btn btn-sm btn-danger" data-act="remote-del" data-name="' + esc(r.name) + '">Delete</button>' +
        "</div>" +
      "</div>" +
    "</div>";
  }).join("") + "</div>";
}

async function refreshRemotes(render){
  try{
    remotes = await api("/api/remotes");
    fillRemoteSelects();
    if(render !== false) renderRemotes();
  }catch(e){
    if(render !== false){
      $("remoteCards").innerHTML = '<div class="notice err">' + esc(e.message) + "</div>";
    }
  }
}

async function testSavedRemote(name){
  remoteTests[name] = { state: "busy", message: "Testing connection..." };
  renderRemotes();
  try{
    var res = await api("/api/remotes/test", jsonBody({ name: name }));
    remoteTests[name] = { state: res.ok ? "ok" : "err", message: res.message };
  }catch(e){
    remoteTests[name] = { state: "err", message: e.message };
  }
  renderRemotes();
}

async function deleteRemote(name){
  // Look up what would break first, so the warning names the affected jobs
  // instead of asking to approve a generic sentence. The server refuses the
  // delete without confirm=true as well, so this cannot be skipped by accident.
  var usage = [];
  try{
    usage = await api("/api/remotes/" + encodeURIComponent(name) + "/usage") || [];
  }catch(e){ /* fall through to the plain warning; the server still guards */ }

  var body, confirmLabel;
  if(usage.length){
    body = '<div class="notice err"><strong>' + usage.length + " job definition" +
      (usage.length === 1 ? "" : "s") + "</strong> still point at <strong>" + esc(name) +
      "</strong>. Deleting it will break " + (usage.length === 1 ? "that job" : "them") +
      " at the next scan.</div>" +
      '<ul class="usage-list">' + usage.map(function(u){
        return "<li><b>" + esc(u.job_name || u.job_id) + "</b> <span>" + esc(u.job_id) +
          " &middot; used as " + esc(u.role) + "</span></li>";
      }).join("") + "</ul>";
    confirmLabel = "Delete anyway";
  }else{
    body = '<div class="notice">No job definition refers to this remote.</div>';
    confirmLabel = "Delete remote";
  }

  var ok = await askConfirm({
    title: usage.length ? "This remote is still in use" : "Delete this remote?",
    subtitle: name,
    body: body,
    confirmLabel: confirmLabel,
    danger: true
  });
  if(!ok) return;

  try{
    await api("/api/remotes/" + encodeURIComponent(name) + "?confirm=true", { method: "DELETE" });
    delete remoteTests[name];
    toast("Remote deleted");
    await refreshRemotes();
  }catch(e){ toast(e.message, true); }
}

/* Remote wizard ------------------------------------------------------------ */
var COMMON = ["sftp","ftp","webdav","s3","drive","onedrive","smb","local"];

async function loadProviders(){
  if(providers || providersError) return;
  try{
    var list = await api("/api/remotes/providers");
    providers = Array.isArray(list) ? list : [];
  }catch(e){
    providers = null;
    providersError = e.message;
  }
}

function providerByName(name){
  return (providers || []).filter(function(p){ return p.Name === name; })[0] || null;
}

// rclone scopes an option to sub-providers with a comma list, optionally
// negated with a leading "!" - for example "!AWS,Ceph" on the s3 backend.
function providerMatches(spec, current){
  if(!spec) return true;
  var negate = spec.charAt(0) === "!";
  var list = (negate ? spec.slice(1) : spec).split(",").map(function(x){ return x.trim(); }).filter(Boolean);
  var hit = list.indexOf(current || "") >= 0;
  return negate ? !hit : hit;
}

function optionHelp(o){
  var h = String(o.Help || "").split("\n\n")[0].replace(/\s+/g, " ").trim();
  return h.length > 200 ? h.slice(0, 200) + "..." : h;
}

// Every rclone backend that scopes options to a sub-provider (s3, storj, koofr,
// oracleobjectstorage) names the discriminator "provider", and Option.Provider
// is matched against its value. It decides which other fields even exist, so it
// always gets a real chooser rather than a free-text box.
function isProviderKey(o){ return o.Name === "provider"; }

// rclone's Hide is a bit field: 1 hides from the command line, 2 hides from the
// configurator. Only bit 2 means "do not ask the user for this".
function hiddenFromConfig(o){ return (Number(o.Hide) || 0) & 2; }

function visibleOptions(advanced){
  if(!wiz.provider) return [];
  var current = wiz.values["provider"] || "";
  return (wiz.provider.Options || []).filter(function(o){
    if(hiddenFromConfig(o)) return false;
    if(!!o.Advanced !== advanced) return false;
    return providerMatches(o.Provider, current);
  });
}

// These options render as a select with no blank entry, so the browser shows
// their first choice as selected. Mirror that into wiz.values, or the sub-
// provider would still read as empty and every provider-scoped field would stay
// hidden. Two passes cover one level of nesting, which is as deep as rclone's
// provider-scoped options go.
function seedDefaults(){
  if(!wiz.provider) return;
  for(var pass = 0; pass < 2; pass++){
    visibleOptions(false).concat(visibleOptions(true)).forEach(function(o){
      if(!isProviderKey(o) && !(o.Required && o.Exclusive)) return;
      if(wiz.values[o.Name]) return;
      var usable = (o.Examples || []).filter(function(ex){
        return providerMatches(ex.Provider, wiz.values["provider"] || "");
      });
      if(usable.length) wiz.values[o.Name] = String(usable[0].Value == null ? "" : usable[0].Value);
    });
  }
}

function optionField(o){
  var key = o.Name;
  var val = wiz.values[key] == null ? "" : wiz.values[key];
  var def = o.DefaultStr == null ? "" : String(o.DefaultStr);
  var isRedacted = wiz.redacted.indexOf(key) >= 0;
  var label = esc(titleCase(key)) + (o.Required ? ' <span class="field-req">*</span>' : "");
  var help = optionHelp(o);
  var control;

  if(o.Type === "bool"){
    var on = val === "true" || (val === "" && def === "true");
    control = '<label class="field-row"><input type="checkbox" data-opt="' + esc(key) + '" data-kind="bool"' +
      (on ? " checked" : "") + '><span class="dim">' + (def === "true" ? "on by default" : "off by default") + "</span></label>";
    return '<div class="field' + (o.Advanced ? "" : " wide") + '">' +
      '<span class="field-label">' + label + "</span>" + control +
      (help ? '<div class="field-help">' + esc(help) + "</div>" : "") + "</div>";
  }

  var examples = o.Examples || [];
  var usable = examples.filter(function(ex){ return providerMatches(ex.Provider, wiz.values["provider"] || ""); });

  if(usable.length && (o.Exclusive || isProviderKey(o))){
    control = '<select data-opt="' + esc(key) + '">' +
      (o.Required || isProviderKey(o) ? "" : '<option value="">Not set</option>') +
      usable.map(function(ex){
        var v = String(ex.Value == null ? "" : ex.Value);
        var text = ex.Help ? v + " - " + String(ex.Help).split("\n")[0] : v;
        return '<option value="' + esc(v) + '"' + (v === val ? " selected" : "") + ">" + esc(text) + "</option>";
      }).join("") + "</select>";
  }else{
    var listId = usable.length ? "dl_" + key.replace(/[^A-Za-z0-9_]/g, "_") : "";
    var type = o.IsPassword ? "password" : "text";
    var placeholder = isRedacted ? "unchanged" : (def || "");
    control = "<input" +
      ' type="' + type + '"' +
      ' data-opt="' + esc(key) + '"' +
      ' value="' + esc(val) + '"' +
      ' placeholder="' + esc(placeholder) + '"' +
      ' autocomplete="' + (o.IsPassword ? "new-password" : "off") + '"' +
      ' spellcheck="false"' +
      (listId ? ' list="' + listId + '"' : "") + ">";
    if(listId){
      control += '<datalist id="' + listId + '">' + usable.map(function(ex){
        return '<option value="' + esc(String(ex.Value == null ? "" : ex.Value)) + '">' +
          esc(String(ex.Help || "").split("\n")[0]) + "</option>";
      }).join("") + "</datalist>";
    }
  }

  return '<label class="field">' +
    '<span class="field-label">' + label + "</span>" + control +
    (help ? '<div class="field-help">' + esc(help) + "</div>" : "") +
  "</label>";
}

function renderProviderList(){
  var host = $("wizList");
  if(!host) return;
  var q = wiz.search.toLowerCase();
  var matches = (providers || []).filter(function(p){
    return !q || p.Name.toLowerCase().indexOf(q) >= 0 || String(p.Description || "").toLowerCase().indexOf(q) >= 0;
  });
  host.innerHTML = matches.length
    ? matches.map(function(p){
        return '<button class="provider-row" data-act="wiz-pick" data-type="' + esc(p.Name) + '">' +
          "<b>" + esc(p.Name) + "</b><span>" + esc(p.Description || "") + "</span></button>";
      }).join("")
    : '<div class="empty-row">No provider matches "' + esc(wiz.search) + '"</div>';
}

function renderWizardStep1(){
  var common = COMMON.map(providerByName).filter(Boolean);
  $("remoteBody").innerHTML =
    (common.length
      ? '<div class="form-section"><div class="form-section-title">Common providers</div>' +
        '<div class="provider-grid">' + common.map(function(p){
          return '<button class="provider" data-act="wiz-pick" data-type="' + esc(p.Name) + '">' +
            '<div class="provider-name">' + esc(p.Name) + "</div>" +
            '<div class="provider-desc" title="' + esc(p.Description || "") + '">' + esc(p.Description || "") + "</div>" +
          "</button>";
        }).join("") + "</div></div>"
      : "") +
    '<div class="form-section"><div class="form-section-title">All providers (' + (providers || []).length + ")</div>" +
      '<input id="wizSearch" placeholder="Search rclone backends" autocomplete="off" spellcheck="false" value="' + esc(wiz.search) + '">' +
      '<div class="provider-list" id="wizList" style="margin-top:10px"></div>' +
    "</div>";
  renderProviderList();
  $("remoteFoot").innerHTML =
    '<button class="btn" data-act="wiz-manual">Enter settings manually</button>' +
    '<span class="spacer"></span>' +
    '<button class="btn" data-act="remote-close">Cancel</button>';
}

function renderManualRows(){
  return wiz.rows.map(function(row, i){
    return '<div class="param-row">' +
      '<input data-row="' + i + '" data-part="k" value="' + esc(row.k) + '" placeholder="key" autocomplete="off" spellcheck="false">' +
      '<input data-row="' + i + '" data-part="v" value="' + esc(row.v) + '" placeholder="value" autocomplete="off" spellcheck="false">' +
      '<button class="btn btn-sm btn-danger" data-act="wiz-del-row" data-row="' + i + '">Remove</button>' +
    "</div>";
  }).join("");
}

function renderWizardStep2(){
  var editing = wiz.mode === "edit";
  var body = "";

  body += '<div class="form-section"><div class="form-section-title">Remote</div><div class="form-grid">' +
    '<label class="field"><span class="field-label">Name <span class="field-req">*</span></span>' +
      '<input id="wizName" value="' + esc(wiz.name) + '"' + (editing ? " disabled" : "") +
      ' placeholder="seedbox" autocomplete="off" spellcheck="false"></label>' +
    '<label class="field"><span class="field-label">Type</span>' +
      (wiz.manual
        ? '<input id="wizType" value="' + esc(wiz.type) + '"' + (editing ? " disabled" : "") + ' placeholder="sftp" autocomplete="off" spellcheck="false">'
        : '<input value="' + esc(wiz.type) + '" disabled>') +
    "</label></div>";
  if(wiz.provider && wiz.provider.Description){
    body += '<div class="field-help" style="margin-top:10px">' + esc(wiz.provider.Description) + "</div>";
  }
  body += "</div>";

  if(wiz.manual){
    body += '<div class="form-section"><div class="form-section-title">Parameters</div>' +
      renderManualRows() +
      '<button class="btn btn-sm" data-act="wiz-add-row">+ Add parameter</button>' +
      '<div class="field-help" style="margin-top:10px">Keys are rclone config keys, for example host, user, pass or port.</div>' +
      "</div>";
  }else{
    seedDefaults();
    var basic = visibleOptions(false);
    var advanced = visibleOptions(true);
    body += '<div class="form-section"><div class="form-section-title">Settings</div>' +
      (basic.length
        ? '<div class="form-grid">' + basic.map(optionField).join("") + "</div>"
        : '<div class="field-help">This provider needs no basic settings.</div>') +
      "</div>";
    if(advanced.length){
      body += '<details class="adv"><summary>Advanced settings (' + advanced.length + ")</summary>" +
        '<div><div class="form-grid">' + advanced.map(optionField).join("") + "</div></div></details>";
    }
  }

  if(editing && wiz.redacted.length){
    // rclone redacts every option it marks sensitive, which is broader than
    // just passwords - on sftp that includes host and user too.
    body += '<div class="notice" style="margin-top:16px">rclone withholds the stored value of ' +
      wiz.redacted.map(function(k){ return "<strong>" + esc(k) + "</strong>"; }).join(", ") +
      ", so these are never sent back to the browser. Leave them blank to keep what is stored. " +
      "Testing checks the remote as currently saved, so save a change before testing it.</div>";
  }

  body += '<div class="notice" id="wizResult" style="margin-top:16px" hidden></div>';
  $("remoteBody").innerHTML = body;

  $("remoteFoot").innerHTML =
    (editing ? "" : '<button class="btn" data-act="wiz-back">Back</button>') +
    '<span class="spacer"></span>' +
    '<button class="btn" data-act="wiz-test" id="wizTestBtn">' + (editing ? "Test saved remote" : "Test connection") + "</button>" +
    '<button class="btn btn-primary" data-act="wiz-save" id="wizSaveBtn">' + (editing ? "Save changes" : "Create remote") + "</button>";

  if(wiz.result) showWizResult(wiz.result.state, wiz.result.message);
}

function showWizResult(state, message){
  wiz.result = { state: state, message: message };
  var box = $("wizResult");
  if(!box) return;
  box.className = "notice " + (state === "busy" ? "busy" : (state === "ok" ? "ok" : "err"));
  box.textContent = message;
  box.hidden = false;
}

function renderWizard(){
  $("remoteModalTitle").textContent = wiz.mode === "edit" ? "Edit remote" : "Add remote";
  $("remoteSteps").innerHTML = wiz.mode === "edit"
    ? '<b class="on">' + esc(wiz.name) + "</b>"
    : '<b class="' + (wiz.step === 1 ? "on" : "") + '">1 Provider</b><span>&rsaquo;</span>' +
      '<b class="' + (wiz.step === 2 ? "on" : "") + '">2 Configure</b>';
  if(wiz.step === 1) renderWizardStep1(); else renderWizardStep2();
}

async function openRemoteWizard(name){
  wiz = {
    mode: name ? "edit" : "create",
    step: 1, provider: null, type: "", name: name || "",
    values: {}, redacted: [], initial: {}, search: "", manual: false,
    rows: [{ k: "", v: "" }], result: null
  };
  $("remoteModal").hidden = false;
  $("remoteBody").innerHTML = '<div class="notice busy">Loading rclone providers...</div>';
  $("remoteFoot").innerHTML = '<span class="spacer"></span><button class="btn" data-act="remote-close">Cancel</button>';

  await loadProviders();

  if(name){
    try{
      var detail = await api("/api/remotes/" + encodeURIComponent(name));
      wiz.type = detail.type || "";
      wiz.values = Object.assign({}, detail.parameters || {});
      wiz.initial = Object.assign({}, detail.parameters || {});
      wiz.redacted = detail.redacted || [];
      wiz.provider = providerByName(wiz.type);
      wiz.manual = !wiz.provider;
      if(wiz.manual){
        wiz.rows = Object.keys(wiz.values).map(function(k){ return { k: k, v: wiz.values[k] }; });
        if(!wiz.rows.length) wiz.rows = [{ k: "", v: "" }];
      }
      wiz.step = 2;
    }catch(e){
      $("remoteBody").innerHTML = '<div class="notice err">' + esc(e.message) + "</div>";
      return;
    }
  }else if(!providers){
    wiz.manual = true;
    wiz.step = 2;
  }
  renderWizard();
  if(providersError && !name){
    showWizResult("err", "Could not read rclone's provider list (" + providersError + "). Falling back to manual settings.");
  }
}

function closeRemoteWizard(){
  $("remoteModal").hidden = true;
  wiz = null;
}

// wizParams turns the form into the flat string map the API expects, dropping
// anything the user left at its rclone default so the config stays minimal.
function wizParams(){
  var params = {};
  if(wiz.manual){
    wiz.rows.forEach(function(row){
      var k = row.k.trim();
      if(k) params[k] = row.v;
    });
  }else{
    var opts = {};
    (wiz.provider && wiz.provider.Options ? wiz.provider.Options : []).forEach(function(o){ opts[o.Name] = o; });
    Object.keys(wiz.values).forEach(function(k){
      var v = wiz.values[k];
      var o = opts[k];
      var def = o && o.DefaultStr != null ? String(o.DefaultStr) : "";
      if(o && o.Type === "bool"){
        if(v !== (def || "false")) params[k] = v;
        return;
      }
      if(v === "" || v == null) return;
      if(v === def && !(o && o.Required)) return;
      params[k] = v;
    });
  }
  if(wiz.mode === "edit"){
    // Only send what actually changed, so untouched secrets stay untouched.
    Object.keys(params).forEach(function(k){
      if(wiz.initial[k] === params[k]) delete params[k];
    });
  }
  return params;
}

function wizType(){
  return wiz.manual && wiz.mode !== "edit" ? String(wiz.type || "").trim() : wiz.type;
}

// Catch blank required fields here rather than making the user wait for a
// connection attempt that rclone would reject outright.
function missingRequired(){
  if(wiz.manual || wiz.mode === "edit") return [];
  return visibleOptions(false).filter(function(o){
    if(!o.Required) return false;
    var v = wiz.values[o.Name];
    var def = o.DefaultStr == null ? "" : String(o.DefaultStr);
    return (v == null || v === "") && def === "";
  }).map(function(o){ return titleCase(o.Name); });
}

async function wizTest(){
  var missing = missingRequired();
  if(missing.length){
    showWizResult("err", "Fill in the required settings first: " + missing.join(", ") + ".");
    return;
  }
  var btn = $("wizTestBtn");
  if(btn) btn.disabled = true;
  showWizResult("busy", "Testing connection...");
  try{
    var payload = wiz.mode === "edit"
      ? { name: wiz.name }
      : { type: wizType(), parameters: wizParams() };
    if(wiz.mode !== "edit" && !payload.type) throw new Error("Pick a provider type first");
    var res = await api("/api/remotes/test", jsonBody(payload));
    showWizResult(res.ok ? "ok" : "err", res.message);
  }catch(e){
    showWizResult("err", e.message);
  }finally{
    if($("wizTestBtn")) $("wizTestBtn").disabled = false;
  }
}

async function wizSave(){
  var missing = missingRequired();
  if(missing.length){
    showWizResult("err", "Fill in the required settings first: " + missing.join(", ") + ".");
    return;
  }
  var btn = $("wizSaveBtn");
  if(btn) btn.disabled = true;
  try{
    var params = wizParams();
    var res;
    if(wiz.mode === "edit"){
      res = await api("/api/remotes/" + encodeURIComponent(wiz.name), {
        method: "PATCH",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ parameters: params })
      });
    }else{
      var name = wiz.name.trim();
      if(!name) throw new Error("Remote name is required");
      if(!wizType()) throw new Error("Provider type is required");
      res = await api("/api/remotes", jsonBody({ name: name, type: wizType(), parameters: params }));
    }
    if(res.question){
      showWizResult("err", "This provider needs an interactive or OAuth step that Copyarr cannot complete yet.");
      return;
    }
    toast(wiz.mode === "edit" ? "Remote updated" : "Remote created");
    delete remoteTests[wiz.name];
    closeRemoteWizard();
    await refreshRemotes();
  }catch(e){
    showWizResult("err", e.message);
  }finally{
    if($("wizSaveBtn")) $("wizSaveBtn").disabled = false;
  }
}

/* Navigation and events ---------------------------------------------------- */
function showPage(name, tab){
  ["dashboard","jobs","remotes"].forEach(function(p){ $("page-" + p).hidden = p !== name; });
  Array.prototype.forEach.call(document.querySelectorAll(".tab"), function(t){ t.classList.remove("active"); });
  if(tab) tab.classList.add("active");
  if(name === "jobs"){ refreshRemotes(false); refreshDefs(); }
  if(name === "remotes") refreshRemotes();
}

function setFilter(f, el){
  jobFilter = f;
  Array.prototype.forEach.call(document.querySelectorAll(".filter"), function(x){ x.classList.remove("active"); });
  if(el) el.classList.add("active");
  renderJobs();
}

async function controlJob(id, action){
  try{
    await api("/api/jobs/" + id + "/" + action, { method: "POST" });
    toast(action + " requested");
    refresh();
  }catch(e){ toast(e.message, true); }
}

document.addEventListener("click", function(ev){
  var t = ev.target.closest("[data-act]");
  if(!t) return;
  var act = t.dataset.act;

  if(act === "scan"){
    api("/api/scan", { method: "POST" })
      .then(function(){ toast("Scan queued"); refreshStatus(); })
      .catch(function(e){ toast(e.message, true); });
  }
  else if(act === "page") showPage(t.dataset.val, t);
  else if(act === "filter") setFilter(t.dataset.val, t);
  else if(act === "job-ctl") controlJob(t.dataset.id, t.dataset.ctl);
  else if(act === "job-new") openJobModal(null);
  else if(act === "job-edit") openJobModal(t.dataset.id);
  else if(act === "job-close") closeJobModal();
  else if(act === "job-save") saveJob();
  else if(act === "job-del") deleteJob();
  else if(act === "job-reset") resetJob();
  else if(act === "confirm-yes") closeConfirm(true);
  else if(act === "confirm-no") closeConfirm(false);
  else if(act === "remote-new") openRemoteWizard(null);
  else if(act === "remote-edit") openRemoteWizard(t.dataset.name);
  else if(act === "remote-del") deleteRemote(t.dataset.name);
  else if(act === "remote-test") testSavedRemote(t.dataset.name);
  else if(act === "remote-close") closeRemoteWizard();
  else if(act === "wiz-pick"){
    wiz.type = t.dataset.type;
    wiz.provider = providerByName(wiz.type);
    wiz.manual = !wiz.provider;
    wiz.values = {};
    wiz.result = null;
    wiz.step = 2;
    renderWizard();
    if($("wizName")) $("wizName").focus();
  }
  else if(act === "wiz-back"){
    wiz.step = 1;
    wiz.result = null;
    wiz.manual = false;
    renderWizard();
  }
  else if(act === "wiz-manual"){
    wiz.manual = true;
    wiz.step = 2;
    wiz.result = null;
    renderWizard();
  }
  else if(act === "wiz-add-row"){ wiz.rows.push({ k: "", v: "" }); renderWizard(); }
  else if(act === "wiz-del-row"){
    wiz.rows.splice(Number(t.dataset.row), 1);
    if(!wiz.rows.length) wiz.rows = [{ k: "", v: "" }];
    renderWizard();
  }
  else if(act === "wiz-test") wizTest();
  else if(act === "wiz-save") wizSave();
});

// Close a modal by clicking its backdrop, but not by clicking inside the panel.
Array.prototype.forEach.call(document.querySelectorAll(".overlay"), function(o){
  o.addEventListener("mousedown", function(ev){
    if(ev.target !== o) return;
    if(o.id === "confirmModal") closeConfirm(false);
    else if(o.id === "jobModal") closeJobModal();
    else closeRemoteWizard();
  });
});

// Escape dismisses the topmost layer only, so it cannot close the editor
// underneath a confirmation that is still waiting for an answer.
document.addEventListener("keydown", function(ev){
  if(ev.key !== "Escape") return;
  if(!$("confirmModal").hidden) closeConfirm(false);
  else if(!$("remoteModal").hidden) closeRemoteWizard();
  else if(!$("jobModal").hidden) closeJobModal();
});

$("defRtEnabled").addEventListener("change", function(){
  $("rtFields").hidden = !this.checked;
});

// The wizard body is re-rendered constantly, so bind to the container once.
$("remoteBody").addEventListener("input", function(ev){
  var t = ev.target;
  if(!wiz) return;
  if(t.id === "wizSearch"){ wiz.search = t.value; renderProviderList(); return; }
  if(t.id === "wizName"){ wiz.name = t.value; return; }
  if(t.id === "wizType"){ wiz.type = t.value; return; }
  if(t.dataset.row != null){
    var row = wiz.rows[Number(t.dataset.row)];
    if(row) row[t.dataset.part] = t.value;
    return;
  }
  if(t.dataset.opt) wiz.values[t.dataset.opt] = t.dataset.kind === "bool" ? (t.checked ? "true" : "false") : t.value;
});

// Changing the sub-provider reveals a different set of options, so redraw.
$("remoteBody").addEventListener("change", function(ev){
  var t = ev.target;
  if(!wiz || !t.dataset) return;
  if(t.dataset.kind === "bool" && t.dataset.opt){
    wiz.values[t.dataset.opt] = t.checked ? "true" : "false";
  }
  if(t.dataset.opt === "provider") renderWizard();
});

/* Boot --------------------------------------------------------------------- */
refresh();
refreshRemotes(false);
setInterval(refresh, 2000);
</script>
</body>
</html>`

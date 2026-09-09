package api

const uiHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Copyarr</title>
<style>
:root{
  color-scheme:light dark;font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
  --bg:#f5f7fb;--panel:#fff;--panel-2:#f8fafc;--text:#172033;--muted:#667085;--line:#e3e8ef;
  --accent:#4f7cff;--accent-soft:#edf3ff;--good:#1c9b62;--good-soft:#eaf8f1;--warn:#c47a12;--warn-soft:#fff6e7;
  --bad:#d34b4b;--bad-soft:#fff0f0;--paused:#7a5cc7;--paused-soft:#f3efff;--shadow:0 8px 30px rgba(27,39,67,.06)
}
@media(prefers-color-scheme:dark){:root{--bg:#0f141b;--panel:#171d26;--panel-2:#1d2530;--text:#edf2f7;--muted:#96a2b4;--line:#2a3442;--accent:#7ca0ff;--accent-soft:#202c49;--good:#58d59b;--good-soft:#15342a;--warn:#f0b04f;--warn-soft:#3a2b16;--bad:#ff8585;--bad-soft:#3a2020;--paused:#b49cff;--paused-soft:#2b2540;--shadow:none}}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font-size:14px}main{max-width:1240px;margin:auto;padding:26px}
button,input,select,textarea{font:inherit}button{border:1px solid var(--line);border-radius:9px;padding:8px 12px;background:var(--panel-2);color:var(--text);cursor:pointer}button:hover{filter:brightness(1.04)}button.primary{background:var(--accent);border-color:var(--accent);color:#fff}button.danger{color:var(--bad)}
header{display:flex;justify-content:space-between;gap:18px;align-items:center;margin-bottom:16px}.brand{display:flex;align-items:center;gap:12px}.logo{width:38px;height:38px;border-radius:11px;background:linear-gradient(135deg,var(--accent),color-mix(in srgb,var(--accent) 55%,#8f61ff));display:grid;place-items:center;color:#fff;font-weight:800;font-size:18px;box-shadow:var(--shadow)}
h1{font-size:24px;line-height:1;margin:0 0 5px}h2{font-size:15px;margin:0}h3{font-size:14px;margin:0}.muted{color:var(--muted);font-size:12.5px}
.header-actions{display:flex;gap:8px;align-items:center}.live{display:inline-flex;align-items:center;gap:7px;font-size:12px;color:var(--muted);padding:7px 10px;border:1px solid var(--line);border-radius:999px;background:var(--panel)}
.dot{width:7px;height:7px;border-radius:50%;background:var(--good);box-shadow:0 0 0 4px color-mix(in srgb,var(--good) 15%,transparent)}.dot.scan{background:var(--warn)}
.tabs{display:flex;gap:6px;margin-bottom:16px;padding:5px;background:var(--panel);border:1px solid var(--line);border-radius:12px;width:max-content;max-width:100%;overflow:auto}.tab{border:0;background:transparent;color:var(--muted);white-space:nowrap}.tab.active{background:var(--accent-soft);color:var(--accent)}
.page{display:none}.page.active{display:block}.grid{display:grid;grid-template-columns:1.45fr .8fr;gap:14px;margin-bottom:14px}.card{background:var(--panel);border:1px solid var(--line);border-radius:14px;box-shadow:var(--shadow)}.card-pad{padding:18px}.card-head{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:14px}
.transfer-empty{min-height:112px;display:flex;align-items:center;justify-content:center;border:1px dashed var(--line);border-radius:11px;background:var(--panel-2);color:var(--muted)}.transfer-name{font-weight:700;font-size:15px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.transfer-meta{display:flex;flex-wrap:wrap;gap:8px 14px;margin-top:6px;color:var(--muted);font-size:12px}
.progress{height:12px;background:var(--panel-2);border-radius:999px;overflow:hidden;margin:14px 0 10px;border:1px solid var(--line)}.progress>div{height:100%;background:linear-gradient(90deg,var(--accent),color-mix(in srgb,var(--accent) 55%,#8f61ff));border-radius:999px;transition:width .35s}
.metrics{display:grid;grid-template-columns:repeat(3,1fr);gap:10px}.metric{background:var(--panel-2);border:1px solid var(--line);border-radius:11px;padding:13px}.metric-value{font-size:24px;font-weight:750}.metric-label{color:var(--muted);font-size:11.5px;margin-top:5px}
.section{margin-bottom:14px}.table-wrap{overflow:auto}table{width:100%;border-collapse:separate;border-spacing:0;font-size:12.5px}th{text-align:left;padding:10px;color:var(--muted);font-weight:600;border-bottom:1px solid var(--line);white-space:nowrap}td{padding:10px;border-bottom:1px solid var(--line);vertical-align:middle}tbody tr:hover{background:color-mix(in srgb,var(--accent) 4%,transparent)}tbody tr:last-child td{border-bottom:0}
.job-name{font-weight:550;max-width:490px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}.badge{display:inline-flex;align-items:center;border-radius:999px;padding:4px 8px;font-size:11px;font-weight:700;text-transform:capitalize}.badge.done{color:var(--good);background:var(--good-soft)}.badge.copying,.badge.queued{color:var(--accent);background:var(--accent-soft)}.badge.retry_wait{color:var(--warn);background:var(--warn-soft)}.badge.failed,.badge.cancelled{color:var(--bad);background:var(--bad-soft)}.badge.paused{color:var(--paused);background:var(--paused-soft)}.badge.other{color:var(--muted);background:var(--panel-2)}
.filters,.actions{display:flex;gap:5px;flex-wrap:wrap}.filter{padding:6px 9px;font-size:11.5px}.filter.active{background:var(--accent-soft);color:var(--accent)}.actions button{padding:5px 8px;font-size:11px}.err{max-width:330px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;color:var(--muted)}
.stack{display:grid;gap:10px}.def-card,.remote-card{padding:14px;border:1px solid var(--line);border-radius:12px;background:var(--panel-2)}.def-top,.remote-top{display:flex;justify-content:space-between;gap:12px;align-items:flex-start}.def-meta{display:flex;gap:8px 14px;flex-wrap:wrap;color:var(--muted);font-size:12px;margin-top:7px}
.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:11px}.form-grid .wide{grid-column:1/-1}label{display:grid;gap:5px;color:var(--muted);font-size:11.5px}input,select,textarea{width:100%;padding:9px 10px;border-radius:9px;border:1px solid var(--line);background:var(--panel);color:var(--text);outline:none}textarea{min-height:120px;font-family:ui-monospace,SFMono-Regular,Menlo,monospace;font-size:12px;resize:vertical}input:focus,select:focus,textarea:focus{border-color:var(--accent);box-shadow:0 0 0 3px color-mix(in srgb,var(--accent) 14%,transparent)}
.form-actions{display:flex;gap:8px;justify-content:flex-end;margin-top:13px}.toggle-row{display:flex;gap:8px;align-items:center}.toggle-row input{width:auto}.callout{padding:11px 12px;border-radius:10px;background:var(--accent-soft);color:var(--muted);font-size:12px;margin-bottom:12px}.split{display:grid;grid-template-columns:1fr 1fr;gap:14px}
#toast{position:fixed;right:20px;bottom:20px;padding:10px 14px;border-radius:10px;background:var(--text);color:var(--panel);display:none;box-shadow:var(--shadow);z-index:20}
@media(max-width:850px){main{padding:16px}.grid,.split{grid-template-columns:1fr}.form-grid{grid-template-columns:1fr}.form-grid .wide{grid-column:auto}}
@media(max-width:620px){header{align-items:flex-start;flex-direction:column}.header-actions{width:100%;justify-content:space-between}.metrics{grid-template-columns:repeat(3,1fr)}.metric{padding:10px}.metric-value{font-size:20px}.hide-sm{display:none}.job-name{max-width:250px}}
</style>
</head>
<body>
<main>
<header>
  <div class="brand"><div class="logo">C</div><div><h1>Copyarr</h1><div class="muted" id="build">loading…</div></div></div>
  <div class="header-actions"><div class="live"><span class="dot" id="liveDot"></span><span id="liveText">Connecting…</span></div><button class="primary" onclick="triggerScan()">Scan now</button></div>
</header>
<nav class="tabs">
  <button class="tab active" onclick="showPage('dashboard',this)">Dashboard</button>
  <button class="tab" onclick="showPage('definitions',this)">Jobs</button>
  <button class="tab" onclick="showPage('remotes',this)">Remotes</button>
</nav>

<section id="page-dashboard" class="page active">
  <div class="grid">
    <section class="card card-pad"><div class="card-head"><h2>Transfer</h2><span class="muted" id="transferHint">Idle</span></div><div id="active" class="transfer-empty">No active transfer</div></section>
    <section class="card card-pad"><div class="card-head"><h2>Overview</h2><span class="muted" id="lastScan">—</span></div><div class="metrics"><div class="metric"><div class="metric-value" id="queueJobs">0</div><div class="metric-label">Queued</div></div><div class="metric"><div class="metric-value" id="doneJobs">0</div><div class="metric-label">Done</div></div><div class="metric"><div class="metric-value" id="failedJobs">0</div><div class="metric-label">Needs attention</div></div></div></section>
  </div>
  <section class="card card-pad"><div class="card-head"><div><h2>Recent executions</h2><div class="muted" id="jobCount">—</div></div><div class="filters"><button class="filter active" onclick="setFilter('all',this)">All</button><button class="filter" onclick="setFilter('active',this)">Active</button><button class="filter" onclick="setFilter('attention',this)">Attention</button><button class="filter" onclick="setFilter('done',this)">Done</button></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>Name</th><th>State</th><th>Attempt</th><th class="hide-sm">Size</th><th class="hide-sm">Next retry / error</th><th>Actions</th></tr></thead><tbody id="jobs"></tbody></table></div></section>
</section>

<section id="page-definitions" class="page">
  <div class="split">
    <section class="card card-pad">
      <div class="card-head"><div><h2>Jobs</h2><div class="muted">Persistent transfer definitions</div></div><button class="primary" onclick="newDefinition()">New job</button></div>
      <div id="definitions" class="stack"><span class="muted">Loading…</span></div>
    </section>
    <section class="card card-pad">
      <div class="card-head"><div><h2 id="defTitle">New job</h2><div class="muted">Each job has its own transfer policy</div></div></div>
      <div class="form-grid">
        <label>ID<input id="defId" placeholder="photos-to-nas"></label>
        <label>Name<input id="defName" placeholder="Photos → NAS"></label>
        <label>Source remote<select id="defSrcRemote"></select></label>
        <label>Source path<input id="defSrcPath" placeholder="/incoming"></label>
        <label>Destination remote<select id="defDstRemote"></select></label>
        <label>Destination path<input id="defDstPath" placeholder="/downloads"></label>
        <label>Mode<select id="defMode"><option value="copy">Copy</option><option value="move">Move</option></select></label>
        <label>Verification<select id="defVerify"><option value="size">Size</option><option value="none">None</option></select></label>
        <label>Stability wait (seconds)<input id="defStability" type="number" min="0" value="600"></label>
        <label>Cleanup after days<input id="defCleanup" type="number" min="0" value="0"></label>
        <label>Multi-thread streams<input id="defStreams" type="number" min="1" value="4"></label>
        <label>Multi-thread cutoff<input id="defCutoff" value="256M"></label>
        <label>Automatic retries<input id="defRetries" type="number" min="0" value="3"></label>
        <label>Retry wait (seconds)<input id="defRetryWait" type="number" min="0" value="300"></label>
        <label class="wide"><span class="toggle-row"><input id="defEnabled" type="checkbox" checked> Enabled</span></label>
      </div>
      <div class="callout" style="margin-top:12px">rTorrent-specific settings on existing jobs are preserved when you edit them here. A dedicated advanced section comes next.</div>
      <div class="form-actions"><button onclick="newDefinition()">Clear</button><button class="primary" onclick="saveDefinition()">Save job</button></div>
    </section>
  </div>
</section>

<section id="page-remotes" class="page">
  <div class="split">
    <section class="card card-pad">
      <div class="card-head"><div><h2>rclone remotes</h2><div class="muted">Stored in Copyarr's writable data directory</div></div><button onclick="refreshRemotes()">Refresh</button></div>
      <div id="remotes" class="stack"><span class="muted">Loading…</span></div>
    </section>
    <section class="card card-pad">
      <div class="card-head"><div><h2>Add remote</h2><div class="muted">Simple backends work now; OAuth wizard follows</div></div></div>
      <div class="callout">Copyarr does not expose rclone's RC server. Remote changes are performed internally against the dedicated rclone config.</div>
      <div class="form-grid">
        <label>Name<input id="remoteName" placeholder="seedbox2"></label>
        <label>Type<select id="remoteType"><option>ftp</option><option>sftp</option><option>webdav</option><option>s3</option><option>drive</option><option>onedrive</option><option>local</option></select></label>
        <label class="wide">Parameters (JSON object)<textarea id="remoteParams" spellcheck="false">{}</textarea></label>
      </div>
      <div class="muted" style="margin-top:8px">Example FTP: {"host":"server.example","user":"martin","pass":"secret","port":"21"}</div>
      <div class="form-actions"><button class="primary" onclick="createRemote()">Create remote</button></div>
      <div id="remoteQuestion" class="callout" style="display:none;margin-top:12px"></div>
    </section>
  </div>
</section>
</main>
<div id="toast"></div>

<script>
const $=id=>document.getElementById(id);
const esc=s=>String(s??'').replace(/[&<>"']/g,m=>({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[m]));
const bytes=n=>{if(!n)return '0 B';const u=['B','KB','MB','GB','TB'];let i=0;while(n>=1000&&i<u.length-1){n/=1000;i++}return n.toFixed(i?1:0)+' '+u[i]};
const dur=s=>{if(s==null)return '—';s=Math.max(0,Math.round(s));return s<60?s+'s':s<3600?Math.floor(s/60)+'m '+s%60+'s':Math.floor(s/3600)+'h '+Math.floor(s%3600/60)+'m'};
const rel=t=>{if(!t)return '—';const d=Math.max(0,Date.now()-new Date(t).getTime()),s=Math.round(d/1000);return s<60?s+'s ago':s<3600?Math.floor(s/60)+'m ago':Math.floor(s/3600)+'h ago'};
let latestJobs=[],jobFilter='all',definitions=[],remoteList=[],editingDefinition=null;
function toast(t){const x=$('toast');x.textContent=t;x.style.display='block';setTimeout(()=>x.style.display='none',2500)}
async function api(url,opt){const r=await fetch(url,opt);if(!r.ok)throw new Error((await r.text()).trim()||r.statusText);const ct=r.headers.get('content-type')||'';return ct.includes('json')?r.json():r.text()}
function showPage(name,el){document.querySelectorAll('.page').forEach(x=>x.classList.remove('active'));document.querySelectorAll('.tab').forEach(x=>x.classList.remove('active'));$('page-'+name).classList.add('active');el.classList.add('active');if(name==='definitions')refreshDefinitions();if(name==='remotes')refreshRemotes()}
async function action(id,a){try{await api('/api/jobs/'+id+'/'+a,{method:'POST'});toast(a+' requested');refresh()}catch(e){toast(e.message)}}
async function triggerScan(){try{await api('/api/scan',{method:'POST'});toast('Scan queued');refreshStatus()}catch(e){toast(e.message)}}
function setFilter(f,el){jobFilter=f;document.querySelectorAll('.filter').forEach(x=>x.classList.remove('active'));el.classList.add('active');renderJobs()}
function badge(state){const known=['done','copying','queued','retry_wait','failed','cancelled','paused'];const cls=known.includes(state)?state:'other';return '<span class="badge '+cls+'">'+esc(state.replace('_',' '))+'</span>'}
function buttons(j){let a=[];if(j.state==='failed'||j.state==='retry_wait')a.push('retry');if(j.state==='paused')a.push('resume','cancel');if(j.state==='queued'||j.state==='copying')a.push('pause','cancel');return a.map(x=>'<button onclick="action('+j.id+',\''+x+'\')">'+x+'</button>').join('')}
function visibleJob(j){if(jobFilter==='done')return j.state==='done';if(jobFilter==='attention')return ['failed','retry_wait','cancelled'].includes(j.state);if(jobFilter==='active')return ['copying','queued','paused'].includes(j.state);return true}
function renderJobs(){const js=latestJobs.filter(visibleJob);$('jobCount').textContent=js.length+' shown · '+latestJobs.length+' loaded';$('jobs').innerHTML=js.length?js.map(j=>'<tr><td class="muted">'+j.id+'</td><td><div class="job-name" title="'+esc(j.display_name)+'">'+esc(j.display_name)+'</div></td><td>'+badge(j.state)+'</td><td>'+(j.attempt_number||j.attempts||0)+'/'+(j.max_attempts||'?')+'</td><td class="hide-sm">'+bytes(j.total_bytes)+'</td><td class="hide-sm err" title="'+esc(j.last_error||'')+'">'+esc(j.next_retry_at?('Retry '+new Date(j.next_retry_at).toLocaleTimeString()):(j.last_error||'—'))+'</td><td><div class="actions">'+buttons(j)+'</div></td></tr>').join(''):'<tr><td colspan="7" class="muted" style="text-align:center;padding:24px">No jobs in this view</td></tr>'}
async function refreshStatus(){try{const s=await api('/api/status');$('build').textContent=(s.version||'dev')+' · '+(s.revision||'unknown').slice(0,12);$('queueJobs').textContent=s.queue?.jobs||0;$('doneJobs').textContent=s.jobs?.done||0;$('failedJobs').textContent=(s.jobs?.failed||0)+(s.jobs?.retry_wait||0)+(s.jobs?.cancelled||0);$('liveText').textContent=s.scanning?'Scanning':'Online';$('liveDot').className='dot'+(s.scanning?' scan':'');const scans=Object.values(s.rule_scans||{}),last=scans.map(x=>x.last_scan_completed_at).filter(Boolean).sort().pop();$('lastScan').textContent=last?'Last scan '+rel(last):'No scans yet';if(!s.active){$('active').className='transfer-empty';$('active').innerHTML='No active transfer';$('transferHint').textContent='Idle';return}const a=s.active,p=Math.max(0,Math.min(100,a.progress_percent||0));$('active').className='';$('transferHint').textContent=a.phase||'transferring';$('active').innerHTML='<div class="transfer-name">'+esc(a.name)+'</div><div class="transfer-meta"><span>'+esc(a.phase)+'</span><span>Attempt '+(a.attempt_number||'?')+'/'+(a.max_attempts||'?')+'</span><span>'+bytes(a.transferred_bytes)+' / '+bytes(a.total_bytes)+'</span></div><div class="progress"><div style="width:'+p+'%"></div></div><div class="transfer-meta"><strong style="color:var(--text)">'+p.toFixed(1)+'%</strong><span>'+bytes(a.speed_bps)+'/s</span><span>ETA '+dur(a.eta_seconds)+'</span><span>'+(a.multi_thread?'Multi-thread':'Single-thread')+'</span></div>'}catch(e){$('build').textContent='API error';$('liveText').textContent='Offline';$('liveDot').style.background='var(--bad)'}}
async function refreshJobs(){try{latestJobs=await api('/api/jobs?limit=60');renderJobs()}catch(e){toast(e.message)}}

function fillRemoteSelects(){const opts=['<option value="">Local filesystem</option>'].concat(remoteList.map(r=>'<option value="'+esc(r.name)+'">'+esc(r.name)+(r.type?' · '+esc(r.type):'')+'</option>')).join('');$('defSrcRemote').innerHTML=opts;$('defDstRemote').innerHTML=opts}
async function refreshDefinitions(){try{definitions=await api('/api/job-definitions');await refreshRemotes(false);$('definitions').innerHTML=definitions.map(d=>'<div class="def-card"><div class="def-top"><div><strong>'+esc(d.name||d.id)+'</strong><div class="def-meta"><span>'+esc((d.source?.remote||'local')+':'+(d.source?.path||''))+'</span><span>→</span><span>'+esc((d.destination?.remote||'local')+':'+(d.destination?.path||''))+'</span><span>'+esc(d.mode||'copy')+'</span></div></div><div class="actions"><button data-id="'+esc(d.id)+'" onclick="editDefinition(this.dataset.id)">Edit</button></div></div></div>').join('')||'<span class="muted">No jobs yet</span>';fillRemoteSelects()}catch(e){toast(e.message)}}
function newDefinition(){editingDefinition=null;$('defTitle').textContent='New job';$('defId').disabled=false;$('defId').value='';$('defName').value='';$('defSrcRemote').value='';$('defSrcPath').value='';$('defDstRemote').value='';$('defDstPath').value='/downloads';$('defMode').value='copy';$('defVerify').value='size';$('defStability').value=600;$('defCleanup').value=0;$('defStreams').value=4;$('defCutoff').value='256M';$('defRetries').value=3;$('defRetryWait').value=300;$('defEnabled').checked=true}
function editDefinition(id){const d=definitions.find(x=>x.id===id);if(!d)return;editingDefinition=structuredClone(d);$('defTitle').textContent='Edit '+(d.name||d.id);$('defId').disabled=true;$('defId').value=d.id;$('defName').value=d.name||d.id;$('defSrcRemote').value=d.source?.remote||'';$('defSrcPath').value=d.source?.path||'';$('defDstRemote').value=d.destination?.remote||'';$('defDstPath').value=d.destination?.path||'';$('defMode').value=d.mode||'copy';$('defVerify').value=d.verification||'size';$('defStability').value=d.stability_seconds||600;$('defCleanup').value=d.cleanup_days||0;$('defStreams').value=d.multi_thread_streams||4;$('defCutoff').value=d.multi_thread_cutoff||'256M';$('defRetries').value=d.retry_count??3;$('defRetryWait').value=d.retry_wait_seconds??300;$('defEnabled').checked=!!d.enabled;window.scrollTo({top:0,behavior:'smooth'})}
async function saveDefinition(){try{const id=$('defId').value.trim();if(!id)throw new Error('Job ID is required');const d=editingDefinition||{};Object.assign(d,{id,name:$('defName').value.trim()||id,enabled:$('defEnabled').checked,source:{remote:$('defSrcRemote').value,path:$('defSrcPath').value.trim()},destination:{remote:$('defDstRemote').value,path:$('defDstPath').value.trim()},mode:$('defMode').value,verification:$('defVerify').value,stability_seconds:Number($('defStability').value),cleanup_days:Number($('defCleanup').value),multi_thread_streams:Number($('defStreams').value),multi_thread_cutoff:$('defCutoff').value.trim()||'256M',retry_count:Number($('defRetries').value),retry_wait_seconds:Number($('defRetryWait').value),initial_behavior:d.initial_behavior||'ignore_existing',rclone_args:d.rclone_args||[]});const method=editingDefinition?'PUT':'POST',url=editingDefinition?'/api/job-definitions/'+encodeURIComponent(id):'/api/job-definitions';await api(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(d)});toast('Job saved');await refreshDefinitions();editDefinition(id)}catch(e){toast(e.message)}}

async function refreshRemotes(render=true){try{remoteList=await api('/api/remotes');fillRemoteSelects();if(render)$('remotes').innerHTML=remoteList.map(r=>'<div class="remote-card"><div class="remote-top"><div><strong>'+esc(r.name)+'</strong><div class="muted">'+esc(r.type||'unknown type')+'</div></div><button class="danger" data-name="'+esc(r.name)+'" onclick="deleteRemote(this.dataset.name)">Delete</button></div></div>').join('')||'<span class="muted">No rclone remotes configured</span>'}catch(e){toast(e.message)}}
async function createRemote(){try{const name=$('remoteName').value.trim(),type=$('remoteType').value;if(!name)throw new Error('Remote name is required');let parameters;try{parameters=JSON.parse($('remoteParams').value||'{}')}catch{throw new Error('Parameters must be valid JSON')}const result=await api('/api/remotes',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name,type,parameters})});if(result.question){$('remoteQuestion').style.display='block';$('remoteQuestion').textContent='This provider needs an interactive/OAuth step. Wizard support is next.'}else{$('remoteQuestion').style.display='none';toast('Remote created');$('remoteName').value='';$('remoteParams').value='{}';refreshRemotes()}}catch(e){toast(e.message)}}
async function deleteRemote(name){if(!confirm('Delete rclone remote "'+name+'"?'))return;try{await api('/api/remotes/'+encodeURIComponent(name),{method:'DELETE'});toast('Remote deleted');refreshRemotes()}catch(e){toast(e.message)}}

async function refresh(){await Promise.all([refreshStatus(),refreshJobs()])}
refresh();refreshDefinitions();setInterval(refresh,2000);
</script>
</body>
</html>`

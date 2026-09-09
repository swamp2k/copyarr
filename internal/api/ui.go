package api

const uiHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Copyarr</title>
<style>
:root{
  color-scheme:light dark;
  font-family:Inter,ui-sans-serif,system-ui,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;
  --bg:#f5f7fb;--panel:#fff;--panel-2:#f8fafc;--text:#172033;--muted:#667085;
  --line:#e3e8ef;--accent:#4f7cff;--accent-soft:#edf3ff;--good:#1c9b62;--good-soft:#eaf8f1;
  --warn:#c47a12;--warn-soft:#fff6e7;--bad:#d34b4b;--bad-soft:#fff0f0;--paused:#7a5cc7;--paused-soft:#f3efff;
  --shadow:0 8px 30px rgba(27,39,67,.06);
}
@media(prefers-color-scheme:dark){
  :root{
    --bg:#0f141b;--panel:#171d26;--panel-2:#1d2530;--text:#edf2f7;--muted:#96a2b4;
    --line:#2a3442;--accent:#7ca0ff;--accent-soft:#202c49;--good:#58d59b;--good-soft:#15342a;
    --warn:#f0b04f;--warn-soft:#3a2b16;--bad:#ff8585;--bad-soft:#3a2020;--paused:#b49cff;--paused-soft:#2b2540;
    --shadow:none;
  }
}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--text);font-size:14px}
main{max-width:1240px;margin:auto;padding:26px}
button,input,select{font:inherit}
button{border:0;border-radius:9px;padding:8px 12px;background:var(--panel-2);color:var(--text);cursor:pointer;border:1px solid var(--line)}
button:hover{filter:brightness(1.04)}
button.primary{background:var(--accent);border-color:var(--accent);color:white}
button.ghost{background:transparent}
header{display:flex;justify-content:space-between;gap:18px;align-items:center;margin-bottom:18px}
.brand{display:flex;align-items:center;gap:12px}
.logo{width:38px;height:38px;border-radius:11px;background:linear-gradient(135deg,var(--accent),color-mix(in srgb,var(--accent) 55%,#8f61ff));display:grid;place-items:center;color:white;font-weight:800;font-size:18px;box-shadow:var(--shadow)}
h1{font-size:24px;line-height:1;margin:0 0 5px}h2{font-size:15px;margin:0}h3{font-size:14px;margin:0}
.muted{color:var(--muted);font-size:12.5px}
.header-actions{display:flex;gap:8px;align-items:center}
.live{display:inline-flex;align-items:center;gap:7px;font-size:12px;color:var(--muted);padding:7px 10px;border:1px solid var(--line);border-radius:999px;background:var(--panel)}
.dot{width:7px;height:7px;border-radius:50%;background:var(--good);box-shadow:0 0 0 4px color-mix(in srgb,var(--good) 15%,transparent)}
.dot.scan{background:var(--warn);box-shadow:0 0 0 4px color-mix(in srgb,var(--warn) 15%,transparent)}
.grid{display:grid;grid-template-columns:1.45fr .8fr;gap:14px;margin-bottom:14px}
.card{background:var(--panel);border:1px solid var(--line);border-radius:14px;box-shadow:var(--shadow)}
.card-pad{padding:18px}
.card-head{display:flex;justify-content:space-between;align-items:center;gap:12px;margin-bottom:14px}
.transfer-empty{min-height:112px;display:flex;align-items:center;justify-content:center;border:1px dashed var(--line);border-radius:11px;background:var(--panel-2);color:var(--muted)}
.transfer-name{font-weight:700;font-size:15px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.transfer-meta{display:flex;flex-wrap:wrap;gap:8px 14px;margin-top:6px;color:var(--muted);font-size:12px}
.progress{height:12px;background:var(--panel-2);border-radius:999px;overflow:hidden;margin:14px 0 10px;border:1px solid var(--line)}
.progress>div{height:100%;background:linear-gradient(90deg,var(--accent),color-mix(in srgb,var(--accent) 55%,#8f61ff));border-radius:999px;transition:width .35s ease}
.metrics{display:grid;grid-template-columns:repeat(3,1fr);gap:10px}
.metric{background:var(--panel-2);border:1px solid var(--line);border-radius:11px;padding:13px}
.metric-value{font-size:24px;font-weight:750;line-height:1.1}
.metric-label{color:var(--muted);font-size:11.5px;margin-top:5px}
.section{margin-bottom:14px}
.rule{padding:14px;border:1px solid var(--line);border-radius:12px;background:var(--panel-2)}
.rule-top{display:flex;align-items:center;justify-content:space-between;gap:12px}
.rule-title{display:flex;align-items:center;gap:8px}
.rule-config{display:grid;grid-template-columns:180px 220px auto;gap:10px;align-items:end;margin-top:13px}
label{display:grid;gap:5px;color:var(--muted);font-size:11.5px}
input,select{width:100%;padding:9px 10px;border-radius:9px;border:1px solid var(--line);background:var(--panel);color:var(--text);outline:none}
input:focus,select:focus{border-color:var(--accent);box-shadow:0 0 0 3px color-mix(in srgb,var(--accent) 14%,transparent)}
.table-wrap{overflow:auto}
table{width:100%;border-collapse:separate;border-spacing:0;font-size:12.5px}
th{text-align:left;padding:10px 10px;color:var(--muted);font-weight:600;border-bottom:1px solid var(--line);white-space:nowrap}
td{padding:10px;border-bottom:1px solid var(--line);vertical-align:middle}
tbody tr:hover{background:color-mix(in srgb,var(--accent) 4%,transparent)}
tbody tr:last-child td{border-bottom:0}
.job-name{font-weight:550;max-width:490px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.badge{display:inline-flex;align-items:center;border-radius:999px;padding:4px 8px;font-size:11px;font-weight:700;text-transform:capitalize}
.badge.done{color:var(--good);background:var(--good-soft)}
.badge.copying,.badge.queued{color:var(--accent);background:var(--accent-soft)}
.badge.retry_wait{color:var(--warn);background:var(--warn-soft)}
.badge.failed,.badge.cancelled{color:var(--bad);background:var(--bad-soft)}
.badge.paused{color:var(--paused);background:var(--paused-soft)}
.badge.other{color:var(--muted);background:var(--panel-2)}
.filters{display:flex;gap:6px;flex-wrap:wrap}
.filter{padding:6px 9px;font-size:11.5px}
.filter.active{background:var(--accent-soft);color:var(--accent);border-color:color-mix(in srgb,var(--accent) 35%,var(--line))}
.actions{display:flex;gap:5px;flex-wrap:wrap}
.actions button{padding:5px 8px;font-size:11px}
.err{max-width:330px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;color:var(--muted)}
.rule-status{display:inline-flex;align-items:center;gap:6px;color:var(--good);font-size:11.5px;font-weight:600}
.rule-status .dot{width:6px;height:6px;box-shadow:none}
#toast{position:fixed;right:20px;bottom:20px;padding:10px 14px;border-radius:10px;background:var(--text);color:var(--panel);display:none;box-shadow:var(--shadow);z-index:10}
@media(max-width:850px){
  main{padding:16px}.grid{grid-template-columns:1fr}.rule-config{grid-template-columns:1fr 1fr}.rule-config button{grid-column:1/-1}
}
@media(max-width:620px){
  header{align-items:flex-start;flex-direction:column}.header-actions{width:100%;justify-content:space-between}
  .metrics{grid-template-columns:repeat(3,1fr)}.metric{padding:10px}.metric-value{font-size:20px}
  .rule-config{grid-template-columns:1fr}.hide-sm{display:none}.job-name{max-width:250px}
}
</style>
</head>
<body>
<main>
<header>
  <div class="brand">
    <div class="logo">C</div>
    <div><h1>Copyarr</h1><div class="muted" id="build">loading…</div></div>
  </div>
  <div class="header-actions">
    <div class="live"><span class="dot" id="liveDot"></span><span id="liveText">Connecting…</span></div>
    <button class="primary" onclick="triggerScan()">Scan now</button>
  </div>
</header>

<div class="grid">
  <section class="card card-pad">
    <div class="card-head"><h2>Transfer</h2><span class="muted" id="transferHint">Idle</span></div>
    <div id="active" class="transfer-empty">No active transfer</div>
  </section>
  <section class="card card-pad">
    <div class="card-head"><h2>Overview</h2><span class="muted" id="lastScan">—</span></div>
    <div class="metrics">
      <div class="metric"><div class="metric-value" id="queueJobs">0</div><div class="metric-label">Queued</div></div>
      <div class="metric"><div class="metric-value" id="doneJobs">0</div><div class="metric-label">Done</div></div>
      <div class="metric"><div class="metric-value" id="failedJobs">0</div><div class="metric-label">Needs attention</div></div>
    </div>
  </section>
</div>

<section class="card card-pad section">
  <div class="card-head"><h2>Rules</h2><span class="muted">Runtime settings are saved in Copyarr</span></div>
  <div id="rules"><span class="muted">Loading…</span></div>
</section>

<section class="card card-pad">
  <div class="card-head">
    <div><h2>Recent jobs</h2><div class="muted" id="jobCount">—</div></div>
    <div class="filters">
      <button class="filter active" data-filter="all" onclick="setFilter('all',this)">All</button>
      <button class="filter" data-filter="active" onclick="setFilter('active',this)">Active</button>
      <button class="filter" data-filter="attention" onclick="setFilter('attention',this)">Attention</button>
      <button class="filter" data-filter="done" onclick="setFilter('done',this)">Done</button>
    </div>
  </div>
  <div class="table-wrap">
    <table>
      <thead><tr><th>ID</th><th>Name</th><th>State</th><th>Attempt</th><th class="hide-sm">Size</th><th class="hide-sm">Next retry / error</th><th>Actions</th></tr></thead>
      <tbody id="jobs"></tbody>
    </table>
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
let latestJobs=[],jobFilter='all';
function toast(t){const x=$('toast');x.textContent=t;x.style.display='block';setTimeout(()=>x.style.display='none',2200)}
async function api(url,opt){const r=await fetch(url,opt);if(!r.ok)throw new Error((await r.text()).trim()||r.statusText);return r.json()}
async function action(id,a){try{await api('/api/jobs/'+id+'/'+a,{method:'POST'});toast(a+' requested');refresh()}catch(e){toast(e.message)}}
async function triggerScan(){try{await api('/api/scan',{method:'POST'});toast('Scan queued');refreshStatus()}catch(e){toast(e.message)}}
async function saveRule(id){
 const count=Number(document.querySelector('[data-count="'+CSS.escape(id)+'"]').value);
 const wait=Number(document.querySelector('[data-wait="'+CSS.escape(id)+'"]').value);
 try{
   await api('/api/rules/'+encodeURIComponent(id),{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({retry_count:count,retry_wait_seconds:wait})});
   toast('Retry policy saved');refreshRules()
 }catch(e){toast(e.message)}
}
function setFilter(f,el){jobFilter=f;document.querySelectorAll('.filter').forEach(x=>x.classList.remove('active'));el.classList.add('active');renderJobs()}
function badge(state){const known=['done','copying','queued','retry_wait','failed','cancelled','paused'];const cls=known.includes(state)?state:'other';return '<span class="badge '+cls+'">'+esc(state.replace('_',' '))+'</span>'}
function buttons(j){
 let a=[];
 if(j.state==='failed'||j.state==='retry_wait')a.push('retry');
 if(j.state==='paused')a.push('resume','cancel');
 if(j.state==='queued'||j.state==='copying')a.push('pause','cancel');
 return a.map(x=>'<button onclick="action('+j.id+',\''+x+'\')">'+x+'</button>').join('');
}
function visibleJob(j){
 if(jobFilter==='done')return j.state==='done';
 if(jobFilter==='attention')return ['failed','retry_wait','cancelled'].includes(j.state);
 if(jobFilter==='active')return ['copying','queued','paused'].includes(j.state);
 return true;
}
function renderJobs(){
 const js=latestJobs.filter(visibleJob);
 $('jobCount').textContent=js.length+' shown · '+latestJobs.length+' loaded';
 $('jobs').innerHTML=js.length?js.map(j=>
   '<tr><td class="muted">'+j.id+'</td>'+
   '<td><div class="job-name" title="'+esc(j.display_name)+'">'+esc(j.display_name)+'</div></td>'+
   '<td>'+badge(j.state)+'</td>'+
   '<td>'+(j.attempt_number||j.attempts||0)+'/'+(j.max_attempts||'?')+'</td>'+
   '<td class="hide-sm">'+bytes(j.total_bytes)+'</td>'+
   '<td class="hide-sm err" title="'+esc(j.last_error||'')+'">'+esc(j.next_retry_at?('Retry '+new Date(j.next_retry_at).toLocaleTimeString()):(j.last_error||'—'))+'</td>'+
   '<td><div class="actions">'+buttons(j)+'</div></td></tr>'
 ).join(''):'<tr><td colspan="7" class="muted" style="text-align:center;padding:24px">No jobs in this view</td></tr>';
}
async function refreshStatus(){
 try{
  const s=await api('/api/status');
  $('build').textContent=(s.version||'dev')+' · '+(s.revision||'unknown').slice(0,12);
  $('queueJobs').textContent=s.queue?.jobs||0;
  $('doneJobs').textContent=s.jobs?.done||0;
  $('failedJobs').textContent=(s.jobs?.failed||0)+(s.jobs?.retry_wait||0)+(s.jobs?.cancelled||0);
  $('liveText').textContent=s.scanning?'Scanning':'Online';
  $('liveDot').className='dot'+(s.scanning?' scan':'');
  const scans=Object.values(s.rule_scans||{}),last=scans.map(x=>x.last_scan_completed_at).filter(Boolean).sort().pop();
  $('lastScan').textContent=last?'Last scan '+rel(last):'No scans yet';

  if(!s.active){
    $('active').className='transfer-empty';
    $('active').innerHTML='No active transfer';
    $('transferHint').textContent='Idle';
    return;
  }
  const a=s.active,p=Math.max(0,Math.min(100,a.progress_percent||0));
  $('active').className='';
  $('transferHint').textContent=esc(a.phase||'transferring');
  $('active').innerHTML=
    '<div class="transfer-name" title="'+esc(a.name)+'">'+esc(a.name)+'</div>'+
    '<div class="transfer-meta"><span>'+esc(a.phase)+'</span><span>Attempt '+(a.attempt_number||'?')+'/'+(a.max_attempts||'?')+'</span><span>'+bytes(a.transferred_bytes)+' / '+bytes(a.total_bytes)+'</span></div>'+
    '<div class="progress"><div style="width:'+p+'%"></div></div>'+
    '<div class="transfer-meta"><strong style="color:var(--text)">'+p.toFixed(1)+'%</strong><span>'+bytes(a.speed_bps)+'/s</span><span>ETA '+dur(a.eta_seconds)+'</span><span>'+((a.multi_thread)?'Multi-thread':'Single-thread')+'</span></div>';
 }catch(e){
   $('build').textContent='API error';
   $('liveText').textContent='Offline';
   $('liveDot').style.background='var(--bad)';
 }
}
async function refreshJobs(){
 try{latestJobs=await api('/api/jobs?limit=60');renderJobs()}catch(e){toast(e.message)}
}
async function refreshRules(){
 try{
  const rs=await api('/api/rules');
  $('rules').innerHTML=rs.map(r=>{
    const wait=Number(r.retry_wait_seconds??300);
    return '<div class="rule">'+
      '<div class="rule-top"><div class="rule-title"><strong>'+esc(r.name||r.id)+'</strong><span class="muted">'+esc(r.id)+'</span></div><span class="rule-status"><span class="dot"></span>'+(r.enabled?'Enabled':'Disabled')+'</span></div>'+
      '<div class="rule-config">'+
        '<label>Automatic retries<input type="number" min="0" data-count="'+esc(r.id)+'" value="'+Number(r.retry_count??3)+'"></label>'+
        '<label>Retry wait<input type="number" min="0" data-wait="'+esc(r.id)+'" value="'+wait+'"><span class="muted">'+dur(wait)+'</span></label>'+
        '<button class="primary" data-rule="'+esc(r.id)+'" onclick="saveRule(this.dataset.rule)">Save changes</button>'+
      '</div></div>'
  }).join('');
 }catch(e){$('rules').textContent=e.message}
}
async function refresh(){await Promise.all([refreshStatus(),refreshJobs()])}
refreshRules();refresh();setInterval(refresh,2000);setInterval(refreshRules,30000);
</script>
</body>
</html>`

package api

const uiHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Copyarr</title>
<style>
:root{color-scheme:light dark;font-family:Inter,system-ui,sans-serif}
body{margin:0;background:Canvas;color:CanvasText}
main{max-width:1180px;margin:auto;padding:24px}
header{display:flex;justify-content:space-between;gap:16px;align-items:baseline;margin-bottom:20px}
h1{margin:0;font-size:28px} h2{font-size:17px;margin:0 0 12px}
.muted{opacity:.65;font-size:13px}
.grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(310px,1fr));gap:14px;margin-bottom:14px}
.card{border:1px solid color-mix(in srgb,CanvasText 18%,transparent);border-radius:12px;padding:16px;background:color-mix(in srgb,Canvas 94%,CanvasText 6%)}
.row{display:flex;gap:12px;align-items:center;flex-wrap:wrap}
.stat{font-size:24px;font-weight:700}
.progress{height:10px;background:color-mix(in srgb,CanvasText 13%,transparent);border-radius:999px;overflow:hidden;margin:10px 0}
.progress>div{height:100%;background:AccentColor;border-radius:inherit;transition:width .3s}
label{display:grid;gap:5px;font-size:13px;flex:1;min-width:130px}
input{padding:8px 10px;border-radius:8px;border:1px solid color-mix(in srgb,CanvasText 25%,transparent);background:Canvas;color:CanvasText}
button{padding:7px 10px;border-radius:8px;border:1px solid color-mix(in srgb,CanvasText 25%,transparent);background:ButtonFace;color:ButtonText;cursor:pointer}
button:hover{filter:brightness(1.05)}
table{width:100%;border-collapse:collapse;font-size:13px}
th,td{text-align:left;padding:9px 8px;border-bottom:1px solid color-mix(in srgb,CanvasText 12%,transparent);vertical-align:top}
th{opacity:.7;font-weight:600}
.state{font-weight:700}.err{max-width:400px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis;opacity:.75}
.actions{display:flex;gap:5px;flex-wrap:wrap}
#toast{position:fixed;right:18px;bottom:18px;padding:10px 14px;border-radius:9px;background:CanvasText;color:Canvas;display:none}
@media(max-width:720px){main{padding:14px}table{display:block;overflow-x:auto}.hide-sm{display:none}}
</style>
</head>
<body>
<main>
<header><div><h1>Copyarr</h1><div class="muted" id="build">loading…</div></div><button onclick="triggerScan()">Scan now</button></header>

<div class="grid">
<section class="card">
<h2>Transfer</h2>
<div id="active"><span class="muted">No active transfer</span></div>
</section>
<section class="card">
<h2>Queue</h2>
<div class="row"><div><div class="stat" id="queueJobs">0</div><div class="muted">queued jobs</div></div><div><div class="stat" id="doneJobs">0</div><div class="muted">done</div></div><div><div class="stat" id="failedJobs">0</div><div class="muted">failed</div></div></div>
</section>
</div>

<section class="card" style="margin-bottom:14px">
<h2>Rules</h2>
<div id="rules"><span class="muted">Loading…</span></div>
</section>

<section class="card">
<h2>Recent jobs</h2>
<div style="overflow-x:auto">
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
function toast(t){const x=$('toast');x.textContent=t;x.style.display='block';setTimeout(()=>x.style.display='none',2200)}
async function api(url,opt){const r=await fetch(url,opt);if(!r.ok)throw new Error(await r.text());return r.json()}
async function action(id,a){try{await api('/api/jobs/'+id+'/'+a,{method:'POST'});toast(a+' requested');refresh()}catch(e){toast(e.message)}}
async function triggerScan(){try{await api('/api/scan',{method:'POST'});toast('Scan queued')}catch(e){toast(e.message)}}
async function saveRule(id){
 const count=Number(document.querySelector('[data-count="'+CSS.escape(id)+'"]').value);
 const wait=Number(document.querySelector('[data-wait="'+CSS.escape(id)+'"]').value);
 try{await api('/api/rules/'+encodeURIComponent(id),{method:'PATCH',headers:{'Content-Type':'application/json'},body:JSON.stringify({retry_count:count,retry_wait_seconds:wait})});toast('Retry policy saved');refreshRules()}catch(e){toast(e.message)}
}
function buttons(j){
 let a=[];
 if(j.state==='failed'||j.state==='retry_wait')a.push('retry');
 if(j.state==='paused')a.push('resume','cancel');
 if(j.state==='queued')a.push('pause','cancel');
 if(j.state==='copying')a.push('pause','cancel');
 return a.map(x=>'<button onclick="action('+j.id+',\''+x+'\')">'+x+'</button>').join('');
}
async function refreshStatus(){
 try{
  const s=await api('/api/status');
  $('build').textContent=(s.version||'dev')+' · '+(s.revision||'unknown').slice(0,12);
  $('queueJobs').textContent=s.queue?.jobs||0;
  $('doneJobs').textContent=s.jobs?.done||0;
  $('failedJobs').textContent=(s.jobs?.failed||0)+(s.jobs?.retry_wait||0);
  if(!s.active){$('active').innerHTML='<span class="muted">No active transfer</span>';return}
  const a=s.active,p=Math.max(0,Math.min(100,a.progress_percent||0));
  $('active').innerHTML='<div><strong>'+esc(a.name)+'</strong></div><div class="muted">'+esc(a.phase)+' · attempt '+(a.attempt_number||'?')+'/'+(a.max_attempts||'?')+'</div><div class="progress"><div style="width:'+p+'%"></div></div><div class="row"><span>'+p.toFixed(1)+'%</span><span>'+bytes(a.speed_bps)+'/s</span><span>ETA '+dur(a.eta_seconds)+'</span></div>';
 }catch(e){$('build').textContent='API error: '+e.message}
}
async function refreshJobs(){
 try{
  const js=await api('/api/jobs?limit=40');
  $('jobs').innerHTML=js.map(j=>'<tr><td>'+j.id+'</td><td>'+esc(j.display_name)+'</td><td class="state">'+esc(j.state)+'</td><td>'+(j.attempt_number||j.attempts||0)+'/'+(j.max_attempts||'?')+'</td><td class="hide-sm">'+bytes(j.total_bytes)+'</td><td class="hide-sm err" title="'+esc(j.last_error||'')+'">'+esc(j.next_retry_at||j.last_error||'')+'</td><td><div class="actions">'+buttons(j)+'</div></td></tr>').join('');
 }catch(e){toast(e.message)}
}
async function refreshRules(){
 try{
  const rs=await api('/api/rules');
  $('rules').innerHTML=rs.map(r=>'<div class="card" style="margin-top:8px"><div class="row" style="justify-content:space-between"><strong>'+esc(r.name||r.id)+'</strong><span class="muted">'+esc(r.id)+'</span></div><div class="row" style="margin-top:12px"><label>Automatic retries<input type="number" min="0" data-count="'+esc(r.id)+'" value="'+Number(r.retry_count??3)+'"></label><label>Retry wait (seconds)<input type="number" min="0" data-wait="'+esc(r.id)+'" value="'+Number(r.retry_wait_seconds??300)+'"></label><button data-rule="'+esc(r.id)+'" onclick="saveRule(this.dataset.rule)">Save</button></div></div>').join('');
 }catch(e){$('rules').textContent=e.message}
}
async function refresh(){await Promise.all([refreshStatus(),refreshJobs()])}
refreshRules();refresh();setInterval(refresh,2000);setInterval(refreshRules,30000);
</script>
</body>
</html>`

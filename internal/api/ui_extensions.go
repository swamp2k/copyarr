package api

import "strings"

// enhanceUI keeps optional dashboard behaviour out of the already-large
// self-contained ui.go. The extra CSS/JS is injected into the page served at
// runtime, so Copyarr still ships as one binary with no frontend build step.
func enhanceUI(html string) string {
	html = strings.Replace(html, "</style>", uiExtensionCSS+"</style>", 1)
	return strings.Replace(html, "</body>", uiExtensionJS+"</body>", 1)
}

const uiExtensionCSS = `
/* Responsive app shell ---------------------------------------------------- */
body{min-height:100vh}
.shell{max-width:1500px;margin:0 0 0 220px;padding:26px 28px 72px}
.tabs{position:fixed;z-index:45;left:0;top:0;bottom:0;width:220px;margin:0;padding:18px 12px;background:color-mix(in srgb,var(--panel) 96%,transparent);border:0;border-right:1px solid var(--line);border-radius:0;display:flex;flex-direction:column;align-items:stretch;gap:5px;overflow:visible;box-shadow:8px 0 30px rgba(0,0,0,.08)}
.tab{width:100%;display:flex;align-items:center;gap:10px;text-align:left;padding:9px 11px;border-radius:9px}
.tab.active{background:color-mix(in srgb,var(--accent) 22%,transparent)}
.nav-icon{width:20px;height:20px;display:grid;place-items:center;font-size:15px;flex:none}
.side-brand{display:flex;align-items:center;gap:10px;padding:2px 6px 18px;margin-bottom:7px;border-bottom:1px solid var(--line)}
.side-brand .logo{width:34px;height:34px;border-radius:10px}
.side-brand strong{font-size:16px}
.side-status{margin-top:auto;border-top:1px solid var(--line);padding:14px 8px 2px;display:grid;gap:6px;color:var(--dim);font-size:11px}
.side-status-line{display:flex;align-items:center;gap:7px}
.side-status .dot{width:7px;height:7px}
.topbar>.brand{display:none}
.mobile-menu-btn,.mobile-drawer,.mobile-drawer-scrim{display:none}

/* rTorrent-style execution table ----------------------------------------- */
th.exec-column{position:relative;user-select:none;-webkit-user-select:none}
th.exec-column[draggable=true]{cursor:grab}
th.exec-column.dragging{opacity:.45}
th.exec-column.drag-over{box-shadow:inset 3px 0 0 var(--accent)}
.exec-head-label{display:inline-flex;align-items:center;gap:5px;border:0;background:transparent;color:inherit;font:inherit;font-weight:inherit;text-transform:inherit;letter-spacing:inherit;padding:0;cursor:pointer;white-space:nowrap}
.exec-head-label:hover{color:var(--text)}
.exec-sort-mark{color:var(--accent-text);font-size:10px}
.exec-column-menu{position:fixed;z-index:120;min-width:220px;max-height:min(520px,75vh);overflow:auto;background:var(--panel);border:1px solid var(--line-strong);border-radius:11px;box-shadow:var(--shadow);padding:6px}
.exec-column-menu-title{font-size:10px;text-transform:uppercase;letter-spacing:.07em;color:var(--dim);padding:7px 9px 5px}
.exec-column-choice{display:flex;align-items:center;gap:9px;width:100%;padding:7px 9px;border-radius:7px;color:var(--muted);font-size:12px;cursor:pointer}
.exec-column-choice:hover{background:var(--panel-2);color:var(--text)}
.exec-column-choice input{width:15px;height:15px;accent-color:var(--accent);flex:none}
.exec-column-hint{border-top:1px solid var(--line);margin-top:5px;padding:8px 9px 5px;color:var(--faint);font-size:10.5px}

/* Existing extensions ---------------------------------------------------- */
.log-follow{display:inline-flex;align-items:center;gap:6px;color:var(--good);font-size:11px}
.log-follow.paused{color:var(--warn)}
.log-follow .dot{width:6px;height:6px}
.log-follow.paused .dot{background:var(--warn)}
.retention-control{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
.retention-control input{width:90px}

@media(max-width:760px){
  .shell{max-width:none;margin:0;padding:14px 14px 92px}
  .topbar>.brand{display:flex}
  .topbar{margin-bottom:14px}
  .topbar-actions{width:auto;display:flex;gap:7px}
  .mobile-menu-btn{display:inline-grid;place-items:center;width:38px;height:38px;border:1px solid var(--line);background:var(--panel);border-radius:9px;color:var(--text);font-size:19px;cursor:pointer}
  .tabs{left:0;right:0;top:auto;bottom:0;width:100%;height:72px;padding:6px 5px max(6px,env(safe-area-inset-bottom));border:0;border-top:1px solid var(--line);display:flex;flex-direction:row;gap:1px;box-shadow:0 -8px 26px rgba(0,0,0,.13)}
  .side-brand,.side-status{display:none}
  .tab{min-width:0;flex:1;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:2px;padding:5px 2px;font-size:9.5px;line-height:1.15;text-align:center}
  .nav-icon{width:auto;height:20px;font-size:15px}
  .mobile-drawer-scrim{position:fixed;display:block;z-index:89;inset:0;background:rgba(2,6,23,.62)}
  .mobile-drawer{position:fixed;display:flex;z-index:90;left:0;top:0;bottom:0;width:min(300px,82vw);padding:18px 14px;background:var(--panel);border-right:1px solid var(--line);box-shadow:var(--shadow);flex-direction:column;transform:translateX(-105%);transition:transform .2s ease}
  .mobile-drawer.open{transform:translateX(0)}
  .mobile-drawer-head{display:flex;align-items:center;justify-content:space-between;padding:0 4px 16px;border-bottom:1px solid var(--line);margin-bottom:9px}
  .mobile-drawer-brand{display:flex;align-items:center;gap:9px;font-weight:600;font-size:16px}
  .mobile-drawer-close{border:0;background:transparent;color:var(--dim);font-size:24px;cursor:pointer}
  .mobile-drawer .drawer-tab{display:flex;align-items:center;gap:11px;width:100%;padding:11px;border:0;border-radius:9px;background:transparent;color:var(--muted);font:inherit;text-align:left;cursor:pointer}
  .mobile-drawer .drawer-tab.active{background:color-mix(in srgb,var(--accent) 22%,transparent);color:var(--accent-text)}
  .mobile-drawer-status{margin-top:auto;border-top:1px solid var(--line);padding:14px 8px;color:var(--dim);font-size:11px;display:grid;gap:7px}
  th.exec-column{touch-action:pan-x pan-y}
}
`

const uiExtensionJS = `
<script>
(function(){
"use strict";

/* Navigation -------------------------------------------------------------- */
var navMeta={
  dashboard:{label:"Dashboard",icon:"▦"},
  jobs:{label:"Jobs",icon:"☷"},
  remotes:{label:"Remotes",icon:"◉"},
  logs:{label:"Logs",icon:"▤"},
  settings:{label:"Settings",icon:"⚙"}
};
function installAppShell(){
  var nav=document.querySelector(".tabs");
  var topbar=document.querySelector(".topbar");
  if(!nav||!topbar||document.querySelector(".side-brand"))return;
  var brand=document.createElement("div");brand.className="side-brand";brand.innerHTML='<div class="logo">C</div><div><strong>Copyarr</strong><div class="dim">Transfer control</div></div>';nav.insertBefore(brand,nav.firstChild);
  Array.prototype.forEach.call(nav.querySelectorAll(".tab"),function(btn){var m=navMeta[btn.dataset.val]||{label:btn.textContent,icon:"•"};btn.innerHTML='<span class="nav-icon">'+m.icon+'</span><span>'+m.label+'</span>';});
  var stat=document.createElement("div");stat.className="side-status";stat.innerHTML='<div class="side-status-line"><span class="dot" id="sideDot"></span><span id="sideOnline">Online</span></div><div id="sideVersion">Copyarr</div><div>Unraid / self-hosted</div>';nav.appendChild(stat);
  var menu=document.createElement("button");menu.className="mobile-menu-btn";menu.type="button";menu.setAttribute("aria-label","Open navigation");menu.innerHTML="☰";topbar.insertBefore(menu,topbar.firstChild);
  var scrim=document.createElement("div");scrim.className="mobile-drawer-scrim";scrim.hidden=true;document.body.appendChild(scrim);
  var drawer=document.createElement("aside");drawer.className="mobile-drawer";drawer.innerHTML='<div class="mobile-drawer-head"><div class="mobile-drawer-brand"><div class="logo">C</div><span>Copyarr</span></div><button class="mobile-drawer-close" aria-label="Close navigation">×</button></div><div id="drawerLinks"></div><div class="mobile-drawer-status"><div class="side-status-line"><span class="dot"></span><span>Online</span></div><div id="drawerVersion">Copyarr</div><div>Unraid / self-hosted</div></div>';document.body.appendChild(drawer);
  var links=drawer.querySelector("#drawerLinks");
  Array.prototype.forEach.call(nav.querySelectorAll(".tab"),function(btn){var m=navMeta[btn.dataset.val];var x=document.createElement("button");x.className="drawer-tab"+(btn.classList.contains("active")?" active":"");x.dataset.page=btn.dataset.val;x.innerHTML='<span class="nav-icon">'+m.icon+'</span><span>'+m.label+'</span>';links.appendChild(x);});
  function close(){drawer.classList.remove("open");scrim.hidden=true;}
  function open(){drawer.classList.add("open");scrim.hidden=false;}
  menu.addEventListener("click",open);scrim.addEventListener("click",close);drawer.querySelector(".mobile-drawer-close").addEventListener("click",close);
  links.addEventListener("click",function(ev){var b=ev.target.closest("[data-page]");if(!b)return;var real=nav.querySelector('.tab[data-val="'+b.dataset.page+'"]');if(real)real.click();close();syncDrawer();});
  nav.addEventListener("click",function(){setTimeout(syncDrawer,0);});
  function syncDrawer(){Array.prototype.forEach.call(drawer.querySelectorAll(".drawer-tab"),function(x){var real=nav.querySelector('.tab[data-val="'+x.dataset.page+'"]');x.classList.toggle("active",!!real&&real.classList.contains("active"));});}
  var originalRefreshStatus=refreshStatus;
  refreshStatus=async function(){await originalRefreshStatus();var sideBuild=document.getElementById("sideVersion");var drawerBuild=document.getElementById("drawerVersion");var build=$("build").textContent;if(sideBuild)sideBuild.textContent=build;if(drawerBuild)drawerBuild.textContent=build;var txt=$("liveText").textContent;var sideOnline=document.getElementById("sideOnline");if(sideOnline)sideOnline.textContent=txt;var sideDot=document.getElementById("sideDot");if(sideDot)sideDot.className=$("liveDot").className;};
}

/* Execution columns ------------------------------------------------------- */
var execColumns={
  id:{label:"ID",value:function(j){return Number(j.id||0);},render:function(j){return '<span class="dim num">'+esc(j.id)+'</span>'; }},
  name:{label:"Name",value:function(j){return String(j.display_name||"").toLowerCase();},render:function(j){return '<div class="cell-name" title="Open transfer details">'+esc(j.display_name||"")+'</div>'; }},
  state:{label:"State",value:function(j){return String(j.state||"");},render:function(j){return badge(j.state||"other");}},
  date:{label:"Transferred",value:function(j){return Date.parse(j.transfer_at||j.started_at||j.created_at||0)||0;},render:function(j){var v=j.transfer_at||j.started_at||j.created_at;return v?'<span class="num">'+esc(new Date(v).toLocaleString())+'</span>':"-";}},
  created:{label:"Created",value:function(j){return Date.parse(j.created_at||0)||0;},render:function(j){return j.created_at?esc(new Date(j.created_at).toLocaleString()):"-";}},
  completed:{label:"Completed",value:function(j){return Date.parse(j.completed_at||0)||0;},render:function(j){return j.completed_at?esc(new Date(j.completed_at).toLocaleString()):"-";}},
  duration:{label:"Duration",value:function(j){return Number(j.duration_seconds||0);},render:function(j){return j.duration_seconds?dur(j.duration_seconds):"-";}},
  size:{label:"Size",value:function(j){return Number(j.total_bytes||0);},render:function(j){return '<span class="num">'+bytes(j.total_bytes||0)+'</span>'; }},
  avg:{label:"Avg speed",value:function(j){return Number(j.avg_speed_bps||0);},render:function(j){return j.avg_speed_bps?'<span class="num">'+bytes(j.avg_speed_bps)+'/s</span>':"-";}},
  peak:{label:"Peak speed",value:function(j){return Number(j.peak_speed_bps||0);},render:function(j){return j.peak_speed_bps?'<span class="num">'+bytes(j.peak_speed_bps)+'/s</span>':"-";}},
  files:{label:"Files",value:function(j){return Number(j.item_count||0);},render:function(j){return '<span class="num">'+Number(j.item_count||0)+'</span>'; }},
  attempts:{label:"Attempts",value:function(j){return Number(j.attempt_number||j.attempts||0);},render:function(j){return '<span class="num">'+(j.attempt_number||j.attempts||0)+'/'+(j.max_attempts||"?")+'</span>'; }},
  reason:{label:"Reason",value:function(j){return String(j.reason||"").toLowerCase();},render:function(j){return esc(j.reason||"-");}},
  rule:{label:"Job",value:function(j){return String(j.rule_id||"").toLowerCase();},render:function(j){return esc(j.rule_id||"-");}},
  kind:{label:"Kind",value:function(j){return String(j.kind||"").toLowerCase();},render:function(j){return esc(j.kind||"-");}},
  path:{label:"Path",value:function(j){return String(j.rel_root||"").toLowerCase();},render:function(j){return '<span title="'+esc(j.rel_root||"")+'">'+esc(j.rel_root||"-")+'</span>'; }},
  destination:{label:"Destination",value:function(j){return String(j.dest_path||"").toLowerCase();},render:function(j){return '<span title="'+esc(j.dest_path||"")+'">'+esc(j.dest_path||"-")+'</span>'; }},
  error:{label:"Next retry / error",value:function(j){return String(j.next_retry_at||j.last_error||"").toLowerCase();},render:function(j){var v=j.next_retry_at?("Retry at "+new Date(j.next_retry_at).toLocaleString()):(j.last_error||"-");return '<span class="cell-err" title="'+esc(j.last_error||"")+'">'+esc(v)+'</span>';}}
};
var execColumnOrder=["id","name","state","date","created","completed","duration","size","avg","peak","files","attempts","reason","rule","kind","path","destination","error"];
var defaultColumns=["date","name","state","size","avg","peak","files","attempts"];
var execPrefs={columns:defaultColumns.slice(),sort:"date",dir:"desc"};
var draggedExecColumn=null;
var columnMenu=null;

function loadExecPrefs(){
  try{var raw=JSON.parse(localStorage.getItem("copyarr:execution-table:v2")||localStorage.getItem("copyarr:execution-table:v1")||"{}");if(Array.isArray(raw.columns))execPrefs.columns=raw.columns.filter(function(k){return !!execColumns[k];});if(execColumns[raw.sort])execPrefs.sort=raw.sort;if(raw.dir==="asc"||raw.dir==="desc")execPrefs.dir=raw.dir;}catch(e){}
  if(!execPrefs.columns.length)execPrefs.columns=defaultColumns.slice();
}
function saveExecPrefs(){try{localStorage.setItem("copyarr:execution-table:v2",JSON.stringify(execPrefs));}catch(e){}}
function execCompare(a,b,key){var av=execColumns[key].value(a),bv=execColumns[key].value(b);if(typeof av==="number"&&typeof bv==="number")return av-bv;return String(av).localeCompare(String(bv),undefined,{numeric:true,sensitivity:"base"});}
function sortExecColumn(key){if(execPrefs.sort===key)execPrefs.dir=execPrefs.dir==="asc"?"desc":"asc";else{execPrefs.sort=key;execPrefs.dir="desc";}saveExecPrefs();renderJobs();}
function setColumnVisible(key,on){if(on){if(execPrefs.columns.indexOf(key)<0)execPrefs.columns.push(key);}else{if(execPrefs.columns.length<=1){toast("Keep at least one column visible",true);return false;}execPrefs.columns=execPrefs.columns.filter(function(x){return x!==key;});if(execPrefs.sort===key)execPrefs.sort=execPrefs.columns[0];}saveExecPrefs();renderJobs();return true;}
function moveExecColumn(source,target){if(!source||!target||source===target)return;var from=execPrefs.columns.indexOf(source),to=execPrefs.columns.indexOf(target);if(from<0||to<0)return;var item=execPrefs.columns.splice(from,1)[0];to=execPrefs.columns.indexOf(target);execPrefs.columns.splice(to,0,item);saveExecPrefs();renderJobs();}

function closeColumnMenu(){if(columnMenu){columnMenu.remove();columnMenu=null;}}
function openColumnMenu(x,y){
  closeColumnMenu();var menu=document.createElement("div");menu.className="exec-column-menu";menu.innerHTML='<div class="exec-column-menu-title">Visible columns</div>'+execColumnOrder.map(function(k){return '<label class="exec-column-choice"><input type="checkbox" data-column-choice="'+k+'" '+(execPrefs.columns.indexOf(k)>=0?"checked":"")+'><span>'+esc(execColumns[k].label)+'</span></label>';}).join("")+'<div class="exec-column-hint">Drag headers to reorder · click a title to sort</div>';document.body.appendChild(menu);columnMenu=menu;
  var r=menu.getBoundingClientRect();menu.style.left=Math.max(8,Math.min(x,window.innerWidth-r.width-8))+"px";menu.style.top=Math.max(8,Math.min(y,window.innerHeight-r.height-8))+"px";
  menu.addEventListener("change",function(ev){var key=ev.target.dataset.columnChoice;if(!key)return;if(!setColumnVisible(key,ev.target.checked))ev.target.checked=true;});
}
document.addEventListener("pointerdown",function(ev){if(columnMenu&&!columnMenu.contains(ev.target))closeColumnMenu();});
document.addEventListener("keydown",function(ev){if(ev.key==="Escape")closeColumnMenu();});

function bindExecutionHeaders(head){
  Array.prototype.forEach.call(head.querySelectorAll("th.exec-column"),function(th){
    var key=th.dataset.execColumn;
    th.addEventListener("dragstart",function(ev){draggedExecColumn=key;th.classList.add("dragging");if(ev.dataTransfer){ev.dataTransfer.effectAllowed="move";ev.dataTransfer.setData("text/plain",key);}});
    th.addEventListener("dragend",function(){draggedExecColumn=null;th.classList.remove("dragging");Array.prototype.forEach.call(head.querySelectorAll(".drag-over"),function(x){x.classList.remove("drag-over");});});
    th.addEventListener("dragover",function(ev){if(!draggedExecColumn||draggedExecColumn===key)return;ev.preventDefault();th.classList.add("drag-over");if(ev.dataTransfer)ev.dataTransfer.dropEffect="move";});
    th.addEventListener("dragleave",function(){th.classList.remove("drag-over");});
    th.addEventListener("drop",function(ev){ev.preventDefault();th.classList.remove("drag-over");moveExecColumn(draggedExecColumn,key);draggedExecColumn=null;});
    th.addEventListener("contextmenu",function(ev){ev.preventDefault();openColumnMenu(ev.clientX,ev.clientY);});
    var timer=null,startX=0,startY=0;
    th.addEventListener("pointerdown",function(ev){if(ev.pointerType!=="touch")return;startX=ev.clientX;startY=ev.clientY;timer=setTimeout(function(){openColumnMenu(startX,startY);timer=null;},550);});
    th.addEventListener("pointermove",function(ev){if(timer&&(Math.abs(ev.clientX-startX)>12||Math.abs(ev.clientY-startY)>12)){clearTimeout(timer);timer=null;}});
    th.addEventListener("pointerup",function(){if(timer){clearTimeout(timer);timer=null;}});
    var label=th.querySelector("[data-exec-sort]");if(label)label.addEventListener("click",function(ev){ev.stopPropagation();sortExecColumn(key);});
  });
}

renderJobs=function(){
  var shown=jobs.filter(visibleJob).slice();shown.sort(function(a,b){var n=execCompare(a,b,execPrefs.sort);return execPrefs.dir==="asc"?n:-n;});$("jobCount").textContent=shown.length+" shown of "+jobs.length+" loaded";var table=$("jobRows").closest("table");var head=table.querySelector("thead tr");
  head.innerHTML=execPrefs.columns.map(function(k){var mark=execPrefs.sort===k?(execPrefs.dir==="asc"?"↑":"↓"):"";return '<th class="exec-column" draggable="true" data-exec-column="'+k+'"><button class="exec-head-label" data-exec-sort="'+k+'" title="Click to sort · drag to reorder · right-click for columns">'+esc(execColumns[k].label)+(mark?'<span class="exec-sort-mark">'+mark+'</span>':"")+'</button></th>';}).join("")+'<th></th>';
  if(!shown.length){$("jobRows").innerHTML='<tr><td colspan="'+(execPrefs.columns.length+1)+'" class="empty-row">No executions match this filter.</td></tr>';bindExecutionHeaders(head);return;}
  $("jobRows").innerHTML=shown.map(function(j){return '<tr data-act="execution-open" data-id="'+j.id+'" style="cursor:pointer">'+execPrefs.columns.map(function(k){return '<td>'+execColumns[k].render(j)+'</td>';}).join("")+'<td><div class="row-actions">'+controls(j)+'</div></td></tr>';}).join("");bindExecutionHeaders(head);
};

/* Log follow -------------------------------------------------------------- */
var globalLogFollow=true;
function logAtBottom(el){return el.scrollHeight-el.scrollTop-el.clientHeight<28;}
function installLogFollow(){var host=$("globalLog");if(!host||$("logFollowState"))return;var select=$("logLevel"),wrap=select.parentElement;var state=document.createElement("button");state.id="logFollowState";state.className="btn btn-sm";state.innerHTML='<span class="log-follow"><span class="dot"></span><span>Following</span></span>';wrap.insertBefore(state,select);function paint(){var x=state.querySelector(".log-follow");x.className="log-follow"+(globalLogFollow?"":" paused");x.querySelector("span:last-child").textContent=globalLogFollow?"Following":"Paused";}host.addEventListener("scroll",function(){globalLogFollow=logAtBottom(host);paint();},{passive:true});state.addEventListener("click",function(){globalLogFollow=true;renderGlobalLogs();host.scrollTop=host.scrollHeight;paint();});paint();}
renderGlobalLogs=function(){var level=$("logLevel").value||"all";var shown=globalLogs.filter(function(x){return level==="all"||String(x.level||"").toUpperCase()===level;});$("logCount").textContent=shown.length+" shown of "+globalLogs.length+" loaded";var host=$("globalLog"),oldTop=host.scrollTop;if(!shown.length){host.innerHTML='<div class="dim">No stored log entries.</div>';return;}var ordered=shown.slice().reverse();host.innerHTML=ordered.map(function(x){var extra="";if(x.fields_json){try{var obj=JSON.parse(x.fields_json);if(Object.keys(obj).length)extra=" "+JSON.stringify(obj);}catch(e){extra=" "+x.fields_json;}}return '<div class="log-line '+esc(String(x.level||"").toUpperCase())+'"><span class="ts">'+esc(new Date(x.timestamp).toLocaleString())+'</span><span class="lvl">'+esc(x.level||"")+'</span><span class="src">'+esc(x.source||"")+'</span><span class="msg">'+esc((x.message||"")+extra)+'</span></div>';}).join("");if(globalLogFollow)host.scrollTop=host.scrollHeight;else host.scrollTop=oldTop;};

/* Retention --------------------------------------------------------------- */
var originalRefreshSettings=refreshSettings;
refreshSettings=async function(){await originalRefreshSettings();try{var s=await api("/api/settings");if($("settingRetention"))$("settingRetention").value=s.log_retention_days||7;}catch(e){}};
function installRetentionSetting(){var toggle=$("settingLogging");if(!toggle||$("settingRetention"))return;var first=toggle.closest(".settings-row");if(!first)return;var row=document.createElement("div");row.className="settings-row";row.innerHTML='<div><h3>Log retention</h3><div class="dim" style="margin-top:4px">Delete persisted application and rclone logs older than this. Default is 7 days.</div></div><div class="retention-control"><input id="settingRetention" type="number" min="1" max="3650" value="7"><span class="dim">days</span><button class="btn" id="saveRetention">Save</button></div>';first.parentNode.insertBefore(row,first.nextSibling);$("saveRetention").addEventListener("click",async function(){var days=Number($("settingRetention").value);try{var s=await api("/api/settings",{method:"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({log_retention_days:days})});$("settingRetention").value=s.log_retention_days;toast("Log retention saved");}catch(e){toast(e.message,true);}});}

loadExecPrefs();installAppShell();installLogFollow();installRetentionSetting();renderJobs();refreshSettings();
})();
</script>
`
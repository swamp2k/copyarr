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
.exec-head{display:flex;align-items:center;gap:4px;white-space:nowrap}
.exec-head-label{border:0;background:transparent;color:inherit;font:inherit;font-weight:inherit;text-transform:inherit;letter-spacing:inherit;padding:0;cursor:pointer}
.exec-head-label:hover{color:var(--text)}
.exec-head-tools{display:inline-flex;gap:2px;margin-left:2px}
.exec-head-btn{display:inline-grid;place-items:center;width:18px;height:18px;padding:0;border:1px solid transparent;border-radius:5px;background:transparent;color:var(--faint);font:inherit;font-size:10px;cursor:pointer}
.exec-head-btn:hover:not(:disabled){color:var(--text);border-color:var(--line);background:var(--panel-2)}
.exec-head-btn:disabled{opacity:.2;cursor:default}
.exec-add{min-width:118px}
.exec-add select{padding:4px 7px;font-size:11px;min-width:118px}
.log-follow{display:inline-flex;align-items:center;gap:6px;color:var(--good);font-size:11px}
.log-follow.paused{color:var(--warn)}
.log-follow .dot{width:6px;height:6px}
.log-follow.paused .dot{background:var(--warn)}
.retention-control{display:flex;gap:8px;align-items:center;flex-wrap:wrap}
.retention-control input{width:90px}
@media(max-width:620px){.exec-head-tools{gap:1px}.exec-head-btn{width:20px;height:20px}.exec-add{min-width:108px}.exec-add select{min-width:108px}}
`

const uiExtensionJS = `
<script>
(function(){
"use strict";

var execColumns = {
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
var defaultColumns=["id","name","state","date","size","avg","peak","files","attempts","error"];
var execPrefs={columns:defaultColumns.slice(),sort:"date",dir:"desc"};

function loadExecPrefs(){
  try{
    var raw=JSON.parse(localStorage.getItem("copyarr:execution-table:v1")||"{}");
    if(Array.isArray(raw.columns)) execPrefs.columns=raw.columns.filter(function(k){return !!execColumns[k];});
    if(execColumns[raw.sort]) execPrefs.sort=raw.sort;
    if(raw.dir==="asc"||raw.dir==="desc") execPrefs.dir=raw.dir;
  }catch(e){}
  if(!execPrefs.columns.length) execPrefs.columns=defaultColumns.slice();
}
function saveExecPrefs(){try{localStorage.setItem("copyarr:execution-table:v1",JSON.stringify(execPrefs));}catch(e){}}
function execCompare(a,b,key){
  var av=execColumns[key].value(a),bv=execColumns[key].value(b);
  if(typeof av==="number"&&typeof bv==="number") return av-bv;
  return String(av).localeCompare(String(bv),undefined,{numeric:true,sensitivity:"base"});
}
function moveExecColumn(key,delta){
  var i=execPrefs.columns.indexOf(key),next=i+delta;
  if(i<0||next<0||next>=execPrefs.columns.length)return;
  var tmp=execPrefs.columns[i];execPrefs.columns[i]=execPrefs.columns[next];execPrefs.columns[next]=tmp;
  saveExecPrefs();renderJobs();
}
function hideExecColumn(key){
  if(execPrefs.columns.length<=1){toast("Keep at least one column visible",true);return;}
  execPrefs.columns=execPrefs.columns.filter(function(x){return x!==key;});
  if(execPrefs.sort===key)execPrefs.sort=execPrefs.columns[0];
  saveExecPrefs();renderJobs();
}
function addExecColumn(key){
  if(!key||!execColumns[key]||execPrefs.columns.indexOf(key)>=0)return;
  execPrefs.columns.push(key);saveExecPrefs();renderJobs();
}
function sortExecColumn(key){
  if(execPrefs.sort===key)execPrefs.dir=execPrefs.dir==="asc"?"desc":"asc";
  else{execPrefs.sort=key;execPrefs.dir="desc";}
  saveExecPrefs();renderJobs();
}

renderJobs=function(){
  var shown=jobs.filter(visibleJob).slice();
  shown.sort(function(a,b){var n=execCompare(a,b,execPrefs.sort);return execPrefs.dir==="asc"?n:-n;});
  $("jobCount").textContent=shown.length+" shown of "+jobs.length+" loaded";
  var table=$("jobRows").closest("table");
  var head=table.querySelector("thead tr");
  head.innerHTML=execPrefs.columns.map(function(k,i){
    var mark=execPrefs.sort===k?(execPrefs.dir==="asc"?" ↑":" ↓"):"";
    return '<th><div class="exec-head">'+
      '<button class="exec-head-label" data-exec-sort="'+k+'" title="Sort by '+esc(execColumns[k].label)+'">'+esc(execColumns[k].label+mark)+'</button>'+
      '<span class="exec-head-tools">'+
        '<button class="exec-head-btn" data-exec-left="'+k+'" title="Move left" '+(i===0?"disabled":"")+'>&larr;</button>'+
        '<button class="exec-head-btn" data-exec-right="'+k+'" title="Move right" '+(i===execPrefs.columns.length-1?"disabled":"")+'>&rarr;</button>'+
        '<button class="exec-head-btn" data-exec-hide="'+k+'" title="Hide column">&times;</button>'+
      '</span></div></th>';
  }).join("")+'<th class="exec-add"><select data-exec-add><option value="">+ Add column</option>'+execColumnOrder.filter(function(k){return execPrefs.columns.indexOf(k)<0;}).map(function(k){return '<option value="'+k+'">'+esc(execColumns[k].label)+'</option>';}).join("")+'</select></th>';
  if(!shown.length){$("jobRows").innerHTML='<tr><td colspan="'+(execPrefs.columns.length+1)+'" class="empty-row">No executions match this filter.</td></tr>';return;}
  $("jobRows").innerHTML=shown.map(function(j){
    return '<tr data-act="execution-open" data-id="'+j.id+'" style="cursor:pointer">'+
      execPrefs.columns.map(function(k){return '<td>'+execColumns[k].render(j)+'</td>';}).join("")+
      '<td><div class="row-actions">'+controls(j)+'</div></td></tr>';
  }).join("");
  Array.prototype.forEach.call(head.querySelectorAll("[data-exec-sort]"),function(el){el.addEventListener("click",function(ev){ev.stopPropagation();sortExecColumn(this.dataset.execSort);});});
  Array.prototype.forEach.call(head.querySelectorAll("[data-exec-left]"),function(el){el.addEventListener("click",function(ev){ev.stopPropagation();moveExecColumn(this.dataset.execLeft,-1);});});
  Array.prototype.forEach.call(head.querySelectorAll("[data-exec-right]"),function(el){el.addEventListener("click",function(ev){ev.stopPropagation();moveExecColumn(this.dataset.execRight,1);});});
  Array.prototype.forEach.call(head.querySelectorAll("[data-exec-hide]"),function(el){el.addEventListener("click",function(ev){ev.stopPropagation();hideExecColumn(this.dataset.execHide);});});
  var add=head.querySelector("[data-exec-add]");if(add)add.addEventListener("change",function(){addExecColumn(this.value);});
};

var globalLogFollow=true;
function logAtBottom(el){return el.scrollHeight-el.scrollTop-el.clientHeight<28;}
function installLogFollow(){
  var host=$("globalLog");if(!host||$("logFollowState"))return;
  var select=$("logLevel");var wrap=select.parentElement;
  var state=document.createElement("button");state.id="logFollowState";state.className="btn btn-sm";state.innerHTML='<span class="log-follow"><span class="dot"></span><span>Following</span></span>';wrap.insertBefore(state,select);
  function paint(){var x=state.querySelector(".log-follow");x.className="log-follow"+(globalLogFollow?"":" paused");x.querySelector("span:last-child").textContent=globalLogFollow?"Following":"Paused";}
  host.addEventListener("scroll",function(){globalLogFollow=logAtBottom(host);paint();},{passive:true});
  state.addEventListener("click",function(){globalLogFollow=true;renderGlobalLogs();host.scrollTop=host.scrollHeight;paint();});
  paint();
}

renderGlobalLogs=function(){
  var level=$("logLevel").value||"all";
  var shown=globalLogs.filter(function(x){return level==="all"||String(x.level||"").toUpperCase()===level;});
  $("logCount").textContent=shown.length+" shown of "+globalLogs.length+" loaded";
  var host=$("globalLog");var oldTop=host.scrollTop;
  if(!shown.length){host.innerHTML='<div class="dim">No stored log entries.</div>';return;}
  var ordered=shown.slice().reverse();
  host.innerHTML=ordered.map(function(x){
    var extra="";if(x.fields_json){try{var obj=JSON.parse(x.fields_json);if(Object.keys(obj).length)extra=" "+JSON.stringify(obj);}catch(e){extra=" "+x.fields_json;}}
    return '<div class="log-line '+esc(String(x.level||"").toUpperCase())+'"><span class="ts">'+esc(new Date(x.timestamp).toLocaleString())+'</span><span class="lvl">'+esc(x.level||"")+'</span><span class="src">'+esc(x.source||"")+'</span><span class="msg">'+esc((x.message||"")+extra)+'</span></div>';
  }).join("");
  if(globalLogFollow)host.scrollTop=host.scrollHeight;else host.scrollTop=oldTop;
};

var originalRefreshSettings=refreshSettings;
refreshSettings=async function(){
  await originalRefreshSettings();
  try{var s=await api("/api/settings");if($("settingRetention"))$("settingRetention").value=s.log_retention_days||7;}catch(e){}
};
function installRetentionSetting(){
  var toggle=$("settingLogging");if(!toggle||$("settingRetention"))return;
  var first=toggle.closest(".settings-row");if(!first)return;
  var row=document.createElement("div");row.className="settings-row";
  row.innerHTML='<div><h3>Log retention</h3><div class="dim" style="margin-top:4px">Delete persisted application and rclone logs older than this. Default is 7 days.</div></div><div class="retention-control"><input id="settingRetention" type="number" min="1" max="3650" value="7"><span class="dim">days</span><button class="btn" id="saveRetention">Save</button></div>';
  first.parentNode.insertBefore(row,first.nextSibling);
  $("saveRetention").addEventListener("click",async function(){var days=Number($("settingRetention").value);try{var s=await api("/api/settings",{method:"PATCH",headers:{"Content-Type":"application/json"},body:JSON.stringify({log_retention_days:days})});$("settingRetention").value=s.log_retention_days;toast("Log retention saved");}catch(e){toast(e.message,true);}});
}

loadExecPrefs();installLogFollow();installRetentionSetting();renderJobs();refreshSettings();
})();
</script>
`
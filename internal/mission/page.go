package mission

// pageHTML is the entire control surface — self-contained, no build step, no framework
// (CLAUDE.md: "Mission Control = static page from argusd. No React/Next build.").
// It polls /mission/state once a second and drives ONE control: inject/stop drift.
const pageHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Argus · Mission Control</title>
<style>
  :root{--bg:#0b0e14;--card:#151a23;--line:#232a37;--tx:#e6e9ef;--mut:#8b93a7;
        --grn:#2ecc71;--red:#e74c3c;--amb:#f39c12}
  *{box-sizing:border-box}
  body{margin:0;background:var(--bg);color:var(--tx);
       font:16px/1.5 -apple-system,BlinkMacSystemFont,"Segoe UI",Roboto,sans-serif}
  .wrap{max-width:1000px;margin:0 auto;padding:40px 24px}
  header{display:flex;align-items:baseline;gap:16px;margin-bottom:8px}
  h1{font-size:30px;margin:0;letter-spacing:.5px}
  .tag{color:var(--mut);font-size:15px}
  .sub{color:var(--mut);margin:0 0 32px}
  .grid{display:grid;grid-template-columns:repeat(3,1fr);gap:16px}
  .tile{background:var(--card);border:1px solid var(--line);border-radius:14px;padding:22px}
  .tile .k{font-size:13px;letter-spacing:1.5px;text-transform:uppercase;color:var(--mut)}
  .tile .v{font-size:26px;font-weight:600;margin-top:10px;display:flex;align-items:center;gap:10px}
  .tile .d{color:var(--mut);font-size:14px;margin-top:6px;min-height:20px}
  .dot{width:12px;height:12px;border-radius:50%;flex:0 0 auto}
  .ok{background:var(--grn)}.bad{background:var(--red)}.warn{background:var(--amb)}
  .tok{color:var(--grn)}.tbad{color:var(--red)}.twarn{color:var(--amb)}
  .action{background:var(--card);border:1px solid var(--line);border-radius:14px;
          padding:20px 22px;margin-top:16px;display:flex;justify-content:space-between;align-items:center}
  .action .k{font-size:13px;letter-spacing:1.5px;text-transform:uppercase;color:var(--mut)}
  .action .v{font-size:20px;font-weight:600;margin-top:6px}
  .action .ago{color:var(--mut);font-size:14px}
  .controls{margin-top:32px;text-align:center}
  button{font:600 20px/1 inherit;color:#fff;border:0;border-radius:12px;padding:20px 40px;
         cursor:pointer;transition:.15s;min-width:280px}
  .inject{background:var(--red)}.inject:hover{filter:brightness(1.1)}
  .stop{background:#2b3242;border:1px solid var(--line)}.stop:hover{filter:brightness(1.2)}
  button:disabled{opacity:.4;cursor:not-allowed}
  .hint{color:var(--mut);font-size:13px;margin-top:12px;min-height:18px}
  .pulse{animation:p 1.4s ease-in-out infinite}
  @keyframes p{0%,100%{opacity:1}50%{opacity:.45}}
</style>
</head>
<body>
<div class="wrap">
  <header>
    <h1>ARGUS · Mission Control</h1><span class="tag">the immune system for AI agents</span>
  </header>
  <p class="sub">Infrastructure can be perfectly healthy while intelligence is catastrophically wrong.</p>

  <div class="grid">
    <div class="tile"><div class="k">System</div>
      <div class="v"><span id="sys-dot" class="dot ok"></span><span id="sys-v">—</span></div>
      <div class="d" id="sys-d"></div></div>
    <div class="tile"><div class="k">PREVENT · Behavior Guard</div>
      <div class="v"><span id="prev-dot" class="dot ok"></span><span id="prev-v">—</span></div>
      <div class="d" id="prev-d"></div></div>
    <div class="tile"><div class="k">LEARN · Behavior Drift</div>
      <div class="v"><span id="learn-dot" class="dot ok"></span><span id="learn-v">—</span></div>
      <div class="d" id="learn-d"></div></div>
  </div>

  <div class="action">
    <div><div class="k">Last Argus action</div><div class="v" id="act-v">—</div></div>
    <div class="ago" id="act-ago"></div>
  </div>

  <div class="controls">
    <button id="chaos" disabled>—</button>
    <div class="hint" id="hint">connecting…</div>
  </div>
</div>

<script>
(function(){
  var drift=false, available=false;
  function ago(ms){
    if(ms==null) return "";
    var s=Math.round(ms/1000);
    if(s<60) return s+"s ago";
    return Math.round(s/60)+"m ago";
  }
  function set(id,t){document.getElementById(id).textContent=t;}
  function dot(id,cls){document.getElementById(id).className="dot "+cls;}
  function render(st){
    set("sys-v", st.system.ok?"Operational":"Down");
    dot("sys-dot", st.system.ok?"ok":"bad");
    set("sys-d", st.system.detail||"");

    var p=st.prevent;
    set("prev-v","Active");
    dot("prev-dot","ok");
    set("prev-d", p.lastDecision ? ("last: "+p.lastDecision+" · "+(p.lastModel||"")+" "+ago(p.agoMs)) : "inline · deterministic");

    var l=st.learn, q=l.quarantined||{}, keys=Object.keys(q);
    if(l.healthy){ set("learn-v","Healthy"); dot("learn-dot","ok"); set("learn-d","watching windowed drift via SigNoz"); }
    else { set("learn-v","Quarantined"); dot("learn-dot","bad");
           set("learn-d", keys.map(function(k){return k+" → "+q[k];}).join(", ")); }

    var a=st.lastAction;
    set("act-v", a.text || "none yet");
    set("act-ago", a.text ? ago(a.agoMs) : "");

    drift = !!st.chaos.drift; available = !!st.chaos.available;
    var b=document.getElementById("chaos");
    b.disabled = !available;
    b.textContent = drift ? "■  Stop drift" : "⚡  Inject drift";
    b.className = drift ? "stop pulse" : "inject";
    set("hint", available ? (drift?"replay engine is emitting hallucinations (gpt-4o)":"replay engine healthy — click to start the incident")
                          : "chaos target not reachable");
  }
  function poll(){
    fetch("/mission/state").then(function(r){return r.json();}).then(render)
      .catch(function(){ set("sys-v","Unreachable"); dot("sys-dot","bad"); set("hint","argusd unreachable"); });
  }
  document.getElementById("chaos").addEventListener("click",function(){
    var b=this; b.disabled=true;
    fetch("/mission/chaos",{method:"POST",headers:{"Content-Type":"application/json"},
      body:JSON.stringify({drift:!drift})}).then(function(){ setTimeout(poll,150); })
      .catch(function(){ b.disabled=false; });
  });
  poll(); setInterval(poll,1000);
})();
</script>
</body>
</html>`

let remote=[],have=new Set(),timer=null;
const post=(u,b)=>fetch(u,{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(b||{})}).then(r=>r.json());
async function openSettings(){$('settings').style.display='block';location.hash='settings';
 const c=await (await fetch('api/config')).json();$('ddir').value=c.data_dir;$('delay').value=c.delay;
 poll();if(!remote.length)fetchRemote(false);timer=timer||setInterval(poll,1500);}
$('openset').onclick=openSettings;
$('closeset').onclick=()=>{$('settings').style.display='none';clearInterval(timer);timer=null;history.replaceState(null,'',location.pathname);loadLib();};
$('saveconf').onclick=async()=>{const d=Math.max(1,+$('delay').value||1.2);$('delay').value=d;
 const r=await post('api/config',{data_dir:$('ddir').value,delay:d});$('confmsg').textContent=r.error||'已保存';loadLib();};
$('sample').onclick=()=>post('api/sample');
async function fetchRemote(refresh){$('rcount').textContent='正在获取…';
 const r=await (await fetch('api/remote'+(refresh?'?refresh':''))).json();
 if(r.error){$('rcount').textContent='获取失败: '+r.error;return;}remote=r;drawRemote();}
function drawRemote(){have=new Set(cat.filter(b=>b.complete).map(b=>b.id));const q=$('rq').value.trim();
 const l=remote.filter(i=>!q||(i.title+i.title_kana+i.artist).includes(q));
 $('rcount').textContent=`共 ${remote.length} 部，已下载 ${remote.filter(i=>have.has(i.id)).length} 部`;
 $('rbody').innerHTML=l.map(i=>`<tr class="${have.has(i.id)?'have':''}"><td><input type="checkbox" value="${i.id}" style="width:auto"></td><td>${i.id}</td><td>${i.title}</td><td>${i.artist}</td><td>${have.has(i.id)?'已下载':''}</td></tr>`).join('');}
$('fetchr').onclick=()=>fetchRemote(false);$('refreshr').onclick=()=>fetchRemote(true);$('rq').oninput=drawRemote;
$('selall').onclick=()=>document.querySelectorAll('#rbody input').forEach(c=>c.checked=true);
$('selnone').onclick=()=>document.querySelectorAll('#rbody input').forEach(c=>c.checked=false);
$('dlsel').onclick=async()=>{const ids=[...document.querySelectorAll('#rbody input:checked')].map(c=>+c.value);
 if(!ids.length)return alert('请先勾选作品');const r=await post('api/download',{ids});if(r.error)alert(r.error);else if(!r.ok)alert('已有下载任务在进行');poll();};
$('dlall').onclick=async()=>{if(!confirm('确定下载全部作品？数据量很大，需要较长时间。'))return;
 const r=await post('api/download',{all:true});if(r.error)alert(r.error);poll();};
$('pause').onclick=()=>post('api/pause').then(poll);$('resume').onclick=()=>post('api/resume').then(poll);
const SN={idle:'空闲',running:'下载中',pausing:'正在暂停…',paused:'已暂停',done:'已完成',error:'出错'};
let lastDone=-1;
async function poll(){const s=await (await fetch('api/status')).json();
 $('stline').textContent=`${SN[s.state]||s.state}${s.current?'：'+s.current.id+' '+s.current.title:''}　${s.message||''}`;
 $('pitem').max=s.item_total||1;$('pitem').value=s.item_done;$('pitemt').textContent=`${s.item_done} / ${s.item_total} 页`;
 $('pall').max=s.total_items||1;$('pall').value=s.done_items;$('pallt').textContent=`${s.done_items} / ${s.total_items} 部，剩余 ${s.queue_len} 部`;
 $('errs').textContent=s.errors.length?`失败 ${s.errors.length} 项：${s.errors.slice(-3).join('；')}`:'';
 $('samplemsg').textContent=s.sample.message||'从 GitHub 最新发布版下载 3 部示例作品';
 if(s.done_items!==lastDone||s.sample.state==='done'){lastDone=s.done_items;await loadLib();if(remote.length)drawRemote();}}

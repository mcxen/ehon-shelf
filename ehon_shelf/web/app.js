let cat=[],book=null,meta=null,pos=0,zoom=1;
const $=id=>document.getElementById(id);
const st={rtl:localStorage.getItem('rtl')!=='0',spread:localStorage.getItem('spread')==='1'};
function loadLib(){return fetch('api/library').then(r=>r.json()).then(c=>{cat=c;$('artist').length=1;
 [...new Set(c.map(b=>b.artist))].sort().forEach(a=>$('artist').add(new Option(a,a)));render();});}
loadLib().then(()=>{
 const h=decodeURIComponent(location.hash.slice(1));if(h&&h!=='settings')openBook(+h);else if(h==='settings')openSettings();});
function render(){const q=$('q').value.trim().toLowerCase(),a=$('artist').value;
 const list=cat.filter(b=>(!a||b.artist===a)&&(!q||(b.title+b.title_kana+b.artist).toLowerCase().includes(q)));
 $('count').textContent=`共 ${list.length} 部`;
 $('shelf').innerHTML=list.map(b=>`<div class="book${b.complete?'':' missing'}" data-id="${b.id}"><div class="c"><img loading="lazy" src="${b.cover}" alt=""></div>
 <div class="t">${b.title}</div><div class="a">${b.artist}${b.date?' · '+b.date:''} · ${b.page_count||0} 图</div></div>`).join('');}
$('q').oninput=render;$('artist').onchange=render;
$('shelf').onclick=e=>{const b=e.target.closest('.book');if(b)openBook(+b.dataset.id);};
async function openBook(id){book=cat.find(b=>b.id===id);if(!book)return;
 meta=await (await fetch(`data/${id}/meta.json`)).json();
 pos=+(localStorage.getItem('pos_'+id)||0);zoom=1;location.hash=id;
 $('rt').textContent=`${meta.title}　${meta.artist}`;$('reader').style.display='flex';show();}
function step(){return st.spread?2:1}
function show(){const n=meta.pages.length;pos=Math.max(0,Math.min(pos,n-1));
 let idx=[pos];if(st.spread&&pos+1<n)idx.push(pos+1);if(st.rtl)idx.reverse();
 $('pages').innerHTML=idx.map(i=>`<img src="data/${meta.id}/${meta.pages[i].file}" alt="">`).join('');
 $('pages').className=zoom>1?'zoomed':'';$('pages').style.transform=zoom===1?'':`scale(${zoom/2})`;
 $('pn').textContent=`第 ${pos+1}${idx.length>1?'-'+(pos+2):''} / ${n} 图`;
 $('dir').textContent=st.rtl?'阅读方向：右至左':'阅读方向：左至右';
 $('spread').textContent=st.spread?'双页模式':'单页模式';
 localStorage.setItem('pos_'+meta.id,pos);}
function fwd(){if(pos+step()<meta.pages.length){pos+=step();show();}}
function back(){pos-=step();show();}
// visual left/right: in RTL, left arrow goes forward
function left(){st.rtl?fwd():back()} function right(){st.rtl?back():fwd()}
$('prev').onclick=left;$('next').onclick=right;
$('dir').onclick=()=>{st.rtl=!st.rtl;localStorage.setItem('rtl',st.rtl?1:0);show();};
$('spread').onclick=()=>{st.spread=!st.spread;localStorage.setItem('spread',st.spread?1:0);show();};
$('zi').onclick=()=>{zoom=Math.min(zoom+1,8);show();};$('zo').onclick=()=>{zoom=Math.max(1,zoom-1);show();};$('zr').onclick=()=>{zoom=1;show();};
$('jump').onchange=e=>{pos=(+e.target.value||1)-1;show();};
$('close').onclick=()=>{$('reader').style.display='none';meta=null;history.replaceState(null,'',location.pathname);};
document.onkeydown=e=>{if(!meta||e.target.tagName==='INPUT')return;
 if(e.key==='ArrowLeft')left();else if(e.key==='ArrowRight')right();
 else if(e.key==='+'||e.key==='=')$('zi').click();else if(e.key==='-')$('zo').click();else if(e.key==='Escape')$('close').click();};

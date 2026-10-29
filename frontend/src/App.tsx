import {useEffect, useRef, useState} from 'react'
import {Activity, ArrowUpRight, Check, Cpu, ExternalLink, Loader2, Monitor, Moon, Pause, Pencil, Play, Plus, Power, RefreshCw, Server as ServerIcon, Settings2, Sun, Thermometer, Trash2, X} from 'lucide-react'
import {EventsOn} from '../wailsjs/runtime/runtime'
import {bridge, emptyConfig, emptySnapshot, type Bridge, type Choice, type Config, type GPU, type Server, type ServerSnapshot, type Snapshot} from './api'

const states: Record<string,string>={live:'实时',empty:'无 GPU',connecting:'连接中',stale:'数据过期',error:'连接失败',disabled:'已暂停'}
const errorText=(e:unknown)=>e instanceof Error?e.message:String(e)
function relative(ts:number,now:number){if(!ts)return '尚无成功采样';const seconds=Math.max(0,Math.floor((now-ts)/1000));return seconds<60?`${seconds} 秒前更新`:`${Math.floor(seconds/60)} 分钟前更新`}
function value(n:number|null,suffix=''){return n===null?'—':`${Math.round(n)}${suffix}`}
function gib(n:number|null){return n===null?'—':(n/1024).toFixed(1)}
function GPUItem({gpu,stale}:{gpu:GPU;stale:boolean}){
 const mem= gpu.memUsed!==null&&gpu.memTotal!==null&&gpu.memTotal>0?gpu.memUsed/gpu.memTotal*100:null
 return <div className={`gpu-row ${stale?'old-data':''}`}>
  <div className="gpu-name"><span className="gpu-index">{gpu.index.toString().padStart(2,'0')}</span><span title={gpu.name}>{gpu.name.replace(/^NVIDIA (GeForce )?/,'')}</span><span className={`temperature ${(gpu.temp??0)>=80?'hot':''}`}><Thermometer size={12}/>{value(gpu.temp,'°C')}</span></div>
  <div className="metrics"><div className="metric"><div className="metric-label"><span>利用率</span><strong>{value(gpu.util,'%')}</strong></div><div className="bar"><i style={{width:`${gpu.util??0}%`}}/></div></div><div className="metric memory"><div className="metric-label"><span>显存</span><strong>{gib(gpu.memUsed)}<em> / {gib(gpu.memTotal)} GiB</em></strong></div><div className="bar"><i style={{width:`${Math.min(100,mem??0)}%`}}/></div></div></div>
 </div>
}
function ServerCard({row,now,busy,onEdit,onToggle,onRetry}:{row:ServerSnapshot;now:number;busy:boolean;onEdit:()=>void;onToggle:()=>void;onRetry:()=>void}){
 const old=row.status!=='live';return <article className={`server-card ${row.status==='disabled'?'disabled':''}`}>
  <div className="card-heading"><span className="server-glyph"><ServerIcon size={18}/></span><div className="server-title"><h2 title={row.server.name}>{row.server.name}</h2><span title={row.server.target}>{row.server.target}</span></div><span className={`status ${row.status}`}><i/>{states[row.status]||row.status}</span></div>
  <div className="card-body">{row.gpus.length?row.gpus.map(g=><GPUItem key={g.uuid} gpu={g} stale={old}/>):<div className="card-empty">{row.status==='connecting'?<Loader2 className="spin" size={23}/>:<Cpu size={23}/>}<span>{row.status==='connecting'?'正在读取 GPU 状态':row.status==='disabled'?'启用后继续监控':row.status==='empty'?'服务器未检测到 NVIDIA GPU':'暂时无法获取 GPU 数据'}</span></div>}</div>
  {row.errorMessage&&<div className="connection-error" title={row.errorMessage}>{row.errorMessage}</div>}
  <footer className="card-footer"><span>{row.status==='disabled'?'监控已暂停':relative(row.lastSuccess,now)}{row.nextRetry>now&&<span> · {Math.ceil((row.nextRetry-now)/1000)} 秒后重试</span>}</span><div className="card-actions"><button className="icon-button" title="立即重试" aria-label={`重试 ${row.server.name}`} disabled={busy||!row.server.enabled} onClick={onRetry}><RefreshCw size={13}/></button><button className="icon-button" title={row.server.enabled?'暂停监控':'继续监控'} aria-label={`${row.server.enabled?'暂停':'启用'} ${row.server.name}`} disabled={busy} onClick={onToggle}>{row.server.enabled?<Pause size={13}/>:<Play size={13}/>}</button><button className="icon-button" title="编辑服务器" aria-label={`编辑 ${row.server.name}`} onClick={onEdit}><Pencil size={13}/></button></div></footer>
 </article>
}
export default function App(){
 const api=useRef<Bridge|null>(null)
 const [config,setConfig]=useState<Config>(emptyConfig),[snapshot,setSnapshot]=useState<Snapshot>(emptySnapshot)
 const [mode,setMode]=useState('main'),[now,setNow]=useState(Date.now()),[loading,setLoading]=useState(true),[error,setError]=useState(''),[fatal,setFatal]=useState(''),[busy,setBusy]=useState(false)
 const [modal,setModal]=useState<'add'|'edit'|'settings'|null>(null),[tab,setTab]=useState<'ssh'|'manual'>('ssh')
 const [choices,setChoices]=useState<Choice[]>([]),[selected,setSelected]=useState<string[]>([]),[discovering,setDiscovering]=useState(false),[discoveryError,setDiscoveryError]=useState('')
 const [form,setForm]=useState<Server>({id:'',name:'',target:'',port:0,enabled:true}),[confirmDelete,setConfirmDelete]=useState(false)
 const [systemDark,setSystemDark]=useState(matchMedia('(prefers-color-scheme: dark)').matches)
 const [version,setVersion]=useState('')
 const modalRef=useRef<HTMLDivElement>(null)
 useEffect(()=>{
  let alive=true;let offData=()=>{};let offMode=()=>{}
  bridge().then(async b=>{if(!alive)return;api.current=b
   if(window.go){offData=EventsOn('monitor:snapshot',(s:Snapshot)=>setSnapshot(prev=>s.revision>=prev.revision?s:prev));offMode=EventsOn('window:mode',(m:string)=>setMode(m))}
   const initial=await b.GetInitialState();if(!alive)return;setConfig(initial.config);setSnapshot(prev=>initial.snapshot.revision>=prev.revision?initial.snapshot:prev);setMode(initial.mode);setFatal(initial.error);setVersion(initial.version)
  }).catch(e=>alive&&setFatal(errorText(e))).finally(()=>alive&&setLoading(false))
  const clock=setInterval(()=>setNow(Date.now()),1000);const media=matchMedia('(prefers-color-scheme: dark)');const change=()=>setSystemDark(media.matches);media.addEventListener('change',change)
  return()=>{alive=false;offData();offMode();clearInterval(clock);media.removeEventListener('change',change)}
 },[])
 useEffect(()=>{document.documentElement.dataset.theme=config.theme==='system'?(systemDark?'dark':'light'):config.theme},[config.theme,systemDark])
 useEffect(()=>{
  if(!modal)return
  const previous=document.activeElement as HTMLElement|null
  const timer=setTimeout(()=>modalRef.current?.querySelector<HTMLElement>('input, select, button')?.focus(),0)
  return()=>{clearTimeout(timer);previous?.focus()}
 },[modal])
 useEffect(()=>{const handler=(e:KeyboardEvent)=>{
  if(e.key==='Escape'){if(busy)return;if(modal)setModal(null);else void api.current?.HideWindow()}
  if(e.key==='Tab'&&modal&&modalRef.current){const nodes=Array.from(modalRef.current.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled)'));const first=nodes[0],last=nodes[nodes.length-1];if(e.shiftKey&&document.activeElement===first){e.preventDefault();last?.focus()}else if(!e.shiftKey&&document.activeElement===last){e.preventDefault();first?.focus()}}
 };document.addEventListener('keydown',handler);return()=>document.removeEventListener('keydown',handler)},[modal,busy])
 async function save(next:Config){if(!api.current||busy)return false;setBusy(true);setError('');try{const saved=await api.current.SaveConfiguration(next);setConfig(saved);const fresh=await api.current.GetInitialState();setSnapshot(prev=>fresh.snapshot.revision>=prev.revision?fresh.snapshot:prev);return true}catch(e){setError(errorText(e));return false}finally{setBusy(false)}}
 async function add(){setModal('add');setTab('ssh');setSelected([]);setError('');setDiscoveryError('');setForm({id:'',name:'',target:'',port:0,enabled:true});setDiscovering(true);try{setChoices(await api.current!.ListSSHConfigConnections())}catch(e){setDiscoveryError(errorText(e))}finally{setDiscovering(false)}}
 function edit(s:Server){setForm({...s});setConfirmDelete(false);setError('');setModal('edit')}
 async function submit(){const servers=modal==='edit'?config.servers.map(s=>s.id===form.id?form:s):[...config.servers,...(tab==='ssh'?selected.map(target=>({id:'',name:target,target,port:0,enabled:true})): [form])];if(await save({...config,servers}))setModal(null)}
 async function remove(){if(!confirmDelete){setConfirmDelete(true);return}if(await save({...config,servers:config.servers.filter(s=>s.id!==form.id)}))setModal(null)}
 const rows=config.servers.map(s=>{const row=snapshot.servers.find(r=>r.server.id===s.id);return row?{...row,server:s}:{server:s,status:s.enabled?'connecting':'disabled',gpus:[],lastSuccess:0,nextRetry:0,errorCode:'',errorMessage:''}})
 const mini=mode==='mini'
 const available=choices.filter(c=>!config.servers.some(s=>s.target===c.target&&s.port===0))
 return <div className={`app ${mini?'mini':''}`}>
  <header className="app-header"><div className="brand drag-region"><span className="brand-icon"><Activity size={21}/></span><div><h1>Server Monitor</h1><p>GPU 实时监控</p></div></div><div className="header-actions"><button className="icon-button" aria-label="设置" title="设置" disabled={loading} onClick={()=>{setError('');setModal('settings')}}><Settings2 size={17}/></button>{mini&&<button className="icon-button" title="打开总览" aria-label="打开总览" onClick={()=>void api.current?.ShowMainWindow()}><ArrowUpRight size={18}/></button>}<button className="icon-button" title="隐藏窗口" aria-label="隐藏窗口" onClick={()=>void api.current?.HideWindow()}><X size={18}/></button></div></header>
  <main className="main-scroll">
   <section className="overview"><div><div className="eyebrow">{mini?'全部服务器':'服务器概览'}</div><p><strong>{snapshot.online}</strong><span> / {snapshot.enabled} 台在线</span><i/> <strong>{snapshot.gpuCount}</strong><span> 张 GPU</span></p></div><button className="button primary" onClick={()=>void add()} disabled={loading||!!fatal}><Plus size={16}/><span>{mini?'添加':'添加服务器'}</span></button></section>
   {fatal&&<div className="banner error-banner" role="alert">{fatal}</div>}
   {error&&!modal&&<div className="banner error-banner" role="alert">{error}<button className="icon-button" aria-label="关闭错误" onClick={()=>setError('')}><X size={14}/></button></div>}
   {loading?<div className="empty-state"><Loader2 className="spin"/><h2>正在加载</h2></div>:rows.length===0?<div className="empty-state"><div className="empty-icon"><ServerIcon size={34}/></div><h2>把服务器放在一起看</h2><p>添加已有的 SSH 连接，实时查看每张 GPU 的状态。</p><button className="button primary" disabled={!!fatal} onClick={()=>void add()}><Plus size={16}/>添加第一台服务器</button><span className="empty-note">免密 SSH · 无需远端部署</span></div>:<div className="server-grid">{rows.map(row=><ServerCard key={row.server.id} row={row} now={now} busy={busy} onEdit={()=>edit(row.server)} onToggle={()=>void save({...config,servers:config.servers.map(s=>s.id===row.server.id?{...s,enabled:!s.enabled}:s)})} onRetry={()=>void api.current?.RetryServer(row.server.id).catch(e=>setError(errorText(e)))}/>)}</div>}
  </main>
  <footer className="app-footer"><span><i className={snapshot.online?'live-dot':'quiet-dot'}/>{snapshot.enabled?`每 ${config.interval} 秒刷新`:'等待添加服务器'}</span>{mini?<button onClick={()=>void api.current?.ShowMainWindow()}>打开总览 <ExternalLink size={12}/></button>:<span className="footer-note">本机 SSH · 只读采集</span>}</footer>
  {modal&&<div className="modal-backdrop" onMouseDown={e=>{if(e.target===e.currentTarget&&!busy)setModal(null)}}><div className="modal" ref={modalRef} role="dialog" aria-modal="true" aria-labelledby="dialog-title">
   <div className="modal-heading"><div><h2 id="dialog-title">{modal==='settings'?'设置':modal==='edit'?'编辑服务器':'添加服务器'}</h2><p>{modal==='settings'?'只保留必要的偏好设置':modal==='edit'?'使用本机 SSH 配置连接服务器':'选择本机已有的免密 SSH 连接'}</p></div><button className="icon-button" aria-label="关闭面板" disabled={busy} onClick={()=>setModal(null)}><X size={18}/></button></div>
   {error&&<div className="banner error-banner" role="alert">{error}</div>}
   {modal==='settings'?<div className="settings-content">
    <label className="setting-row"><span><strong>刷新间隔</strong><small>所有已启用的服务器</small></span><select aria-label="刷新间隔" disabled={busy} value={config.interval} onChange={e=>void save({...config,interval:Number(e.target.value)})}>{[1,2,5,10].map(n=><option key={n} value={n}>{n} 秒</option>)}</select></label>
    <div className="setting-row"><span><strong>外观</strong><small>跟随系统或选择主题</small></span><div className="theme-options">{([{id:'system',title:'跟随系统',icon:Monitor},{id:'light',title:'浅色',icon:Sun},{id:'dark',title:'深色',icon:Moon}] as const).map(t=><button key={t.id} title={t.title} aria-label={t.title} aria-pressed={config.theme===t.id} disabled={busy} className={config.theme===t.id?'selected':''} onClick={()=>void save({...config,theme:t.id})}><t.icon size={17}/></button>)}</div></div>
    <div className="about-row"><span>Server Monitor <small>{version}</small></span><span className="attribution">基于 NVSmiBar · MIT</span></div>
    <button className="button quit-button" disabled={busy} onClick={()=>void api.current?.Quit()}><Power size={15}/>退出应用</button>
   </div>:<>
    {modal==='add'&&<div className="tabs"><button className={tab==='ssh'?'selected':''} onClick={()=>setTab('ssh')}>SSH 配置</button><button className={tab==='manual'?'selected':''} onClick={()=>setTab('manual')}>手动添加</button></div>}
    {modal==='add'&&tab==='ssh'?<div className="ssh-list">{discovering?<div className="discovery-empty"><Loader2 size={20} className="spin"/>读取 SSH 配置…</div>:discoveryError?<div className="banner error-banner">{discoveryError}</div>:available.length?available.map(c=><label className="ssh-choice" key={c.target}><input type="checkbox" checked={selected.includes(c.target)} onChange={e=>setSelected(prev=>e.target.checked?[...prev,c.target]:prev.filter(v=>v!==c.target))}/><ServerIcon size={17}/><span>{c.name}</span>{selected.includes(c.target)&&<Check size={15}/>}</label>):<div className="discovery-empty"><ServerIcon size={24}/><span>没有可添加的 SSH 别名</span><small>可切换到“手动添加”，或在 ~/.ssh/config 中配置连接。</small></div>}</div>:<form className="manual-form" onSubmit={e=>{e.preventDefault();void submit()}}><label>SSH 别名或 user@host<input autoComplete="off" aria-label="SSH 目标" placeholder="例如 gpu-lab" value={form.target} onChange={e=>setForm({...form,target:e.target.value})}/></label><div className="form-columns"><label>显示名称<input aria-label="显示名称" placeholder="默认使用 SSH 别名" value={form.name} onChange={e=>setForm({...form,name:e.target.value})}/></label><label>端口 <span>可选</span><input aria-label="端口" type="number" min="1" max="65535" placeholder="使用 SSH 配置" value={form.port||''} onChange={e=>setForm({...form,port:e.target.value?Number(e.target.value):0})}/></label></div><p className="form-hint">使用系统 SSH 的密钥和跳板机配置，无需在应用中填写密码。</p><button type="submit" hidden/></form>}
    <div className="modal-footer">{modal==='edit'?<button className={`button danger ${confirmDelete?'confirm':''}`} disabled={busy} onClick={()=>void remove()}><Trash2 size={14}/>{confirmDelete?'确认移除':'移除'}</button>:<span className="selection-note">{tab==='ssh'?`已选择 ${selected.length} 台`:''}</span>}<div><button className="button secondary" disabled={busy} onClick={()=>setModal(null)}>取消</button><button className="button primary" disabled={busy||(modal==='add'&&tab==='ssh'?selected.length===0:!form.target.trim())} onClick={()=>void submit()}>{busy?<Loader2 size={15} className="spin"/>:modal==='edit'?'保存':'添加并监控'}</button></div></div>
   </>}
  </div></div>}
 </div>
}

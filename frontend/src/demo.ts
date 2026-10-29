// Development-only fixtures. Vite excludes this module from production builds.
import type {Bridge, Config, InitialState, Snapshot} from './api'
let config: Config={version:1,interval:2,theme:'system',servers:[
 {id:'a',name:'训练工作站',target:'gpu-lab-a',port:0,enabled:true},
 {id:'b',name:'推理服务器',target:'gpu-lab-b',port:0,enabled:true},
 {id:'c',name:'备用节点',target:'gpu-lab-c',port:0,enabled:true},
]}
function snapshot(): Snapshot {return {revision:1,timestamp:Date.now(),enabled:config.servers.filter(s=>s.enabled).length,online:config.servers.filter(s=>s.enabled&&s.id!=='c').length,gpuCount:6,servers:config.servers.map((s,i)=>({server:s,status:!s.enabled?'disabled':s.id==='c'?'error':'live',lastSuccess:s.id==='c'?0:Date.now(),nextRetry:0,errorCode:s.id==='c'?'timeout':'',errorMessage:s.id==='c'?'连接超时，请检查网络或 VPN':'',gpus:s.id==='c'?[]:Array.from({length:i===0?4:2},(_,j)=>({uuid:`GPU-${s.id}-${j}`,index:j,name:i===0?'NVIDIA GeForce RTX 4090':'NVIDIA A100-SXM4-80GB',util:[87,32,0,95][j],memUsed:[21820,7520,1024,23400][j],memTotal:i===0?24576:81920,temp:[64,48,36,72][j]}))}))}}
export const demoBridge: Bridge={
 async GetInitialState():Promise<InitialState>{return {config,snapshot:snapshot(),mode:new URLSearchParams(location.search).get('mode')||'main',error:'',version:'0.1.0-dev'}},
 async SaveConfiguration(c){config=structuredClone(c);config.servers=config.servers.map(s=>({...s,id:s.id||crypto.randomUUID()}));return config},
 async ListSSHConfigConnections(){return [{name:'gpu-lab-d',target:'gpu-lab-d'},{name:'gpu-lab-e',target:'gpu-lab-e'}]},
 async RetryServer(){}, async ShowMainWindow(){},async ShowMiniWindow(){},async HideWindow(){},async Quit(){},
}

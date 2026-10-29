export interface Server {
  id: string
  name: string
  target: string
  port: number
  enabled: boolean
}
export interface Config {
  version: number
  servers: Server[]
  interval: number
  theme: 'system' | 'light' | 'dark'
}
export interface GPU {
  uuid: string
  index: number
  name: string
  util: number | null
  memUsed: number | null
  memTotal: number | null
  temp: number | null
}
export interface ServerSnapshot {
  server: Server
  status: string
  gpus: GPU[]
  idleGPUIndices: number[]
  lastSuccess: number
  nextRetry: number
  errorCode: string
  errorMessage: string
}
export interface Snapshot {
  revision: number
  timestamp: number
  servers: ServerSnapshot[]
  online: number
  enabled: number
  gpuCount: number
}
export interface InitialState {
  config: Config
  snapshot: Snapshot
  mode: string
  error: string
  version: string
}
export interface Choice {
  name: string
  target: string
}
export interface Bridge {
  GetInitialState(): Promise<InitialState>
  SaveConfiguration(c: Config): Promise<Config>
  ListSSHConfigConnections(): Promise<Choice[]>
  RetryServer(id: string): Promise<void>
  ShowMainWindow(): Promise<void>
  ShowMiniWindow(): Promise<void>
  HideWindow(): Promise<void>
  Quit(): Promise<void>
}
declare global {
  interface Window {
    go?: { main: { App: Bridge } }
  }
}
export const emptyConfig: Config = { version: 1, servers: [], interval: 2, theme: 'system' }
export const emptySnapshot: Snapshot = {
  revision: 0,
  timestamp: 0,
  servers: [],
  online: 0,
  enabled: 0,
  gpuCount: 0,
}
export async function bridge(): Promise<Bridge> {
  if (window.go?.main?.App) return window.go.main.App
  if (import.meta.env.DEV && new URLSearchParams(location.search).has('demo'))
    return (await import('./demo')).demoBridge
  throw new Error('请在 Server Monitor 桌面应用中打开此界面')
}

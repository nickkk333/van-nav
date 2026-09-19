// 与后端 types 包对应的数据结构定义

export interface Tool {
  id: number
  name: string
  url: string
  logo: string
  catelog: string
  desc: string
  sort: number
  hide: boolean
  default: boolean
}

export interface Catelog {
  id: number
  name: string
  sort: number
  hide: boolean
}

export interface Setting {
  id: number
  favicon: string
  title: string
  govRecord: string
  logo192: string
  logo512: string
  hideAdmin: boolean
  hideGithub: boolean
  hideToggleJumpTarget: boolean
  jumpTargetBlank: boolean
}

export interface SiteConfig {
  id: number
  noImageMode: boolean
  compactMode: boolean
}

export interface Token {
  id: number
  name: string
  value: string
  disabled: number
}

export interface SearchEngine {
  id: number
  name: string
  baseUrl: string
  queryParam: string
  logo: string
  sort: number
  enabled: boolean
}

export interface User {
  id: number
  name: string
  password?: string
}

export interface ApiResult<T = any> {
  success: boolean
  message?: string
  errorMessage?: string
  data: T
}

// 前台首页数据
export interface HomeData {
  tools: Tool[]
  catelogs: string[]
  setting: Setting
  siteConfig: SiteConfig
}

// 管理后台数据
export interface AdminData {
  tools: Tool[]
  catelogs: Catelog[]
  setting: Setting
  siteConfig: SiteConfig
  user: User
  tokens: Token[]
}

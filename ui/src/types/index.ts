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

// 抓取网址得到的信息（后台添加工具时自动填充用）
export interface UrlInfo {
  name: string
  title: string
  description: string
  logo: string
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
  /** 首页背景图地址（后台上传返回的 url 或外链地址），留空表示不展示背景图 */
  backgroundImage: string
}

export interface SiteConfig {
  id: number
  noImageMode: boolean
  compactMode: boolean
  /** 首页每行展示的网站数量 */
  cardsPerRow: number
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

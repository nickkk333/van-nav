// 与后端 types 包对应的数据结构定义

export interface Tool {
  id: number
  name: string
  url: string
  /** 图标网址：保存可以下载到图片的 url 地址 */
  logo: string
  /** 图标图片名：图片下载保存到 data 目录（data/images）后的文件名，前台优先按它读本地图片 */
  logoName: string
  /** 图标占位字符：图标缺失或加载失败时显示的字符，只在前台虚拟卡片上使用（如搜索引擎卡片取引擎名），缺省时用 name 的首字符 */
  logoText?: string
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
  /** 该分类下的工具是否展示在主页默认栏 */
  default: boolean
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
  /** 首页背景图地址（后台上传返回的 url 或外链地址），留空时前台使用必应每日壁纸作为背景 */
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

// 导入导出的备份数据：所有工具、分类、搜索引擎与 api token
// 图标只保存网址（不包含图片内容），导入后服务端会按网址自动获取图片
export interface BackupData {
  /** 备份文件格式版本 */
  version: number
  /** 导出时间（RFC3339） */
  exportedAt?: string
  tools: Tool[]
  catelogs: Catelog[]
  searchEngines: SearchEngine[]
  apiTokens: Token[]
}

/** 导入结果：各类数据导入的条数 */
export interface BackupImportResult {
  tools: number
  catelogs: number
  searchEngines: number
  apiTokens: number
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

// 与后端 types 包对应的数据结构定义

export interface Tool {
  id: number
  name: string
  url: string
  /** 内网网址：选填，留空表示没有内网地址；首页点击卡片时优先访问它，检测不通时回退到 url */
  internalUrl: string
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
  /** 前台首页是否隐藏便签入口（类似 Github 按钮开关，默认显示） */
  hideMemo: boolean
  jumpTargetBlank: boolean
  /** 首页背景图地址（后台上传返回的 url 或外链地址），留空时前台使用必应每日壁纸作为背景 */
  backgroundImage: string
  /** 默认图标地址（后台上传返回的 url 或外链地址，相当于替换内置的 default.png）：
   *  工具/搜索引擎没有自己的图标时前台显示它，留空时前台用名称首字符占位 */
  defaultLogo: string
  /** 默认搜索引擎 id：回车无匹配卡片或按 Ctrl+Enter 时使用；0 表示自动（第一个启用的引擎） */
  defaultSearchEngine: number
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

// 便签：单文档模式（多端通过后端同步）
export interface Memo {
  id: number
  content: string
  mode: 'simple' | 'rich'
  updatedAt: number
}

// 便签历史版本
export interface MemoVersion {
  id: number
  content: string
  mode: 'simple' | 'rich'
  createdAt: number
}

// 便签附件（上传的文件 / 粘贴的图片）
export interface MemoFile {
  /** 服务端保存的文件名（附件地址的最后一段） */
  name: string
  /** 可直接访问的附件地址 */
  url: string
  /** 上传时的原始文件名（附件卡片上显示） */
  original: string
  /** 文件大小（字节） */
  size: number
  /** 是否图片：图片在便签里内联显示，其它文件显示「图标 + 文件名」卡片 */
  isImage: boolean
}

// 导入导出的备份数据：所有工具、分类、搜索引擎与便签
// 图标只保存网址（不包含图片内容），导入后服务端会按网址自动获取图片
export interface BackupData {
  /** 备份文件格式版本 */
  version: number
  /** 导出时间（RFC3339） */
  exportedAt?: string
  tools: Tool[]
  catelogs: Catelog[]
  searchEngines: SearchEngine[]
  memo?: Memo | null
  memoVersions?: MemoVersion[]
}

/** 导入结果：各类数据导入的条数（api token 不参与导入，故无此项） */
export interface BackupImportResult {
  tools: number
  catelogs: number
  searchEngines: number
  memoVersions?: number
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

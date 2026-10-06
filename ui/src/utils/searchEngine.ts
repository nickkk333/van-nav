import { fetchGetDefaultSearchEngine, fetchGetEnabledSearchEngines } from '../api'
import type { SearchEngine, Tool } from '../types'

// 搜索引擎缓存
let searchEnginesCache: SearchEngine[] = []
let cacheExpiry = 0
const CACHE_DURATION = 5 * 60 * 1000 // 5 分钟缓存

const defaultEngines: SearchEngine[] = [
  {
    id: 1,
    name: '百度',
    baseUrl: 'https://www.baidu.com/s',
    queryParam: 'wd',
    logo: '/baidu.ico',
    sort: 1,
    enabled: true,
  },
  {
    id: 2,
    name: 'Bing',
    baseUrl: 'https://cn.bing.com/search',
    queryParam: 'q',
    logo: '/bing.ico',
    sort: 2,
    enabled: true,
  },
  {
    id: 3,
    name: 'Google',
    baseUrl: 'https://www.google.com/search',
    queryParam: 'q',
    logo: '/google.ico',
    sort: 3,
    enabled: true,
  },
]

/** 获取启用的搜索引擎（带缓存） */
const getEnabledSearchEngines = async (): Promise<SearchEngine[]> => {
  const now = Date.now()
  if (searchEnginesCache.length > 0 && now < cacheExpiry) {
    return searchEnginesCache
  }
  try {
    searchEnginesCache = await fetchGetEnabledSearchEngines()
    cacheExpiry = now + CACHE_DURATION
  } catch (error) {
    console.error('获取搜索引擎失败，使用默认配置:', error)
    searchEnginesCache = defaultEngines
    cacheExpiry = now + CACHE_DURATION
  }
  return searchEnginesCache
}

const generateSearchUrl = (baseUrl: string, queryParam: string, searchString: string) => {
  const separator = baseUrl.includes('?') ? '&' : '?'
  return `${baseUrl}${separator}${queryParam}=${encodeURIComponent(searchString)}`
}

/** 根据搜索关键字生成搜索引擎卡片 */
export const generateSearchEngineCards = async (searchString: string): Promise<Tool[]> => {
  const keyword = searchString.trim()
  if (!keyword) {
    return []
  }
  try {
    const engines = await getEnabledSearchEngines()
    return engines
      .filter((engine) => engine.enabled)
      .sort((a, b) => a.sort - b.sort)
      .map((engine) => ({
        id: 8800880000 + engine.id, // 使用特定 ID 前缀避免与工具冲突
        name: `使用 ${engine.name} 搜索`,
        url: generateSearchUrl(engine.baseUrl, engine.queryParam, keyword),
        desc: `在 ${engine.name} 中搜索 「${keyword}」`,
        logo: engine.logo,
        // 虚拟卡片没有本机图片名，直接按搜索引擎的 logo 地址显示
        logoName: '',
        // 图标缺失或加载失败时用引擎名首字符占位（卡片名是「使用 xxx 搜索」，直接用会一直是「使」）
        logoText: engine.name,
        catelog: '默认',
        sort: engine.sort,
        hide: false,
        default: false,
      }))
  } catch (error) {
    console.error('生成搜索引擎卡片失败:', error)
    return []
  }
}

// 解析后的「默认搜索引擎」缓存（来自公开接口 /searchEngines/default，后端按设置解析）
// 回车（考虑启用）与 Ctrl+Enter（不考虑启用）两种解析分别缓存
let enterEngineCache: SearchEngine | null = null
let enterEngineExpiry = 0
let ctrlEngineCache: SearchEngine | null = null
let ctrlEngineExpiry = 0

/** 获取解析后的默认搜索引擎（带缓存）：后端按后台设置解析，未登录也能用 */
const getDefaultEngine = async (ignoreEnabled: boolean): Promise<SearchEngine> => {
  const now = Date.now()
  if (ignoreEnabled) {
    if (ctrlEngineCache && now < ctrlEngineExpiry) return ctrlEngineCache
  } else {
    if (enterEngineCache && now < enterEngineExpiry) return enterEngineCache
  }
  let engine: SearchEngine
  try {
    engine = await fetchGetDefaultSearchEngine(ignoreEnabled)
  } catch (error) {
    console.error('获取默认搜索引擎失败，使用内置百度:', error)
    engine = defaultEngines[0]
  }
  if (ignoreEnabled) {
    ctrlEngineCache = engine
    ctrlEngineExpiry = now + CACHE_DURATION
  } else {
    enterEngineCache = engine
    enterEngineExpiry = now + CACHE_DURATION
  }
  return engine
}

/**
 * 解析应使用哪个搜索引擎作为「默认」，返回它对给定关键词的搜索地址；keyword 为空时返回 null。
 * 解析交由后端公开接口 /searchEngines/default 完成：
 * - ignoreEnabled=false（回车）：自动优先第一个启用的、无启用则用所有已存在引擎的第一个；特定引擎不论启用与否直接用选中的；
 * - ignoreEnabled=true（Ctrl+Enter）：不考虑是否有启用的，自动用所有已存在引擎的第一个、特定用选中的。
 */
export const getDefaultSearchEngineUrl = async (keyword: string, ignoreEnabled = false): Promise<string | null> => {
  const k = keyword.trim()
  if (!k) return null
  const engine = await getDefaultEngine(ignoreEnabled)
  return generateSearchUrl(engine.baseUrl, engine.queryParam, k)
}

/** 清除缓存（管理员修改搜索引擎配置后可调用） */
export const clearSearchEngineCache = () => {
  searchEnginesCache = []
  cacheExpiry = 0
  enterEngineCache = null
  enterEngineExpiry = 0
  ctrlEngineCache = null
  ctrlEngineExpiry = 0
}
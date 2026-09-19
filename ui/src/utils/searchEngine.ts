import { fetchGetEnabledSearchEngines } from '../api'
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
    logo: 'baidu.ico',
    sort: 1,
    enabled: true,
  },
  {
    id: 2,
    name: 'Bing',
    baseUrl: 'https://cn.bing.com/search',
    queryParam: 'q',
    logo: 'bing.ico',
    sort: 2,
    enabled: true,
  },
  {
    id: 3,
    name: 'Google',
    baseUrl: 'https://www.google.com/search',
    queryParam: 'q',
    logo: 'google.ico',
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

/** 清除缓存（管理员修改搜索引擎配置后可调用） */
export const clearSearchEngineCache = () => {
  searchEnginesCache = []
  cacheExpiry = 0
}
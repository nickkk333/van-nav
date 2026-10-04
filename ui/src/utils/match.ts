import { ref } from 'vue'

/**
 * pinyin-pro 按需懒加载：字典约 400KB（含 dist 分包与 match 分包），首屏不需要，
 * 只有用户真正输入中文/拼音搜索时才加载，避免进首屏主包。
 *
 * multiSearch 保持同步签名（首页 filteredData / 后台 filteredTools 都是同步 computed，
 * 改异步要重构两处调用方，成本高风险大）：拼音分包未就绪前先用原生包含匹配顶着，
 * 分包就绪后通过 pinyinReady 触发 computed 重新计算，补上拼音匹配结果。
 */

/** 拼音分包是否就绪：调用方在 computed 里读取一次即可建立依赖，分包加载完自动重算 */
export const pinyinReady = ref(false)

type PinyinMatchFn = (source: string, target: string) => unknown

let pinyinMatchFn: PinyinMatchFn | null = null
let loadPromise: Promise<PinyinMatchFn | null> | null = null

/** 触发拼音分包加载（幂等，多次调用共用同一个 import），空闲时预加载、首次搜索时兜底加载 */
export const ensurePinyinLoaded = (): Promise<PinyinMatchFn | null> => {
  if (pinyinMatchFn) {
    return Promise.resolve(pinyinMatchFn)
  }
  if (!loadPromise) {
    loadPromise = import('pinyin-pro')
      .then((m) => {
        pinyinMatchFn = m.match as PinyinMatchFn
        pinyinReady.value = true
        return pinyinMatchFn
      })
      .catch(() => {
        loadPromise = null
        return null
      })
  }
  return loadPromise
}

// 模块加载后空闲时预热：用户大概率会搜索，提前拉分包但不阻塞首屏渲染
if (typeof window !== 'undefined') {
  const preload = () => ensurePinyinLoaded()
  const ric = (window as unknown as { requestIdleCallback?: (cb: () => void, opts?: { timeout: number }) => void }).requestIdleCallback
  if (typeof ric === 'function') {
    ric.call(window, preload, { timeout: 3000 })
  } else {
    window.setTimeout(preload, 1500)
  }
}

/**
 * 同时支持原生包含和拼音匹配的模糊搜索
 * 例如：输入 "bd" 可以匹配到 "百度"（拼音分包就绪后生效，就绪前按原生包含匹配）
 */
export const multiSearch = (source?: string | null, target?: string | null): boolean => {
  if (!source || !target) {
    return false
  }
  const s = String(source).toLowerCase()
  const t = String(target).toLowerCase().trim()
  if (!t) {
    return true
  }
  if (s.includes(t)) {
    return true
  }
  if (pinyinMatchFn) {
    try {
      return Boolean(pinyinMatchFn(s, t))
    } catch {
      return false
    }
  }
  // 拼音分包还没回来：后台触发加载（就绪后 pinyinReady 翻转，computed 自动重算补上结果）
  void ensurePinyinLoaded()
  return false
}
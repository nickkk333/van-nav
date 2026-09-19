import { match as pinyinMatch } from 'pinyin-pro'

/**
 * 同时支持原生包含和拼音匹配的模糊搜索
 * 例如：输入 "bd" 可以匹配到 "百度"
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
  try {
    return Boolean(pinyinMatch(s, t))
  } catch {
    return false
  }
}
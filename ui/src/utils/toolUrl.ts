import { getJumpTarget } from './setting'

/**
 * 工具卡片的跳转地址处理：
 * 卡片配置了内网网址时，点击先检测内网网址是否可达，可达就打开内网网址，不可达（超时 / 连不上）回退到外网网址。
 */

/** 内网网址检测的超时时间（毫秒）：内网一般很快，等太久会让点击有明显延迟 */
const PROBE_TIMEOUT = 1200

/** 检测结果的缓存时间（毫秒）：避免每次点击都重新探测，内网状态短暂变化也能在过期后重新判断 */
const PROBE_TTL = 60 * 1000

/** 规范化用户填写的内网网址：去掉首尾空格，缺少协议时补 http://（内网地址基本是 http） */
export const normalizeInternalUrl = (raw?: string | null): string => {
  const text = String(raw ?? '').trim()
  if (!text) {
    return ''
  }
  if (/^https?:\/\//i.test(text)) {
    return text
  }
  if (text.startsWith('//')) {
    return `http:${text}`
  }
  return `http://${text}`
}

type ReachabilityCache = { reachable: boolean; at: number }

const reachabilityCache = new Map<string, ReachabilityCache>()
/** 正在探测中的请求：同一地址被同时点击 / 悬停预热时复用同一个 Promise */
const pendingProbes = new Map<string, Promise<boolean>>()

/** 取出缓存里的检测结果；没有结果或已过期时返回 undefined */
export const getCachedInternalReachability = (raw?: string | null): boolean | undefined => {
  const target = normalizeInternalUrl(raw)
  if (!target) {
    return undefined
  }
  const cached = reachabilityCache.get(target)
  if (!cached) {
    return undefined
  }
  if (Date.now() - cached.at > PROBE_TTL) {
    reachabilityCache.delete(target)
    return undefined
  }
  return cached.reachable
}

/**
 * 检测内网网址是否可达。
 * 用 no-cors 的 HEAD 请求探测：只要能拿到响应（哪怕状态码是 401/404）就算可达，
 * 连不上 / DNS 解析失败 / 超时都算不可达。no-cors 响应内容读不到，这里也只关心能不能连上。
 */
export const probeInternalUrl = (raw?: string | null): Promise<boolean> => {
  const target = normalizeInternalUrl(raw)
  if (!target) {
    return Promise.resolve(false)
  }
  const cached = getCachedInternalReachability(target)
  if (cached !== undefined) {
    return Promise.resolve(cached)
  }
  // 页面是 https 而内网是 http 时浏览器会按混合内容直接拦掉，探测必然失败，直接当作不可达（回退外网网址）
  if (window.location.protocol === 'https:' && target.startsWith('http://')) {
    reachabilityCache.set(target, { reachable: false, at: Date.now() })
    return Promise.resolve(false)
  }
  const pending = pendingProbes.get(target)
  if (pending) {
    return pending
  }
  const probe = new Promise<boolean>((resolve) => {
    const finish = (reachable: boolean) => {
      reachabilityCache.set(target, { reachable, at: Date.now() })
      pendingProbes.delete(target)
      resolve(reachable)
    }
    if (typeof fetch !== 'function' || typeof AbortController !== 'function') {
      finish(false)
      return
    }
    const controller = new AbortController()
    const timer = window.setTimeout(() => {
      controller.abort()
      finish(false)
    }, PROBE_TIMEOUT)
    fetch(target, {
      method: 'HEAD',
      mode: 'no-cors',
      cache: 'no-store',
      credentials: 'omit',
      redirect: 'follow',
      signal: controller.signal,
    }).then(
      () => {
        window.clearTimeout(timer)
        finish(true)
      },
      () => {
        window.clearTimeout(timer)
        finish(false)
      }
    )
  })
  pendingProbes.set(target, probe)
  return probe
}

/** 悬停预热：鼠标移到卡片上时先探测一次，点击时通常已有结果，可以直接跳转（异步跳转会丢失用户手势） */
export const prefetchInternalReachability = (raw?: string | null) => {
  const target = normalizeInternalUrl(raw)
  if (!target || getCachedInternalReachability(target) !== undefined) {
    return
  }
  void probeInternalUrl(target)
}

/** 按「内网优先，不通回退外网」的规则算出最终要打开的地址；没有内网网址时直接返回外网网址 */
export const resolveToolTargetUrl = async (url?: string | null, internalUrl?: string | null): Promise<string> => {
  const external = String(url ?? '').trim()
  const internal = normalizeInternalUrl(internalUrl)
  if (!internal) {
    return external
  }
  const reachable = await probeInternalUrl(internal)
  if (reachable) {
    return internal
  }
  return external || internal
}

/** 用当前的跳转方式打开地址：新标签页被浏览器拦截时退化为原地跳转 */
export const openToolUrl = (url?: string | null) => {
  const target = String(url ?? '').trim()
  if (!target) {
    return
  }
  if (getJumpTarget() !== 'blank') {
    window.location.href = target
    return
  }
  const opened = window.open(target, '_blank')
  if (opened) {
    opened.opener = null
    return
  }
  window.location.href = target
}

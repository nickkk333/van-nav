import { ref } from 'vue'

/**
 * 首页配色与显示模式是两件独立的事：
 * - ColorTheme（自动 / 浅色 / 深色）由管理页「系统配置」下发，本地只缓存用于首屏
 * - HomeMode（仅搜索 / 卡片搜索）由首页右下角按钮切换，属于设备端偏好
 * 旧版把「仅显示搜索框」塞进主题四态循环，改配色会连带改变首页布局，这里彻底拆开。
 */
export type ColorTheme = 'auto' | 'light' | 'dark'
export type HomeMode = 'search' | 'card'

const COLOR_THEME_KEY = 'colorTheme'
const HOME_MODE_KEY = 'homeMode'
/** 旧版主题 key：值可能是 auto/light/dark/search，用于平滑迁移 */
const LEGACY_THEME_KEY = 'theme'

/** 卡片搜索模式下，鼠标移动后卡片与分类名的持续显示时间（秒） */
export const DEFAULT_CARD_SHOW_DURATION = 5
export const MIN_CARD_SHOW_DURATION = 1
export const MAX_CARD_SHOW_DURATION = 60

/** 根据当前时间与系统偏好推断自动主题 */
export const decodeAuto = (): 'auto-dark' | 'auto-light' => {
  const hour = new Date().getHours()
  const night = hour > 18 || hour < 8
  if (typeof window === 'undefined') {
    return night ? 'auto-dark' : 'auto-light'
  }
  if (night || window.matchMedia('(prefers-color-scheme: dark)').matches) {
    return 'auto-dark'
  }
  return 'auto-light'
}

export const decodeTheme = (t: ColorTheme): string => (t === 'auto' ? decodeAuto() : t)

/** 应用配色：使用 element-plus 约定的 html.dark 类名 */
export const applyTheme = (t: string) => {
  document.documentElement.classList.toggle('dark', t.includes('dark'))
}

const readStorage = (key: string): string => {
  try {
    return localStorage.getItem(key) || ''
  } catch {
    // 隐私模式等读不到：回落默认值
    return ''
  }
}

const writeStorage = (key: string, value: string) => {
  try {
    localStorage.setItem(key, value)
  } catch {
    // 隐私模式等存不了：忽略
  }
}

export const getColorTheme = (): ColorTheme => {
  const saved = readStorage(COLOR_THEME_KEY)
  if (saved === 'auto' || saved === 'light' || saved === 'dark') return saved
  // 旧版把主题存成 4 态；'search' 是布局模式，配色按自动处理
  const legacy = readStorage(LEGACY_THEME_KEY)
  if (legacy === 'auto' || legacy === 'light' || legacy === 'dark') return legacy
  return 'auto'
}

/** 当前配色主题（响应式）；首页加载服务端设置后会用 setColorTheme 覆盖 */
export const colorTheme = ref<ColorTheme>(getColorTheme())

/** 应用配色主题：服务端设置下发、管理页保存后调用，顺带更新本地首屏缓存 */
export const setColorTheme = (theme: string) => {
  const value: ColorTheme = theme === 'light' || theme === 'dark' ? theme : 'auto'
  colorTheme.value = value
  writeStorage(COLOR_THEME_KEY, value)
  applyTheme(decodeTheme(value))
}

export const getHomeMode = (): HomeMode => {
  const saved = readStorage(HOME_MODE_KEY)
  if (saved === 'search' || saved === 'card') return saved
  // 旧版「仅显示搜索框」对应新的「仅搜索」模式
  if (readStorage(LEGACY_THEME_KEY) === 'search') return 'search'
  return 'card'
}

/** 当前首页模式（响应式）：仅搜索 / 卡片搜索 */
export const homeMode = ref<HomeMode>(getHomeMode())

export const setHomeMode = (mode: HomeMode) => {
  homeMode.value = mode
  writeStorage(HOME_MODE_KEY, mode)
}

export const nextHomeMode = (mode: HomeMode): HomeMode => (mode === 'search' ? 'card' : 'search')

export const homeModeLabel = (mode: HomeMode) => (mode === 'search' ? '仅搜索' : '卡片搜索')

/** 初始化配色：首屏先用本地缓存，首页拿到服务端设置后会再覆盖一次 */
export const initTheme = () => {
  applyTheme(decodeTheme(getColorTheme()))
  if (typeof window === 'undefined') return
  window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
    if (colorTheme.value === 'auto') applyTheme(decodeAuto())
  })
}

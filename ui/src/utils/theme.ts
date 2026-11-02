import { ref } from 'vue'

export type ThemeMode = 'auto' | 'light' | 'dark' | 'search'

const THEME_KEY = 'theme'

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

/** 解析主题：'search'（仅显示搜索框）使用自动主题 */
export const decodeTheme = (t: ThemeMode): string => {
  const effective = t === 'search' ? 'auto' : t
  return effective === 'auto' ? decodeAuto() : effective
}

/** 应用主题：使用 element-plus 约定的 html.dark 类名 */
export const applyTheme = (t: string) => {
  const el = document.documentElement
  el.classList.toggle('dark', t.includes('dark'))
}

export const getThemeMode = (): ThemeMode => {
  try {
    const saved = localStorage.getItem(THEME_KEY)
    if (saved === 'dark' || saved === 'light' || saved === 'auto' || saved === 'search') {
      return saved
    }
  } catch {
    /* 隐私模式等读不到：回落 auto */
  }
  return 'auto'
}

/** 当前主题模式（响应式）：DarkSwitch 据此渲染图标，HomeView 据此判断「仅显示搜索框」 */
export const themeMode = ref<ThemeMode>(getThemeMode())

export const setThemeMode = (mode: ThemeMode) => {
  themeMode.value = mode
  try {
    localStorage.setItem(THEME_KEY, mode)
  } catch {
    /* 隐私模式等存不了：忽略 */
  }
  applyTheme(decodeTheme(mode))
  return decodeTheme(mode)
}

export const nextThemeMode = (mode: ThemeMode): ThemeMode => {
  if (mode === 'auto') return 'light'
  if (mode === 'light') return 'dark'
  if (mode === 'dark') return 'search'
  return 'auto'
}

export const themeModeLabel = (mode: ThemeMode) => {
  if (mode === 'light') return '浅色'
  if (mode === 'dark') return '深色'
  if (mode === 'search') return '仅显示搜索框'
  return '自动'
}

/** 初始化主题，并在自动模式下监听系统/时间变化 */
export const initTheme = () => {
  const mode = getThemeMode()
  applyTheme(decodeTheme(mode))
  if ((mode === 'auto' || mode === 'search') && typeof window !== 'undefined') {
    window
      .matchMedia('(prefers-color-scheme: dark)')
      .addEventListener('change', () => applyTheme(decodeTheme(getThemeMode())))
  }
}

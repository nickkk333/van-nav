export type ThemeMode = 'auto' | 'light' | 'dark'

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

export const decodeTheme = (t: ThemeMode): string => {
  return t === 'auto' ? decodeAuto() : t
}

/** 应用主题：使用 element-plus 约定的 html.dark 类名 */
export const applyTheme = (t: string) => {
  const el = document.documentElement
  el.classList.toggle('dark', t.includes('dark'))
}

export const getThemeMode = (): ThemeMode => {
  const saved = localStorage.getItem(THEME_KEY)
  if (saved === 'dark' || saved === 'light' || saved === 'auto') {
    return saved
  }
  return 'auto'
}

export const setThemeMode = (mode: ThemeMode) => {
  localStorage.setItem(THEME_KEY, mode)
  applyTheme(decodeTheme(mode))
  return decodeTheme(mode)
}

export const nextThemeMode = (mode: ThemeMode): ThemeMode => {
  if (mode === 'auto') return 'light'
  if (mode === 'light') return 'dark'
  return 'auto'
}

export const themeModeLabel = (mode: ThemeMode) => {
  if (mode === 'light') return '浅色'
  if (mode === 'dark') return '深色'
  return '自动'
}

/** 初始化主题，并在自动模式下监听系统/时间变化 */
export const initTheme = () => {
  const mode = getThemeMode()
  applyTheme(decodeTheme(mode))
  if (mode === 'auto' && typeof window !== 'undefined') {
    window
      .matchMedia('(prefers-color-scheme: dark)')
      .addEventListener('change', () => applyTheme(decodeTheme(getThemeMode())))
  }
}

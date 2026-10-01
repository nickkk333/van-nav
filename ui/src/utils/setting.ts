export type JumpTarget = 'blank' | 'self'

/** 首页每行展示的网站数量默认值（与后端 service.DefaultCardsPerRow 保持一致） */
export const DEFAULT_CARDS_PER_ROW = 5

/** 首页每行展示的网站数量上限（与后端 service.MaxCardsPerRow 保持一致） */
export const MAX_CARDS_PER_ROW = 12

/**
 * 首页默认背景图：未在后台配置背景图时使用的必应每日壁纸
 * 图片由服务端启动时下载保存到 data 目录（必应壁纸.jpg），此处地址与后端路由 /api/bingWallpaper 保持一致
 */
export const DEFAULT_BACKGROUND_IMAGE = '/api/bingWallpaper'

const JUMP_TARGET_KEY = 'jumpTarget'
const INITED_KEY = 'initedServerJumpTarget'

export const setJumpTarget = (target: JumpTarget) => {
  window.localStorage.setItem(JUMP_TARGET_KEY, target)
}

export const getJumpTarget = (): JumpTarget => {
  return (window.localStorage.getItem(JUMP_TARGET_KEY) as JumpTarget) || 'blank'
}

/** 切换跳转方式（原地跳转 / 新标签页） */
export const toggleJumpTarget = () => {
  const current = getJumpTarget()
  setJumpTarget(current === 'blank' ? 'self' : 'blank')
  return getJumpTarget()
}

/** 首次进入时使用服务端配置初始化跳转方式 */
export const initServerJumpTargetConfig = (setting: { jumpTargetBlank?: boolean } | null | undefined) => {
  if (window.localStorage.getItem(INITED_KEY)) {
    return
  }
  window.localStorage.setItem(INITED_KEY, 'true')
  if (!setting || setting.jumpTargetBlank === undefined || setting.jumpTargetBlank === true) {
    setJumpTarget('blank')
  } else {
    setJumpTarget('self')
  }
}

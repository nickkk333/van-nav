export const isLogin = () => {
  return Boolean(localStorage.getItem('_token'))
}

/** 网络图片走后端代理，避免跨域和加载慢 */
export const getLogoUrl = (url: string) => {
  if (!url) {
    return ''
  }
  if (url.startsWith('http')) {
    return `/api/img?url=${url}`
  }
  return url
}

/** 空分类统一显示为“未分类” */
export const displayCatelog = (catelog?: string | null) => {
  if (catelog === null || catelog === undefined || String(catelog).trim() === '') {
    return '未分类'
  }
  return catelog
}

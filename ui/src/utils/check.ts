export const isLogin = () => {
  return Boolean(localStorage.getItem('_token'))
}

/** 默认图标：前端公共资源 default.png（源文件在 ui/public/default.png，构建时拷贝到 ./public 供 go:embed 内嵌） */
export const DEFAULT_LOGO_URL = `${import.meta.env.BASE_URL}default.png`

/** 网络图片走后端代理，避免跨域和加载慢；url 需要转义，否则带 ?a=1&b=2 的地址会被 & 截断
 *  logo 为空时返回默认图标 default.png */
export const getLogoUrl = (url: string) => {
  if (!url) {
    return DEFAULT_LOGO_URL
  }
  if (url.startsWith('http')) {
    return `/api/img?url=${encodeURIComponent(url)}`
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

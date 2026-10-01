import type { Tool } from '../types'

export const isLogin = () => {
  return Boolean(localStorage.getItem('_token'))
}

/** 默认图标：前端公共资源 default.png（源文件在 ui/public/default.png，构建时拷贝到 ./public 供 go:embed 内嵌） */
export const DEFAULT_LOGO_URL = `${import.meta.env.BASE_URL}default.png`

/** 本机图片的访问前缀（图片保存在 data/images 下），与后端 service.UploadUrlPrefix 保持一致 */
export const UPLOADED_IMAGE_PREFIX = '/api/uploadedImage/'

/** 本地图片地址：图片名指向 data 目录（data/images）里的图片文件 */
export const getLocalLogoUrl = (logoName: string) => `${UPLOADED_IMAGE_PREFIX}${logoName}`

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

/** 工具图标地址：优先读保存到本机的图片（logoName 指向 data 目录里的文件），
 *  其次按图标网址（logo）显示，都没有时用默认图标 default.png */
export const getToolLogoUrl = (tool: Pick<Tool, 'logo' | 'logoName'>) => {
  const logoName = (tool.logoName || '').trim()
  if (logoName) {
    return getLocalLogoUrl(logoName)
  }
  return getLogoUrl(tool.logo)
}

/** 空分类统一显示为“未分类” */
export const displayCatelog = (catelog?: string | null) => {
  if (catelog === null || catelog === undefined || String(catelog).trim() === '') {
    return '未分类'
  }
  return catelog
}

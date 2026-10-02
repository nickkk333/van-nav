import { ref } from 'vue'
import type { Tool } from '../types'

export const isLogin = () => {
  return Boolean(localStorage.getItem('_token'))
}

/** 默认图标：前端公共资源 default.png（源文件在 ui/public/default.png，构建时拷贝到 ./public 供 go:embed 内嵌） */
export const DEFAULT_LOGO_URL = `${import.meta.env.BASE_URL}default.png`

/**
 * 后台上传的自定义默认图标（设置里的 defaultLogo）：上传后相当于替换内置的 default.png，
 * 工具/搜索引擎没有自己的图标时前台显示它；没有上传时前台用名称首字符占位（LogoFallback）
 * 用 ref 存放，设置加载完成后已经渲染出来的卡片会自动更新
 */
export const customDefaultLogo = ref('')

/** 设置加载后同步自定义默认图标地址（空串表示前台改用名称首字符占位） */
export const setCustomDefaultLogo = (url?: string | null) => {
  customDefaultLogo.value = (url || '').trim()
}

/** 本机图片的访问前缀（图片保存在 data/images 下），与后端 service.UploadUrlPrefix 保持一致 */
export const UPLOADED_IMAGE_PREFIX = '/api/uploadedImage/'

/** 本地图片地址：图片名指向 data 目录（data/images）里的图片文件 */
export const getLocalLogoUrl = (logoName: string) => `${UPLOADED_IMAGE_PREFIX}${logoName}`

/** 网络图片走后端代理，避免跨域和加载慢；url 需要转义，否则带 ?a=1&b=2 的地址会被 & 截断
 *  logo 为空时返回默认图标 default.png；图片后端取不到（缓存里没有，例如外链 404）时接口返回 404，前台 <img> 触发 error 后用名称首字符占位 */
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
 *  其次按图标网址（logo）显示，都没有时返回默认图标 default.png（前台会用名称首字符占位，见 hasToolLogo） */
export const getToolLogoUrl = (tool: Pick<Tool, 'logo' | 'logoName'>) => {
  const logoName = (tool.logoName || '').trim()
  if (logoName) {
    return getLocalLogoUrl(logoName)
  }
  return getLogoUrl(tool.logo)
}

/** 是否配置了图标：本机图片名（logoName）或图标网址（logo）任一不为空即视为有图标；
 *  两者都为空时前台不再显示默认图标 default.png，改用名称首字符占位（LogoFallback） */
export const hasToolLogo = (tool: Pick<Tool, 'logo' | 'logoName'>) =>
  Boolean((tool.logoName || '').trim()) || Boolean((tool.logo || '').trim())

/** 空分类统一显示为“未分类” */
export const displayCatelog = (catelog?: string | null) => {
  if (catelog === null || catelog === undefined || String(catelog).trim() === '') {
    return '未分类'
  }
  return catelog
}

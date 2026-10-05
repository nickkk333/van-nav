import axios from 'axios'
import type { AdminData, ApiResult, BackupData, BackupImportResult, Catelog, HomeData, SearchEngine, Token, Tool, UrlInfo, User } from '../types'

export const http = axios.create({
  baseURL: '/api',
  timeout: 30000,
})

// 请求拦截器：带上登录凭证
http.interceptors.request.use((config) => {
  const token = window.localStorage.getItem('_token')
  if (token) {
    config.headers.Authorization = token
  }
  return config
})

// 响应拦截器：登录失效时回到登录页
http.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error?.response?.status === 401) {
      window.localStorage.removeItem('_token')
      if (window.location.pathname !== '/login') {
        window.location.href = '/login'
      }
    }
    return Promise.reject(error)
  }
)

/** 把未知异常转成可读信息 */
export const resolveError = (err: unknown, fallback = '请求失败'): string => {
  if (axios.isAxiosError(err)) {
    const msg = (err.response?.data as { errorMessage?: string } | undefined)?.errorMessage
    if (msg) return msg
    return err.message || fallback
  }
  if (err instanceof Error && err.message) {
    return err.message
  }
  return fallback
}

// ==================== 前台接口 ====================

/** 获取前台全部数据 */
export const fetchList = async (): Promise<HomeData> => {
  const { data } = await http.get<ApiResult<HomeData>>('/')
  return data.data
}

/** 获取启用的搜索引擎（前台搜索用） */
export const fetchGetEnabledSearchEngines = async (): Promise<SearchEngine[]> => {
  const { data } = await http.get<ApiResult<SearchEngine[]>>('/searchEngines')
  return data.data || []
}

/** 获取必应每日壁纸的图片描述（首页搜索框 placeholder 显示，取不到时返回空串） */
export const fetchBingWallpaperTitle = async (): Promise<string> => {
  try {
    const { data } = await http.get<ApiResult<{ title: string }>>('/bingWallpaperInfo')
    return data.data?.title || ''
  } catch {
    // 描述只是锦上添花，接口失败（离线、首次下载未完成等）时静默回落到默认 placeholder
    return ''
  }
}

export const login = async (name: string, password: string): Promise<ApiResult<{ user: User; token: string }>> => {
  const { data } = await http.post<ApiResult<{ user: User; token: string }>>('/login', { name, password })
  return data
}

export const logout = async (): Promise<ApiResult> => {
  const { data } = await http.get<ApiResult>('/logout')
  return data
}

// ==================== 后台接口 ====================

export const fetchAdminData = async (): Promise<AdminData> => {
  const { data } = await http.get<ApiResult<AdminData>>('/admin/all')
  return data.data
}

// 工具管理
export const fetchAddTool = async (payload: Partial<Tool>): Promise<ApiResult<{ id: number }>> => {
  const { data } = await http.post<ApiResult<{ id: number }>>('/admin/tool', payload)
  return data
}

export const fetchUpdateTool = async (payload: Partial<Tool>): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>(`/admin/tool/${payload.id}`, payload)
  return data
}

export const fetchDeleteTool = async (id: number): Promise<ApiResult> => {
  const { data } = await http.delete<ApiResult>(`/admin/tool/${id}`)
  return data
}

/**
 * 给工具上传本地图标：图片按工具名称命名保存到 data 目录（data/images，与按网址自动下载的图标命名一致），
 * 后端同时把该工具的图标网址与 logo 图片名改成这张图片，返回图标图片名与本地地址
 */
export const fetchUploadToolLogo = async (id: number, file: File): Promise<{ name: string; url: string }> => {
  const formData = new FormData()
  formData.append('file', file)
  const { data } = await http.post<ApiResult<{ name: string; url: string }>>(`/admin/tool/${id}/logo`, formData)
  return data.data ?? { name: '', url: '' }
}

/** 抓取网址信息（标题/描述/图标），用于添加工具时自动填充 */
export const fetchGetUrlInfo = async (url: string): Promise<UrlInfo> => {
  const { data } = await http.get<ApiResult<UrlInfo>>('/admin/urlInfo', { params: { url } })
  return data.data
}

export const fetchUpdateToolsSort = async (updates: { id: number; sort: number }[]): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/tools/sort', updates)
  return data
}

/** 导出全部数据（工具、分类、搜索引擎、api token），图标只导出图标网址（工具的图片名与搜索引擎的图标不导出） */
export const fetchExportAll = async (): Promise<BackupData> => {
  const { data } = await http.get<ApiResult<BackupData>>('/admin/exportAll')
  return data.data
}

/** 导入全部数据（工具、分类、搜索引擎、api token），导入后服务端会自动获取图标 */
export const fetchImportAll = async (payload: BackupData): Promise<ApiResult<BackupImportResult>> => {
  const { data } = await http.post<ApiResult<BackupImportResult>>('/admin/importAll', payload)
  return data
}

// 分类管理
export const fetchAddCateLog = async (payload: Partial<Catelog>): Promise<ApiResult> => {
  const { data } = await http.post<ApiResult>('/admin/catelog', payload)
  return data
}

export const fetchUpdateCateLog = async (payload: Partial<Catelog>): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>(`/admin/catelog/${payload.id}`, payload)
  return data
}

export const fetchDeleteCatelog = async (id: number): Promise<ApiResult> => {
  const { data } = await http.delete<ApiResult>(`/admin/catelog/${id}`)
  return data
}

export const fetchUpdateCatelogsSort = async (updates: { id: number; sort: number }[]): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/catelogs/sort', updates)
  return data
}

// 设置
export const fetchUpdateSetting = async (payload: Record<string, unknown>): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/setting', payload)
  return data
}

export const fetchUpdateSiteConfig = async (payload: Record<string, unknown>): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/siteConfig', payload)
  return data
}

export const fetchUpdateUser = async (payload: Record<string, unknown>): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/user', payload)
  return data
}

/**
 * 上传图片（背景图 / logo 等），返回图片的可访问地址
 * 后端限制单张图片不超过 5MB，支持的格式见 service/upload.go
 */
export const fetchUploadImage = async (file: File): Promise<string> => {
  const formData = new FormData()
  formData.append('file', file)
  const { data } = await http.post<ApiResult<{ url: string }>>('/admin/uploadImage', formData)
  return data.data?.url ?? ''
}

// API Token
export const fetchAddApiToken = async (payload: { name: string }): Promise<ApiResult> => {
  const { data } = await http.post<ApiResult>('/admin/apiToken', payload)
  return data
}

export const fetchDeleteApiToken = async (id: number): Promise<ApiResult> => {
  const { data } = await http.delete<ApiResult>(`/admin/apiToken/${id}`)
  return data
}

// 搜索引擎管理
export const fetchGetAllSearchEngines = async (): Promise<SearchEngine[]> => {
  const { data } = await http.get<ApiResult<SearchEngine[]>>('/admin/searchEngine')
  return data.data || []
}

export const fetchAddSearchEngine = async (payload: Partial<SearchEngine>): Promise<ApiResult> => {
  const { data } = await http.post<ApiResult>('/admin/searchEngine', payload)
  return data
}

/** 把搜索引擎的 logo 外链下载保存到本地（文件名为搜索引擎名称），返回本地可访问地址 */
export const fetchSaveSearchEngineLogo = async (payload: { name: string; logo: string }): Promise<string> => {
  const { data } = await http.post<ApiResult<{ url: string }>>('/admin/searchEngine/logo', payload)
  return data.data?.url ?? ''
}

export const fetchUpdateSearchEngine = async (payload: Partial<SearchEngine>): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>(`/admin/searchEngine/${payload.id}`, payload)
  return data
}

export const fetchDeleteSearchEngine = async (id: number): Promise<ApiResult> => {
  const { data } = await http.delete<ApiResult>(`/admin/searchEngine/${id}`)
  return data
}

export const fetchUpdateSearchEnginesSort = async (updates: { id: number; sort: number }[]): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/searchEngines/sort', updates)
  return data
}

export type { AdminData, ApiResult, Catelog, HomeData, SearchEngine, Token, Tool, User }
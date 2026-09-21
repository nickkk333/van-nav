import axios from 'axios'
import type { AdminData, ApiResult, Catelog, HomeData, SearchEngine, Token, Tool, UrlInfo, User } from '../types'

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

/** 抓取网址信息（标题/描述/图标），用于添加工具时自动填充 */
export const fetchGetUrlInfo = async (url: string): Promise<UrlInfo> => {
  const { data } = await http.get<ApiResult<UrlInfo>>('/admin/urlInfo', { params: { url } })
  return data.data
}

export const fetchUpdateToolsSort = async (updates: { id: number; sort: number }[]): Promise<ApiResult> => {
  const { data } = await http.put<ApiResult>('/admin/tools/sort', updates)
  return data
}

export const fetchExportTools = async (): Promise<Tool[]> => {
  const { data } = await http.get<ApiResult<Tool[]>>('/admin/exportTools')
  return data.data
}

export const fetchImportTools = async (payload: Tool[]): Promise<ApiResult> => {
  const { data } = await http.post<ApiResult>('/admin/importTools', payload)
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
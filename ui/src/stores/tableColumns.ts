import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchAdminData, fetchUpdateAdminTableColumns } from '../api'
import { useAdminStore } from './admin'
import type { SiteConfig } from '../types'

/** 后台可配置列的表格标识 */
export type TableKey = 'tools' | 'categories' | 'searchEngines' | 'apiToken'
/** 隐藏列配置：按表格分组，值为隐藏的列 key 数组 */
export type HiddenColumns = Partial<Record<TableKey, string[]>>

/** 描述表格里一列的可选项 */
export interface TableColumnOption {
  key: string
  label: string
}

export const useTableColumnsStore = defineStore('tableColumns', () => {
  const hidden = ref<HiddenColumns>({})
  const saving = ref(false)

  /** 从服务端下发的 siteConfig 解析隐藏列；空值 / 脏数据 = 全部显示 */
  const syncFromSiteConfig = (siteConfig?: Partial<SiteConfig> | null) => {
    const raw = siteConfig?.adminHiddenColumns
    if (!raw) {
      hidden.value = {}
      return
    }
    try {
      const parsed: unknown = JSON.parse(raw)
      if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
        const next: HiddenColumns = {}
        Object.entries(parsed as Record<string, unknown>).forEach(([key, value]) => {
          if (Array.isArray(value)) {
            next[key as TableKey] = value.filter((item): item is string => typeof item === 'string')
          }
        })
        hidden.value = next
        return
      }
    } catch {
      /* 配置损坏时回落全部显示 */
    }
    hidden.value = {}
  }

  /**
   * 从服务端重新拉取列配置，用于多设备端同步（其它设备改动后，切回本设备自动生效）。
   * 直接调接口而不写入 adminStore，避免覆盖表格里尚未保存的行编辑
   */
  const refresh = async () => {
    try {
      const data = await fetchAdminData()
      syncFromSiteConfig(data?.siteConfig)
    } catch {
      /* 网络异常时保留当前配置，不影响使用 */
    }
  }

  /** 某列是否显示（默认全部显示） */
  const isVisible = (table: TableKey, column: string): boolean =>
    !(hidden.value[table] ?? []).includes(column)

  /** 保存某张表的隐藏列：本地立即生效，再写入服务端；失败时回滚本地状态 */
  const setHidden = async (table: TableKey, columns: string[]) => {
    const previous = hidden.value
    const next: HiddenColumns = { ...hidden.value }
    if (columns.length) next[table] = columns
    else delete next[table]
    hidden.value = next
    const payload = JSON.stringify(next)
    saving.value = true
    try {
      await fetchUpdateAdminTableColumns(payload)
      // 让内存里的 siteConfig 与服务端保持一致，避免界面其它地方读到旧值
      const admin = useAdminStore()
      if (admin.store.siteConfig) {
        admin.store.siteConfig.adminHiddenColumns = payload
      }
    } catch (e) {
      hidden.value = previous
      throw e
    } finally {
      saving.value = false
    }
  }

  return { hidden, saving, syncFromSiteConfig, refresh, isVisible, setHidden }
})

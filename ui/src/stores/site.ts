import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchList } from '../api'
import { setCustomDefaultLogo } from '../utils/check'
import { DEFAULT_CARDS_PER_ROW } from '../utils/setting'
import type { HomeData, Setting, SiteConfig, Tool } from '../types'

const defaultSetting: Setting = {
  id: 0,
  favicon: 'favicon.ico',
  title: 'Van Nav',
  govRecord: '',
  logo192: 'logo192.png',
  logo512: 'logo512.png',
  hideAdmin: false,
  hideGithub: false,
  hideToggleJumpTarget: false,
  jumpTargetBlank: true,
  backgroundImage: '',
  defaultLogo: '',
  defaultSearchEngine: 0,
}

const defaultSiteConfig: SiteConfig = {
  id: 0,
  noImageMode: false,
  compactMode: false,
  cardsPerRow: DEFAULT_CARDS_PER_ROW,
}

export const useSiteStore = defineStore('site', () => {
  const data = ref<HomeData>({
    tools: [],
    catelogs: [],
    setting: { ...defaultSetting },
    siteConfig: { ...defaultSiteConfig },
  })
  const loading = ref(false)

  /** 用新的工具列表覆盖本地数据（首页拖拽排序 / 右键删除后即时刷新界面） */
  const setTools = (tools: Tool[]) => {
    data.value = { ...data.value, tools }
  }

  const load = async () => {
    loading.value = true
    try {
      const res = await fetchList()
      const setting = { ...defaultSetting, ...(res?.setting ?? {}) }
      // 同步后台上传的默认图标：卡片没配图标时用它显示，没上传时用名称首字符占位（见 utils/check.ts）
      setCustomDefaultLogo(setting.defaultLogo)
      data.value = {
        tools: res?.tools ?? [],
        catelogs: res?.catelogs ?? [],
        setting,
        siteConfig: { ...defaultSiteConfig, ...(res?.siteConfig ?? {}) },
      }
      return data.value
    } finally {
      loading.value = false
    }
  }

  return { data, loading, load, setTools }
})
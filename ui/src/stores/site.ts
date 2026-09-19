import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchList } from '../api'
import type { HomeData, Setting, SiteConfig } from '../types'

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
}

const defaultSiteConfig: SiteConfig = {
  id: 0,
  noImageMode: false,
  compactMode: false,
}

export const useSiteStore = defineStore('site', () => {
  const data = ref<HomeData>({
    tools: [],
    catelogs: [],
    setting: { ...defaultSetting },
    siteConfig: { ...defaultSiteConfig },
  })
  const loading = ref(false)

  const load = async () => {
    loading.value = true
    try {
      const res = await fetchList()
      data.value = {
        tools: res?.tools ?? [],
        catelogs: res?.catelogs ?? [],
        setting: { ...defaultSetting, ...(res?.setting ?? {}) },
        siteConfig: { ...defaultSiteConfig, ...(res?.siteConfig ?? {}) },
      }
      return data.value
    } finally {
      loading.value = false
    }
  }

  return { data, loading, load }
})
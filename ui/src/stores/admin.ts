import { defineStore } from 'pinia'
import { ref } from 'vue'
import { fetchAdminData } from '../api'
import type { AdminData } from '../types'

export const useAdminStore = defineStore('admin', () => {
  const store = ref<Partial<AdminData>>({})
  const loading = ref(false)
  const loaded = ref(false)

  /** 加载后台数据，默认命中缓存，传 true 强制刷新 */
  const load = async (force = false) => {
    if (loaded.value && !force) {
      return store.value
    }
    loading.value = true
    try {
      store.value = await fetchAdminData()
      loaded.value = true
      return store.value
    } finally {
      loading.value = false
    }
  }

  const reset = () => {
    store.value = {}
    loaded.value = false
  }

  return { store, loading, loaded, load, reset }
})
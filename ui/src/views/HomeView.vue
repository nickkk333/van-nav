<template>
  <div class="app">
    <div class="app-bg" aria-hidden="true"></div>
    <div class="main">
      <div class="topbar">
        <SearchBar
          ref="searchBarRef"
          :model-value="searchText"
          @update:model-value="onSearchInput"
          @search="onSearchSubmit"
        />
        <TagSelector :tags="tags" :curr-tag="currTag" @change="handleSetCurrTag" />
      </div>
      <div class="content-wraper">
        <div class="content cards" :class="{ 'compact-grid': siteConfig.compactMode }">
          <Loading v-if="loading" />
          <ToolCard
            v-for="(item, index) in filteredData"
            :key="item.id + '-' + index"
            :tool="item"
            :index="index"
            :is-searching="isSearching"
            :no-image-mode="siteConfig.noImageMode"
            :compact-mode="siteConfig.compactMode"
            @click="handleCardClick"
          />
        </div>
        <div v-if="!loading && filteredData.length === 0" class="empty-tip">没有找到匹配的结果</div>
      </div>
      <div class="record-wraper">
        <a href="https://beian.miit.gov.cn" target="_blank" rel="noreferrer">{{ setting.govRecord }}</a>
      </div>
      <GithubLink v-if="showGithub" />
      <DarkSwitch :hide-github="!showGithub" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import SearchBar from '../components/SearchBar.vue'
import TagSelector from '../components/TagSelector.vue'
import ToolCard from '../components/ToolCard.vue'
import GithubLink from '../components/GithubLink.vue'
import DarkSwitch from '../components/DarkSwitch.vue'
import Loading from '../components/Loading.vue'
import { useSiteStore } from '../stores/site'
import { multiSearch } from '../utils/match'
import { generateSearchEngineCards } from '../utils/searchEngine'
import { initServerJumpTargetConfig, toggleJumpTarget } from '../utils/setting'
import type { Tool } from '../types'

const DEFAULT_TAG = '默认'
const ADMIN_TAG = '管理后台'

const site = useSiteStore()

const searchText = ref('')
const searchString = ref('')
const currTag = ref(DEFAULT_TAG)
const engineCards = ref<Tool[]>([])
const loading = ref(true)
const searchBarRef = ref<InstanceType<typeof SearchBar>>()

const setting = computed(() => site.data.setting)
const siteConfig = computed(() => site.data.siteConfig)
const isSearching = computed(() => searchString.value.trim() !== '')
const showGithub = computed(() => setting.value.hideGithub !== true)

/** 默认标签 + 所有分类 */
const tags = computed(() => {
  const list = (site.data.catelogs ?? []).filter((tag) => tag !== DEFAULT_TAG)
  return [DEFAULT_TAG, ...list]
})

const filteredData = computed<Tool[]>(() => {
  const tools = site.data.tools ?? []
  const searching = isSearching.value
  const localResult = tools.filter((item) => {
    // 主题中隐藏了跳转方式卡片
    if (setting.value.hideToggleJumpTarget && item.url === 'toggleJumpTarget') {
      return false
    }
    // 搜索时在所有工具中查找，否则按当前分类过滤
    if (!searching) {
      if (currTag.value === DEFAULT_TAG) {
        return item.default === true
      }
      return item.catelog === currTag.value
    }
    return (
      multiSearch(item.name, searchString.value) ||
      multiSearch(item.catelog, searchString.value) ||
      multiSearch(item.url, searchString.value)
    )
  })
  return searching ? [...localResult, ...engineCards.value] : localResult
})

// 关键字变化时生成搜索引擎卡片
watch(searchString, async (value) => {
  try {
    engineCards.value = await generateSearchEngineCards(value)
  } catch (error) {
    console.error('加载搜索引擎卡片失败:', error)
    engineCards.value = []
  }
})

// 输入框内容变化（由 SearchBar 触发，避免与 resetSearch 相互递归）
const onSearchInput = (value: string) => {
  searchText.value = value
  if (value && value.trim() !== '') {
    currTag.value = DEFAULT_TAG
    searchString.value = value.trim()
  } else {
    resetSearch()
  }
}

const resetSearch = (notSetTag = false) => {
  searchText.value = ''
  searchString.value = ''
  const tagInLocalStorage = window.localStorage.getItem('tag')
  if (!notSetTag && tagInLocalStorage && tagInLocalStorage !== '' && tagInLocalStorage !== ADMIN_TAG) {
    currTag.value = tagInLocalStorage
  }
}

const handleSetCurrTag = (tag: string) => {
  currTag.value = tag
  if (tag !== ADMIN_TAG) {
    window.localStorage.setItem('tag', tag)
  }
  resetSearch(true)
}

const handleCardClick = async (tool: Tool) => {
  resetSearch()
  if (tool.url === 'toggleJumpTarget') {
    toggleJumpTarget()
    await site.load()
  }
}

/** 打开当前结果列表的第一项（回车或点击搜索按钮） */
const openFirstResult = () => {
  const cards = filteredData.value
  if (cards.length) {
    window.open(cards[0].url, '_blank')
    resetSearch()
  }
}

/** 点击搜索按钮：有关键词时打开第一条结果，否则聚焦搜索框 */
const onSearchSubmit = () => {
  if (isSearching.value) {
    openFirstResult()
    return
  }
  searchBarRef.value?.focus()
}

/** 回车打开第一条结果，Ctrl/Cmd + 数字打开对应结果 */
const onKeyEnter = (ev: KeyboardEvent) => {
  const cards = filteredData.value
  if (ev.key === 'Enter' || ev.keyCode === 13) {
    openFirstResult()
    return
  }
  if (ev.ctrlKey || ev.metaKey) {
    const num = Number(ev.key)
    if (Number.isNaN(num) || ev.key.trim() === '') {
      return
    }
    ev.preventDefault()
    const index = num - 1
    if (index >= 0 && index < cards.length) {
      window.open(cards[index].url, '_blank')
      resetSearch()
    }
  }
}

const bindKeyboard = () => {
  document.removeEventListener('keydown', onKeyEnter)
  if (isSearching.value) {
    document.addEventListener('keydown', onKeyEnter)
  }
}

watch(searchString, bindKeyboard)

/** 动态设置标题和 favicon */
const applySiteMeta = () => {
  document.title = setting.value.title || 'Van Nav'
  const href = setting.value.favicon || '/logo192.png'
  let link = document.querySelector<HTMLLinkElement>("link[rel~='icon']")
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }
  link.href = href
}

const init = async () => {
  loading.value = true
  try {
    await site.load()
    initServerJumpTargetConfig(site.data.setting)
    const tagInLocalStorage = window.localStorage.getItem('tag')
    if (tagInLocalStorage && (site.data.catelogs ?? []).includes(tagInLocalStorage)) {
      currTag.value = tagInLocalStorage
    }
    applySiteMeta()
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

onMounted(init)
onBeforeUnmount(() => document.removeEventListener('keydown', onKeyEnter))
</script>


<template>
  <div class="app">
    <div class="app-bg" :style="bgStyle" aria-hidden="true"></div>
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
        <div
          class="content cards"
          :class="{ 'compact-grid': siteConfig.compactMode, 'grouped-grid': groupByCatelog }"
          :style="gridStyle"
        >
          <Loading v-if="loading" />
          <div v-for="group in cardGroups" :key="group.key" class="cards-group-wraper">
            <div v-if="group.name" class="cards-group-divider" aria-hidden="true">
              <span class="cards-group-divider-line"></span>
              <span class="cards-group-divider-name">{{ displayCatelog(group.name) }}</span>
              <span class="cards-group-divider-line"></span>
            </div>
            <div class="cards-group">
              <ToolCard
                v-for="item in group.items"
                :key="item.tool.id + '-' + item.index"
                :tool="item.tool"
                :index="item.index"
                :is-searching="isSearching"
                :no-image-mode="siteConfig.noImageMode"
                :compact-mode="siteConfig.compactMode"
                @click="handleCardClick"
              />
            </div>
          </div>
        </div>
        <div v-if="!loading && filteredData.length === 0" class="empty-tip">没有找到匹配的结果</div>
      </div>
      <div class="record-wraper">
        <a href="https://beian.miit.gov.cn" target="_blank" rel="noreferrer">{{ setting.govRecord }}</a>
      </div>
      <div class="float-actions">
        <DarkSwitch />
        <GithubLink v-if="showGithub" />
        <AdminLink />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import type { CSSProperties } from 'vue'
import SearchBar from '../components/SearchBar.vue'
import TagSelector from '../components/TagSelector.vue'
import ToolCard from '../components/ToolCard.vue'
import GithubLink from '../components/GithubLink.vue'
import AdminLink from '../components/AdminLink.vue'
import DarkSwitch from '../components/DarkSwitch.vue'
import Loading from '../components/Loading.vue'
import { useSiteStore } from '../stores/site'
import { displayCatelog } from '../utils/check'
import { multiSearch } from '../utils/match'
import { generateSearchEngineCards } from '../utils/searchEngine'
import { DEFAULT_CARDS_PER_ROW, MAX_CARDS_PER_ROW, initServerJumpTargetConfig, toggleJumpTarget } from '../utils/setting'
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

/** 首页全屏背景图：地址来自后台设置，留空时保持纯色背景 */
const bgStyle = computed<CSSProperties>(() => {
  // 过滤掉可能破坏 url() 语法的字符，避免异常地址影响整个样式
  const url = (setting.value.backgroundImage ?? '').replace(/["'()\\\s]/g, '')
  return url ? { backgroundImage: `url("${url}")` } : {}
})

/** 每行展示的网站数量，小屏自动收敛，避免卡片过窄 */
const cardsPerRow = computed(() => {
  const value = Number(siteConfig.value.cardsPerRow)
  if (!Number.isFinite(value) || value < 1) {
    return DEFAULT_CARDS_PER_ROW
  }
  return Math.min(Math.floor(value), MAX_CARDS_PER_ROW)
})

/** 通过 CSS 变量把每行数量传给网格布局 */
const gridStyle = computed<CSSProperties>(() => ({
  '--cards-per-row': String(cardsPerRow.value),
  '--cards-per-row-md': String(Math.min(cardsPerRow.value, 3)),
  '--cards-per-row-sm': String(Math.min(cardsPerRow.value, 2)),
}))

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

/** 首页渲染用的卡片项：index 同时用于搜索序号与列表 key */
interface CardItem {
  tool: Tool
  index: number
}

/** 卡片分组：默认栏中每个分类为一组，其余情况只有一组 */
interface CardGroup {
  key: string
  /** 分组名称（分类名）；仅默认栏分组时存在，用于渲染 “---分类名---” 分隔栏 */
  name?: string
  items: CardItem[]
}

/** 默认栏（默认标签且未搜索）需要按分类分组展示 */
const groupByCatelog = computed(() => !isSearching.value && currTag.value === DEFAULT_TAG)

/**
 * 默认栏按分类顺序分组：同一分类的工具排在一起，不同分类各占一块并用 “---分类名---” 分隔栏区分
 * （每组单独换行，块与块之间的距离由样式中的 --cards-group-gap 加大）。
 * 分类顺序以接口返回的 catelogs（后台排序后的顺序）为准；
 * 不在分类列表中的工具（例如未分类）统一追加到最后。
 */
const cardGroups = computed<CardGroup[]>(() => {
  const list = filteredData.value
  let index = 0
  const toItems = (tools: Tool[]): CardItem[] => tools.map((tool) => ({ tool, index: index++ }))

  if (!groupByCatelog.value) {
    return [{ key: 'all', items: toItems(list) }]
  }

  const catelogs = site.data.catelogs ?? []
  const groups = new Map<string, Tool[]>()
  catelogs.forEach((name) => groups.set(name, []))
  const rest: Tool[] = []
  list.forEach((tool) => {
    const group = groups.get(tool.catelog)
    if (group) {
      group.push(tool)
    } else {
      rest.push(tool)
    }
  })

  const result: CardGroup[] = []
  const appendGroup = (name: string, tools: Tool[]) => {
    // 空分类不渲染，避免产生多余的分隔栏与组间距
    if (!tools.length) {
      return
    }
    result.push({ key: `${result.length}-${name}`, name, items: toItems(tools) })
  }
  groups.forEach((tools, name) => appendGroup(name, tools))
  appendGroup('未分类', rest)
  return result
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


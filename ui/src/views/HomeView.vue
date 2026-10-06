<template>
  <div class="app">
    <div class="app-bg" :style="bgStyle" aria-hidden="true"></div>
    <div class="main">
      <div class="topbar">
        <SearchBar
          ref="searchBarRef"
          :model-value="searchText"
          :placeholder="searchPlaceholder"
          :completion="domainCompletion"
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
            <div class="cards-group" :ref="(el) => setGroupEl(group.key, el)">
              <ToolCard
                v-for="item in group.items"
                :key="item.tool.id + '-' + item.index"
                :tool="item.tool"
                :index="item.index"
                :is-searching="isSearching"
                :no-image-mode="siteConfig.noImageMode"
                :compact-mode="siteConfig.compactMode"
                :editable="canEdit"
                @click="handleCardClick"
                @contextmenu="handleCardContextMenu"
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
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchBingWallpaperTitle, fetchDeleteTool, fetchUpdateToolsSort, resolveError } from '../api'
import { useCardSortable } from '../composables/useCardSortable'
import { useSiteStore } from '../stores/site'
import { displayCatelog, isLogin } from '../utils/check'
import { multiSearch, pinyinReady } from '../utils/match'
import { generateSearchEngineCards, getDefaultSearchEngineUrl } from '../utils/searchEngine'
import { DEFAULT_BACKGROUND_IMAGE, DEFAULT_CARDS_PER_ROW, MAX_CARDS_PER_ROW, initServerJumpTargetConfig, toggleJumpTarget } from '../utils/setting'
import type { Tool } from '../types'

const DEFAULT_TAG = '默认'
const ADMIN_TAG = '管理后台'

const site = useSiteStore()

const searchText = ref('')
const searchString = ref('')
// 搜索框提示语（置灰 placeholder）：默认提示，背景使用必应壁纸时换成当天壁纸的图片描述
const searchPlaceholder = ref('按任意键直接开始搜索')
const currTag = ref(DEFAULT_TAG)
const engineCards = ref<Tool[]>([])
const loading = ref(true)
const searchBarRef = ref<InstanceType<typeof SearchBar>>()

// ==================== 域名补全（搜索框 ghost 提示 + Tab 补全 + 回车直访） ====================
/** 从工具 url 提取域名（取 host 并去掉前导 www.），仅用域名部分参与匹配与补全 */
const domainOf = (url?: string | null): string => {
  if (!url) return ''
  let normalized = String(url).trim()
  if (!/^https?:\/\//i.test(normalized)) normalized = `http://${normalized}`
  try {
    const host = new URL(normalized).hostname
    return host.replace(/^www\./i, '').toLowerCase()
  } catch {
    return ''
  }
}

/** 所有卡片去重后的域名列表（按出现顺序），作为域名补全的匹配来源 */
const domainList = computed<string[]>(() => {
  const seen = new Set<string>()
  const list: string[] = []
  for (const tool of site.data.tools ?? []) {
    const d = domainOf(tool.url)
    if (d && !seen.has(d)) {
      seen.add(d)
      list.push(d)
    }
  }
  return list
})

/** 输入文本与某个卡片域名前缀匹配时，返回完整域名（Tab 补全的目标） */
const matchedDomain = computed<string>(() => {
  const text = searchText.value.trim().toLowerCase()
  if (!text || /\s/.test(text)) return ''
  for (const d of domainList.value) {
    if (d.startsWith(text)) return d
  }
  return ''
})

/** 搜索框里灰显的未补全域名后缀，如输入 git 时显示 hub.com */
const domainCompletion = computed<string>(() => {
  const text = searchText.value.trim().toLowerCase()
  return matchedDomain.value && text ? matchedDomain.value.slice(text.length) : ''
})

/** 判断文本是否像域名（含点、至少两段、无空格、无协议/路径符号） */
const isDomainLike = (text: string): boolean => {
  const t = text.trim()
  return t !== '' && !/\s/.test(t) && /^[\w-]+(\.[\w-]+)+$/.test(t)
}
/** 给裸域名补上 https:// 协议 */
const withProtocol = (text: string): string => {
  const t = text.trim()
  return /^https?:\/\//i.test(t) ? t : `https://${t}`
}

/**
 * 回车 / 点搜索按钮：
 * - 输入像域名则直接访问该域名；
 * - 否则优先打开第一个匹配的工具卡片（跳过搜索引擎卡片）；
 * - 没有任何工具卡片时，用默认搜索引擎搜索当前词。
 */
const submitSearch = async () => {
  const text = searchText.value.trim()
  if (text && isDomainLike(text)) {
    window.open(withProtocol(text), '_blank')
    resetSearch()
    return
  }
  if (firstToolCardUrl.value) {
    window.open(firstToolCardUrl.value, '_blank')
    resetSearch()
    return
  }
  const url = await getDefaultSearchEngineUrl(text, false)
  if (url) {
    window.open(url, '_blank')
    resetSearch()
  }
}

const setting = computed(() => site.data.setting)
const siteConfig = computed(() => site.data.siteConfig)
const isSearching = computed(() => searchString.value.trim() !== '')
const showGithub = computed(() => setting.value.hideGithub !== true)

/** 首页全屏背景图：优先使用后台配置的地址，未配置时使用服务端下载的必应每日壁纸 */
const bgStyle = computed<CSSProperties>(() => {
  // 过滤掉可能破坏 url() 语法的字符，避免异常地址影响整个样式
  const url = (setting.value.backgroundImage || DEFAULT_BACKGROUND_IMAGE).replace(/["'()\\\s]/g, '')
  return url ? { backgroundImage: `url("${url}")` } : {}
})

/** 当前背景是否是必应每日壁纸（未配置自定义背景，或显式填了壁纸地址） */
const usingBingWallpaper = computed(() => !setting.value.backgroundImage || setting.value.backgroundImage === DEFAULT_BACKGROUND_IMAGE)

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

/** 过滤后的工具卡片（不含搜索引擎虚拟卡片）：用于卡片展示与「回车打开第一个工具卡片」 */
const localToolResults = computed<Tool[]>(() => {
  // 订阅拼音分包就绪状态：分包加载完自动重算，补上拼音匹配结果（见 utils/match.ts）
  void pinyinReady.value
  const tools = site.data.tools ?? []
  const searching = isSearching.value
  return tools.filter((item) => {
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
})

const filteredData = computed<Tool[]>(() => {
  return isSearching.value ? [...localToolResults.value, ...engineCards.value] : localToolResults.value
})

/** 回车优先打开的第一个工具卡片地址（不含搜索引擎卡片）；没有则为空 */
const firstToolCardUrl = computed(() => localToolResults.value[0]?.url ?? '')

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

// ==================== 首页管理（登录后可用：分类内拖拽排序 / 右键删除） ====================

/** 登录后首页卡片支持拖拽排序与右键删除，未登录时保持只读 */
const canEdit = ref(isLogin())
/** 搜索结果是临时的（还包含搜索引擎虚拟卡片），搜索时不参与拖拽排序 */
const canSort = computed(() => canEdit.value && !isSearching.value)

/** 首页全部工具（服务端已按排序值升序返回） */
const tools = computed<Tool[]>(() => site.data.tools ?? [])

/** 各分组容器：分组 key -> DOM 元素，拖拽只在分组内部进行 */
const groupEls = new Map<string, HTMLElement>()
const setGroupEl = (key: string, el: unknown) => {
  if (el instanceof HTMLElement) {
    groupEls.set(key, el)
  } else {
    groupEls.delete(key)
  }
}

/** 本地按新的排序值刷新工具列表（排序值相同时保持原有顺序） */
const applyToolsSort = (updates: { id: number; sort: number }[]) => {
  const sortMap = new Map(updates.map((item) => [item.id, item.sort]))
  const next = tools.value.map((tool) => {
    const sort = sortMap.get(tool.id)
    return sort === undefined ? tool : { ...tool, sort }
  })
  next.sort((a, b) => (a.sort ?? 0) - (b.sort ?? 0))
  site.setTools(next)
}

/**
 * 计算拖拽后要提交的排序数据：
 * 优先把该分组原有的排序值按从小到大依次分配给新顺序（只改该分组，其他分类不受影响）；
 * 分组内排序值有重复时（例如手工改过数据库）改用整表顺序重排成 1 开始依次递增，保证新顺序能保存下来。
 */
const buildSortUpdates = (order: Tool[], values: number[]) => {
  if (new Set(values).size === values.length) {
    return order.map((tool, index) => ({ id: tool.id, sort: values[index] }))
  }
  const ids = new Set(order.map((tool) => tool.id))
  const queue = [...order]
  const list = tools.value.map((tool) => (ids.has(tool.id) ? (queue.shift() as Tool) : tool))
  return list.map((tool, index) => ({ id: tool.id, sort: index + 1 }))
}

/** 分组内拖拽结束：先按新顺序刷新本地，再把该分组的排序提交到服务端 */
const persistSort = async (key: string, oldIndex: number, newIndex: number) => {
  const group = cardGroups.value.find((item) => item.key === key)
  if (!group) {
    return
  }
  const order = group.items.map((item) => item.tool)
  const [moved] = order.splice(oldIndex, 1)
  if (!moved) {
    return
  }
  order.splice(newIndex, 0, moved)

  const values = group.items.map((item) => item.tool.sort ?? 0).sort((a, b) => a - b)
  const updates = buildSortUpdates(order, values)
  applyToolsSort(updates)
  try {
    await fetchUpdateToolsSort(updates)
    ElMessage({ message: '排序已更新', type: 'success', grouping: true, duration: 1200 })
  } catch (error) {
    ElMessage.error(resolveError(error, '排序更新失败'))
    // 失败时以服务端数据为准
    await site.load()
  }
}

useCardSortable({
  source: cardGroups,
  enabled: canSort,
  getContainers: () => groupEls,
  onEnd: persistSort,
})

/** 右键删除工具（内置的跳转方式卡片、搜索引擎虚拟卡片不参与删除） */
const handleCardContextMenu = async (tool: Tool) => {
  if (!canEdit.value || tool.url === 'toggleJumpTarget') {
    return
  }
  // 搜索时展示的搜索引擎卡片是虚拟数据，没有对应的工具记录
  if (!tools.value.some((item) => item.id === tool.id)) {
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除「${tool.name}」吗？`, '删除工具', {
      type: 'warning',
      confirmButtonText: '删除',
      cancelButtonText: '取消',
    })
  } catch {
    // 取消删除
    return
  }
  try {
    await fetchDeleteTool(tool.id)
    // 用剩余工具覆盖本地列表，界面立即去掉这张卡片
    ElMessage.success('删除成功')
    site.setTools(tools.value.filter((item) => item.id !== tool.id))
  } catch (error) {
    ElMessage.error(resolveError(error, '删除失败'))
  }
}

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

/** 点击搜索按钮：有关键词时打开第一条结果，否则聚焦搜索框 */
const onSearchSubmit = () => {
  if (isSearching.value) {
    submitSearch()
    return
  }
  searchBarRef.value?.focus()
}

/** 回车打开第一条结果，Ctrl/Cmd + 数字打开对应结果，Ctrl/Cmd + Enter 用第一个搜索引擎卡片搜索 */
const onKeyEnter = (ev: KeyboardEvent) => {
  // Ctrl/Cmd + Enter：在新标签打开搜索结果页，保留导航页
  if ((ev.ctrlKey || ev.metaKey) && (ev.key === 'Enter' || ev.keyCode === 13)) {
    ev.preventDefault()
    const text = searchText.value.trim()
    if (text && isDomainLike(text)) {
      window.open(withProtocol(text), '_blank')
      resetSearch()
      return
    }
    getDefaultSearchEngineUrl(text, true).then((url) => {
      if (url) {
        window.open(url, '_blank')
        resetSearch()
      }
    })
    return
  }
  const cards = filteredData.value
  if (ev.key === 'Enter' || ev.keyCode === 13) {
    submitSearch()
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
  // 登录状态决定首页是否提供拖拽排序 / 右键删除
  canEdit.value = isLogin()
  try {
    await site.load()
    initServerJumpTargetConfig(site.data.setting)
    const tagInLocalStorage = window.localStorage.getItem('tag')
    if (tagInLocalStorage && (site.data.catelogs ?? []).includes(tagInLocalStorage)) {
      currTag.value = tagInLocalStorage
    }
    applySiteMeta()
    // 搜索框 placeholder 换成必应壁纸的图片描述：不阻塞首屏，异步拿到再替换
    // （自定义背景图时不展示必应描述，保持默认提示）
    if (usingBingWallpaper.value) {
      fetchBingWallpaperTitle().then((title) => {
        if (title) searchPlaceholder.value = title
      })
    }
  } catch (error) {
    console.error(error)
  } finally {
    loading.value = false
  }
}

onMounted(init)
onBeforeUnmount(() => document.removeEventListener('keydown', onKeyEnter))
</script>


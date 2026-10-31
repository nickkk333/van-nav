<template>
  <div class="app">
    <div class="app-bg" :style="bgStyle" aria-hidden="true"></div>
    <div class="main" :class="{ 'hide-content': effectiveHide }">
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
        <div v-if="editMode" class="edit-mode-bar">
          <span class="edit-mode-text">设置模式：拖动卡片排序，点击卡片右上角 × 删除</span>
          <button type="button" class="edit-mode-done" @click="exitEditMode">完成</button>
        </div>
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
                :editable="canEdit && manageableIds.has(item.tool.id)"
                :edit-mode="editMode"
                @click="handleCardClick"
                @contextmenu="handleCardContextMenu"
                @delete="handleDeleteCard"
                @longpress="enterEditMode"
              />
            </div>
          </div>
        </div>
        <div v-if="!loading && filteredData.length === 0" class="empty-tip">没有找到匹配的结果</div>
      </div>
      <div class="record-wraper">
        <a href="https://beian.miit.gov.cn" target="_blank" rel="noreferrer">{{ setting.govRecord }}</a>
      </div>
      <div class="float-actions" :class="{ 'rail-open': railOpen }">
        <DarkSwitch />
        <ProductivityDock :show-memo="showMemo" />
        <button
          type="button"
          class="content-toggle"
          :class="{ checked: !hideContent }"
          :title="hideContent ? '仅显示搜索框（点击显示全部内容）' : '显示全部内容（点击仅显示搜索框）'"
          :aria-label="hideContent ? '显示全部内容' : '仅显示搜索框'"
          @click="toggleContent"
        >
          <svg v-if="!hideContent" viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/></svg>
          <svg v-else viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="3" y="3" width="18" height="18" rx="2" ry="2"/></svg>
        </button>
        <GithubLink v-if="showGithub" />
        <AdminLink v-if="showAdmin" />
      </div>
    </div>
    <!-- 卡片右键菜单：只提供一个「设置」入口，避免和浏览器菜单混在一起 -->
    <Teleport to="body">
      <div
        v-if="cardMenu.visible"
        class="card-context-menu"
        :style="{ left: `${cardMenu.x}px`, top: `${cardMenu.y}px` }"
        @pointerdown.stop
        @contextmenu.prevent
      >
        <button type="button" class="card-context-menu-item" @click="enterEditMode">
          <el-icon><Setting /></el-icon>
          <span>设置</span>
        </button>
      </div>
    </Teleport>
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
import ProductivityDock from '../components/productivity/ProductivityDock.vue'
import { isTypingContext } from '../utils/keyboard'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Setting } from '@element-plus/icons-vue'
import { fetchBingWallpaperTitle, fetchDeleteTool, fetchUpdateToolsSort, resolveError } from '../api'
import { useCardSortable } from '../composables/useCardSortable'
import { useSiteStore } from '../stores/site'
import { displayCatelog, isLogin } from '../utils/check'
import { multiSearch, pinyinReady } from '../utils/match'
import { generateSearchEngineCards, getDefaultSearchEngineUrl } from '../utils/searchEngine'
import { DEFAULT_BACKGROUND_IMAGE, DEFAULT_CARDS_PER_ROW, MAX_CARDS_PER_ROW, initServerJumpTargetConfig } from '../utils/setting'
import { normalizeInternalUrl, resolveToolTargetUrl } from '../utils/toolUrl'
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

/** 一键隐藏内容、仅显示搜索框的偏好（存 localStorage，刷新后保持） */
const hideContent = ref(localStorage.getItem('van_nav_hide_content') === '1')
const toggleContent = () => {
  hideContent.value = !hideContent.value
  localStorage.setItem('van_nav_hide_content', hideContent.value ? '1' : '0')
}
// 搜索框有内容时临时显示内容（卡片），不隐藏
const hasSearchText = computed(() => searchText.value.trim().length > 0)
const effectiveHide = computed(() => hideContent.value && !hasSearchText.value)
const searchBarRef = ref<InstanceType<typeof SearchBar>>()

// ==================== 右下角图标列：默认隐藏，鼠标贴到屏幕右边框才滑出 ====================
/** 热区宽度：鼠标进入屏幕右侧这么多像素内即显示图标列 */
const RAIL_HOT_ZONE = 70
/** 触摸设备没有 hover，保持常显，否则点不到 Github / 管理后台 */
const railHoverable = window.matchMedia('(hover: hover) and (pointer: fine)').matches
const railOpen = ref(!railHoverable)

const onRailMove = (e: MouseEvent) => {
  const open = e.clientX >= window.innerWidth - RAIL_HOT_ZONE
  if (open !== railOpen.value) railOpen.value = open
}

const hideRail = () => {
  railOpen.value = false
}

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
/** 给裸域名补上 https:// 协议（已带协议的完整地址原样返回） */
const withProtocol = (text: string): string => {
  const t = text.trim()
  return /^https?:\/\//i.test(t) ? t : `https://${t}`
}
/** 是否以 http:// 或 https:// 开头的完整地址（直接访问，不再补协议） */
const isHttpUrl = (text: string): boolean => /^https?:\/\//i.test(text.trim())

/**
 * 回车 / 点搜索按钮：
 * - 输入以 http:// 或 https:// 开头则直接访问该完整地址；
 * - 输入像域名则补上 https:// 直接访问该域名；
 * - 否则优先打开第一个匹配的工具卡片（跳过搜索引擎卡片）；
 * - 没有任何工具卡片时，用默认搜索引擎搜索当前词。
 */
const submitSearch = async () => {
  const text = searchText.value.trim()
  if (text && (isHttpUrl(text) || isDomainLike(text))) {
    window.open(withProtocol(text), '_blank')
    resetSearch()
    return
  }
  if (firstToolCard.value) {
    void openCardUrl(firstToolCard.value)
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
const showAdmin = computed(() => setting.value.hideAdmin !== true)

// ==================== 首页管理（登录后可用：分类内拖拽排序 / 右键删除） ====================

/** 登录后首页卡片支持右键设置、设置模式里拖拽排序与删除，未登录时保持只读 */
const canEdit = ref(isLogin())
/**
 * 首页设置模式：卡片右键 → 设置 进入。
 * 只有该模式下才显示删除角标、长按卡片可拖动排序；未进入设置时拖动无反应。
 */
const editMode = ref(false)
/** 搜索结果是临时的（还包含搜索引擎虚拟卡片），搜索时不参与拖拽排序 */
const canSort = computed(() => canEdit.value && editMode.value && !isSearching.value)

/** 便签是私人数据（后端接口需要登录），未登录时不显示入口 */
const showMemo = computed(() => canEdit.value && setting.value.hideMemo !== true)

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

/** 回车优先打开的第一个工具卡片（不含搜索引擎卡片）；没有则为空 */
const firstToolCard = computed<Tool | undefined>(() => localToolResults.value[0])

/**
 * 在新标签页打开卡片地址：配置了内网网址时先探测内网是否可达，可达打开内网网址，不通回退外网网址。
 * 探测是异步的，先同步开一个空标签页再写入地址，避免异步 window.open 被浏览器拦截。
 */
const openCardUrl = async (tool: Pick<Tool, 'url' | 'internalUrl'>) => {
  if (!normalizeInternalUrl(tool.internalUrl)) {
    window.open(tool.url, '_blank')
    return
  }
  const preOpened = window.open('', '_blank')
  const url = await resolveToolTargetUrl(tool.url, tool.internalUrl)
  if (!url) {
    preOpened?.close()
    return
  }
  if (preOpened && !preOpened.closed) {
    preOpened.opener = null
    preOpened.location.replace(url)
    return
  }
  window.open(url, '_blank')
}

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

// ==================== 首页设置（登录后可用：右键设置 / 设置模式内拖动排序与删除） ====================

/** 首页全部工具（服务端已按排序值升序返回） */
const tools = computed<Tool[]>(() => site.data.tools ?? [])

/** 真实工具 id 集合：搜索引擎虚拟卡片、内置跳转卡片不参与设置与删除 */
const manageableIds = computed(() => new Set(tools.value.map((tool) => tool.id)))

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

// 只有进入设置模式后才允许拖动排序（短按/长按在普通模式下都不会触发拖拽）
useCardSortable({
  source: cardGroups,
  enabled: canSort,
  getContainers: () => groupEls,
  onEnd: persistSort,
})

/** 卡片右键菜单：位置 + 显示状态 */
const cardMenu = ref({ visible: false, x: 0, y: 0 })

const closeCardMenu = () => {
  cardMenu.value.visible = false
}

/** 右键菜单里的「设置」：进入设置模式（显示删除角标、启用拖动排序） */
const enterEditMode = () => {
  closeCardMenu()
  if (!canEdit.value) {
    return
  }
  editMode.value = true
}

const exitEditMode = () => {
  editMode.value = false
}

/** 右键卡片弹出「设置」按钮；未登录或虚拟卡片保留浏览器默认菜单 */
const handleCardContextMenu = (tool: Tool, event: MouseEvent) => {
  if (!canEdit.value || !manageableIds.value.has(tool.id)) {
    return
  }
  // 菜单约 120x44，贴近窗口边缘时向内收，避免被裁掉
  cardMenu.value = {
    visible: true,
    x: Math.min(event.clientX, window.innerWidth - 128),
    y: Math.min(event.clientY, window.innerHeight - 52),
  }
}

/** 点击删除角标：先给出删除提示，确认后再删除 */
const handleDeleteCard = async (tool: Tool) => {
  if (!canEdit.value || !manageableIds.value.has(tool.id)) {
    return
  }
  try {
    await ElMessageBox.confirm(`确定删除「${tool.name}」吗？删除后不可恢复。`, '删除卡片', {
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
    site.setTools(tools.value.filter((item) => item.id !== tool.id))
    ElMessage.success('删除成功')
  } catch (error) {
    ElMessage.error(resolveError(error, '删除失败'))
  }
}

/** 点空白处 / 滚动时收起右键菜单；Esc 收起菜单并退出设置模式 */
const onGlobalPointerDown = () => closeCardMenu()
const onGlobalScroll = () => closeCardMenu()
const onGlobalKeydown = (event: KeyboardEvent) => {
  if (event.key !== 'Escape') {
    return
  }
  closeCardMenu()
  if (editMode.value) {
    editMode.value = false
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

const handleCardClick = () => {
  resetSearch()
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
  // 弹窗（便签）与输入框中的按键不触发搜索快捷键，回车等交给弹窗自己
  if (isTypingContext(ev)) return
  // 焦点在按钮/链接上时交给元素自身激活，避免与搜索快捷键重复触发
  const evTarget = ev.target as HTMLElement | null
  if (evTarget && (evTarget.tagName === 'BUTTON' || evTarget.tagName === 'A' || evTarget.closest('button, a[href]'))) return
  // Ctrl/Cmd + Enter：在新标签打开搜索结果页，保留导航页
  if ((ev.ctrlKey || ev.metaKey) && (ev.key === 'Enter' || ev.keyCode === 13)) {
    ev.preventDefault()
    const text = searchText.value.trim()
    if (text && (isHttpUrl(text) || isDomainLike(text))) {
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
      void openCardUrl(cards[index])
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

onMounted(() => {
  init()
  // 图标列贴边显示：只监听鼠标（触摸设备保持常显）
  if (railHoverable) {
    document.addEventListener('mousemove', onRailMove)
    document.addEventListener('mouseleave', hideRail)
  }
  // 右键菜单：点空白处 / 滚动 / Esc 时收起
  window.addEventListener('pointerdown', onGlobalPointerDown)
  window.addEventListener('scroll', onGlobalScroll, true)
  window.addEventListener('keydown', onGlobalKeydown)
})
onBeforeUnmount(() => {
  document.removeEventListener('keydown', onKeyEnter)
  document.removeEventListener('mousemove', onRailMove)
  document.removeEventListener('mouseleave', hideRail)
  window.removeEventListener('pointerdown', onGlobalPointerDown)
  window.removeEventListener('scroll', onGlobalScroll, true)
  window.removeEventListener('keydown', onGlobalKeydown)
})
</script>


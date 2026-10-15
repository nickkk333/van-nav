<template>
  <a
    class="card-box"
    :class="{ 'card-editable': editable, 'is-dragging': isDragging }"
    :href="isToggleCard ? undefined : tool.url"
    :target="target"
    rel="noreferrer"
    @pointerdown="onPointerDown"
    @pointerup="endPress"
    @pointercancel="endPress"
    @click="onClick"
    @contextmenu="onContextMenu"
  >
    <span v-if="showNumIndex" class="card-index">{{ index + 1 }}</span>
    <div class="card-content" :class="{ 'compact-mode': compactMode }">
      <div v-if="!noImageMode" class="card-left">
        <LogoFallback v-if="showNameInitial" class="card-image-error" :name="initialName" />
        <template v-else>
          <span v-if="showLoading" class="card-loading-spinner" />
          <img
            :src="imageSrc"
            :alt="tool.name"
            loading="lazy"
            :style="{ opacity: imageLoaded ? 1 : 0.1 }"
            @load="onImageLoad"
            @error="onImageError"
          />
        </template>
      </div>
      <div class="card-right">
        <div class="card-right-top">
          <span class="card-right-title" :title="tool.name">{{ tool.name }}</span>
          <span v-if="!compactMode" class="card-tag" :title="displayCatelog(tool.catelog)">
            {{ displayCatelog(tool.catelog) }}
          </span>
        </div>
        <div v-if="!compactMode" class="card-right-bottom" :title="tool.desc">{{ tool.desc }}</div>
      </div>
    </div>
  </a>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import LogoFallback from './LogoFallback.vue'
import { displayCatelog, customDefaultLogo, DEFAULT_LOGO_URL, getToolLogoUrl, hasToolLogo } from '../utils/check'
import { getJumpTarget } from '../utils/setting'
import type { Tool } from '../types'

const props = defineProps<{
  tool: Tool
  index: number
  isSearching: boolean
  noImageMode?: boolean
  compactMode?: boolean
  /** 登录后首页卡片支持拖拽排序与右键删除 */
  editable?: boolean
}>()

const emit = defineEmits<{
  (e: 'click', tool: Tool): void
  (e: 'contextmenu', tool: Tool): void
}>()

const imageLoaded = ref(false)
const imageError = ref(false)
const showLoading = ref(true)
let timer: ReturnType<typeof setTimeout> | null = null

const isToggleCard = computed(() => props.tool.url === 'toggleJumpTarget')

/**
 * 拖拽（sortablejs）结束后浏览器还会补发一次 click，
 * 锚点会因此执行默认跳转，既把页面带走了，也会打断排序提交的时机。
 * 这里用「按下点到松开点的位移」判定：移动超过阈值就吞掉这次 click。
 * 同一位移判定还驱动 is-dragging：平时（含悬停 / 单击 / 长按不动）一律箭头，
 * 只有真正产生位移的拖动才切换出手型（纯 CSS :active 区分不了“按住不动”和“拖动”）。
 */
const CLICK_MOVE_THRESHOLD = 5
let pointerStart: { x: number; y: number } | null = null
let activePointerId: number | null = null
/** 按下后是否已产生位移（拖动中为 true，长按不动为 false） */
const didMove = ref(false)
/** 拖动中：模板据此挂 is-dragging 类显示手型 */
const isDragging = ref(false)

const setDragging = (value: boolean) => {
  if (isDragging.value === value) {
    return
  }
  isDragging.value = value
  // 拖动时鼠标可能正好悬停在别的卡片上，全局类保证拖动全程都是手型
  document.body.classList.toggle('card-dragging', value)
}

const onPointerMove = (event: Event) => {
  if (!(event instanceof PointerEvent)) {
    return
  }
  if (event.pointerId !== activePointerId || !pointerStart) {
    return
  }
  if (
    Math.abs(event.clientX - pointerStart.x) > CLICK_MOVE_THRESHOLD ||
    Math.abs(event.clientY - pointerStart.y) > CLICK_MOVE_THRESHOLD
  ) {
    didMove.value = true
    setDragging(true)
  }
}

const onPointerDown = (event: PointerEvent) => {
  // 只跟踪左键 / 触摸主按压：右键（删除菜单）不进入拖动手型
  if (event.pointerType === 'mouse' && event.button !== 0) {
    return
  }
  activePointerId = event.pointerId
  pointerStart = { x: event.clientX, y: event.clientY }
  didMove.value = false
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', endPress)
  window.addEventListener('pointercancel', endPress)
}

/** 按压结束：手型复位为箭头，但保留 didMove 给随后到达的 click 做拖拽判定 */
const endPress = (event?: Event) => {
  if (event instanceof PointerEvent && event.pointerId !== activePointerId) {
    return
  }
  activePointerId = null
  pointerStart = null
  setDragging(false)
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', endPress)
  window.removeEventListener('pointercancel', endPress)
}

const onClick = (event: MouseEvent) => {
  const moved = didMove.value
  didMove.value = false
  if (moved) {
    // 拖拽/滑动收尾的 click：阻止 <a> 的默认跳转，也不触发业务点击逻辑
    event.preventDefault()
    return
  }
  emit('click', props.tool)
}

/** 登录后拦截右键菜单用于删除工具，未登录时保留浏览器默认菜单（可新标签页打开等） */
const onContextMenu = (event: MouseEvent) => {
  if (!props.editable) {
    return
  }
  event.preventDefault()
  emit('contextmenu', props.tool)
}

// 跳转方式卡片用的是内置图片（相对路径），直接取原值；
// 其余卡片优先读保存到本机的图片，没配置图标的卡片用后台上传的默认图标（相当于替换内置的 default.png）
const imageSrc = computed(() => {
  if (isToggleCard.value) {
    return props.tool.logo || DEFAULT_LOGO_URL
  }
  return hasToolLogo(props.tool) ? getToolLogoUrl(props.tool) : customDefaultLogo.value
})
// 名称首字符占位：没有图标也没有上传默认图标（图片地址为空），或图片加载失败（含默认图标加载失败）时显示
// 跳转方式卡片一定用内置图片，不参与该逻辑
const showNameInitial = computed(() => (isToggleCard.value ? false : !imageSrc.value || imageError.value))
// 占位字符：虚拟卡片（如搜索引擎卡片）可用 logoText 指定，缺省取名称首字符
const initialName = computed(() => props.tool.logoText || props.tool.name)

const target = computed(() => (getJumpTarget() === 'blank' ? '_blank' : '_self'))
const showNumIndex = computed(() => props.index < 10 && props.isSearching)

const resetImageState = () => {
  imageLoaded.value = false
  imageError.value = false
  showLoading.value = true
  if (timer) {
    clearTimeout(timer)
  }
  // 10 秒超时保护
  timer = setTimeout(() => {
    showLoading.value = false
  }, 10000)
}

const onImageLoad = () => {
  imageLoaded.value = true
  showLoading.value = false
}

const onImageError = () => {
  imageError.value = true
  showLoading.value = false
}

watch(imageSrc, resetImageState, { immediate: true })

onBeforeUnmount(() => {
  endPress()
  if (timer) {
    clearTimeout(timer)
  }
})
</script>

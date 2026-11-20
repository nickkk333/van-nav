<template>
  <div class="about-page">
    <header class="about-header">
      <button type="button" class="about-back" aria-label="返回" @click="goBack">
        <el-icon :size="16"><ArrowLeft /></el-icon>
        <span>返回</span>
      </button>
      <h1>关于应用</h1>
    </header>

    <main class="about-main">
      <section class="about-card">
        <img class="about-logo" :src="logoSrc" alt="应用图标" @error="onLogoError" />

        <h2 class="about-name">{{ appName }}</h2>
        <p class="about-desc">{{ ABOUT_INFO.description }}</p>

        <div class="about-info">
          <div class="about-info-row">
            <span>当前版本</span>
            <a class="about-info-link" :href="ABOUT_INFO.releaseUrl" target="_blank" rel="noopener noreferrer">
              v{{ ABOUT_INFO.version }}
            </a>
          </div>
          <div class="about-info-row">
            <span>开发者</span>
            <a class="about-info-link" :href="ABOUT_INFO.developerUrl" target="_blank" rel="noopener noreferrer">
              {{ ABOUT_INFO.developer }}
            </a>
          </div>
        </div>

        <div v-if="qrList.length" class="about-support">
          <h3>赞赏支持</h3>
          <p>{{ ABOUT_SUPPORT_TIP }}</p>
          <div class="about-qr-list">
            <figure v-for="qr in qrList" :key="qr.key" class="about-qr-item">
              <img
                v-if="!failedQr[qr.key]"
                class="about-qr-image"
                :src="qr.src"
                :alt="qr.label"
                @error="failedQr[qr.key] = true"
              />
              <div v-else class="about-qr-placeholder">
                <el-icon :size="26"><Picture /></el-icon>
                <span>暂未配置</span>
              </div>
              <figcaption>{{ qr.label }}</figcaption>
            </figure>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ArrowLeft, Picture } from '@element-plus/icons-vue'
import { useSiteStore } from '../stores/site'
import { setColorTheme } from '../utils/theme'
import { ABOUT_INFO, ABOUT_SUPPORT_TIP } from '../utils/about'

const router = useRouter()
const site = useSiteStore()

const FALLBACK_LOGO = `${import.meta.env.BASE_URL}logo192.png`
const logoFailed = ref(false)

/** 站点设置里的 192x192 图标优先，未配置或加载失败时回落到内置图标 */
const logoSrc = computed(() => {
  if (logoFailed.value) return FALLBACK_LOGO
  const logo = (site.data.setting.logo192 || '').trim()
  if (!logo) return FALLBACK_LOGO
  if (/^(https?:)?\/\//i.test(logo) || logo.startsWith('/') || logo.startsWith('data:')) {
    return logo
  }
  return `${import.meta.env.BASE_URL}${logo}`
})

const onLogoError = () => {
  logoFailed.value = true
}

/** 站点标题可以自定义，没配置时用默认应用名 */
const appName = computed(() => (site.data.setting.title || '').trim() || ABOUT_INFO.name)

const failedQr = reactive<Record<string, boolean>>({})

const qrList = computed(() => {
  const list: { key: string; label: string; src: string }[] = []
  if (ABOUT_INFO.wechatQr) list.push({ key: 'wechat', label: '微信', src: ABOUT_INFO.wechatQr })
  if (ABOUT_INFO.alipayQr) list.push({ key: 'alipay', label: '支付宝', src: ABOUT_INFO.alipayQr })
  return list
})

/** 有站内历史时返回上一页，直接打开 /about 时回到首页 */
const goBack = () => {
  if (window.history.length > 1 && window.history.state?.back) {
    router.back()
    return
  }
  router.push('/')
}

onMounted(async () => {
  document.title = '关于应用'
  try {
    await site.load()
    // 直接打开 /about 时也按服务端配置应用配色
    setColorTheme(site.data.setting.colorTheme)
    document.title = `关于应用 - ${appName.value}`
  } catch (error) {
    console.error(error)
  }
})
</script>

import { onBeforeUnmount, ref } from 'vue'

/** 手机 / 窄屏断点，与 styles/index.css 里的后台适配断点保持一致 */
const MOBILE_QUERY = '(max-width: 768px)'

/**
 * 当前是否为手机（窄屏）布局，窗口尺寸变化时自动更新。
 * 需要在组件的 setup 中调用（内部注册了卸载时的事件解绑）。
 */
export const useIsMobile = () => {
  const query = window.matchMedia(MOBILE_QUERY)
  const isMobile = ref(query.matches)

  const onChange = (event: MediaQueryListEvent) => {
    isMobile.value = event.matches
  }
  query.addEventListener('change', onChange)
  onBeforeUnmount(() => query.removeEventListener('change', onChange))

  return isMobile
}

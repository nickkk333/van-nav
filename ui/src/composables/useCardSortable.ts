import { nextTick, onBeforeUnmount, onMounted, watch } from 'vue'
import type { Ref } from 'vue'
import Sortable from 'sortablejs'

interface UseCardSortableOptions {
  /** 分组 / 顺序等变化后需要重新绑定拖拽的数据来源 */
  source: Ref<unknown>
  /** 是否启用拖拽（未登录或搜索时不启用） */
  enabled: Ref<boolean>
  /** 取当前所有分组容器：分组 key -> 容器元素 */
  getContainers: () => Map<string, HTMLElement>
  /** 拖拽结束回调（分组 key、起始下标、目标下标） */
  onEnd: (key: string, oldIndex: number, newIndex: number) => void
}

/**
 * 让首页的分类卡片分组支持拖拽排序（基于 sortablejs，只能在分组内部拖动）
 */
export const useCardSortable = (options: UseCardSortableOptions) => {
  const { source, enabled, getContainers, onEnd } = options
  let instances: Sortable[] = []

  const destroy = () => {
    instances.forEach((instance) => instance.destroy())
    instances = []
  }

  const mount = () => {
    destroy()
    if (!enabled.value) {
      return
    }
    getContainers().forEach((element, key) => {
      instances.push(
        Sortable.create(element, {
          animation: 150,
          ghostClass: 'sortable-ghost',
          onEnd: (evt) => {
            const { oldIndex, newIndex } = evt
            if (oldIndex === undefined || newIndex === undefined || oldIndex === newIndex) {
              return
            }
            onEnd(key, oldIndex, newIndex)
          },
        })
      )
    })
  }

  onMounted(() => nextTick(mount))
  watch([source, enabled], () => nextTick(mount), { flush: 'post' })
  onBeforeUnmount(destroy)

  return { mount, destroy }
}
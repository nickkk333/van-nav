import { nextTick, onBeforeUnmount, onMounted, watch } from 'vue'
import type { Ref } from 'vue'
import Sortable from 'sortablejs'

interface UseTableSortableOptions {
  /** el-table 组件实例 */
  tableRef: Ref<any>
  /** 表格数据，用于数据变化后重新绑定拖拽 */
  rows: Ref<unknown[]>
  /** 拖拽结束回调 */
  onEnd: (oldIndex: number, newIndex: number) => void
  /** 拖拽手柄选择器 */
  handle?: string
}

/**
 * 让 element-plus 的 el-table 支持行拖拽排序（基于 sortablejs）
 */
export const useTableSortable = (options: UseTableSortableOptions) => {
  const { tableRef, rows, onEnd, handle = '.drag-handle' } = options
  let instance: Sortable | null = null

  const destroy = () => {
    instance?.destroy()
    instance = null
  }

  const mount = () => {
    const root: HTMLElement | undefined = tableRef.value?.$el ?? tableRef.value
    const tbody = root?.querySelector('.el-table__body-wrapper tbody') as HTMLElement | null
    if (!tbody) {
      return
    }
    destroy()
    instance = Sortable.create(tbody, {
      handle,
      animation: 150,
      ghostClass: 'sortable-ghost',
      onEnd: (evt) => {
        const { oldIndex, newIndex } = evt
        if (oldIndex === undefined || newIndex === undefined || oldIndex === newIndex) {
          return
        }
        onEnd(oldIndex, newIndex)
      },
    })
  }

  onMounted(() => {
    nextTick(mount)
  })

  watch(rows, () => nextTick(mount), { flush: 'post' })

  onBeforeUnmount(destroy)

  return { mount, destroy }
}
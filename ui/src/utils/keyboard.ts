/**
 * 全局键盘快捷键的「让路」判断：
 * 便签等弹窗与任何可编辑元素中的按键，不应被
 * 「任意键聚焦搜索框」「回车执行搜索」之类的全局快捷键抢走焦点或误触发。
 *
 * 搜索框（#search-bar）自身不算：它的输入与回车正是全局搜索要处理的。
 */
/** Element Plus 的 el-dialog 关闭后遮罩仍留在 DOM 中（仅 v-show 隐藏），
 *  因此不能只判断 .el-overlay 是否存在，必须过滤掉不可见的遮罩。 */
const hasVisibleOverlay = (): boolean => {
  const overlays = document.querySelectorAll<HTMLElement>('.el-overlay')
  for (const el of overlays) {
    // display:none 的元素没有任何 client rect
    if (el.getClientRects().length > 0) return true
  }
  return false
}

export const isTypingContext = (ev: Event): boolean => {
  // 已被其他组件处理（弹窗内部快捷键等）
  if (ev.defaultPrevented) return true
  const target = ev.target as HTMLElement | null
  if (target && !target.closest('#search-bar')) {
    const tag = target.tagName
    // 输入框、多行文本、下拉框
    if (tag === 'INPUT' || tag === 'TEXTAREA' || tag === 'SELECT') return true
    // 富文本可编辑区
    if (target.isContentEditable) return true
    // el-dialog / ElMessageBox 等遮罩内（含日期面板、按钮等非输入控件）
    if (target.closest('.el-overlay, .el-message-box__wrapper')) return true
  }
  // 兜底：确有弹窗遮罩「可见」时全局快捷键才让路（事件来自 body 等情况）
  return hasVisibleOverlay()
}

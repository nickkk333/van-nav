<template>
  <div v-if="showCalendar || showMemo">
    <div class="pv-dock">
      <el-tooltip v-if="showCalendar" content="日历" placement="left"><button type="button" class="pv-dock-btn" aria-label="日历" @click="open()"><el-icon :size="20"><Calendar /></el-icon><i v-if="hasUpcoming" class="pv-dock-dot" aria-hidden="true"></i></button></el-tooltip>
      <el-tooltip v-if="showMemo" content="便签" placement="left"><button type="button" class="pv-dock-btn" aria-label="便签" @click="openMemo"><el-icon :size="20"><EditPen /></el-icon></button></el-tooltip>
    </div>
    <!-- 日历主弹窗：右边缘调宽、底边缘调高、右下角同调；尺寸记入 localStorage，双击右下角恢复默认 -->
    <el-dialog v-model="visible" title="日历" :width="dlgWidth" top="8vh" class="pv-dlg" body-class="pv-dlg-body" :style="dlgStyle" append-to-body destroy-on-close>
      <CalendarPanel ref="calRef" @add="onCalAdd" @edit="onCalEdit" @remove="onCalRemove" @toggle-done="onCalToggleDone" />
      <span class="pv-rz pv-rz-e" title="拖动调整宽度" @pointerdown="startResize('e', $event)"></span>
      <span class="pv-rz pv-rz-s" title="拖动调整高度" @pointerdown="startResize('s', $event)"></span>
      <span class="pv-rz pv-rz-se" title="拖动调整大小（双击恢复默认）" @pointerdown="startResize('se', $event)" @dblclick="onSeDblClick"></span>
    </el-dialog>
    <!-- 便签：独立浮动窗口，可拖动移动、贴边吸附成图标、点击展开 -->
    <MemoWindow v-if="showMemo" ref="memoWinRef" />
    <el-dialog v-model="evOpen" :title="evEditing?'编辑事件':'添加事件'" width="400px" append-to-body>
      <el-form label-width="52px" size="small" @submit.prevent>
        <el-form-item label="标题"><el-input v-model="evForm.title" maxlength="60" placeholder="例如：发版 / 生日" /></el-form-item>
        <el-form-item label="日期"><el-date-picker v-model="evForm.date" type="date" value-format="YYYY-MM-DD" /></el-form-item>
        <el-form-item label="时间"><div class="pv-time-row"><el-time-picker v-model="evForm.startTime" format="HH:mm" value-format="HH:mm" placeholder="开始" /><span>~</span><el-time-picker v-model="evForm.endTime" format="HH:mm" value-format="HH:mm" placeholder="结束" /></div></el-form-item>
        <el-form-item label="备注"><el-input v-model="evForm.desc" type="textarea" :rows="2" maxlength="200" /></el-form-item>
        <el-form-item label="颜色"><el-color-picker v-model="evForm.color" :predefine="evColors" /></el-form-item>
      </el-form>
      <template #footer><el-button size="small" @click="evOpen=false">取消</el-button><el-button size="small" type="primary" :loading="evSaving" @click="saveEvent">保存</el-button></template>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, EditPen } from '@element-plus/icons-vue'
import CalendarPanel from './CalendarPanel.vue'
import MemoWindow from './MemoWindow.vue'
import { fetchAddCalendarEvent, fetchDeleteCalendarEvent, fetchGetCalendarEvents, fetchToggleCalendarEventDone, fetchUpdateCalendarEvent, resolveError } from '../../api'
import type { CalendarEvent } from '../../types'
// 首页默认显示日历、便签，后台设置里可以分别关掉（开启「隐藏」后这里对应的入口不渲染）
const props = defineProps<{ showCalendar?: boolean; showMemo?: boolean }>()
const showCalendar = computed(()=>props.showCalendar!==false)
const showMemo = computed(()=>props.showMemo!==false)
const visible = ref(false)
const calRef = ref<InstanceType<typeof CalendarPanel>|null>(null)
const memoWinRef = ref<InstanceType<typeof MemoWindow>|null>(null)
const open = () => { visible.value=true; refreshUpcoming() }
// 便签使用独立浮动窗口：点击入口切换展开 / 关闭
const openMemo = () => memoWinRef.value?.toggle()

// ==================== 弹窗宽高调整（拖右/下/右下角把手，尺寸存 localStorage） ====================
const SIZE_KEY = 'pv-dialog-size'
const DEFAULT_WIDTH = 480
const clamp = (v: number, min: number, max: number) => Math.min(Math.max(v, min), Math.max(min, max))
/** 宽高上下限跟随窗口：宽度不超出视口；高度不越过 top=8vh 之后的剩余空间 */
const sizeLimits = () => {
  const maxW = Math.max(240, window.innerWidth - 24)
  const maxH = Math.max(240, Math.floor(window.innerHeight * 0.92) - 58)
  return { minW: Math.min(320, maxW), maxW, minH: Math.min(320, maxH), maxH }
}
/** 弹窗宽度（px），默认与原来的 width="480px" 一致 */
const dlgWidth = ref(DEFAULT_WIDTH)
/** 弹窗高度（px）；null 表示还没手动调过，高度继续保持「随内容自适应」 */
const dlgHeight = ref<number | null>(null)
const dlgStyle = computed(() => (dlgHeight.value == null ? undefined : { height: `${dlgHeight.value}px` }))
const loadSize = () => {
  try {
    const saved = JSON.parse(window.localStorage.getItem(SIZE_KEY) || 'null')
    if (typeof saved?.w === 'number') dlgWidth.value = saved.w
    if (typeof saved?.h === 'number') dlgHeight.value = saved.h
  } catch { /* 存储内容损坏：当作没存过，用默认尺寸 */ }
}
const saveSize = () => {
  try {
    window.localStorage.setItem(SIZE_KEY, JSON.stringify({ w: dlgWidth.value, h: dlgHeight.value }))
  } catch { /* 隐私模式等存不了：忽略 */ }
}
/** 窗口尺寸变化后把当前弹窗尺寸收回可视范围内 */
const clampSize = () => {
  const { minW, maxW, minH, maxH } = sizeLimits()
  dlgWidth.value = clamp(dlgWidth.value, minW, maxW)
  if (dlgHeight.value != null) dlgHeight.value = clamp(dlgHeight.value, minH, maxH)
}
loadSize()
clampSize()
type ResizeDir = 'e' | 's' | 'se'
/** 右下角把手的按下序列里是否发生过拖动（用来区分「双击恢复默认」和「拖了两下」） */
let seMoved = false
const startResize = (dir: ResizeDir, ev: PointerEvent) => {
  if (ev.pointerType === 'mouse' && ev.button !== 0) return
  const el = (ev.currentTarget as HTMLElement | null)?.closest('.el-dialog') as HTMLElement | null
  if (!el) return
  if (dir === 'se') seMoved = false
  const startW = el.offsetWidth
  const startH = el.offsetHeight
  const startX = ev.clientX
  const startY = ev.clientY
  // 只有真的纵向拖动过才把高度固定下来，误点一下边缘不会破坏「高度自适应」
  let heightLocked = dlgHeight.value != null
  document.body.style.cursor = dir === 'e' ? 'col-resize' : dir === 's' ? 'row-resize' : 'nwse-resize'
  document.body.style.userSelect = 'none'
  try { (ev.currentTarget as HTMLElement).setPointerCapture(ev.pointerId) } catch { /* 捕获失败也不影响拖动 */ }
  const onMove = (e2: PointerEvent) => {
    const dx = e2.clientX - startX
    const dy = e2.clientY - startY
    if (dir === 'se' && (Math.abs(dx) > 3 || Math.abs(dy) > 3)) seMoved = true
    const { minW, maxW, minH, maxH } = sizeLimits()
    if (dir !== 's') dlgWidth.value = clamp(startW + dx, minW, maxW)
    if (dir !== 'e') {
      if (!heightLocked) { dlgHeight.value = startH; heightLocked = true }
      dlgHeight.value = clamp(startH + dy, minH, maxH)
    }
  }
  const stopResize = () => {
    window.removeEventListener('pointermove', onMove)
    window.removeEventListener('pointerup', stopResize)
    window.removeEventListener('pointercancel', stopResize)
    document.body.style.cursor = ''
    document.body.style.userSelect = ''
    saveSize()
  }
  window.addEventListener('pointermove', onMove)
  window.addEventListener('pointerup', stopResize)
  window.addEventListener('pointercancel', stopResize)
}
/** 双击右下角恢复默认：宽 480、高度自适应 */
const resetSize = () => {
  dlgWidth.value = DEFAULT_WIDTH
  dlgHeight.value = null
  saveSize()
}
const onSeDblClick = () => { if (!seMoved) resetSize() }
// 日历图标右上角未读式小红点：当前时间落在某事件时间范围内才显示，只提示“有”，不显示条数
// 全天事件（无时间）当天全天提示；打勾（已完成）视为已结束，不提示
const upcomingEvents = ref<CalendarEvent[]>([])
const hasUpcoming = computed(()=>{
  const now = new Date()
  const pad = (n:number)=>String(n).padStart(2,'0')
  const todayStr = `${now.getFullYear()}-${pad(now.getMonth()+1)}-${pad(now.getDate())}`
  const nowHM = `${pad(now.getHours())}:${pad(now.getMinutes())}`
  return upcomingEvents.value.some((ev)=>{
    if (!ev?.date || ev.done) return false
    if (ev.date !== todayStr) return false
    const start = (ev.startTime || '').trim()
    const end = (ev.endTime || '').trim()
    // 全天事件：当天全天提示
    if (!start && !end) return true
    // 只有开始：开始后（含开始整分钟）提示
    if (start && !end) return nowHM >= start
    // 只有结束：结束前提示
    if (!start && end) return nowHM < end
    // 有起止：开始后、结束前提示
    return nowHM >= start && nowHM < end
  })
})
const refreshUpcoming = async ()=>{
  if (!showCalendar.value) { upcomingEvents.value = []; return }
  try { upcomingEvents.value = await fetchGetCalendarEvents() } catch { /* 提醒是锦上添花，失败时静默不显示小红点 */ }
}
// 只在刷新时重算：挂载、新增/编辑/删除/打勾、打开日历后刷新；不做 60s 轮询
onMounted(() => {
  refreshUpcoming()
  window.addEventListener('resize', clampSize)
})
onUnmounted(() => window.removeEventListener('resize', clampSize))
const evOpen = ref(false); const evEditing = ref<CalendarEvent|null>(null); const evSaving = ref(false)
const evColors = ['#409eff','#67c23a','#e6a23c','#f56c6c','#909399','#9b59b6','#00bcd4']
const evForm = ref({title:'',date:'',startTime:'',endTime:'',desc:'',color:'#409eff'})
const onCalAdd = (date:string)=>{ evEditing.value=null; evForm.value={title:'',date,startTime:'',endTime:'',desc:'',color:'#409eff'}; evOpen.value=true }
const onCalEdit = (ev:CalendarEvent)=>{ evEditing.value=ev; evForm.value={title:ev.title,date:ev.date,startTime:ev.startTime||'',endTime:ev.endTime||'',desc:ev.desc||'',color:ev.color||'#409eff'}; evOpen.value=true }
const onCalRemove = async (ev:CalendarEvent)=>{ try{ await ElMessageBox.confirm(`确定删除「${ev.title}」吗？`,'删除事件',{type:'warning',confirmButtonText:'删除',cancelButtonText:'取消'}) }catch{ return } try{ await fetchDeleteCalendarEvent(ev.id); ElMessage.success('已删除'); await calRef.value?.reload(); await refreshUpcoming() }catch(e){ ElMessage.error(resolveError(e,'删除失败')) } }
const onCalToggleDone = async (ev:CalendarEvent, done:boolean)=>{ try{ await fetchToggleCalendarEventDone(ev.id, done); await calRef.value?.reload(); await refreshUpcoming() }catch(e){ ElMessage.error(resolveError(e,'更新失败')) } }
const saveEvent = async ()=>{ if(!evForm.value.title.trim()||!evForm.value.date){ ElMessage.warning('标题与日期不能为空'); return } evSaving.value=true; try{ let date=evForm.value.date; if(evEditing.value) { const r=await fetchUpdateCalendarEvent(evEditing.value.id,{...evForm.value}); date=r.date } else { const r=await fetchAddCalendarEvent({...evForm.value}); date=r.date } evOpen.value=false; ElMessage.success('已保存'); await calRef.value?.reload(date); await refreshUpcoming() }catch(e){ ElMessage.error(resolveError(e,'保存失败')) }finally{ evSaving.value=false } }
</script>

<template>
  <div v-if="showCalendar || showMemo">
    <div class="pv-dock">
      <el-tooltip v-if="showCalendar" content="日历" placement="left"><button type="button" class="pv-dock-btn" aria-label="日历" @click="open('calendar')"><el-icon :size="20"><Calendar /></el-icon></button></el-tooltip>
      <el-tooltip v-if="showMemo" content="便签" placement="left"><button type="button" class="pv-dock-btn" aria-label="便签" @click="open('memo')"><el-icon :size="20"><EditPen /></el-icon></button></el-tooltip>
    </div>
    <el-dialog v-model="visible" :title="title" width="480px" top="8vh" append-to-body destroy-on-close>
      <CalendarPanel v-if="tab==='calendar'" ref="calRef" @add="onCalAdd" @edit="onCalEdit" @remove="onCalRemove" />
      <MemoPanel v-else ref="memoRef" />
    </el-dialog>
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
import { computed, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Calendar, EditPen } from '@element-plus/icons-vue'
import CalendarPanel from './CalendarPanel.vue'
import MemoPanel from './MemoPanel.vue'
import { fetchAddCalendarEvent, fetchDeleteCalendarEvent, fetchUpdateCalendarEvent, resolveError } from '../../api'
import type { CalendarEvent } from '../../types'
// 首页默认显示日历、便签，后台设置里可以分别关掉（开启「隐藏」后这里对应的入口不渲染）
const props = defineProps<{ showCalendar?: boolean; showMemo?: boolean }>()
const showCalendar = computed(()=>props.showCalendar!==false)
const showMemo = computed(()=>props.showMemo!==false)
type Tab = 'calendar' | 'memo'
const tab = ref<Tab>('calendar'); const visible = ref(false)
const title = computed(()=>tab.value==='calendar'?'日历':'便签')
const calRef = ref<InstanceType<typeof CalendarPanel>|null>(null)
const memoRef = ref<InstanceType<typeof MemoPanel>|null>(null)
const open = (t:Tab)=>{ tab.value=t; visible.value=true; if(t==='memo') memoRef.value?.reload?.() }
const evOpen = ref(false); const evEditing = ref<CalendarEvent|null>(null); const evSaving = ref(false)
const evColors = ['#409eff','#67c23a','#e6a23c','#f56c6c','#909399','#9b59b6','#00bcd4']
const evForm = ref({title:'',date:'',startTime:'',endTime:'',desc:'',color:'#409eff'})
const onCalAdd = (date:string)=>{ evEditing.value=null; evForm.value={title:'',date,startTime:'',endTime:'',desc:'',color:'#409eff'}; evOpen.value=true }
const onCalEdit = (ev:CalendarEvent)=>{ evEditing.value=ev; evForm.value={title:ev.title,date:ev.date,startTime:ev.startTime||'',endTime:ev.endTime||'',desc:ev.desc||'',color:ev.color||'#409eff'}; evOpen.value=true }
const onCalRemove = async (ev:CalendarEvent)=>{ try{ await ElMessageBox.confirm(`确定删除「${ev.title}」吗？`,'删除事件',{type:'warning',confirmButtonText:'删除',cancelButtonText:'取消'}) }catch{ return } try{ await fetchDeleteCalendarEvent(ev.id); ElMessage.success('已删除'); await calRef.value?.reload() }catch(e){ ElMessage.error(resolveError(e,'删除失败')) } }
const saveEvent = async ()=>{ if(!evForm.value.title.trim()||!evForm.value.date){ ElMessage.warning('标题与日期不能为空'); return } evSaving.value=true; try{ let date=evForm.value.date; if(evEditing.value) { const r=await fetchUpdateCalendarEvent(evEditing.value.id,{...evForm.value}); date=r.date } else { const r=await fetchAddCalendarEvent({...evForm.value}); date=r.date } evOpen.value=false; ElMessage.success('已保存'); await calRef.value?.reload(date) }catch(e){ ElMessage.error(resolveError(e,'保存失败')) }finally{ evSaving.value=false } }
</script>

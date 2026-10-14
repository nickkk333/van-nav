<template>
  <div class="pv-cal">
    <div class="pv-cal-head">
      <span class="pv-today-num">{{ todayNum }}</span>
      <div class="pv-today-info"><b>{{ todayLabel }}</b><span>{{ lunarLabel }}</span></div>
    </div>
    <div class="pv-cal-nav">
      <el-button size="small" text @click="shiftMonth(-1)"><el-icon><ArrowLeft /></el-icon></el-button>
      <button type="button" class="pv-cal-title" @click="goToday">{{ viewYear }}.{{ viewMonth + 1 }}</button>
      <el-button size="small" text @click="shiftMonth(1)"><el-icon><ArrowRight /></el-icon></el-button>
    </div>
    <div class="pv-week"><span v-for="d in weeks" :key="d">{{ d }}</span></div>
    <div class="pv-grid" v-loading="loading">
      <div v-for="cell in cells" :key="cell.key" class="pv-cell" :class="{ today: cell.today, sel: cell.date === selected }" @click="cell.date && pick(cell.date)">
        <span class="pv-day">{{ cell.day }}</span><span class="pv-nong">{{ cell.sub }}</span><i v-if="cell.hasEvent" class="pv-dot"></i>
      </div>
    </div>
    <div class="pv-daybox">
      <div class="pv-dayhead"><span>{{ selectedLabel }}（{{ dayEvents.length }}）</span><el-button size="small" type="primary" @click="emit('add', selected)">添加</el-button></div>
      <div v-if="!dayEvents.length" class="pv-empty">当天没有安排</div>
      <div v-for="ev in dayEvents" :key="ev.id" class="pv-ev">
        <i :style="{ backgroundColor: ev.color || '#409eff' }"></i>
        <div class="pv-ev-main"><b>{{ ev.title }}</b><span v-if="ev.startTime || ev.desc">{{ ev.startTime }}{{ ev.endTime ? '~' + ev.endTime : '' }}{{ ev.desc ? ' · ' + ev.desc : '' }}</span></div>
        <el-button size="small" text @click="emit('edit', ev)">编辑</el-button>
        <el-button size="small" text type="danger" @click="emit('remove', ev)">删除</el-button>
      </div>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { ArrowLeft, ArrowRight } from '@element-plus/icons-vue'
import { fetchGetCalendarEvents } from '../../api'
import { getJieQi, lunarFestival, solarToLunar } from '../../utils/lunar'
import type { CalendarEvent } from '../../types'
const emit = defineEmits<{ add: [date: string]; edit: [ev: CalendarEvent]; remove: [ev: CalendarEvent] }>()
const FEST: Record<string,string> = {'01-01':'元旦','02-14':'情人','03-08':'妇女','05-01':'劳动','05-04':'青年','06-01':'儿童','07-01':'建党','08-01':'建军','09-10':'教师','10-01':'国庆','12-25':'圣诞'}
const pad = (n:number)=>String(n).padStart(2,'0')
const ds = (d:Date)=>`${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}`
const weeks = ['日','一','二','三','四','五','六']
const view = ref(new Date()); const vy = computed(()=>view.value.getFullYear()); const vm = computed(()=>view.value.getMonth())
const viewYear = computed(()=>vy.value);const viewMonth = computed(()=>vm.value);const mk = computed(()=>`${vy.value}-${pad(vm.value+1)}`)
const selected = ref(ds(new Date())); const events = ref<CalendarEvent[]>([]); const loading = ref(false)
const dates = computed(()=>new Set(events.value.map(e=>e.date)))
const dayEvents = computed(()=>events.value.filter(e=>e.date===selected.value))
const todayNum = computed(()=>new Date().getDate())
const todayLabel = computed(()=>{const n=new Date();return `${n.getFullYear()}年${n.getMonth()+1}月${n.getDate()}日 周${'日一二三四五六'[n.getDay()]}`})
const lunarLabel = computed(()=>{const n=new Date();const l=solarToLunar(n.getFullYear(),n.getMonth()+1,n.getDate());return `${l.month}${l.day}`})
const selectedLabel = computed(()=>{const p=selected.value.split('-');return `${p[0]}年${Number(p[1])}月${Number(p[2])}日`})
const cells = computed(()=>{
  const y=vy.value,m=vm.value,first=new Date(y,m,1).getDay(),days=new Date(y,m+1,0).getDate(),today=ds(new Date())
  const list:{key:string;day:number|'';date:string;today:boolean;sub:string;hasEvent:boolean}[]=[]
  for(let i=0;i<first;i++) list.push({key:'p'+i,day:'',date:'',today:false,sub:'',hasEvent:false})
  for(let d=1;d<=days;d++){
    const date=`${y}-${pad(m+1)}-${pad(d)}`
    const sf=FEST[`${pad(m+1)}-${pad(d)}`]||''; const lf=lunarFestival(y,m+1,d); const jq=getJieQi(y,m+1,d)
    const l=solarToLunar(y,m+1,d); const sub=sf||lf||jq||(l.day==='初一'?l.month:l.day)
    list.push({key:date,day:d,date,today:date===today,sub,hasEvent:dates.value.has(date)})
  }
  return list
})
const load = async ()=>{ loading.value=true; try{ events.value=await fetchGetCalendarEvents(mk.value) }catch(e){ console.error(e) }finally{ loading.value=false } }
const shiftMonth = (n:number)=>{ view.value=new Date(vy.value,vm.value+n,1) }
const goToday = ()=>{ const n=new Date(); view.value=new Date(n.getFullYear(),n.getMonth(),1); selected.value=ds(n) }
const pick = (d:string)=>{ selected.value=d }
const reload = async (date?:string)=>{ if(date&&date.slice(0,7)!==mk.value) view.value=new Date(Number(date.slice(0,4)),Number(date.slice(5,7))-1,1); if(date) selected.value=date; await load() }
watch(mk,load); onMounted(load); defineExpose({ reload })
</script>


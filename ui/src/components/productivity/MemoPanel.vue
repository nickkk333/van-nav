<template>
  <div class="pv-memo">
    <div class="pv-memo-bar">
      <span class="pv-save" :class="saveState">{{ saveText }}</span>
      <div class="pv-memo-actions">
        <el-button size="small" text @click="showVersions">历史版本</el-button>
        <el-button size="small" type="primary" :loading="saving" @click="save(true)">保存</el-button>
      </div>
    </div>
    <el-input v-model="content" type="textarea" :rows="12" placeholder="记录灵感、账号备注、常用文本……输入后自动保存" @input="onInput" />
    <el-dialog v-model="verOpen" title="历史版本" width="420px" append-to-body>
      <div v-if="!versions.length" class="pv-empty">暂无历史版本</div>
      <div v-for="v in versions" :key="v.id" class="pv-ver">
        <div class="pv-ver-main"><b>{{ fmtTime(v.createdAt) }}</b><span>{{ preview(v.content) }}</span></div>
        <el-button size="small" text type="primary" @click="restore(v)">恢复</el-button>
        <el-button size="small" text type="danger" @click="delVer(v)">删除</el-button>
      </div>
    </el-dialog>
  </div>
</template>
<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { fetchDeleteMemoVersion, fetchGetMemo, fetchGetMemoVersions, fetchRestoreMemoVersion, fetchSaveMemo, resolveError } from '../../api'
import type { MemoVersion } from '../../types'
const content = ref(''); const serverContent = ref(''); const loading = ref(false); const saving = ref(false)
const versions = ref<MemoVersion[]>([]); const verOpen = ref(false)
let timer: number | null = null
const dirty = computed(()=>content.value!==serverContent.value)
const saveState = computed(()=>saving.value?'saving':(dirty.value?'dirty':'saved'))
const saveText = computed(()=>saving.value?'保存中…':(dirty.value?'有未保存修改':'已同步'))
const load = async ()=>{ loading.value=true; try{ const m=await fetchGetMemo(); content.value=m.content||''; serverContent.value=content.value }catch(e){ console.error(e) }finally{ loading.value=false } }
const save = async (tip=false)=>{ if(saving.value) return; if(!dirty.value){ if(tip) ElMessage.success('已是最新'); return } saving.value=true; try{ const m=await fetchSaveMemo({content:content.value,mode:'simple'}); serverContent.value=m.content||'' }catch(e){ ElMessage.error(resolveError(e,'保存失败')) }finally{ saving.value=false } }
const onInput = ()=>{ if(timer) window.clearTimeout(timer); timer=window.setTimeout(()=>save(false),1200) }
const showVersions = async ()=>{ try{ versions.value=await fetchGetMemoVersions() }catch(e){ ElMessage.error(resolveError(e,'加载失败')) } verOpen.value=true }
const preview = (s:string)=>(s||'').replace(/\s+/g,' ').slice(0,40)||'(空)'
const fmtTime = (t:number)=>new Date(t).toLocaleString()
const restore = async (v:MemoVersion)=>{ try{ await ElMessageBox.confirm('恢复该版本？当前内容会先归档，可撤销。','恢复版本',{type:'warning'}) }catch{ return } try{ const m=await fetchRestoreMemoVersion(v.id); content.value=m.content||''; serverContent.value=content.value; verOpen.value=false; ElMessage.success('已恢复') }catch(e){ ElMessage.error(resolveError(e,'恢复失败')) } }
const delVer = async (v:MemoVersion)=>{ try{ await fetchDeleteMemoVersion(v.id); versions.value=versions.value.filter(x=>x.id!==v.id) }catch(e){ ElMessage.error(resolveError(e,'删除失败')) } }
const reload = ()=>load()
onMounted(load); onUnmounted(()=>{ if(timer) window.clearTimeout(timer) }); defineExpose({ reload })
</script>

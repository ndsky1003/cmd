<template>
    <div class="file-container">
        <div class="toolbar">
            <button class="btn" @click="goUp">⬆</button>
            <div class="breadcrumb">
                <span class="crumb" @click="jumpBread(-1)">📁</span>
                <span v-for="(p, i) in paths" :key="i" class="crumb" @click="jumpBread(i)">/ {{ p }}</span>
            </div>
            <button class="btn" @click="mkdir">📁 新建文件夹</button>
            <button class="btn" @click="selectFile">📄 上传文件</button>
        </div>
        <div class="grid">
            <div v-if="loading" class="msg">加载中...</div>
            <div v-else-if="files.length === 0" class="msg">目录为空</div>
            <FileContainerItem
                v-for="(item, i) in files" :key="i" :item="item"
                @select="onSelect" @open="onOpen"
            />
        </div>
        <input ref="fileInput" type="file" multiple style="display:none" @change="onUpload" />
        <FileViewer :item="viewerItem" @close="viewerItem = null" />
    </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { ListDir, Mkdir, SaveFile } from '../../wailsjs/go/main/App'
import FileContainerItem from './FileContainerItem.vue'
import FileViewer from './FileViewer.vue'

const emit = defineEmits(['disconnect'])
const paths = ref([])
const files = ref([])
const loading = ref(false)
const fileInput = ref(null)
const viewerItem = ref(null)

onMounted(() => loadFiles())

function destPath() {
    return paths.value.join('/')
}

async function loadFiles() {
    loading.value = true
    try {
        const res = await ListDir(destPath() || '/')
        files.value = res || []
    } catch (e) {
        files.value = []
    } finally {
        loading.value = false
    }
}

function jumpBread(i) {
    if (i === -1) paths.value = []
    else paths.value = paths.value.slice(0, i + 1)
    loadFiles()
}

function goUp() {
    if (paths.value.length === 0) return
    paths.value.pop()
    loadFiles()
}

function onSelect(item) {
    if (item.IsDir) {
        paths.value.push(item.Name)
        loadFiles()
    }
}

function onOpen(item) {
    const full = destPath() ? destPath() + '/' + item.Name : item.Name
    viewerItem.value = { ...item, _fullPath: full }
}

async function mkdir() {
    const name = prompt('输入文件夹名称:')
    if (!name || !name.trim()) return
    const full = destPath() ? destPath() + '/' + name : name
    try {
        await Mkdir(full)
        loadFiles()
    } catch (e) {
        alert('创建失败: ' + e)
    }
}

function selectFile() {
    fileInput.value.value = ''
    fileInput.value.click()
}

async function onUpload(e) {
    const files = e.target.files
    if (!files || files.length === 0) return
    for (const file of files) {
        const buf = await file.arrayBuffer()
        const data = new Uint8Array(buf)
        const filename = destPath() ? destPath() + '/' + file.name : file.name
        try {
            await SaveFile(filename, Array.from(data))
        } catch (e) {
            alert('上传失败: ' + file.name + ' - ' + e)
        }
    }
    loadFiles()
}
</script>

<style scoped>
.file-container { display: flex; flex-direction: column; height: 100%; }
.toolbar {
    display: flex; align-items: center; gap: 8px; padding: 8px 0;
    border-bottom: 1px solid rgba(255,255,255,0.1); flex-shrink: 0;
}
.btn {
    padding: 6px 14px; border: none; border-radius: 4px; cursor: pointer;
    background: rgba(255,255,255,0.1); color: white; font-size: 13px; white-space: nowrap;
}
.btn:hover { background: rgba(255,255,255,0.2); }
.breadcrumb { flex: 1; font-size: 14px; overflow: hidden; white-space: nowrap; }
.crumb {
    cursor: pointer; display: inline; padding: 2px 4px; border-radius: 3px;
}
.crumb:hover { background: rgba(255,255,255,0.1); }
.grid {
    display: flex; flex-wrap: wrap; align-content: flex-start;
    flex: 1; overflow-y: auto; padding: 4px 0;
}
.msg { width: 100%; text-align: center; padding: 40px; color: rgba(255,255,255,0.3); }
</style>

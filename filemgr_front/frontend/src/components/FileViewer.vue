<template>
    <div v-if="visible" class="viewer-overlay" @click.self="close">
        <div class="viewer-dialog">
            <div class="viewer-header">
                <span class="viewer-title">{{ filename }}</span>
                <button class="viewer-close" @click="close">✕</button>
            </div>
            <div class="viewer-body">
                <img v-if="isImage" :src="imageSrc" class="viewer-img" />
                <pre v-else class="viewer-text"><code>{{ content }}</code></pre>
            </div>
        </div>
    </div>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import { ReadFile } from '../../wailsjs/go/main/App'
import { OpenFile } from '../../wailsjs/go/main/App'

const props = defineProps({
    item: { type: Object, default: null }
})
const emit = defineEmits(['close'])

const visible = ref(false)
const content = ref('')
const imageSrc = ref('')

const textExts = ['.txt','.md','.json','.xml','.yaml','.yml','.toml','.go','.py','.js','.ts','.css','.html','.sh','.log','.conf','.cfg','.ini','.csv','.env','.gitignore','.dockerfile','.makefile','.gradle','.properties']
const imgExts = ['.png','.jpg','.jpeg','.gif','.svg','.bmp','.webp']

const isText = computed(() => textExts.includes(props.item?.Ext))
const isImage = computed(() => imgExts.includes(props.item?.Ext))
const filename = computed(() => props.item?.Name || '')

function close() {
    visible.value = false
    emit('close')
}

async function open(item) {
    if (!item || item.IsDir) return
    const path = item._fullPath || item.Name
    visible.value = true

    if (isText.value) {
        content.value = '加载中...'
        try {
            const data = await ReadFile(path)
            content.value = new TextDecoder().decode(new Uint8Array(data))
        } catch (e) {
            content.value = '读取失败: ' + e
        }
    } else if (isImage.value) {
        try {
            const data = await ReadFile(path)
            const blob = new Blob([new Uint8Array(data)])
            imageSrc.value = URL.createObjectURL(blob)
        } catch (e) {
            content.value = '读取失败: ' + e
            imageSrc.value = ''
        }
    } else {
        visible.value = false
        try {
            await OpenFile(path)
        } catch (e) {
            alert('打开失败: ' + e)
        }
    }
}

watch(() => props.item, (val) => { if (val) open(val) })
</script>

<style scoped>
.viewer-overlay {
    position: fixed; inset: 0; z-index: 1000;
    background: rgba(0,0,0,0.6); display: flex;
    align-items: center; justify-content: center;
}
.viewer-dialog {
    width: 80vw; height: 80vh; display: flex; flex-direction: column;
    background: #1e1e2e; border-radius: 8px; overflow: hidden;
}
.viewer-header {
    display: flex; align-items: center; padding: 8px 16px;
    background: rgba(255,255,255,0.05); flex-shrink: 0;
}
.viewer-title { flex: 1; font-size: 14px; color: rgba(255,255,255,0.8); }
.viewer-close {
    border: none; background: none; color: rgba(255,255,255,0.5);
    font-size: 18px; cursor: pointer; padding: 4px 8px;
}
.viewer-close:hover { color: white; }
.viewer-body { flex: 1; overflow: auto; padding: 16px; }
.viewer-text {
    margin: 0; font-family: monospace; font-size: 13px; line-height: 1.5;
    white-space: pre-wrap; word-break: break-all;
}
.viewer-img { max-width: 100%; display: block; margin: auto; }
</style>

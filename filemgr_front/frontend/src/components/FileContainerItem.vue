<template>
    <div class="file-item" @click="onClick" @dblclick="onDblClick">
        <div class="icon">{{ icon }}</div>
        <div class="name">{{ item.Name }}</div>
    </div>
</template>

<script setup>
import { computed } from 'vue'

const props = defineProps({
    item: { type: Object, required: true }
})
const emit = defineEmits(['select', 'open'])

const iconMap = {
    '.txt': '📄', '.md': '📝', '.json': '📋', '.log': '📋',
    '.sh': '⚡', '.go': '🔵', '.py': '🐍', '.js': '🟨',
    '.html': '🌐', '.css': '🎨', '.xml': '📋', '.yaml': '📋',
    '.jpg': '🖼', '.jpeg': '🖼', '.png': '🖼', '.gif': '🖼', '.svg': '🖼',
    '.mp3': '🎵', '.wav': '🎵',
    '.mp4': '🎬', '.avi': '🎬', '.mkv': '🎬',
    '.zip': '📦', '.tar': '📦', '.gz': '📦', '.7z': '📦', '.rar': '📦',
    '.pdf': '📕', '.doc': '📘', '.docx': '📘',
}

const icon = computed(() => {
    if (props.item.IsDir) return '📁'
    return iconMap[props.item.Ext] || '📄'
})

function onClick() {
    emit('select', props.item)
}
function onDblClick() {
    if (!props.item.IsDir) emit('open', props.item)
}
</script>

<style scoped>
.file-item {
    display: flex; flex-direction: column; align-items: center;
    justify-content: center; width: 100px; height: 110px;
    margin: 8px; padding: 8px; border-radius: 8px;
    cursor: pointer; background: rgba(255,255,255,0.03);
    transition: background 0.2s;
}
.file-item:hover { background: rgba(255,255,255,0.1); }
.icon { font-size: 36px; line-height: 1; }
.name {
    font-size: 12px; text-align: center; word-break: break-all;
    overflow: hidden; text-overflow: ellipsis; display: -webkit-box;
    -webkit-line-clamp: 2; -webkit-box-orient: vertical; margin-top: 4px;
}
</style>

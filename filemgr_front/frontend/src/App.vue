<template>
    <div id="app-root">
        <div v-if="!connected" class="connect-panel">
            <h2>连接到文件管理器</h2>
            <div class="connect-row">
                <input v-model="addr" placeholder="地址" />
                <input v-model="name" placeholder="服务名" />
                <input v-model="secret" type="password" placeholder="密钥" />
                <button class="btn primary" @click="connect">连接</button>
                <span class="status" :class="{ error: connErr }">{{ connMsg }}</span>
            </div>
        </div>
        <FileContainer v-else @disconnect="disconnect" />
        <div v-if="connected" class="top-bar">
            <span class="conn-info">{{ connLabel }}</span>
            <button class="btn small" @click="disconnect">断开</button>
        </div>
    </div>
</template>

<script setup>
import { ref } from 'vue'
import { Connect, Disconnect } from '../wailsjs/go/main/App'
import FileContainer from './components/FileContainer.vue'

const addr = ref('127.0.0.1:18083')
const name = ref('filemgr')
const secret = ref('')
const connected = ref(false)
const connErr = ref(false)
const connMsg = ref('')
const connLabel = ref('')

async function connect() {
    connMsg.value = '连接中...'
    connErr.value = false
    try {
        await Connect(addr.value, name.value, secret.value)
        connected.value = true
        connLabel.value = `${name.value} @ ${addr.value}`
    } catch (e) {
        connMsg.value = '连接失败: ' + e
        connErr.value = true
    }
}

async function disconnect() {
    try { await Disconnect() } catch (_) {}
    connected.value = false
    connMsg.value = ''
}
</script>

<style scoped>
#app-root { height: 100vh; display: flex; flex-direction: column; }
.connect-panel {
    max-width: 700px; margin: 40px auto; padding: 24px;
    background: rgba(255,255,255,0.05); border-radius: 8px; text-align: center;
}
h2 { margin: 0 0 16px; font-size: 18px; }
.connect-row { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.connect-row input {
    flex: 1; min-width: 120px; padding: 8px 12px; border: none; border-radius: 4px;
    background: rgba(255,255,255,0.9); outline: none;
}
.btn {
    padding: 8px 20px; border: none; border-radius: 4px; cursor: pointer;
    background: rgba(255,255,255,0.1); color: white;
}
.btn.primary { background: #4caf50; font-weight: bold; }
.btn.primary:hover { background: #45a049; }
.btn.small { padding: 4px 12px; font-size: 12px; background: #e53935; }
.btn.small:hover { background: #c62828; }
.status { line-height: 34px; font-size: 13px; color: #4caf50; }
.status.error { color: #e53935; }
.top-bar {
    display: flex; align-items: center; gap: 8px; padding: 4px 12px;
    background: rgba(0,0,0,0.3); flex-shrink: 0;
}
.conn-info { flex: 1; font-size: 12px; color: rgba(255,255,255,0.5); }
</style>

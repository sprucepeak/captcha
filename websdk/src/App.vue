<template>
  <div class="demo-container">
    <header class="demo-header">
      <h1>Captcha Vue3 + TS SDK Demo</h1>
      <p>行为验证码 Vue3 SDK 演示工程</p>
    </header>

    <!-- Control bar -->
    <div class="demo-controls">
      <div class="control-group">
        <label>选择验证码类型：</label>
        <button
          v-for="t in types"
          :key="t.key"
          :class="{ active: currentType === t.key }"
          class="btn-type"
          @click="changeType(t.key)"
        >
          {{ t.label }}
        </button>
      </div>

      <div class="control-group">
        <label>展示模式：</label>
        <button
          :class="{ active: !dialogVisible }"
          class="btn-mode"
          @click="dialogVisible = false"
        >
          嵌入式 (Inline)
        </button>
        <button
          :class="{ active: dialogVisible }"
          class="btn-mode"
          @click="dialogVisible = true"
        >
          弹窗式 (Modal)
        </button>
      </div>
    </div>

    <!-- Main Content Area -->
    <main class="demo-main">
      <!-- Inline Mode -->
      <div v-if="!dialogVisible" class="captcha-card">
        <Captcha
          ref="captchaRef"
          :key="currentType"
          :type="currentType"
          :width="340"
          @success="onSuccess"
          @fail="onFail"
        />
      </div>

      <!-- Modal Mode -->
      <div v-else class="modal-trigger-area">
        <button class="btn-primary" @click="showModal = true">点击弹出验证码</button>

        <div v-if="showModal" class="modal-overlay" @click.self="showModal = false">
          <div class="modal-content">
            <Captcha
              :key="currentType"
              :type="currentType"
              :width="340"
              @success="onModalSuccess"
              @fail="onFail"
              @close="showModal = false"
            />
          </div>
        </div>
      </div>

      <!-- Logs panel -->
      <div class="log-panel">
        <div class="log-header">
          <span>事件交互日志</span>
          <button class="btn-clear" @click="logs = []">清空</button>
        </div>
        <div class="log-body">
          <div v-if="logs.length === 0" class="log-empty">暂无事件产生，请操作验证码...</div>
          <div
            v-for="(log, idx) in logs"
            :key="idx"
            class="log-item"
            :class="`log-${log.type}`"
          >
            <span class="log-time">[{{ log.time }}]</span>
            <span class="log-msg">{{ log.msg }}</span>
          </div>
        </div>
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue';
import type { CaptchaType } from './types';
import Captcha from './components/Captcha.vue';

const types: { key: CaptchaType; label: string }[] = [
  { key: 'SLIDER', label: '滑块拼图 (SLIDER)' },
  { key: 'SLIDER_V2', label: '滑块验证V2 (SLIDER_V2)' },
  { key: 'ROTATE', label: '旋转图片 (ROTATE)' },
  { key: 'ROTATE_DEGREE', label: '角度验证 (ROTATE_DEGREE)' },
  { key: 'CONCAT', label: '滑动还原 (CONCAT)' },
  { key: 'WORD_IMAGE_CLICK', label: '文字点选 (WORD_CLICK)' },
  { key: 'SHAPE_CLICK', label: '图形点选 (SHAPE_CLICK)' },
  { key: 'DRAG_DROP', label: '自由拖拽 (DRAG_DROP)' }
];

const currentType = ref<CaptchaType>('SLIDER');
const dialogVisible = ref(false);
const showModal = ref(false);
const captchaRef = ref();

interface LogItem {
  time: string;
  type: 'info' | 'success' | 'error';
  msg: string;
}

const logs = ref<LogItem[]>([]);

const addLog = (type: 'info' | 'success' | 'error', msg: string) => {
  const now = new Date();
  const time = now.toTimeString().split(' ')[0] + '.' + String(now.getMilliseconds()).padStart(3, '0');
  logs.value.unshift({ time, type, msg });
};

const changeType = (t: CaptchaType) => {
  currentType.value = t;
  addLog('info', `切换验证码类型为: ${t}`);
};

const onSuccess = (data: any) => {
  addLog('success', `验证通过！响应数据: ${JSON.stringify(data)}`);
};

const onModalSuccess = (data: any) => {
  addLog('success', `弹窗验证通过！`);
  setTimeout(() => {
    showModal.value = false;
  }, 1000);
};

const onFail = (msg: string) => {
  addLog('error', `验证失败: ${msg}`);
};
</script>

<style>
* {
  margin: 0;
  padding: 0;
  box-sizing: border-box;
}

body {
  background-color: #f0f2f5;
  color: #333;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
}

.demo-container {
  max-width: 900px;
  margin: 40px auto;
  padding: 0 20px;
}

.demo-header {
  text-align: center;
  margin-bottom: 28px;
}

.demo-header h1 {
  font-size: 26px;
  color: #1f2937;
  margin-bottom: 8px;
}

.demo-header p {
  color: #6b7280;
  font-size: 14px;
}

.demo-controls {
  background: #ffffff;
  padding: 16px 20px;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  margin-bottom: 24px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.control-group {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
}

.control-group label {
  font-size: 14px;
  font-weight: 500;
  color: #4b5563;
  min-width: 120px;
}

.btn-type, .btn-mode {
  padding: 6px 14px;
  background: #f3f4f6;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  cursor: pointer;
  font-size: 13px;
  transition: all 0.2s;
}

.btn-type:hover, .btn-mode:hover {
  background: #e5e7eb;
}

.btn-type.active, .btn-mode.active {
  background: #1890ff;
  border-color: #1890ff;
  color: #ffffff;
}

.demo-main {
  display: grid;
  grid-template-columns: 380px 1fr;
  gap: 24px;
}

@media (max-width: 768px) {
  .demo-main {
    grid-template-columns: 1fr;
  }
}

.captcha-card {
  display: flex;
  justify-content: center;
}

.modal-trigger-area {
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  border-radius: 8px;
  padding: 40px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
}

.btn-primary {
  padding: 10px 24px;
  background: #1890ff;
  color: #ffffff;
  border: none;
  border-radius: 6px;
  font-size: 15px;
  cursor: pointer;
  transition: background 0.2s;
}

.btn-primary:hover {
  background: #40a9ff;
}

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.modal-content {
  animation: modal-fade 0.2s ease-out;
}

@keyframes modal-fade {
  from { transform: scale(0.9); opacity: 0; }
  to { transform: scale(1); opacity: 1; }
}

.log-panel {
  background: #ffffff;
  border-radius: 8px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
  display: flex;
  flex-direction: column;
  height: 380px;
}

.log-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #f0f0f0;
  font-size: 14px;
  font-weight: 600;
}

.btn-clear {
  background: none;
  border: none;
  color: #8c8c8c;
  cursor: pointer;
  font-size: 12px;
}

.btn-clear:hover {
  color: #ff4d4f;
}

.log-body {
  padding: 12px;
  overflow-y: auto;
  font-family: monospace;
  font-size: 12px;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.log-empty {
  color: #9ca3af;
  text-align: center;
  margin-top: 40px;
}

.log-item {
  display: flex;
  gap: 8px;
  line-height: 1.5;
}

.log-time {
  color: #9ca3af;
  flex-shrink: 0;
}

.log-info .log-msg { color: #1f2937; }
.log-success .log-msg { color: #52c41a; font-weight: bold; }
.log-error .log-msg { color: #f5222d; font-weight: bold; }
</style>


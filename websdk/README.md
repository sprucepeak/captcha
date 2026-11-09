# Captcha Vue3 + TypeScript SDK & Demo

使用 **Vue 3 (Composition API / `<script setup>`) + TypeScript + Vite** 构建的现代化前端行为验证码 SDK 与完整演示工程。支持 **滑块拼图 (SLIDER)**、**滑块验证V2 (SLIDER_V2)**、**旋转验证 (ROTATE)**、**角度验证 (ROTATE_DEGREE)**、**滑动还原 (CONCAT)** 和 **文字点选 (WORD_IMAGE_CLICK)**。

---

## 核心特性

- 🎯 **完整支持 4 种验证码类型**：
  - **滑块拼图 (SLIDER)**：按比例自适应凹槽对齐与实时滑块联动
  - **旋转验证 (ROTATE)**：角度与滑动拖拽实时映射计算
  - **滑动还原 (CONCAT)**：错位切片实时移动拼接对齐
  - **文字点选 (WORD_IMAGE_CLICK)**：字符提示横幅、画板点击序号打点与提交核销
- 🛡️ **高拟真轨迹采集**：
  - 自动记录按压 (`down`)、移动 (`move`)、抬起 (`up`)、点击 (`click`) 坐标与毫秒时间差 (`t`)
  - 采样坐标自动根据自然图像分辨率进行等比缩放转换，完全对齐后端 `BasicCaptchaTrackValidator` 防刷算法
- 📱 **多端自适应**：
  - 同时支持 PC 端鼠标拖拽/点击与移动端触控 Touch 事件
- 📦 **轻量 SDK 导出**：
  - 支持 ES Module、UMD 与完整 TypeScript 类型定义（`.d.ts`）导出
  - 支持在其他 Vue3 项目中以组件形式直接引入

---

## 目录结构

```
websdk/
├── index.html                  # 演示页 HTML 容器
├── package.json
├── tsconfig.json
├── vite.config.ts              # 演示工程 Vite 配置 (内置 /captcha 代理到 Go 后端)
├── vite.config.lib.ts          # SDK 打包配置文件
├── src/
│   ├── index.ts                # SDK 统一导出入口 (Captcha 组件、类型、Composable)
│   ├── types/
│   │   └── index.ts            # CaptchaVO, TrackData, ApiResponse 等完整 TS 类型定义
│   ├── composables/
│   │   └── useCaptchaTrack.ts  # 行为轨迹采集与计算 Composable
│   ├── components/
│   │   └── Captcha.vue   # 核心 Vue3 验证码组件
│   ├── App.vue                 # 交互式 Demo 演示主界面 (支持 Inline 与 Modal 弹窗模式)
│   └── main.ts                 # Demo 应用启动入口
└── dist/                       # SDK 构建输出 (ES / UMD / CSS / d.ts)
```

---

## 快速使用

### 1. 运行本地开发与 Demo 演示

```bash
cd websdk
npm install
npm run dev
```

默认在浏览器打开 `http://localhost:3000`。

> 提示：本地配合运行 `e:\workspace\research\captcha\go` 启动后端服务（端口 8080），`vite.config.ts` 已内置将 `/captcha` 请求代理至 `http://localhost:8080`。

### 2. 构建 SDK 库文件

```bash
npm run build:sdk
```

构建后产出物位于 `websdk/dist`：

- `index.js` (ES Module)
- `index.umd.cjs` (UMD)
- `index.d.ts` (TypeScript 声明文件)
- `style.css` (组件样式文件)

---

## 在其他 Vue3 项目中使用

### 安装或引入组件

```vue
<template>
  <div class="login-box">
    <!-- 使用 Captcha 组件 -->
    <Captcha
      type="SLIDER"
      :width="320"
      @success="handleSuccess"
      @fail="handleFail"
    />
  </div>
</template>

<script setup lang="ts">
import { Captcha } from "captcha-vue3";
import "captcha-vue3/dist/style.css";

const handleSuccess = (res: any) => {
  console.log("验证成功:", res);
};

const handleFail = (msg: string) => {
  console.warn("验证失败:", msg);
};
</script>
```

### 自定义接口请求（可选）

如果后端接口地址或响应结构非默认，可直接通过 `requestCaptchaData` 和 `validCaptchaData` 属性注入自定义方法：

```vue
<Captcha
  :width="340"
  :request-captcha-data="myGetCaptchaApi"
  :valid-captcha-data="myVerifyApi"
  @success="onSuccess"
/>
```

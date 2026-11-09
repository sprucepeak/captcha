<template>
  <div class="tac-captcha" :style="{ width: `${width}px` }">
    <!-- Header: Tip or Action bar -->
    <div class="tac-header">
      <div v-if="captchaData?.type === 'WORD_IMAGE_CLICK' || captchaData?.type === 'SHAPE_CLICK'" class="tac-word-tip">
        <span>请依次点击：</span>
        <img v-if="captchaData?.templateImage" :src="captchaData.templateImage" class="tac-tip-banner" alt="tip" />
      </div>
      <div v-else class="tac-title">
        <span v-if="captchaData?.type === 'ROTATE'">请拖动滑块使图片拼接完整</span>
        <span v-else-if="captchaData?.type === 'ROTATE_DEGREE'">请拖动滑块将图片摆正</span>
        <span v-else-if="captchaData?.type === 'CONCAT'">请拖动滑块拼合完整图片</span>
        <span v-else-if="captchaData?.type === 'SLIDER_V2'">请沿轨迹拖动滑块完成拼图</span>
        <span v-else-if="captchaData?.type === 'DRAG_DROP'">请将拼图块自由拖拽放置到缺口中</span>
        <span v-else>拖动滑块完成拼图</span>
      </div>
      <div class="tac-actions">
        <span v-if="showRefresh" class="tac-icon-btn" title="刷新" @click="refresh">↻</span>
        <span v-if="showClose" class="tac-icon-btn" title="关闭" @click="$emit('close')">✕</span>
      </div>
    </div>

    <!-- Image Board -->
    <div
      ref="boardRef"
      class="tac-board"
      :style="{ height: `${boardHeight}px` }"
      @click="handleBoardClick"
    >
      <div v-if="loading" class="tac-loading">
        <div class="tac-spinner"></div>
        <span>加载中...</span>
      </div>

      <template v-else-if="captchaData">
        <!-- Rotate Degree (ROTATE_DEGREE: 单独圆形图片居中旋转摆正) -->
        <div v-if="captchaData.type === 'ROTATE_DEGREE'" class="tac-rotate-degree-wrap">
          <div class="tac-rotate-circle-container">
            <img
              :src="captchaData.backgroundImage"
              class="tac-rotate-degree-img"
              :style="{ transform: `rotate(${rotateDeg}deg)` }"
              alt="rotate degree"
              draggable="false"
            />
            <!-- 拖动时的辅助对齐十字虚线 -->
            <div v-if="isDragging" class="tac-crosshair-line tac-crosshair-h"></div>
            <div v-if="isDragging" class="tac-crosshair-line tac-crosshair-v"></div>
          </div>
        </div>

        <!-- Default Background Image (SLIDER / ROTATE / CONCAT / WORD_IMAGE_CLICK) -->
        <img
          v-else
          :src="captchaData.backgroundImage"
          class="tac-bg-img"
          alt="background"
          draggable="false"
        />

        <!-- Slider Template Image (SLIDER) -->
        <img
          v-if="captchaData.type === 'SLIDER' && captchaData.templateImage"
          :src="captchaData.templateImage"
          class="tac-slider-template"
          :style="{
            left: `${sliderLeft}px`,
            width: `${sliderTplWidth}px`,
            height: `${boardHeight}px`
          }"
          alt="template"
          draggable="false"
        />

        <!-- SliderV2 Non-Linear / Curve Slider (SLIDER_V2: 曲线爬升滑块) -->
        <img
          v-else-if="captchaData.type === 'SLIDER_V2' && captchaData.templateImage"
          :src="captchaData.templateImage"
          class="tac-slider-v2-template"
          :style="{
            left: `${sliderV2Pos.x}px`,
            top: `${sliderV2Pos.y}px`,
            width: `${sliderV2Width}px`,
            height: `${sliderV2Height}px`
          }"
          alt="slider v2 template"
          draggable="false"
        />

        <!-- Rotate Template Image (ROTATE) -->
        <div
          v-else-if="captchaData.type === 'ROTATE' && captchaData.templateImage"
          class="tac-rotate-template-wrap"
        >
          <img
            :src="captchaData.templateImage"
            class="tac-rotate-template"
            :style="{
              transform: `rotate(${rotateDeg}deg)`
            }"
            alt="rotate template"
            draggable="false"
          />
        </div>

        <!-- Concat Split Slide (CONCAT) -->
        <div
          v-else-if="captchaData.type === 'CONCAT'"
          class="tac-concat-mask"
          :style="{
            height: `${concatTopHeight}px`,
            backgroundImage: `url(${captchaData.backgroundImage})`,
            backgroundSize: `${width}px ${boardHeight}px`,
            backgroundPositionX: `${concatLeft}px`,
            backgroundPositionY: '0px'
          }"
        ></div>

        <!-- DragDrop Free Floating Puzzle Piece (DRAG_DROP) -->
        <div
          v-else-if="captchaData.type === 'DRAG_DROP' && captchaData.templateImage"
          class="tac-drag-piece"
          :style="{
            left: `${dragDropPos.x}px`,
            top: `${dragDropPos.y}px`,
            width: `${dragDropSize.w}px`,
            height: `${dragDropSize.h}px`,
            cursor: isDraggingPiece ? 'grabbing' : 'grab'
          }"
          @mousedown.stop="onPieceDragStart"
          @touchstart.stop.prevent="onPieceDragStart"
        >
          <img
            :src="captchaData.templateImage"
            class="tac-drag-piece-img"
            alt="drag tile"
            draggable="false"
          />
        </div>

        <!-- Word / Shape Click Markers -->
        <div v-if="captchaData.type === 'WORD_IMAGE_CLICK' || captchaData.type === 'SHAPE_CLICK'" class="tac-click-layer">
          <span
            v-for="(pt, idx) in clickPoints"
            :key="idx"
            class="tac-click-dot"
            :style="{ left: `${pt.x - 11}px`, top: `${pt.y - 11}px` }"
          >
            {{ idx + 1 }}
          </span>
        </div>
      </template>

      <!-- Feedback Tip banner -->
      <transition name="tac-fade">
        <div
          v-if="tipState.show"
          class="tac-tip"
          :class="`tac-tip-${tipState.type}`"
        >
          {{ tipState.text }}
        </div>
      </transition>
    </div>

    <!-- Bottom Slider / Button bar -->
    <div class="tac-footer">
      <!-- Tip bar for DRAG_DROP -->
      <div v-if="captchaData?.type === 'DRAG_DROP'" class="tac-dragdrop-tip-bar">
        <span>按住上方拼图块拖拽至缺口处松开</span>
      </div>

      <!-- Slider bar for SLIDER, SLIDER_V2, ROTATE, CONCAT -->
      <div
        v-else-if="captchaData?.type !== 'WORD_IMAGE_CLICK' && captchaData?.type !== 'SHAPE_CLICK'"
        ref="trackBarRef"
        class="tac-track-bar"
      >
        <div class="tac-track-bg" :style="{ width: `${handleLeft + 20}px` }"></div>
        <div class="tac-track-tip">按住左边滑块拖动完成验证</div>
        <div
          ref="handleRef"
          class="tac-slider-handle"
          :style="{ transform: `translateX(${handleLeft}px)` }"
          @mousedown="onDragStart"
          @touchstart.prevent="onDragStart"
        >
          <span class="tac-arrow">→</span>
        </div>
      </div>

      <!-- Confirm Button for Word Click & Shape Click -->
      <button
        v-else
        class="tac-confirm-btn"
        :disabled="clickPoints.length === 0"
        @click="submitClickPoints"
      >
        确认 ({{ clickPoints.length }})
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted, nextTick } from 'vue';
import type { CaptchaProps, CaptchaEmits, CaptchaVO, CaptchaTrackPoint, CaptchaTrackData } from '../types';
import { useCaptchaTrack } from '../composables/useCaptchaTrack';
import { collectClientEnv, encryptPayload } from '../composables/useSecurity';

const props = withDefaults(defineProps<CaptchaProps>(), {
  type: 'SLIDER',
  width: 310,
  showRefresh: true,
  showClose: true
});

const emit = defineEmits<CaptchaEmits>();

const boardRef = ref<HTMLDivElement>();
const trackBarRef = ref<HTMLDivElement>();
const handleRef = ref<HTMLDivElement>();

const loading = ref(false);
const captchaData = ref<CaptchaVO | null>(null);

// Scale ratio from natural background size to display size
const scaleRatio = computed(() => {
  if (!captchaData.value || !captchaData.value.backgroundImageWidth) return 1;
  return props.width / captchaData.value.backgroundImageWidth;
});

const boardHeight = computed(() => {
  if (!captchaData.value) return 160;
  return Math.round(captchaData.value.backgroundImageHeight * scaleRatio.value);
});

// Slider width scaled
const sliderTplWidth = computed(() => {
  if (!captchaData.value || !captchaData.value.templateImageWidth) return 50;
  return Math.round(captchaData.value.templateImageWidth * scaleRatio.value);
});

// Interactive state
const handleLeft = ref(0);
const isDragging = ref(false);
const dragStartX = ref(0);
const maxHandleDrag = computed(() => {
  if (!trackBarRef.value) return props.width - 40;
  return trackBarRef.value.clientWidth - 40;
});

// Actual slider left mapped to image board
// 滑动拼图块的横向移动位移映射：与整幅背景图宽度严格等比联动
const sliderLeft = computed(() => {
  if (maxHandleDrag.value <= 0) return 0;
  return (handleLeft.value / maxHandleDrag.value) * props.width;
});

// Rotate degree mapped (0 到 360 度)
const rotateDeg = computed(() => {
  if (maxHandleDrag.value <= 0) return 0;
  return (handleLeft.value / maxHandleDrag.value) * 360;
});

// Concat top cut height (对齐Java StandardConcatImageCaptchaGenerator与web sdk concat.js)
// bgImageSplit 顶部切片的高度在原图是 bgH - randomY，所以展示高度必须是 (bgH - randomY) * scale
const concatTopHeight = computed(() => {
  if (!captchaData.value) return 0;
  const bgH = captchaData.value.backgroundImageHeight;
  const randomY = (captchaData.value.data?.randomY ?? (captchaData.value as any).randomY ?? 0) as number;
  return Math.round((bgH - randomY) * scaleRatio.value);
});

// Concat offset mapped: 背景图位移距离
const concatLeft = computed(() => {
  if (maxHandleDrag.value <= 0) return 0;
  return (handleLeft.value / maxHandleDrag.value) * props.width;
});

// SliderV2 曲线轨迹二维坐标计算
const sliderV2Width = computed(() => {
  if (!captchaData.value || !captchaData.value.templateImageWidth) return 48;
  return Math.round(captchaData.value.templateImageWidth * scaleRatio.value);
});

const sliderV2Height = computed(() => {
  if (!captchaData.value || !captchaData.value.templateImageHeight) return 48;
  return Math.round(captchaData.value.templateImageHeight * scaleRatio.value);
});

// 依据用户拖动百分比进度 t 计算二次贝塞尔曲线上的 (x, y) 坐标
const sliderV2Pos = computed(() => {
  if (!captchaData.value || maxHandleDrag.value <= 0) {
    return { x: 0, y: 0 };
  }
  const data = captchaData.value.data || {};
  const scale = scaleRatio.value;

  const bgW = captchaData.value.backgroundImageWidth;
  const bgH = captchaData.value.backgroundImageHeight;

  // 严格从原始数据中获取起始点与终点缺口坐标
  const rawStartY = ((data.startY ?? (bgH / 2)) as number);
  const rawTargetX = ((data.targetX ?? (data.randomX ?? (bgW * 0.7))) as number);
  const rawTargetY = ((data.targetY ?? (data.randomY ?? (bgH / 2))) as number);
  const rawControlY = ((data.controlY ?? (rawTargetY - 25)) as number);

  // t 是滑块拖动手柄占整个轨道宽度的百分比进度 (0.0 ~ 1.0)
  const t = Math.max(0, Math.min(1, handleLeft.value / maxHandleDrag.value));

  // 1. X 坐标随手柄 1:1 线性位移
  // 当 t = rawTargetX / bgW 时，x 恰好等于 targetX (rawTargetX * scale)，即阴影缺口的绝对位置！
  const x = t * props.width;

  // 2. Y 坐标在到达缺口前沿抛物线产生优雅微弧，到达缺口时高度与缺口 100% 绝对重合
  const targetRatio = rawTargetX / bgW;
  const p = targetRatio > 0 ? Math.min(1, t / targetRatio) : t;
  const oneMinusP = 1 - p;
  const y = (oneMinusP * oneMinusP * rawStartY + 2 * oneMinusP * p * rawControlY + p * p * rawTargetY) * scale;

  return { x: Math.round(x), y: Math.round(y) };
});

// DragDrop 自由拖拽拼图状态
const isDraggingPiece = ref(false);
const dragPiecePos = reactive({ x: 0, y: 0 });
const dragPieceOffset = reactive({ x: 0, y: 0 });

const dragDropSize = computed(() => {
  if (!captchaData.value || !captchaData.value.templateImageWidth) return { w: 50, h: 50 };
  return {
    w: Math.round(captchaData.value.templateImageWidth * scaleRatio.value),
    h: Math.round((captchaData.value.templateImageHeight || captchaData.value.templateImageWidth) * scaleRatio.value)
  };
});

// 重置/初始化拖拽拼图块位置到后端返回的 initX, initY
const initDragPiecePos = () => {
  if (!captchaData.value) {
    dragPiecePos.x = 10;
    dragPiecePos.y = 10;
    return;
  }
  const data = captchaData.value.data || {};
  const scale = scaleRatio.value;
  const initX = Math.round(((data.initX ?? 20) as number) * scale);
  const initY = Math.round(((data.initY ?? 100) as number) * scale);
  dragPiecePos.x = initX;
  dragPiecePos.y = initY;
};

const dragDropPos = computed(() => {
  return { x: dragPiecePos.x, y: dragPiecePos.y };
});

// Word / Shape Click state
const clickPoints = ref<{ x: number; y: number }[]>([]);

// Feedback Tip
const tipState = reactive({
  show: false,
  type: 'success' as 'success' | 'error',
  text: ''
});

const showTip = (type: 'success' | 'error', text: string) => {
  tipState.show = true;
  tipState.type = type;
  tipState.text = text;
  setTimeout(() => {
    tipState.show = false;
  }, 1500);
};

// Track recording hook
const {
  startTime,
  trackList,
  startTrack,
  addMoveTrack,
  addClickTrack,
  resetTrack
} = useCaptchaTrack();

// Fetch captcha data
const fetchCaptcha = async () => {
  loading.value = true;
  resetAll();
  try {
    if (props.requestCaptchaData) {
      const res = await props.requestCaptchaData();
      if ('data' in res && res.data) {
        captchaData.value = res.data as CaptchaVO;
      } else {
        captchaData.value = res as CaptchaVO;
      }
    } else {
      // Default fallback fetch to /captcha/gen
      const resp = await fetch(`/captcha/gen?type=${props.type}`);
      const json = await resp.json();
      if (json.success && json.data) {
        captchaData.value = json.data;
      } else {
        showTip('error', json.msg || '获取验证码失败');
      }
    }
    if (captchaData.value?.type === 'DRAG_DROP') {
      initDragPiecePos();
    }
  } catch (err: any) {
    showTip('error', err.message || '网络异常');
  } finally {
    loading.value = false;
  }
};

const resetAll = () => {
  handleLeft.value = 0;
  isDragging.value = false;
  isDraggingPiece.value = false;
  clickPoints.value = [];
  resetTrack();
};

const refresh = () => {
  emit('refresh');
  fetchCaptcha();
};

// Mouse / Touch Drag handlers
const onDragStart = (e: MouseEvent | TouchEvent) => {
  if (loading.value || !captchaData.value) return;
  isDragging.value = true;
  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
  const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY;
  dragStartX.value = clientX - handleLeft.value;

  // Track start
  startTrack(clientX, clientY, 'down');

  window.addEventListener('mousemove', onDragMove);
  window.addEventListener('mouseup', onDragEnd);
  window.addEventListener('touchmove', onDragMove, { passive: false });
  window.addEventListener('touchend', onDragEnd);
};

const onDragMove = (e: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return;
  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
  const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY;

  let x = clientX - dragStartX.value;
  if (x < 0) x = 0;
  if (x > maxHandleDrag.value) x = maxHandleDrag.value;

  handleLeft.value = x;
  addMoveTrack(clientX, clientY);
};

const onDragEnd = async (e: MouseEvent | TouchEvent) => {
  if (!isDragging.value) return;
  isDragging.value = false;

  window.removeEventListener('mousemove', onDragMove);
  window.removeEventListener('mouseup', onDragEnd);
  window.removeEventListener('touchmove', onDragMove);
  window.removeEventListener('touchend', onDragEnd);

  const clientX = 'changedTouches' in e ? e.changedTouches[0].clientX : (e as MouseEvent).clientX;
  const clientY = 'changedTouches' in e ? e.changedTouches[0].clientY : (e as MouseEvent).clientY;

  if (!captchaData.value) return;

  // 轨迹缩放与对齐：
  const bgW = captchaData.value.backgroundImageWidth;
  const bgH = captchaData.value.backgroundImageHeight;
  const maxDrag = maxHandleDrag.value;
  const first = trackList.value[0] || { x: clientX, y: clientY };

  // 计算拖动比例 (0 到 1)
  const dragRatio = maxDrag > 0 ? handleLeft.value / maxDrag : 0;
  // 映射到背景图宽度的终点 X 偏移量
  const targetDeltaX = dragRatio * bgW;

  const rawLastDelta = (trackList.value[trackList.value.length - 1]?.x || clientX) - first.x;
  const trackScale = rawLastDelta !== 0 ? targetDeltaX / rawLastDelta : 1;

  // 格式化轨迹点，首点相对原点为 0
  const formattedTrackList: CaptchaTrackPoint[] = trackList.value.map(pt => ({
    x: Math.round((pt.x - first.x) * trackScale),
    y: Math.round((pt.y - first.y) * 0.5), // 保留 Y 轴扰动，防止触发 check4SameY 机器人判定
    type: pt.type,
    t: pt.t
  }));

  const stopTime = Date.now();
  formattedTrackList.push({
    x: Math.round(targetDeltaX),
    y: Math.round((clientY - first.y) * 0.5),
    type: 'up',
    t: stopTime - startTime.value
  });

  const trackData = {
    bgImageWidth: bgW,
    bgImageHeight: bgH,
    startTime: startTime.value,
    stopTime,
    trackList: formattedTrackList
  };

  await verify(trackData);
};

// DragDrop Piece Event Handlers
const onPieceDragStart = (e: MouseEvent | TouchEvent) => {
  if (loading.value || !captchaData.value || captchaData.value.type !== 'DRAG_DROP') return;
  const boardRect = boardRef.value?.getBoundingClientRect();
  if (!boardRect) return;

  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
  const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY;

  isDraggingPiece.value = true;
  const currentX = dragPiecePos.x;
  const currentY = dragPiecePos.y;

  // 记录点击点距离拼图块自身左上角的绝对偏移，避免任何位置跳动
  dragPieceOffset.x = clientX - (boardRect.left + currentX);
  dragPieceOffset.y = clientY - (boardRect.top + currentY);

  startTrack(clientX, clientY, 'down');

  window.addEventListener('mousemove', onPieceDragMove);
  window.addEventListener('mouseup', onPieceDragEnd);
  window.addEventListener('touchmove', onPieceDragMove, { passive: false });
  window.addEventListener('touchend', onPieceDragEnd);
};

const onPieceDragMove = (e: MouseEvent | TouchEvent) => {
  if (!isDraggingPiece.value || !boardRef.value) return;
  const boardRect = boardRef.value.getBoundingClientRect();
  const clientX = 'touches' in e ? e.touches[0].clientX : e.clientX;
  const clientY = 'touches' in e ? e.touches[0].clientY : e.clientY;

  let nx = clientX - boardRect.left - dragPieceOffset.x;
  let ny = clientY - boardRect.top - dragPieceOffset.y;

  // 拖动范围允许稍微超出 5px 边缘缓冲，确保边界贴边缺口也能 100% 完整重合覆盖
  const maxW = Math.round(props.width - dragDropSize.value.w + 6);
  const maxH = Math.round(boardHeight.value - dragDropSize.value.h + 6);

  if (nx < -6) nx = -6;
  if (nx > maxW) nx = maxW;
  if (ny < -6) ny = -6;
  if (ny > maxH) ny = maxH;

  dragPiecePos.x = nx;
  dragPiecePos.y = ny;

  addMoveTrack(clientX, clientY);
};

const onPieceDragEnd = async (e: MouseEvent | TouchEvent) => {
  if (!isDraggingPiece.value || !captchaData.value) return;
  isDraggingPiece.value = false;

  window.removeEventListener('mousemove', onPieceDragMove);
  window.removeEventListener('mouseup', onPieceDragEnd);
  window.removeEventListener('touchmove', onPieceDragMove);
  window.removeEventListener('touchend', onPieceDragEnd);

  const scale = scaleRatio.value;
  const bgW = captchaData.value.backgroundImageWidth;
  const bgH = captchaData.value.backgroundImageHeight;

  // 释放时的实际坐标还原到背景图原尺寸
  const finalX = Math.round(dragPiecePos.x / scale);
  const finalY = Math.round(dragPiecePos.y / scale);

  const clientX = 'changedTouches' in e ? e.changedTouches[0].clientX : (e as MouseEvent).clientX;
  const clientY = 'changedTouches' in e ? e.changedTouches[0].clientY : (e as MouseEvent).clientY;
  const first = trackList.value[0] || { x: clientX, y: clientY };

  const formattedTrackList: CaptchaTrackPoint[] = trackList.value.map(pt => ({
    x: Math.round((pt.x - first.x) / scale),
    y: Math.round((pt.y - first.y) / scale),
    type: pt.type,
    t: pt.t
  }));

  const stopTime = Date.now();
  formattedTrackList.push({
    x: finalX,
    y: finalY,
    type: 'up',
    t: stopTime - startTime.value
  });

  const trackData = {
    bgImageWidth: bgW,
    bgImageHeight: bgH,
    startTime: startTime.value,
    stopTime,
    trackList: formattedTrackList
  };

  await verify(trackData);
};

// Word / Shape Click Handler
const handleBoardClick = (e: MouseEvent) => {
  if (
    (captchaData.value?.type !== 'WORD_IMAGE_CLICK' && captchaData.value?.type !== 'SHAPE_CLICK') ||
    loading.value
  )
    return;
  const rect = boardRef.value?.getBoundingClientRect();
  if (!rect) return;

  const clickX = e.clientX - rect.left;
  const clickY = e.clientY - rect.top;

  clickPoints.value.push({ x: clickX, y: clickY });
  addClickTrack(clickX, clickY);
};

const submitClickPoints = async () => {
  if (!captchaData.value || clickPoints.value.length === 0) return;
  const scale = scaleRatio.value;
  const bgW = captchaData.value.backgroundImageWidth;
  const bgH = captchaData.value.backgroundImageHeight;

  const pointsTrack: CaptchaTrackPoint[] = clickPoints.value.map((pt, i) => ({
    x: Math.round(pt.x / scale),
    y: Math.round(pt.y / scale),
    type: 'click',
    t: i * 200 + 100
  }));

  const stopTime = Date.now();
  const trackData = {
    bgImageWidth: bgW,
    bgImageHeight: bgH,
    startTime: startTime.value || Date.now() - 1000,
    stopTime,
    trackList: pointsTrack
  };

  await verify(trackData);
};

// Submit verification to backend
const verify = async (trackData: any) => {
  if (!captchaData.value) return;
  try {
    let result: any;
    if (props.validCaptchaData) {
      result = await props.validCaptchaData(captchaData.value.id, trackData);
    } else {
      // 收集浏览器反爬环境特征
      const env = collectClientEnv();
      const timestamp = captchaData.value.timestamp || Date.now();

      // 使用动态密钥加密轨迹与指纹
      const payload = await encryptPayload(captchaData.value.id, timestamp, {
        track: trackData,
        env
      });

      const bodyData: any = {
        id: captchaData.value.id,
        track: trackData,
        timestamp,
        env
      };
      if (payload) {
        bodyData.payload = payload;
      }

      const resp = await fetch('/captcha/check', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(bodyData)
      });
      result = await resp.json();
    }

    const isSuccess = typeof result === 'boolean' ? result : !!(result?.success || result?.code === 200);

    if (isSuccess) {
      showTip('success', '验证通过！');
      emit('success', result);
    } else {
      const msg = result?.msg || '验证失败，请重试';
      showTip('error', msg);
      emit('fail', msg);
      setTimeout(() => {
        refresh();
      }, 800);
    }
  } catch (err: any) {
    showTip('error', err.message || '验证请求异常');
    emit('fail', err.message);
    setTimeout(() => {
      refresh();
    }, 800);
  }
};

onMounted(() => {
  fetchCaptcha();
});

onUnmounted(() => {
  resetAll();
});

// Expose public API
defineExpose({
  refresh,
  reset: resetAll
});
</script>

<style scoped>
.tac-captcha {
  position: relative;
  box-sizing: border-box;
  background: #ffffff;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
  padding: 12px;
  user-select: none;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Helvetica, Arial, sans-serif;
}

.tac-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
  min-height: 24px;
}

.tac-title {
  font-size: 14px;
  color: #333333;
  font-weight: 500;
}

.tac-word-tip {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: #333;
}

.tac-tip-banner {
  height: 28px;
  border-radius: 4px;
  border: 1px solid #e0e0e0;
}

.tac-actions {
  display: flex;
  gap: 8px;
}

.tac-icon-btn {
  cursor: pointer;
  color: #8c8c8c;
  font-size: 16px;
  line-height: 1;
  transition: color 0.2s;
}

.tac-icon-btn:hover {
  color: #1890ff;
}

/* Board */
.tac-board {
  position: relative;
  overflow: hidden;
  border-radius: 6px;
  background: #f5f5f5;
  cursor: pointer;
}

.tac-bg-img {
  width: 100%;
  height: 100%;
  display: block;
  object-fit: cover;
}

/* ROTATE_DEGREE: 单独圆形图片居中展示与旋转 */
.tac-rotate-degree-wrap {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f8fafc;
}

.tac-rotate-circle-container {
  position: relative;
  width: 180px;
  height: 180px;
  border-radius: 50%;
  overflow: hidden;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  background: #fff;
  display: flex;
  align-items: center;
  justify-content: center;
}

/* 十字虚线辅助对齐线 */
.tac-crosshair-line {
  position: absolute;
  pointer-events: none;
  z-index: 10;
}

.tac-crosshair-h {
  top: 50%;
  left: 0;
  width: 100%;
  height: 0;
  border-top: 1.5px dashed rgba(255, 255, 255, 0.9);
  filter: drop-shadow(0 0 1px rgba(0, 0, 0, 0.7));
  transform: translateY(-50%);
}

.tac-crosshair-v {
  top: 0;
  left: 50%;
  width: 0;
  height: 100%;
  border-left: 1.5px dashed rgba(255, 255, 255, 0.9);
  filter: drop-shadow(0 0 1px rgba(0, 0, 0, 0.7));
  transform: translateX(-50%);
}

.tac-rotate-degree-img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  display: block;
  object-fit: cover;
  transition: transform 0.05s linear;
  user-select: none;
}

.tac-slider-template {
  position: absolute;
  top: 0;
  pointer-events: none;
  z-index: 2;
}

/* SliderV2 曲线轨迹浮动滑块样式 */
.tac-slider-v2-template {
  position: absolute;
  pointer-events: none;
  z-index: 3;
}

.tac-rotate-template-wrap {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  pointer-events: none;
  z-index: 2;
}

.tac-rotate-template {
  height: 100%;
  width: auto;
  display: block;
  transition: transform 0.05s linear;
}

.tac-concat-mask {
  position: absolute;
  top: 0;
  left: 0;
  width: 100%;
  overflow: hidden;
  pointer-events: none;
  z-index: 2;
  background-repeat: repeat-x;
}

.tac-click-layer {
  position: absolute;
  inset: 0;
  z-index: 5;
}

/* 自由拖拽拼图块样式 */
.tac-drag-piece {
  position: absolute;
  z-index: 6;
  user-select: none;
  filter: drop-shadow(0 4px 8px rgba(0, 0, 0, 0.35));
}

.tac-drag-piece-img {
  width: 100%;
  height: 100%;
  display: block;
  pointer-events: none;
}

/* 拖拽底部提示栏 */
.tac-dragdrop-tip-bar {
  height: 40px;
  background: #f7f9fa;
  border: 1px solid #e1e4e8;
  border-radius: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #666;
}

.tac-click-dot {
  position: absolute;
  width: 22px;
  height: 22px;
  line-height: 22px;
  text-align: center;
  background: #1890ff;
  color: #fff;
  font-size: 12px;
  font-weight: bold;
  border-radius: 50%;
  border: 2px solid #ffffff;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.3);
  animation: tac-pop 0.2s ease-out;
}

@keyframes tac-pop {
  0% { transform: scale(0.5); opacity: 0; }
  100% { transform: scale(1); opacity: 1; }
}

.tac-tip {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 6px 12px;
  font-size: 13px;
  color: #fff;
  z-index: 10;
  text-align: center;
}

.tac-tip-success {
  background: rgba(82, 196, 26, 0.9);
}

.tac-tip-error {
  background: rgba(245, 34, 45, 0.9);
}

.tac-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: #f5f5f5;
  color: #8c8c8c;
  font-size: 13px;
}

.tac-spinner {
  width: 24px;
  height: 24px;
  border: 3px solid #d9d9d9;
  border-top-color: #1890ff;
  border-radius: 50%;
  animation: tac-spin 0.8s linear infinite;
}

@keyframes tac-spin {
  to { transform: rotate(360deg); }
}

/* Footer / Track Bar */
.tac-footer {
  margin-top: 10px;
}

.tac-track-bar {
  position: relative;
  height: 40px;
  background: #f0f2f5;
  border-radius: 20px;
  border: 1px solid #e4e7ed;
  overflow: hidden;
}

.tac-track-bg {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  background: #e6f7ff;
  border: 1px solid #91d5ff;
  border-radius: 20px;
}

.tac-track-tip {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  color: #8c8c8c;
  pointer-events: none;
}

.tac-slider-handle {
  position: absolute;
  top: 0;
  left: 0;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: #ffffff;
  box-shadow: 0 2px 6px rgba(0, 0, 0, 0.15);
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: grab;
  transition: box-shadow 0.2s, background-color 0.2s;
  z-index: 3;
}

.tac-slider-handle:active {
  cursor: grabbing;
  background-color: #1890ff;
}

.tac-slider-handle:active .tac-arrow {
  color: #ffffff;
}

.tac-arrow {
  font-size: 16px;
  color: #595959;
  font-weight: bold;
}

.tac-confirm-btn {
  width: 100%;
  height: 38px;
  background: #1890ff;
  color: #fff;
  border: none;
  border-radius: 6px;
  font-size: 14px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.tac-confirm-btn:disabled {
  background: #d9d9d9;
  cursor: not-allowed;
}

.tac-confirm-btn:not(:disabled):hover {
  background: #40a9ff;
}

.tac-fade-enter-active,
.tac-fade-leave-active {
  transition: opacity 0.3s;
}

.tac-fade-enter-from,
.tac-fade-leave-to {
  opacity: 0;
}
</style>

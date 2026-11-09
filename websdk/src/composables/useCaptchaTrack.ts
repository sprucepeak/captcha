import { ref } from 'vue';
import type { CaptchaTrackPoint, CaptchaTrackData } from '../types';

export function useCaptchaTrack() {
  const startTime = ref<number>(0);
  const trackList = ref<CaptchaTrackPoint[]>([]);

  const startTrack = (x: number, y: number, type: 'down' | 'click' = 'down') => {
    startTime.value = Date.now();
    trackList.value = [
      {
        x: Math.round(x),
        y: Math.round(y),
        type,
        t: 0
      }
    ];
  };

  const addMoveTrack = (x: number, y: number) => {
    if (startTime.value === 0) return;
    trackList.value.push({
      x: Math.round(x),
      y: Math.round(y),
      type: 'move',
      t: Date.now() - startTime.value
    });
  };

  const addClickTrack = (x: number, y: number) => {
    const now = Date.now();
    if (startTime.value === 0) {
      startTime.value = now;
    }
    trackList.value.push({
      x: Math.round(x),
      y: Math.round(y),
      type: 'click',
      t: now - startTime.value
    });
  };

  const endTrack = (
    x: number,
    y: number,
    bgWidth: number,
    bgHeight: number,
    tplWidth?: number,
    tplHeight?: number
  ): CaptchaTrackData => {
    const stopTime = Date.now();
    trackList.value.push({
      x: Math.round(x),
      y: Math.round(y),
      type: 'up',
      t: stopTime - startTime.value
    });

    const data: CaptchaTrackData = {
      bgImageWidth: bgWidth,
      bgImageHeight: bgHeight,
      templateImageWidth: tplWidth,
      templateImageHeight: tplHeight,
      startTime: startTime.value,
      stopTime,
      trackList: [...trackList.value]
    };

    // Reset
    startTime.value = 0;
    trackList.value = [];

    return data;
  };

  const resetTrack = () => {
    startTime.value = 0;
    trackList.value = [];
  };

  return {
    startTime,
    trackList,
    startTrack,
    addMoveTrack,
    addClickTrack,
    endTrack,
    resetTrack
  };
}


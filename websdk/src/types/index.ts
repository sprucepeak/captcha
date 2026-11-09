export type CaptchaType =
  | 'SLIDER'
  | 'SLIDER_V2'
  | 'ROTATE'
  | 'ROTATE_DEGREE'
  | 'CONCAT'
  | 'WORD_IMAGE_CLICK'
  | 'SHAPE_CLICK'
  | 'DRAG_DROP';

export interface CaptchaTrackPoint {
  x: number;
  y: number;
  type: 'down' | 'move' | 'up' | 'click';
  t: number;
}

export interface CaptchaTrackData {
  bgImageWidth: number;
  bgImageHeight: number;
  templateImageWidth?: number;
  templateImageHeight?: number;
  startTime: number;
  stopTime: number;
  trackList: CaptchaTrackPoint[];
  data?: string;
}

export interface CaptchaVO {
  id: string;
  type: CaptchaType;
  backgroundImage: string;
  templateImage?: string;
  backgroundImageWidth: number;
  backgroundImageHeight: number;
  templateImageWidth?: number;
  templateImageHeight?: number;
  timestamp?: number;
  data?: Record<string, any>;
}

export interface ApiResponse<T = any> {
  code: number;
  msg: string;
  success: boolean;
  data?: T;
}

export interface CaptchaProps {
  type?: CaptchaType;
  requestCaptchaData?: () => Promise<ApiResponse<CaptchaVO> | CaptchaVO>;
  validCaptchaData?: (id: string, track: CaptchaTrackData) => Promise<ApiResponse<any> | boolean>;
  width?: number; // Container display width, default 310
  showRefresh?: boolean;
  showClose?: boolean;
}

export interface CaptchaEmits {
  (e: 'success', data: any): void;
  (e: 'fail', msg: string): void;
  (e: 'refresh'): void;
  (e: 'close'): void;
}


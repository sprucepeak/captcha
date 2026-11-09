package model

import (
	"strings"
	"time"
)

// Captcha types
const (
	TypeSlider          = "SLIDER"
	TypeSliderV2        = "SLIDER_V2"        // 滑块验证V2 (非线性/曲线轨迹滑块)
	TypeRotate          = "ROTATE"           // 旋转图片 (内圈圆旋转拼合周围背景)
	TypeRotateDegree    = "ROTATE_DEGREE"    // 角度验证 (整图/主体旋转摆正)
	TypeConcat          = "CONCAT"
	TypeWordImageClick  = "WORD_IMAGE_CLICK" // 文字点选
	TypeShapeClick      = "SHAPE_CLICK"      // 图形/形状点选 (矢量几何/图标点选)
	TypeDragDrop        = "DRAG_DROP"        // 自由拖拽验证码 (全图自由放置拼图)
	TypePow             = "POW"

	DefaultTag = "default"

	TemplateActiveImageName = "active.png"
	TemplateFixedImageName  = "fixed.png"
	TemplateMaskImageName   = "mask.png"

	TrackTypeClick = "click"
	TrackTypeMove  = "move"
	TrackTypeDown  = "down"
	TrackTypeUp    = "up"
)

// NormalizeType normalizes captcha type string (case-insensitive & aliases)
func NormalizeType(t string) string {
	switch strings.ToUpper(strings.TrimSpace(t)) {
	case "SLIDER", "SLIDE":
		return TypeSlider
	case "SLIDER_V2", "SLIDE_V2", "V2", "SLIDER_CURVE", "CURVE":
		return TypeSliderV2
	case "ROTATE":
		return TypeRotate
	case "ROTATE_DEGREE", "DEGREE", "ANGLE":
		return TypeRotateDegree
	case "CONCAT":
		return TypeConcat
	case "WORD_IMAGE_CLICK", "WORD_CLICK", "CLICK":
		return TypeWordImageClick
	case "SHAPE_CLICK", "SHAPE", "CLICK_SHAPE":
		return TypeShapeClick
	case "DRAG_DROP", "DRAG", "DROP":
		return TypeDragDrop
	case "POW":
		return TypePow
	default:
		return strings.ToUpper(strings.TrimSpace(t))
	}
}

// Resource represents an image or font resource
type Resource struct {
	Type  string `json:"type"`  // "classpath", "file", "bytes", "url", etc.
	Data  string `json:"data"`  // path, text, etc.
	Bytes []byte `json:"-"`     // optional in-memory or embedded bytes
	Tag   string `json:"tag"`
}

// GenerateParam represents parameters for captcha generation
type GenerateParam struct {
	Type                string                 `json:"type"`
	BackgroundImageTag  string                 `json:"backgroundImageTag"`
	TemplateImageTag    string                 `json:"templateImageTag"`
	FontTag             string                 `json:"fontTag"`
	Obfuscate           bool                   `json:"obfuscate"`
	CheckClickCount     int                    `json:"checkClickCount"`
	InterferenceCount   int                    `json:"interferenceCount"`
	Extra               map[string]interface{} `json:"extra"`
}

// TolerantBox 定义上下左右四向容差
// 支持两种模式：
// 1. 绝对像素模式：数值 >= 1.0 时作为物理像素（如 Left: 8.0 表示允许向左偏离 8px）
// 2. 相对百分比模式：数值 < 1.0 时作为百分比（如 Left: 0.02 表示允许向左偏离宽度的 2%）
type TolerantBox struct {
	Top    float64 `json:"top"`    // 向上允许偏差
	Bottom float64 `json:"bottom"` // 向下允许偏差
	Left   float64 `json:"left"`   // 向左允许偏差
	Right  float64 `json:"right"`  // 向右允许偏差
}

// NewSymmetricTolerant 创建对称四向容差（上下左右使用相同数值，可为像素如 8.0，亦可为百分比如 0.02）
func NewSymmetricTolerant(v float64) TolerantBox {
	return TolerantBox{Top: v, Bottom: v, Left: v, Right: v}
}

// NewBoxTolerant 创建独立四向容差
func NewBoxTolerant(top, bottom, left, right float64) TolerantBox {
	return TolerantBox{Top: top, Bottom: bottom, Left: left, Right: right}
}

// ImageCaptchaInfo holds intermediate generated captcha data
type ImageCaptchaInfo struct {
	Type                  string                 `json:"type"`
	BackgroundImage       string                 `json:"backgroundImage"` // Base64
	TemplateImage         string                 `json:"templateImage"`   // Base64
	BackgroundImageTag    string                 `json:"backgroundImageTag"`
	TemplateImageTag      string                 `json:"templateImageTag"`
	BackgroundImageWidth  int                    `json:"backgroundImageWidth"`
	BackgroundImageHeight int                    `json:"backgroundImageHeight"`
	TemplateImageWidth    int                    `json:"templateImageWidth"`
	TemplateImageHeight   int                    `json:"templateImageHeight"`
	RandomX               int                    `json:"randomX"`
	RandomY               int                    `json:"randomY"`
	Tolerant              float64                `json:"tolerant"`
	TolerantBox           *TolerantBox           `json:"tolerantBox,omitempty"` // 上下左右四向容差（优先使用）
	ViewData              map[string]interface{} `json:"viewData"`
	Expand                interface{}            `json:"expand"`
}

// ImageCaptchaVO is returned to the client
type ImageCaptchaVO struct {
	ID                    string                 `json:"id"`
	Type                  string                 `json:"type"`
	BackgroundImage       string                 `json:"backgroundImage"`
	TemplateImage         string                 `json:"templateImage,omitempty"`
	BackgroundImageTag    string                 `json:"backgroundImageTag,omitempty"`
	TemplateImageTag      string                 `json:"templateImageTag,omitempty"`
	BackgroundImageWidth  int                    `json:"backgroundImageWidth"`
	BackgroundImageHeight int                    `json:"backgroundImageHeight"`
	TemplateImageWidth    int                    `json:"templateImageWidth,omitempty"`
	TemplateImageHeight   int                    `json:"templateImageHeight,omitempty"`
	Timestamp             int64                  `json:"timestamp"`
	Data                  map[string]interface{} `json:"data,omitempty"`
}

// Track represents a single point in user track
type Track struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	T    float64 `json:"t"`
	Type string  `json:"type"`
}

// ImageCaptchaTrack represents full track submitted by user
type ImageCaptchaTrack struct {
	BgImageWidth  int     `json:"bgImageWidth"`
	BgImageHeight int     `json:"bgImageHeight"`
	StartTime     int64   `json:"startTime"`
	StopTime      int64   `json:"stopTime"`
	TrackList     []Track `json:"trackList"`
	Data          string  `json:"data,omitempty"`
}

// MatchParam represents parameters passed when verifying
type MatchParam struct {
	Track ImageCaptchaTrack `json:"track"`
}

// ClickPoint represents a click check point
type ClickPoint struct {
	X      int    `json:"x"`
	Y      int    `json:"y"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Tip    string `json:"tip"`
}

// ApiResponse represents unified response
type ApiResponse[T any] struct {
	Code    int    `json:"code"`
	Msg     string `json:"msg"`
	Success bool   `json:"success"`
	Data    T      `json:"data"`
}

func Success[T any](data T) ApiResponse[T] {
	return ApiResponse[T]{
		Code:    200,
		Msg:     "success",
		Success: true,
		Data:    data,
	}
}

func Error[T any](code int, msg string) ApiResponse[T] {
	return ApiResponse[T]{
		Code:    code,
		Msg:     msg,
		Success: false,
	}
}

func CheckFail[T any](msg string) ApiResponse[T] {
	return ApiResponse[T]{
		Code:    50001,
		Msg:     msg,
		Success: false,
	}
}

// NowMillis returns current Unix millis
func NowMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}


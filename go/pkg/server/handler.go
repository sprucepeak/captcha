package server

import (
	"encoding/json"
	"net/http"

	"captcha/pkg/application"
	"captcha/pkg/model"
	"captcha/pkg/security"
)

// CaptchaHttpHandler provides standard HTTP endpoints for captcha
type CaptchaHttpHandler struct {
	app application.CaptchaApplication
}

func NewCaptchaHttpHandler(app application.CaptchaApplication) *CaptchaHttpHandler {
	return &CaptchaHttpHandler{app: app}
}

func (h *CaptchaHttpHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/captcha/gen", h.handleGenerate)
	mux.HandleFunc("/captcha/check", h.handleMatching)
}

func (h *CaptchaHttpHandler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if r.Method == http.MethodOptions {
		return
	}

	cType := r.URL.Query().Get("type")
	if cType == "" {
		cType = model.TypeSlider
	}
	cType = model.NormalizeType(cType)

	res := h.app.GenerateCaptcha(&model.GenerateParam{
		Type: cType,
	})

	json.NewEncoder(w).Encode(res)
}

type CheckRequest struct {
	ID        string                  `json:"id"`
	Track     model.ImageCaptchaTrack `json:"track"`
	Payload   string                  `json:"payload,omitempty"`   // AES 加密密文
	Timestamp int64                   `json:"timestamp,omitempty"` // 时间戳
	Env       *security.ClientEnvInfo `json:"env,omitempty"`       // 浏览器反爬环境特征
}

func (h *CaptchaHttpHandler) handleMatching(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")

	if r.Method == http.MethodOptions {
		return
	}

	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		json.NewEncoder(w).Encode(model.CheckFail[interface{}]("invalid request format"))
		return
	}

	// 1. 如果提交了加密 Payload，使用动态密钥解密还原轨迹与环境指纹
	trackToMatch := req.Track
	if req.Payload != "" && req.Timestamp > 0 {
		type EncryptedPacket struct {
			Track model.ImageCaptchaTrack `json:"track"`
			Env   *security.ClientEnvInfo `json:"env"`
		}
		packet, err := security.DecryptTrackPayload[EncryptedPacket](req.Payload, req.ID, req.Timestamp)
		if err != nil {
			json.NewEncoder(w).Encode(model.CheckFail[interface{}]("安全报文校验失败"))
			return
		}
		trackToMatch = packet.Track
		if packet.Env != nil {
			req.Env = packet.Env
		}
	}

	// 2. 检查浏览器反爬环境指纹（如检测自动化驱动 Webdriver）
	if req.Env != nil {
		if ok, reason := security.VerifyClientEnv(req.Env); !ok {
			json.NewEncoder(w).Encode(model.CheckFail[interface{}](reason))
			return
		}
	}

	res := h.app.Matching(req.ID, &model.MatchParam{
		Track: trackToMatch,
	})

	json.NewEncoder(w).Encode(res)
}

// GinHandlerFunc returns a standard Gin handler for quick integration:
//
//	r.GET("/captcha/gen", captcha.GinGenHandler(app))
//	r.POST("/captcha/check", captcha.GinCheckHandler(app))
func (h *CaptchaHttpHandler) GinGenHandler() http.HandlerFunc {
	return h.handleGenerate
}

func (h *CaptchaHttpHandler) GinCheckHandler() http.HandlerFunc {
	return h.handleMatching
}


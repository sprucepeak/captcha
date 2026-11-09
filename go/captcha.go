// Package captcha provides an all-in-one CAPTCHA SDK for Go.
// Supporting SLIDER, SLIDER_V2, ROTATE, ROTATE_DEGREE, CONCAT, and WORD_IMAGE_CLICK captcha types.
package captcha

import (
	"captcha/pkg/application"
	"captcha/pkg/generator"
	"captcha/pkg/model"
	"captcha/pkg/server"
	"captcha/pkg/store"
	"captcha/pkg/validator"
	"net/http"
	"time"
)

// Captcha types
const (
	TypeSlider         = model.TypeSlider
	TypeSliderV2       = model.TypeSliderV2
	TypeRotate         = model.TypeRotate
	TypeRotateDegree   = model.TypeRotateDegree
	TypeConcat         = model.TypeConcat
	TypeWordImageClick = model.TypeWordImageClick
	TypeShapeClick     = model.TypeShapeClick
	TypeDragDrop       = model.TypeDragDrop

	DefaultTag = model.DefaultTag
)

// Re-export core types
type (
	// CaptchaApplication is the main entrypoint
	CaptchaApplication = application.CaptchaApplication

	// TACBuilder builds CaptchaApplication fluently
	TACBuilder = application.TACBuilder

	// Model definitions
	GenerateParam      = model.GenerateParam
	ImageCaptchaVO     = model.ImageCaptchaVO
	ImageCaptchaInfo   = model.ImageCaptchaInfo
	ImageCaptchaTrack  = model.ImageCaptchaTrack
	MatchParam         = model.MatchParam
	Track              = model.Track
	ClickPoint         = model.ClickPoint
	ApiResponse[T any] = model.ApiResponse[T]

	// Stores & Interfaces
	CacheStore    = store.CacheStore
	ResourceStore = store.ResourceStore
	Resource      = model.Resource
	ResourceMap   = store.ResourceMap

	// Generator & Validator
	CaptchaGenerator = generator.CaptchaGenerator
	Validator        = validator.Validator

	// HTTP Handler
	CaptchaHttpHandler = server.CaptchaHttpHandler
)

// Builder returns a new fluent builder for CaptchaApplication (supports functional options)
func Builder(opts ...application.Option) *TACBuilder {
	return application.Builder(opts...)
}

// Re-export functional options
var (
	WithResourceStore    = application.WithResourceStore
	WithCacheStore       = application.WithCacheStore
	WithValidator        = application.WithValidator
	WithExpire           = application.WithExpire
	WithDefaultTolerant  = application.WithDefaultTolerant
	WithBuiltinResources = application.WithBuiltinResources
)

// NewDefault creates a ready-to-use CaptchaApplication with embedded builtin resources
// Zero external files required!
func NewDefault() CaptchaApplication {
	return application.Builder(
		application.WithBuiltinResources(),
		application.WithDefaultTolerant(0.04),
		application.WithExpire(2*time.Minute),
	).Build()
}

// NewHttpHandler returns standard net/http handler for captcha endpoints
func NewHttpHandler(app CaptchaApplication) *server.CaptchaHttpHandler {
	return server.NewCaptchaHttpHandler(app)
}

// RegisterRoutes registers /captcha/gen and /captcha/check onto http.ServeMux
func RegisterRoutes(mux *http.ServeMux, app CaptchaApplication) {
	handler := server.NewCaptchaHttpHandler(app)
	handler.RegisterRoutes(mux)
}

// GinGenHandler returns a http.HandlerFunc compatible with Gin r.GET("/captcha/gen", gin.WrapF(captcha.GinGenHandler(app)))
func GinGenHandler(app CaptchaApplication) http.HandlerFunc {
	return server.NewCaptchaHttpHandler(app).GinGenHandler()
}

// GinCheckHandler returns a http.HandlerFunc compatible with Gin r.POST("/captcha/check", gin.WrapF(captcha.GinCheckHandler(app)))
func GinCheckHandler(app CaptchaApplication) http.HandlerFunc {
	return server.NewCaptchaHttpHandler(app).GinCheckHandler()
}

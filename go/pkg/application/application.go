package application

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"captcha/pkg/generator"
	"captcha/pkg/model"
	"captcha/pkg/store"
	"captcha/pkg/validator"
)

// CaptchaApplication defines main captcha API
type CaptchaApplication interface {
	GenerateCaptcha(param *model.GenerateParam) model.ApiResponse[*model.ImageCaptchaVO]
	Matching(id string, matchParam *model.MatchParam) model.ApiResponse[interface{}]
	MatchingPercentage(id string, percentage float64) bool
}

type DefaultCaptchaApplication struct {
	generator     generator.CaptchaGenerator
	validator     validator.Validator
	cacheStore    store.CacheStore
	defaultExpire time.Duration
}

func NewDefaultCaptchaApplication(
	gen generator.CaptchaGenerator,
	val validator.Validator,
	cache store.CacheStore,
	expire time.Duration,
) *DefaultCaptchaApplication {
	if expire <= 0 {
		expire = 2 * time.Minute
	}
	return &DefaultCaptchaApplication{
		generator:     gen,
		validator:     val,
		cacheStore:    cache,
		defaultExpire: expire,
	}
}

func (app *DefaultCaptchaApplication) GenerateCaptcha(param *model.GenerateParam) model.ApiResponse[*model.ImageCaptchaVO] {
	if param == nil {
		param = &model.GenerateParam{Type: model.TypeSlider}
	}
	// 容错处理：统一转为大写且去除多余空格
	param.Type = model.NormalizeType(param.Type)
	if param.Type == "" {
		param.Type = model.TypeSlider
	}

	info, err := app.generator.Generate(param)
	if err != nil {
		return model.Error[*model.ImageCaptchaVO](500, err.Error())
	}

	// Generate unique captcha ID: type_uuid
	b := make([]byte, 16)
	rand.Read(b)
	id := fmt.Sprintf("%s_%s", param.Type, hex.EncodeToString(b))

	// Generate verification data
	validData, err := app.validator.GenerateValidData(info)
	if err != nil {
		return model.Error[*model.ImageCaptchaVO](500, err.Error())
	}

	now := time.Now().UnixMilli()
	validData["timestamp"] = now

	// Cache verification data
	app.cacheStore.Set(id, validData, app.defaultExpire)

	vo := &model.ImageCaptchaVO{
		ID:                    id,
		Type:                  info.Type,
		BackgroundImage:       info.BackgroundImage,
		TemplateImage:         info.TemplateImage,
		BackgroundImageTag:    info.BackgroundImageTag,
		TemplateImageTag:      info.TemplateImageTag,
		BackgroundImageWidth:  info.BackgroundImageWidth,
		BackgroundImageHeight: info.BackgroundImageHeight,
		TemplateImageWidth:    info.TemplateImageWidth,
		TemplateImageHeight:   info.TemplateImageHeight,
		Timestamp:             now,
		Data:                  info.ViewData,
	}

	return model.Success(vo)
}

func (app *DefaultCaptchaApplication) Matching(id string, matchParam *model.MatchParam) model.ApiResponse[interface{}] {
	if id == "" || matchParam == nil {
		return model.CheckFail[interface{}]("参数错误")
	}

	// One-time verification: get and remove from cache
	val, ok := app.cacheStore.GetAndRemove(id)
	if !ok || val == nil {
		return model.Error[interface{}](ApiResponseExpired, "验证码已失效或不存在")
	}

	validData, ok := val.(map[string]interface{})
	if !ok {
		return model.Error[interface{}](500, "验证数据格式错误")
	}

	return app.validator.Validate(&matchParam.Track, validData)
}

func (app *DefaultCaptchaApplication) MatchingPercentage(id string, percentage float64) bool {
	val, ok := app.cacheStore.GetAndRemove(id)
	if !ok || val == nil {
		return false
	}
	validData, ok := val.(map[string]interface{})
	if !ok {
		return false
	}
	oriPercentage, ok := validData["percentage"].(float64)
	if !ok {
		return false
	}
	tolerant, ok := validData["tolerant"].(float64)
	if !ok || tolerant <= 0 {
		tolerant = 0.04
	}
	diff := percentage - oriPercentage
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerant
}

const ApiResponseExpired = 10002


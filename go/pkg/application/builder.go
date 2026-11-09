package application

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"captcha/pkg/generator"
	"captcha/pkg/imageutil"
	"captcha/pkg/model"
	"captcha/pkg/store"
	"captcha/pkg/validator"
	"captcha/resources"
)

// Option defines functional configuration for TACBuilder
type Option func(*TACBuilder)

// WithResourceStore sets a custom resource store
func WithResourceStore(rs store.ResourceStore) Option {
	return func(b *TACBuilder) {
		b.resourceStore = rs
	}
}

// WithCacheStore sets a custom cache store (e.g. Memory, Redis)
func WithCacheStore(cs store.CacheStore) Option {
	return func(b *TACBuilder) {
		b.cacheStore = cs
	}
}

// WithValidator sets a custom validator
func WithValidator(v validator.Validator) Option {
	return func(b *TACBuilder) {
		b.validator = v
	}
}

// WithExpire sets captcha session expiration time
func WithExpire(expire time.Duration) Option {
	return func(b *TACBuilder) {
		b.expire = expire
	}
}

// WithDefaultTolerant sets global default tolerance percentage
func WithDefaultTolerant(tol float64) Option {
	return func(b *TACBuilder) {
		b.defaultTolerant = tol
	}
}

// WithBuiltinResources embeds all default background images, templates, and fonts
func WithBuiltinResources() Option {
	return func(b *TACBuilder) {
		b.AddBuiltinResources()
	}
}

// TACBuilder builds CaptchaApplication fluently like Java TACBuilder or functional options
type TACBuilder struct {
	resourceStore   store.ResourceStore
	cacheStore      store.CacheStore
	validator       validator.Validator
	expire          time.Duration
	defaultTolerant float64
	fontPath        string
}

func Builder(opts ...Option) *TACBuilder {
	b := &TACBuilder{
		resourceStore:   store.NewMemoryResourceStore(),
		cacheStore:      store.NewMemoryCacheStore(),
		expire:          2 * time.Minute,
		defaultTolerant: 0.04,
	}
	for _, opt := range opts {
		opt(b)
	}
	return b
}

func (b *TACBuilder) SetCacheStore(cs store.CacheStore) *TACBuilder {
	b.cacheStore = cs
	return b
}

func (b *TACBuilder) SetResourceStore(rs store.ResourceStore) *TACBuilder {
	b.resourceStore = rs
	return b
}

func (b *TACBuilder) SetValidator(v validator.Validator) *TACBuilder {
	b.validator = v
	return b
}

func (b *TACBuilder) SetExpire(exp time.Duration) *TACBuilder {
	b.expire = exp
	return b
}

func (b *TACBuilder) SetDefaultTolerant(tol float64) *TACBuilder {
	b.defaultTolerant = tol
	return b
}

func (b *TACBuilder) AddResource(captchaType string, res *model.Resource) *TACBuilder {
	b.resourceStore.AddResource(captchaType, res)
	return b
}

func (b *TACBuilder) AddTemplate(captchaType, tag string, tpl store.ResourceMap) *TACBuilder {
	b.resourceStore.AddTemplate(captchaType, tag, tpl)
	return b
}

// AddResourceDir scans a directory and adds images as backgrounds
func (b *TACBuilder) AddResourceDir(dirPath, tag string, captchaTypes ...string) *TACBuilder {
	b.resourceStore.AddResourceDir(dirPath, tag, captchaTypes...)
	return b
}

// AddBuiltinResources loads all default resources embedded inside the Go package (zero external file dependency)
func (b *TACBuilder) AddBuiltinResources() *TACBuilder {
	// 1. All background images from resource/ directory (1.jpg .. 19.jpg)
	if entries, err := resources.BuiltinFS.ReadDir("resource"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			data, err := resources.BuiltinFS.ReadFile("resource/" + entry.Name())
			if err != nil {
				continue
			}
			for _, t := range []string{model.TypeSlider, model.TypeSliderV2, model.TypeRotate, model.TypeRotateDegree, model.TypeConcat, model.TypeWordImageClick} {
				b.resourceStore.AddResource(t, &model.Resource{
					Type:  "builtin",
					Bytes: data,
					Tag:   model.DefaultTag,
				})
			}
		}
	}

	// 2. Slider templates (slider_1 .. slider_5)
	for i := 1; i <= 5; i++ {
		dirName := fmt.Sprintf("slider_%d", i)
		tag := model.DefaultTag
		if i > 1 {
			tag = dirName
		}
		activePath := fmt.Sprintf("template/%s/active.png", dirName)
		fixedPath := fmt.Sprintf("template/%s/fixed.png", dirName)
		actData, errAct := resources.BuiltinFS.ReadFile(activePath)
		fixData, errFix := resources.BuiltinFS.ReadFile(fixedPath)
		if errAct == nil && errFix == nil {
			b.resourceStore.AddTemplate(model.TypeSlider, tag, store.ResourceMap{
				model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: actData, Tag: tag},
				model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: fixData, Tag: tag},
			})
			b.resourceStore.AddTemplate(model.TypeSliderV2, tag, store.ResourceMap{
				model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: actData, Tag: tag},
				model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: fixData, Tag: tag},
			})
			if tag != model.DefaultTag {
				// Also allow picking from default tag pool randomly
				b.resourceStore.AddTemplate(model.TypeSlider, model.DefaultTag, store.ResourceMap{
					model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: actData, Tag: tag},
					model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: fixData, Tag: tag},
				})
				b.resourceStore.AddTemplate(model.TypeSliderV2, model.DefaultTag, store.ResourceMap{
					model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: actData, Tag: tag},
					model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: fixData, Tag: tag},
				})
			}
		}
	}

	// 2.1 Star template for SliderV2 (star_1)
	starAct, errStarAct := resources.BuiltinFS.ReadFile("template/star_1/active.png")
	starFix, errStarFix := resources.BuiltinFS.ReadFile("template/star_1/fixed.png")
	if errStarAct == nil && errStarFix == nil {
		b.resourceStore.AddTemplate(model.TypeSliderV2, model.DefaultTag, store.ResourceMap{
			model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: starAct, Tag: model.DefaultTag},
			model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: starFix, Tag: model.DefaultTag},
		})
		b.resourceStore.AddTemplate(model.TypeSliderV2, "star_1", store.ResourceMap{
			model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: starAct, Tag: "star_1"},
			model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: starFix, Tag: "star_1"},
		})
	}

	// 3. Rotate templates (rotate_1 .. rotate_5)
	for i := 1; i <= 5; i++ {
		dirName := fmt.Sprintf("rotate_%d", i)
		tag := model.DefaultTag
		if i > 1 {
			tag = dirName
		}
		activePath := fmt.Sprintf("template/%s/active.png", dirName)
		fixedPath := fmt.Sprintf("template/%s/fixed.png", dirName)
		actData, errAct := resources.BuiltinFS.ReadFile(activePath)
		fixData, errFix := resources.BuiltinFS.ReadFile(fixedPath)
		if errAct == nil && errFix == nil {
			b.resourceStore.AddTemplate(model.TypeRotate, tag, store.ResourceMap{
				model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: actData, Tag: tag},
				model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: fixData, Tag: tag},
			})
			if tag != model.DefaultTag {
				b.resourceStore.AddTemplate(model.TypeRotate, model.DefaultTag, store.ResourceMap{
					model.TemplateActiveImageName: &model.Resource{Type: "builtin", Bytes: actData, Tag: tag},
					model.TemplateFixedImageName:  &model.Resource{Type: "builtin", Bytes: fixData, Tag: tag},
				})
			}
		}
	}

	// 5. fonts (SIMSUN.TTC and all open-source fonts)
	if entries, err := resources.BuiltinFS.ReadDir("template/fonts"); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			fontData, err := resources.BuiltinFS.ReadFile("template/fonts/" + entry.Name())
			if err != nil {
				continue
			}
			fnt, err := imageutil.ParseFontBytes(fontData)
			if err == nil {
				imageutil.AddFontToPool(fnt)
			}
		}
	} else if fontData, err := resources.BuiltinFS.ReadFile("template/fonts/SIMSUN.TTC"); err == nil {
		fnt, err := imageutil.ParseFontBytes(fontData)
		if err == nil {
			imageutil.SetDefaultFont(fnt)
		}
	}

	return b
}

func (b *TACBuilder) AddDefaultTemplate(resourceBaseDir string) *TACBuilder {
	// Add background resource 1.jpg
	bgPath := filepath.Join(resourceBaseDir, "resource", "1.jpg")
	if _, err := os.Stat(bgPath); err == nil {
		for _, t := range []string{model.TypeSlider, model.TypeRotate, model.TypeRotateDegree, model.TypeConcat, model.TypeWordImageClick} {
			b.resourceStore.AddResource(t, &model.Resource{
				Type: "file",
				Data: bgPath,
				Tag:  model.DefaultTag,
			})
		}
	}

	// Add slider_1 template
	s1Active := filepath.Join(resourceBaseDir, "template", "slider_1", "active.png")
	s1Fixed := filepath.Join(resourceBaseDir, "template", "slider_1", "fixed.png")
	if fileExists(s1Active) && fileExists(s1Fixed) {
		b.resourceStore.AddTemplate(model.TypeSlider, model.DefaultTag, store.ResourceMap{
			model.TemplateActiveImageName: &model.Resource{Type: "file", Data: s1Active, Tag: model.DefaultTag},
			model.TemplateFixedImageName:  &model.Resource{Type: "file", Data: s1Fixed, Tag: model.DefaultTag},
		})
	}

	// Add slider_2 template
	s2Active := filepath.Join(resourceBaseDir, "template", "slider_2", "active.png")
	s2Fixed := filepath.Join(resourceBaseDir, "template", "slider_2", "fixed.png")
	if fileExists(s2Active) && fileExists(s2Fixed) {
		b.resourceStore.AddTemplate(model.TypeSlider, "slider_2", store.ResourceMap{
			model.TemplateActiveImageName: &model.Resource{Type: "file", Data: s2Active, Tag: "slider_2"},
			model.TemplateFixedImageName:  &model.Resource{Type: "file", Data: s2Fixed, Tag: "slider_2"},
		})
	}

	// Add rotate_1 template
	r1Active := filepath.Join(resourceBaseDir, "template", "rotate_1", "active.png")
	r1Fixed := filepath.Join(resourceBaseDir, "template", "rotate_1", "fixed.png")
	if fileExists(r1Active) && fileExists(r1Fixed) {
		b.resourceStore.AddTemplate(model.TypeRotate, model.DefaultTag, store.ResourceMap{
			model.TemplateActiveImageName: &model.Resource{Type: "file", Data: r1Active, Tag: model.DefaultTag},
			model.TemplateFixedImageName:  &model.Resource{Type: "file", Data: r1Fixed, Tag: model.DefaultTag},
		})
	}

	// Add fonts
	fontFile := filepath.Join(resourceBaseDir, "template", "fonts", "SIMSUN.TTC")
	if fileExists(fontFile) {
		b.fontPath = fontFile
		fnt, err := imageutil.LoadFont(fontFile)
		if err == nil {
			imageutil.SetDefaultFont(fnt)
		} else {
			fmt.Println("warn: load default font failed:", err)
		}
	}

	return b
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func (b *TACBuilder) Build() CaptchaApplication {
	multiGen := generator.NewMultiGenerator()
	multiGen.Register(model.TypeSlider, generator.NewSliderGenerator(b.resourceStore))
	multiGen.Register(model.TypeSliderV2, generator.NewSliderV2Generator(b.resourceStore))
	multiGen.Register(model.TypeRotate, generator.NewRotateGenerator(b.resourceStore))
	multiGen.Register(model.TypeRotateDegree, generator.NewRotateDegreeGenerator(b.resourceStore))
	multiGen.Register(model.TypeConcat, generator.NewConcatGenerator(b.resourceStore))
	multiGen.Register(model.TypeWordImageClick, generator.NewWordClickGenerator(b.resourceStore))
	multiGen.Register(model.TypeShapeClick, generator.NewShapeClickGenerator(b.resourceStore))
	multiGen.Register(model.TypeDragDrop, generator.NewDragDropGenerator(b.resourceStore))

	if b.validator == nil {
		b.validator = validator.NewBasicTrackValidator(b.defaultTolerant)
	}

	return NewDefaultCaptchaApplication(multiGen, b.validator, b.cacheStore, b.expire)
}

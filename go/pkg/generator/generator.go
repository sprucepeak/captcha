package generator

import (
	"captcha/pkg/imageutil"
	"captcha/pkg/model"
	"captcha/pkg/store"
	"fmt"
	"image"
	"math"
	"math/rand"
)

// CaptchaGenerator generates captcha images and metadata
type CaptchaGenerator interface {
	Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error)
}

// MultiGenerator manages multiple generators by type
type MultiGenerator struct {
	generators map[string]CaptchaGenerator
}

func NewMultiGenerator() *MultiGenerator {
	return &MultiGenerator{
		generators: make(map[string]CaptchaGenerator),
	}
}

func (m *MultiGenerator) Register(captchaType string, gen CaptchaGenerator) {
	m.generators[captchaType] = gen
}

func (m *MultiGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	gen, ok := m.generators[param.Type]
	if !ok {
		return nil, fmt.Errorf("unsupported captcha generator for type: %s", param.Type)
	}
	return gen.Generate(param)
}

// -------------------------------------------------------------
// Slider Generator
// -------------------------------------------------------------
type SliderGenerator struct {
	resourceStore store.ResourceStore
}

func NewSliderGenerator(rs store.ResourceStore) *SliderGenerator {
	return &SliderGenerator{resourceStore: rs}
}

func (g *SliderGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeSlider, param.BackgroundImageTag)
	if err != nil {
		return nil, err
	}
	tplRes, err := g.resourceStore.GetRandomTemplate(model.TypeSlider, param.TemplateImageTag)
	if err != nil {
		return nil, err
	}

	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, fmt.Errorf("load bg image error: %w", err)
	}
	fixedRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateFixedImageName])
	if err != nil {
		return nil, fmt.Errorf("load fixed image error: %w", err)
	}
	activeRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateActiveImageName])
	if err != nil {
		return nil, fmt.Errorf("load active image error: %w", err)
	}

	bg := imageutil.LoadRGBA(bgRaw)
	fixed := imageutil.LoadRGBA(fixedRaw)
	active := imageutil.LoadRGBA(activeRaw)

	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()
	fixW, fixH := fixed.Bounds().Dx(), fixed.Bounds().Dy()

	// Random X and Y
	minX := fixW + 5
	maxX := bgW - fixW - 10
	if maxX <= minX {
		maxX = minX + 10
	}
	randomX := minX + rand.Intn(maxX-minX)
	randomY := rand.Intn(bgH - fixH)

	// 1. Cut template from background
	cutImg := imageutil.CutImage(bg, fixed, randomX, randomY)

	// 2. Overlay fixed groove onto background
	imageutil.OverlayImage(bg, fixed, randomX, randomY)

	// 3. 增强防AI视觉识别：在背景上随机叠加 1~2 个混淆假缺口，并拉开高度差 (Y 轴错位)
	obfCount := 1
	if param.Obfuscate {
		obfCount = 2
	}
	for i := 0; i < obfCount; i++ {
		obfX := fixW + rand.Intn(bgW-fixW*2)
		// 确保 X 轴错开至少一个滑块宽度
		if math.Abs(float64(obfX-randomX)) > float64(fixW) {
			// 在 Y 轴上做微小的高低落差（仅错位 4~10px，视觉平缓自然同时仍能干扰整行像素特征）
			yOffset := 4 + rand.Intn(7)
			if rand.Intn(2) == 0 {
				yOffset = -yOffset
			}
			obfY := randomY + yOffset
			// 边界约束
			if obfY < 10 {
				obfY = 10
			}
			if obfY > bgH-fixH-10 {
				obfY = bgH - fixH - 10
			}
			imageutil.OverlayImage(bg, fixed, obfX, obfY)
		}
	}

	// 4. Overlay active frame on cutImg
	imageutil.OverlayImage(cutImg, active, 0, 0)

	// 5. Create full-height transparent slider matrix image
	matrixTemplate := imageutil.LoadRGBA(image.NewRGBA(image.Rect(0, 0, active.Bounds().Dx(), bgH)))
	imageutil.OverlayImage(matrixTemplate, cutImg, 0, randomY)

	bgBase64, err := imageutil.ToBase64JPEG(bg, 85)
	if err != nil {
		return nil, err
	}
	tplBase64, err := imageutil.ToBase64PNG(matrixTemplate)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeSlider,
		BackgroundImage:       bgBase64,
		TemplateImage:         tplBase64,
		BackgroundImageTag:    bgRes.Tag,
		TemplateImageTag:      tplRes[model.TemplateActiveImageName].Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    matrixTemplate.Bounds().Dx(),
		TemplateImageHeight:   bgH,
		RandomX:               randomX,
		RandomY:               randomY,
		Tolerant:              0.02,
		TolerantBox:           &model.TolerantBox{Left: 6.0, Right: 6.0, Top: 6.0, Bottom: 6.0}, // 左右中心对齐允许偏差各 6px (原图)，折算到前端屏幕约 ±3px
	}, nil
}

// -------------------------------------------------------------
// SliderV2 Generator (滑块验证V2: 曲线/非水平滑块)
// -------------------------------------------------------------
type SliderV2Generator struct {
	resourceStore store.ResourceStore
}

func NewSliderV2Generator(rs store.ResourceStore) *SliderV2Generator {
	return &SliderV2Generator{resourceStore: rs}
}

func (g *SliderV2Generator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeSliderV2, param.BackgroundImageTag)
	if err != nil {
		bgRes, err = g.resourceStore.GetRandomResource(model.TypeSlider, param.BackgroundImageTag)
		if err != nil {
			return nil, err
		}
	}
	tplRes, err := g.resourceStore.GetRandomTemplate(model.TypeSliderV2, param.TemplateImageTag)
	if err != nil {
		// 回退使用普通 slider 模板
		tplRes, err = g.resourceStore.GetRandomTemplate(model.TypeSlider, param.TemplateImageTag)
		if err != nil {
			return nil, err
		}
	}

	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	fixedRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateFixedImageName])
	if err != nil {
		return nil, err
	}
	activeRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateActiveImageName])
	if err != nil {
		return nil, err
	}

	bg := imageutil.LoadRGBA(bgRaw)
	fixed := imageutil.LoadRGBA(fixedRaw)
	active := imageutil.LoadRGBA(activeRaw)

	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()
	tplW, tplH := fixed.Bounds().Dx(), fixed.Bounds().Dy()

	// 1. 目标缺口 (randomX, randomY): 随机分布在背景图中间安全区域
	randomX := tplW + 20 + rand.Intn(bgW-tplW*2-30)
	randomY := tplH/2 + 20 + rand.Intn(bgH-tplH-40)

	// 2. 曲线起点 (startX, startY): 固定在左侧，Y 与目标缺口完全一致（水平滑动）
	startX := 10
	startY := randomY // 起点 Y 与终点 Y 相同，确保视觉上滑块 Y 轴固定不变

	// 3. 贝塞尔控制点 (CX, CY): 显著增强曲线弧度（50~80px），根据缺口纵坐标自适应向上/向下大幅度弧线，终点保证 100% 重合对齐
	controlX := (startX + randomX) / 2
	arcOffset := 55 + rand.Intn(36) // 较大幅度弧度 55~90px
	var controlY int
	if randomY >= (bgH-tplH)/2 {
		// 缺口在偏下部分，向上做大拱形弧线
		controlY = randomY - arcOffset
		if controlY < 10 {
			controlY = 10
		}
	} else {
		// 缺口在偏上部分，向下做深凹弧线
		controlY = randomY + arcOffset
		if controlY > bgH-tplH-10 {
			controlY = bgH - tplH - 10
		}
	}

	// 4. 镂空目标缺口并叠加阴影 (仅保留这一个真实目标缺口)
	cutImage := imageutil.CutImage(bg, fixed, randomX, randomY)
	imageutil.OverlayImage(bg, fixed, randomX, randomY)

	// 5. 将滑块 active 覆盖到切片上
	imageutil.OverlayImage(cutImage, active, 0, 0)

	// 6. 独立滑块图片（只导出滑块本身尺寸，前端根据曲线公式自由计算 top / left）
	bgBase64, err := imageutil.ToBase64JPEG(bg, 85)
	if err != nil {
		return nil, err
	}
	tplBase64, err := imageutil.ToBase64PNG(cutImage)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeSliderV2,
		BackgroundImage:       bgBase64,
		TemplateImage:         tplBase64,
		BackgroundImageTag:    bgRes.Tag,
		TemplateImageTag:      tplRes[model.TemplateActiveImageName].Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    tplW,
		TemplateImageHeight:   tplH,
		RandomX:               randomX,
		RandomY:               randomY,
		Tolerant:              0.02,
		TolerantBox:           &model.TolerantBox{Left: 6.0, Right: 6.0, Top: 6.0, Bottom: 6.0}, // 左右中心对齐允许偏差各 6px (原图)，折算到前端屏幕约 ±3px
		ViewData: map[string]interface{}{
			"startX":   startX,
			"startY":   startY,
			"targetX":  randomX,
			"targetY":  randomY,
			"controlX": controlX,
			"controlY": controlY,
		},
	}, nil
}

// -------------------------------------------------------------
// Rotate Generator
// -------------------------------------------------------------
type RotateGenerator struct {
	resourceStore store.ResourceStore
}

func NewRotateGenerator(rs store.ResourceStore) *RotateGenerator {
	return &RotateGenerator{resourceStore: rs}
}

func (g *RotateGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeRotate, param.BackgroundImageTag)
	if err != nil {
		return nil, err
	}
	tplRes, err := g.resourceStore.GetRandomTemplate(model.TypeRotate, param.TemplateImageTag)
	if err != nil {
		return nil, err
	}

	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	fixedRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateFixedImageName])
	if err != nil {
		return nil, err
	}
	activeRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateActiveImageName])
	if err != nil {
		return nil, err
	}

	bg := imageutil.LoadRGBA(bgRaw)
	fixed := imageutil.LoadRGBA(fixedRaw)
	active := imageutil.LoadRGBA(activeRaw)

	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()
	fixW, fixH := fixed.Bounds().Dx(), fixed.Bounds().Dy()

	centerX := bgW/2 - fixW/2
	centerY := bgH/2 - fixH/2

	// Cut out center circle
	cutImg := imageutil.CutImage(bg, fixed, centerX, centerY)

	// Overlay fixed template on bg
	imageutil.OverlayImage(bg, fixed, centerX, centerY)
	// Overlay active template on cutImg
	imageutil.OverlayImage(cutImg, active, 0, 0)

	// Random rotate angle
	// randomX determines the target degree
	randomX := fixW + rand.Intn(bgW-fixW*2)
	degree := 360.0 - float64(randomX)/(float64(bgW)/360.0)

	// Matrix template: transparent square
	matrixTemplate := imageutil.LoadRGBA(image.NewRGBA(image.Rect(0, 0, cutImg.Bounds().Dx(), bgH)))
	imageutil.CenterOverlayAndRotateImage(matrixTemplate, cutImg, degree)

	bgBase64, err := imageutil.ToBase64JPEG(bg, 85)
	if err != nil {
		return nil, err
	}
	tplBase64, err := imageutil.ToBase64PNG(matrixTemplate)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeRotate,
		BackgroundImage:       bgBase64,
		TemplateImage:         tplBase64,
		BackgroundImageTag:    bgRes.Tag,
		TemplateImageTag:      tplRes[model.TemplateActiveImageName].Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    matrixTemplate.Bounds().Dx(),
		TemplateImageHeight:   bgH,
		RandomX:               randomX,
		RandomY:               0,
		Tolerant:              0.04,
		ViewData: map[string]interface{}{
			"degree": degree,
		},
	}, nil
}

// -------------------------------------------------------------
// RotateDegree Generator (角度验证：圆形独立图片旋转摆正)
// -------------------------------------------------------------
type RotateDegreeGenerator struct {
	resourceStore store.ResourceStore
}

func NewRotateDegreeGenerator(rs store.ResourceStore) *RotateDegreeGenerator {
	return &RotateDegreeGenerator{resourceStore: rs}
}

func (g *RotateDegreeGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeRotateDegree, param.BackgroundImageTag)
	if err != nil {
		// 回退尝试使用 ROTATE 的资源
		bgRes, err = g.resourceStore.GetRandomResource(model.TypeRotate, param.BackgroundImageTag)
		if err != nil {
			return nil, err
		}
	}
	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	bg := imageutil.LoadRGBA(bgRaw)
	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()

	// 裁切为正圆（直径取背景图宽高较小值的 85% ~ 90%）
	radius := bgH / 2
	if bgW/2 < radius {
		radius = bgW / 2
	}
	circleImg := imageutil.CropCircle(bg, radius)
	circleDia := circleImg.Bounds().Dx()

	// 随机旋转角度 (50 ~ 310 度，避开微小偏移或接近一整圈)
	// randomX 对应前端滑块拖动到的终点位置
	minAngle := 50.0
	maxAngle := 310.0
	randAngle := minAngle + rand.Float64()*(maxAngle-minAngle)
	randomX := int((randAngle / 360.0) * float64(bgW))
	degree := 360.0 - (float64(randomX)/float64(bgW))*360.0

	// 初始倾斜旋转该正圆图片
	rotatedCircle := imageutil.RotateImage(circleImg, degree)

	// 导出为透明 PNG
	bgBase64, err := imageutil.ToBase64PNG(rotatedCircle)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeRotateDegree,
		BackgroundImage:       bgBase64, // 倾斜的圆形图片
		TemplateImage:         "",
		BackgroundImageTag:    bgRes.Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    circleDia,
		TemplateImageHeight:   circleDia,
		RandomX:               randomX,
		RandomY:               0,
		Tolerant:              0.02,
		ViewData: map[string]interface{}{
			"degree": degree,
		},
	}, nil
}

// -------------------------------------------------------------
// Concat Generator
// -------------------------------------------------------------
type ConcatGenerator struct {
	resourceStore store.ResourceStore
}

func NewConcatGenerator(rs store.ResourceStore) *ConcatGenerator {
	return &ConcatGenerator{resourceStore: rs}
}

func (g *ConcatGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeConcat, param.BackgroundImageTag)
	if err != nil {
		return nil, err
	}
	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	bgImage := imageutil.LoadRGBA(bgRaw)

	bgW, bgH := bgImage.Bounds().Dx(), bgImage.Bounds().Dy()

	spacingY := bgH / 4
	randomY := spacingY + rand.Intn(bgH-spacingY*2)
	bgSplitTop, bgSplitBottom := imageutil.SplitImage(randomY, true, bgImage)

	spacingX := bgW / 8
	randomX := spacingX + rand.Intn(bgW-bgW/5-spacingX)
	bgTopLeft, bgTopRight := imageutil.SplitImage(randomX, false, bgSplitTop)

	// Concat top split swapped
	sliderImg := imageutil.ConcatImage(true, bgTopLeft.Bounds().Dx()+bgTopRight.Bounds().Dx(), bgTopLeft.Bounds().Dy(), bgTopRight, bgTopLeft)
	fullImg := imageutil.ConcatImage(false, bgSplitBottom.Bounds().Dx(), sliderImg.Bounds().Dy()+bgSplitBottom.Bounds().Dy(), sliderImg, bgSplitBottom)

	bgBase64, err := imageutil.ToBase64JPEG(fullImg, 85)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeConcat,
		BackgroundImage:       bgBase64,
		TemplateImage:         "",
		BackgroundImageTag:    bgRes.Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		RandomX:               randomX,
		RandomY:               randomY,
		Tolerant:              0.06,
		ViewData: map[string]interface{}{
			"randomY": randomY,
		},
	}, nil
}

// -------------------------------------------------------------
// WordImageClick Generator
// -------------------------------------------------------------
type WordClickGenerator struct {
	resourceStore store.ResourceStore
}

func NewWordClickGenerator(rs store.ResourceStore) *WordClickGenerator {
	return &WordClickGenerator{resourceStore: rs}
}

func (g *WordClickGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeWordImageClick, param.BackgroundImageTag)
	if err != nil {
		return nil, err
	}
	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	bg := imageutil.CloneRGBA(imageutil.LoadRGBA(bgRaw))
	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()

	checkCount := param.CheckClickCount
	if checkCount <= 0 {
		checkCount = 4
	}
	interfCount := param.InterferenceCount
	if interfCount <= 0 {
		interfCount = 2
	}
	total := checkCount + interfCount

	// 优先选取经典成语/四字短语提升点选语义关联性（如日中天、风和日丽等）
	chars := make([]string, total)
	phrase := imageutil.RandomIdiomOrPoemPhrase()
	phraseRunes := []rune(phrase)
	for i := 0; i < total; i++ {
		if i < len(phraseRunes) && i < checkCount {
			chars[i] = string(phraseRunes[i])
		} else {
			chars[i] = imageutil.RandomChineseChar()
		}
	}

	type block struct {
		x, y, w, h int
	}
	var blocks []block

	fontSize := 52.0
	itemSize := int(fontSize * 1.5) // ~78px
	gridW := itemSize + 16
	gridH := itemSize + 16

	xNum := bgW / gridW
	yNum := bgH / gridH
	for x := 0; x < xNum; x++ {
		for y := 0; y < yNum; y++ {
			blocks = append(blocks, block{
				x: x*gridW + 8,
				y: y*gridH + 8,
				w: itemSize,
				h: itemSize,
			})
		}
	}

	rand.Shuffle(len(blocks), func(i, j int) {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	})

	var clickPoints []model.ClickPoint

	// 提示栏字体与背景图中的文字字体故意不使用同一种字体（增强对抗性和辨识趣味性）
	tipFont := imageutil.GetRandomFont()

	for i := 0; i < total; i++ {
		b := blocks[i]
		deg := float64(rand.Intn(40) - 20)
		col := imageutil.RandomColor()
		// 背景图字渲染：可让各文字在字体池中多样化或者统一采用独立于 tipFont 的字体
		currentWordFont := imageutil.GetDistinctFont(tipFont)
		wordImg, err := imageutil.DrawWordImg(chars[i], currentWordFont, fontSize, col, b.w, b.h, deg)
		if err != nil {
			return nil, err
		}
		imageutil.OverlayImage(bg, wordImg, b.x, b.y)

		// Record the check points
		if i < checkCount {
			clickPoints = append(clickPoints, model.ClickPoint{
				X:      b.x + b.w/2,
				Y:      b.y + b.h/2,
				Width:  b.w,
				Height: b.h,
				Tip:    chars[i],
			})
		}
	}

	// Generate Tip Banner image for user (使用独立的 tipFont)
	var tipText string
	for _, p := range clickPoints {
		tipText += p.Tip
	}
	tipImg, err := imageutil.DrawTipImage(tipText, tipFont, 28)
	if err != nil {
		return nil, err
	}

	bgBase64, err := imageutil.ToBase64JPEG(bg, 85)
	if err != nil {
		return nil, err
	}
	tipBase64, err := imageutil.ToBase64PNG(tipImg)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeWordImageClick,
		BackgroundImage:       bgBase64,
		TemplateImage:         tipBase64,
		BackgroundImageTag:    bgRes.Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    tipImg.Bounds().Dx(),
		TemplateImageHeight:   tipImg.Bounds().Dy(),
		Expand:                clickPoints,
		Tolerant:              0.06,
		ViewData: map[string]interface{}{
			"tip": tipText,
		},
	}, nil
}

// -------------------------------------------------------------
// ShapeClickGenerator (图形/形状点选验证码)
// -------------------------------------------------------------
type ShapeClickGenerator struct {
	resourceStore store.ResourceStore
}

func NewShapeClickGenerator(rs store.ResourceStore) *ShapeClickGenerator {
	return &ShapeClickGenerator{resourceStore: rs}
}

func (g *ShapeClickGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeShapeClick, param.BackgroundImageTag)
	if err != nil {
		bgRes, err = g.resourceStore.GetRandomResource(model.TypeSlider, param.BackgroundImageTag)
		if err != nil {
			return nil, err
		}
	}
	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	bg := imageutil.CloneRGBA(imageutil.LoadRGBA(bgRaw))
	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()

	checkCount := param.CheckClickCount
	if checkCount <= 0 {
		checkCount = 3
	}
	interfCount := param.InterferenceCount
	if interfCount <= 0 {
		interfCount = 3 // 默认增加至 3 个干扰图形（共 6 个几何图形）
	}
	total := checkCount + interfCount

	// 选取不重复的几何图形
	shapes := imageutil.DistinctShapes(total)
	for len(shapes) < total {
		shapes = append(shapes, imageutil.RandomShapeType())
	}

	type block struct {
		x, y, w, h int
	}
	var blocks []block
	itemSize := 64
	gridW := itemSize + 24
	gridH := itemSize + 24
	xNum := bgW / gridW
	yNum := bgH / gridH
	for x := 0; x < xNum; x++ {
		for y := 0; y < yNum; y++ {
			blocks = append(blocks, block{
				x: x*gridW + 16,
				y: y*gridH + 16,
				w: itemSize,
				h: itemSize,
			})
		}
	}
	rand.Shuffle(len(blocks), func(i, j int) {
		blocks[i], blocks[j] = blocks[j], blocks[i]
	})

	var clickPoints []model.ClickPoint
	tipBannerW := checkCount * 56
	tipBannerH := 44
	tipBanner := imageutil.LoadRGBA(image.NewRGBA(image.Rect(0, 0, tipBannerW, tipBannerH)))

	for i := 0; i < total; i++ {
		b := blocks[i]
		col := imageutil.RandomColor()
		// 1. 全向随机旋转角度 (0° ~ 360°)
		deg := float64(rand.Intn(360))
		// 2. 动态随机尺寸扰动 (50px ~ 70px)
		curSize := 50 + rand.Intn(21)
		shapeImg := imageutil.DrawShapeIcon(shapes[i], curSize, curSize, col)
		rotatedShape := imageutil.RotateImage(shapeImg, deg)
		// 居中绘制在 block 网格单元内
		offsetX := b.x + (b.w-curSize)/2
		offsetY := b.y + (b.h-curSize)/2
		imageutil.OverlayImage(bg, rotatedShape, offsetX, offsetY)

		if i < checkCount {
			clickPoints = append(clickPoints, model.ClickPoint{
				X:      offsetX + curSize/2,
				Y:      offsetY + curSize/2,
				Width:  curSize,
				Height: curSize,
				Tip:    shapes[i],
			})

			// 绘制在提示栏上 (尺寸 36x36，提示栏带有 ±15° 轻微姿态，增加立体辨识对抗性)
			tipShape := imageutil.DrawShapeIcon(shapes[i], 36, 36, col)
			tipDeg := float64(rand.Intn(31) - 15) // ±15° 微倾斜
			rotatedTip := imageutil.RotateImage(tipShape, tipDeg)
			imageutil.OverlayImage(tipBanner, rotatedTip, i*56+10, 4)
		}
	}

	// 在上方提示栏 (tipBanner) 上强化干扰：3 条贯穿干扰折线 + 8 个随机干扰噪点
	imageutil.DrawNoiseLines(tipBanner, 3, nil)
	imageutil.DrawNoisePoints(tipBanner, 8, nil)

	bgBase64, err := imageutil.ToBase64JPEG(bg, 85)
	if err != nil {
		return nil, err
	}
	tipBase64, err := imageutil.ToBase64PNG(tipBanner)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeShapeClick,
		BackgroundImage:       bgBase64,
		TemplateImage:         tipBase64,
		BackgroundImageTag:    bgRes.Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    tipBannerW,
		TemplateImageHeight:   tipBannerH,
		Expand:                clickPoints,
		Tolerant:              0.06,
		ViewData: map[string]interface{}{
			"count": checkCount,
		},
	}, nil
}

// -------------------------------------------------------------
// DragDropGenerator (自由拖拽验证码)
// -------------------------------------------------------------
type DragDropGenerator struct {
	resourceStore store.ResourceStore
}

func NewDragDropGenerator(rs store.ResourceStore) *DragDropGenerator {
	return &DragDropGenerator{resourceStore: rs}
}

func (g *DragDropGenerator) Generate(param *model.GenerateParam) (*model.ImageCaptchaInfo, error) {
	bgRes, err := g.resourceStore.GetRandomResource(model.TypeDragDrop, param.BackgroundImageTag)
	if err != nil {
		bgRes, err = g.resourceStore.GetRandomResource(model.TypeSlider, param.BackgroundImageTag)
		if err != nil {
			return nil, err
		}
	}
	tplRes, err := g.resourceStore.GetRandomTemplate(model.TypeDragDrop, param.TemplateImageTag)
	if err != nil {
		tplRes, err = g.resourceStore.GetRandomTemplate(model.TypeSlider, param.TemplateImageTag)
		if err != nil {
			return nil, err
		}
	}

	bgRaw, err := imageutil.LoadResourceImage(bgRes)
	if err != nil {
		return nil, err
	}
	fixedRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateFixedImageName])
	if err != nil {
		return nil, err
	}
	activeRaw, err := imageutil.LoadResourceImage(tplRes[model.TemplateActiveImageName])
	if err != nil {
		return nil, err
	}

	bg := imageutil.LoadRGBA(bgRaw)
	fixed := imageutil.LoadRGBA(fixedRaw)
	active := imageutil.LoadRGBA(activeRaw)

	bgW, bgH := bg.Bounds().Dx(), bg.Bounds().Dy()
	tplW, tplH := fixed.Bounds().Dx(), fixed.Bounds().Dy()

	// 目标缺口可以在画板内部安全区域内自由随机（留出上下左右边距，防止缺口太靠边缘导致拼图块无法完全拖入）
	marginH := tplW + 20
	if bgW <= marginH*2 {
		marginH = tplW / 2
	}
	marginV := tplH + 20
	if bgH <= marginV*2 {
		marginV = tplH / 2
	}

	randomX := marginH/2 + rand.Intn(bgW-tplW-marginH/2)
	randomY := 20 + rand.Intn(bgH-tplH-40)

	// 1. 从目标位置抠图
	cutImg := imageutil.CutImage(bg, fixed, randomX, randomY)
	// 2. 覆盖阴影坑位
	imageutil.OverlayImage(bg, fixed, randomX, randomY)
	// 3. 覆盖高亮 active 边框
	imageutil.OverlayImage(cutImg, active, 0, 0)

	// 4. 起始初始位置：固定在画板下方或角落（让用户自由拖拽放置到目标阴影）
	initX := 20
	initY := bgH - tplH - 15
	if initY < 10 {
		initY = 10
	}

	bgBase64, err := imageutil.ToBase64JPEG(bg, 85)
	if err != nil {
		return nil, err
	}
	tplBase64, err := imageutil.ToBase64PNG(cutImg)
	if err != nil {
		return nil, err
	}

	return &model.ImageCaptchaInfo{
		Type:                  model.TypeDragDrop,
		BackgroundImage:       bgBase64,
		TemplateImage:         tplBase64,
		BackgroundImageTag:    bgRes.Tag,
		TemplateImageTag:      tplRes[model.TemplateActiveImageName].Tag,
		BackgroundImageWidth:  bgW,
		BackgroundImageHeight: bgH,
		TemplateImageWidth:    tplW,
		TemplateImageHeight:   tplH,
		RandomX:               randomX,
		RandomY:               randomY,
		Tolerant:              0.02,
		TolerantBox:           &model.TolerantBox{Left: 6.0, Right: 6.0, Top: 6.0, Bottom: 6.0}, // 水平与垂直中心对齐允许偏差各 6px (原图)，折算前端屏幕约 ±3px
		ViewData: map[string]interface{}{
			"initX":   initX,
			"initY":   initY,
			"targetX": randomX,
			"targetY": randomY,
		},
	}, nil
}

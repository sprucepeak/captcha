package validator

import (
	"captcha/pkg/model"
	"fmt"
	"log"
	"math"
)

// Validator validates user submission against stored valid data
type Validator interface {
	GenerateValidData(info *model.ImageCaptchaInfo) (map[string]interface{}, error)
	Validate(track *model.ImageCaptchaTrack, validData map[string]interface{}) model.ApiResponse[interface{}]
}

// SimpleValidator implements standard slider / click validation
type SimpleValidator struct {
	DefaultTolerant float64
}

func NewSimpleValidator(defaultTolerant float64) *SimpleValidator {
	if defaultTolerant <= 0 {
		defaultTolerant = 0.05
	}
	return &SimpleValidator{DefaultTolerant: defaultTolerant}
}

func (v *SimpleValidator) GenerateValidData(info *model.ImageCaptchaInfo) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	data["type"] = info.Type
	tolerant := info.Tolerant
	if tolerant <= 0 {
		tolerant = v.DefaultTolerant
	}
	data["tolerant"] = tolerant

	// 保存四向容差对象（如果指定了 TolerantBox，则优先存入）
	if info.TolerantBox != nil {
		data["tolerantBox"] = *info.TolerantBox
	}

	switch info.Type {
	case model.TypeSlider, model.TypeSliderV2, model.TypeRotate, model.TypeRotateDegree, model.TypeConcat:
		percentage := float64(info.RandomX) / float64(info.BackgroundImageWidth)
		data["percentage"] = percentage
		data["randomX"] = info.RandomX
		data["randomY"] = info.RandomY
		data["bgW"] = info.BackgroundImageWidth
		data["bgH"] = info.BackgroundImageHeight
		data["templateWidth"] = info.TemplateImageWidth
		data["templateHeight"] = info.TemplateImageHeight
	case model.TypeDragDrop:
		data["targetX"] = float64(info.RandomX) / float64(info.BackgroundImageWidth)
		data["targetY"] = float64(info.RandomY) / float64(info.BackgroundImageHeight)
		data["randomX"] = info.RandomX
		data["randomY"] = info.RandomY
		data["bgW"] = info.BackgroundImageWidth
		data["bgH"] = info.BackgroundImageHeight
		data["templateWidth"] = info.TemplateImageWidth
		data["templateHeight"] = info.TemplateImageHeight
	case model.TypeWordImageClick, model.TypeShapeClick:
		points, ok := info.Expand.([]model.ClickPoint)
		if !ok {
			return nil, fmt.Errorf("invalid expand data for %s", info.Type)
		}
		var validPoints [][2]float64
		for _, pt := range points {
			vx := float64(pt.X) / float64(info.BackgroundImageWidth)
			vy := float64(pt.Y) / float64(info.BackgroundImageHeight)
			validPoints = append(validPoints, [2]float64{vx, vy})
		}
		data["points"] = validPoints
		if len(points) > 0 {
			// calculate tolerant based on click point width
			minLeft := (float64(points[0].X) - float64(points[0].Width)/2.0) / float64(info.BackgroundImageWidth)
			pointTol := validPoints[0][0] - minLeft
			if pointTol > 0 {
				data["tolerant"] = pointTol
			}
		}
	}
	return data, nil
}

func (v *SimpleValidator) Validate(track *model.ImageCaptchaTrack, validData map[string]interface{}) model.ApiResponse[interface{}] {
	cType, _ := validData["type"].(string)
	tolerant, ok := validData["tolerant"].(float64)
	if !ok || tolerant <= 0 {
		tolerant = v.DefaultTolerant
	}

	if track.BgImageWidth <= 0 || track.BgImageHeight <= 0 {
		return model.CheckFail[interface{}]("验证码背景图片宽高参数错误")
	}

	switch cType {
	case model.TypeSlider, model.TypeSliderV2, model.TypeRotate, model.TypeRotateDegree, model.TypeConcat:
		return v.validateSlider(track, validData, tolerant)
	case model.TypeDragDrop:
		return v.validateDragDrop(track, validData, tolerant)
	case model.TypeWordImageClick, model.TypeShapeClick:
		return v.validateClick(track, validData, tolerant)
	default:
		return model.CheckFail[interface{}](fmt.Sprintf("unsupported captcha type: %s", cType))
	}
}

// resolveTolerantBox 解析四向容差
// 自动兼容两种模式：
// 1. 如果配置了 TolerantBox（或上下左右任一值 > 0），以该配置为准；否则使用 tolerant 单值对称构建；
// 2. 模式判定：若数值 >= 1.0，视为绝对物理像素；若数值 < 1.0，视为百分比，乘以背景宽高折算为像素。
func resolveTolerantBox(validData map[string]interface{}, defaultTol float64, bgW, bgH int) (topPx, bottomPx, leftPx, rightPx float64) {
	var box model.TolerantBox

	if rawBox, exists := validData["tolerantBox"]; exists {
		if b, ok := rawBox.(model.TolerantBox); ok {
			box = b
		} else if bMap, ok := rawBox.(map[string]interface{}); ok {
			// json 解码或者 map 传递兼容
			if v, ok := bMap["top"].(float64); ok {
				box.Top = v
			}
			if v, ok := bMap["bottom"].(float64); ok {
				box.Bottom = v
			}
			if v, ok := bMap["left"].(float64); ok {
				box.Left = v
			}
			if v, ok := bMap["right"].(float64); ok {
				box.Right = v
			}
		}
	}

	// 如果没有配置四向容差，则回退使用单个 tolerant 建立对称四向容差
	if box.Top <= 0 && box.Bottom <= 0 && box.Left <= 0 && box.Right <= 0 {
		tol, ok := validData["tolerant"].(float64)
		if !ok || tol <= 0 {
			tol = defaultTol
		}
		box = model.NewSymmetricTolerant(tol)
	}

	// 将各项数值根据 >= 1.0 (像素) 或 < 1.0 (百分比) 换算为统一的物理像素阈值
	leftPx = toPixel(box.Left, float64(bgW))
	rightPx = toPixel(box.Right, float64(bgW))
	topPx = toPixel(box.Top, float64(bgH))
	bottomPx = toPixel(box.Bottom, float64(bgH))

	return topPx, bottomPx, leftPx, rightPx
}

func toPixel(val, baseSize float64) float64 {
	if val <= 0 {
		return 0
	}
	if val >= 1.0 {
		// 绝对像素模式
		return val
	}
	// 百分比模式：乘以基准尺寸折算为像素
	return val * baseSize
}

func (v *SimpleValidator) validateDragDrop(track *model.ImageCaptchaTrack, validData map[string]interface{}, tolerant float64) model.ApiResponse[interface{}] {
	bgW := track.BgImageWidth
	bgH := track.BgImageHeight
	if bgW <= 0 || bgH <= 0 {
		return model.CheckFail[interface{}]("背景图片尺寸异常")
	}

	// 目标缺口物理绝对坐标 (像素)
	var targetPxX, targetPxY float64
	if rx, ok := validData["randomX"].(int); ok {
		targetPxX = float64(rx)
	} else if rx, ok := validData["randomX"].(float64); ok {
		targetPxX = rx
	} else if tx, ok := validData["targetX"].(float64); ok {
		targetPxX = tx * float64(bgW)
	}

	if ry, ok := validData["randomY"].(int); ok {
		targetPxY = float64(ry)
	} else if ry, ok := validData["randomY"].(float64); ok {
		targetPxY = ry
	} else if ty, ok := validData["targetY"].(float64); ok {
		targetPxY = ty * float64(bgH)
	}

	if len(track.TrackList) == 0 {
		return model.CheckFail[interface{}]("没有解析到拖拽轨迹")
	}
	lastTrack := track.TrackList[len(track.TrackList)-1]
	userPxX := lastTrack.X
	userPxY := lastTrack.Y

	// 模板尺寸（用于计算中心点）
	tplW, tplH := 0, 0
	if tw, ok := validData["templateWidth"].(int); ok && tw > 0 {
		tplW = tw
	} else if tw, ok := validData["templateWidth"].(float64); ok && tw > 0 {
		tplW = int(tw)
	}
	if th, ok := validData["templateHeight"].(int); ok && th > 0 {
		tplH = th
	} else if th, ok := validData["templateHeight"].(float64); ok && th > 0 {
		tplH = int(th)
	}

	// 兼容偏移量 compatibleOffset（支持从 validData 配置或默认 0）
	compatibleOffsetX, compatibleOffsetY := 0.0, 0.0
	if ox, ok := validData["offsetX"].(float64); ok {
		compatibleOffsetX = ox
	} else if ox, ok := validData["offset"].(float64); ok {
		compatibleOffsetX = ox
	}
	if oy, ok := validData["offsetY"].(float64); ok {
		compatibleOffsetY = oy
	}

	// 计算滑块中心点与阴影缺口中心点
	// 阴影中心 = 缺口左边缘/上边缘 + 模板尺寸的一半
	// 滑块中心 = 用户放置绝对坐标 + 兼容偏移量 + 模板尺寸的一半
	shadowCenterX := targetPxX + float64(tplW)/2.0
	shadowCenterY := targetPxY + float64(tplH)/2.0

	sliderCenterX := userPxX + compatibleOffsetX + float64(tplW)/2.0
	sliderCenterY := userPxY + compatibleOffsetY + float64(tplH)/2.0

	// 计算四向中心偏差 (像素)
	// deltaX > 0 表示偏右，deltaX < 0 表示偏左
	// deltaY > 0 表示偏下，deltaY < 0 表示偏上
	deltaX := sliderCenterX - shadowCenterX
	deltaY := sliderCenterY - shadowCenterY

	topPx, bottomPx, leftPx, rightPx := resolveTolerantBox(validData, v.DefaultTolerant, bgW, bgH)

	log.Printf("[DRAG_DROP] sliderCenter=(%.1f, %.1f) shadowCenter=(%.1f, %.1f) | delta=(%.2f, %.2f) | box(L=%.1f, R=%.1f, T=%.1f, B=%.1f)",
		sliderCenterX, sliderCenterY, shadowCenterX, shadowCenterY, deltaX, deltaY, leftPx, rightPx, topPx, bottomPx)

	// 水平方向镜像/对称容差校验：-leftPx <= deltaX <= rightPx
	if deltaX < -leftPx || deltaX > rightPx {
		if deltaX < -leftPx {
			return model.CheckFail[interface{}](fmt.Sprintf("拼图偏左 %.1fpx，请对齐缺口", math.Abs(deltaX)))
		}
		return model.CheckFail[interface{}](fmt.Sprintf("拼图偏右 %.1fpx，请对齐缺口", deltaX))
	}

	// 垂直方向镜像/对称容差校验：-topPx <= deltaY <= bottomPx
	if deltaY < -topPx || deltaY > bottomPx {
		if deltaY < -topPx {
			return model.CheckFail[interface{}](fmt.Sprintf("拼图偏上 %.1fpx，请对齐缺口", math.Abs(deltaY)))
		}
		return model.CheckFail[interface{}](fmt.Sprintf("拼图偏下 %.1fpx，请对齐缺口", deltaY))
	}

	return model.Success[interface{}](true)
}

func (v *SimpleValidator) validateSlider(track *model.ImageCaptchaTrack, validData map[string]interface{}, tolerant float64) model.ApiResponse[interface{}] {
	bgW := track.BgImageWidth
	bgH := track.BgImageHeight
	if bgW <= 0 {
		return model.CheckFail[interface{}]("背景图片尺寸异常")
	}

	if len(track.TrackList) == 0 {
		return model.CheckFail[interface{}]("没有解析到滑动轨迹")
	}
	firstTrack := track.TrackList[0]
	lastTrack := track.TrackList[len(track.TrackList)-1]

	// 目标缺口绝对水平像素
	var targetPxX float64
	if rx, ok := validData["randomX"].(int); ok {
		targetPxX = float64(rx)
	} else if rx, ok := validData["randomX"].(float64); ok {
		targetPxX = rx
	} else if p, ok := validData["percentage"].(float64); ok {
		targetPxX = p * float64(bgW)
	}

	// 模板宽度（用于计算中心点）
	tplW := 0
	if tw, ok := validData["templateWidth"].(int); ok && tw > 0 {
		tplW = tw
	} else if tw, ok := validData["templateWidth"].(float64); ok && tw > 0 {
		tplW = int(tw)
	}

	// 兼容偏移量 compatibleOffset（支持从 validData 配置或默认 0）
	compatibleOffset := 0.0
	if offset, ok := validData["offset"].(float64); ok {
		compatibleOffset = offset
	} else if offset, ok := validData["offset"].(int); ok {
		compatibleOffset = float64(offset)
	}

	// 用户滑动绝对像素距离
	userSlidePx := lastTrack.X - firstTrack.X

	// 计算滑块中心点与阴影缺口中心点
	// 阴影中心 = 缺口左边缘 + 模板宽度的一半
	// 滑块中心 = 用户滑动距离 + 兼容偏移量 + 模板宽度的一半
	shadowCenterX := targetPxX + float64(tplW)/2.0
	sliderCenterX := userSlidePx + compatibleOffset + float64(tplW)/2.0

	// 中心偏差 (deltaCenterX > 0 偏右，< 0 偏左)
	deltaCenterX := sliderCenterX - shadowCenterX

	_, _, leftPx, rightPx := resolveTolerantBox(validData, v.DefaultTolerant, bgW, bgH)

	log.Printf("[SLIDER] sliderCenter=%.1fpx shadowCenter=%.1fpx (offset=%.1fpx) | deltaCenterX=%.2fpx | allowed=[-%.1fpx, +%.1fpx]",
		sliderCenterX, shadowCenterX, compatibleOffset, deltaCenterX, leftPx, rightPx)

	// 镜像/对称容差校验：-leftPx <= deltaCenterX <= rightPx
	if deltaCenterX < -leftPx || deltaCenterX > rightPx {
		return model.CheckFail[interface{}]("滑动未对齐缺口，请重试")
	}

	return model.Success[interface{}](map[string]interface{}{
		"percentage": userSlidePx / float64(bgW),
	})
}

func (v *SimpleValidator) validateClick(track *model.ImageCaptchaTrack, validData map[string]interface{}, tolerant float64) model.ApiResponse[interface{}] {
	pointsRaw, ok := validData["points"].([][2]float64)
	if !ok {
		return model.CheckFail[interface{}]("未读取到点选验证数据")
	}

	var clickTracks []model.Track
	for _, t := range track.TrackList {
		if t.Type == model.TrackTypeClick || t.Type == "" {
			clickTracks = append(clickTracks, t)
		}
	}

	if len(clickTracks) != len(pointsRaw) {
		return model.CheckFail[interface{}]("点选次数与要求不符")
	}

	for i, pt := range pointsRaw {
		userX := clickTracks[i].X / float64(track.BgImageWidth)
		userY := clickTracks[i].Y / float64(track.BgImageHeight)

		// 使用欧氏距离 (Euclidean Distance) 判定圆形热区点击范围，相比切比雪夫矩形判定更符合人眼与手指点击习惯
		dx := userX - pt[0]
		dy := userY - pt[1]
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist > tolerant {
			return model.CheckFail[interface{}]("点选位置不匹配")
		}
	}
	return model.Success[interface{}](true)
}

// -------------------------------------------------------------
// BasicTrackValidator adds trajectory detection
// exactly reproducing BasicCaptchaTrackValidator in Java
// -------------------------------------------------------------
type BasicTrackValidator struct {
	*SimpleValidator
}

func NewBasicTrackValidator(defaultTolerant float64) *BasicTrackValidator {
	return &BasicTrackValidator{
		SimpleValidator: NewSimpleValidator(defaultTolerant),
	}
}

func (b *BasicTrackValidator) Validate(track *model.ImageCaptchaTrack, validData map[string]interface{}) model.ApiResponse[interface{}] {
	// First run basic value matching
	res := b.SimpleValidator.Validate(track, validData)
	if !res.Success {
		return res
	}

	cType, _ := validData["type"].(string)

	// 点选类行为时间与反作弊检测
	if cType == model.TypeWordImageClick || cType == model.TypeShapeClick {
		return b.validateClickBehavior(track)
	}

	if cType != model.TypeSlider && cType != model.TypeSliderV2 && cType != model.TypeRotate && cType != model.TypeRotateDegree && cType != model.TypeConcat {
		return res
	}

	// track checks
	startSlidingTime := track.StartTime
	endSlidingTime := track.StopTime
	bgWidth := track.BgImageWidth
	trackList := track.TrackList

	// 1. Sliding time < 150ms => fail
	if endSlidingTime-startSlidingTime < 150 {
		return model.CheckFail[interface{}]("滑动速度过快，请重试")
	}

	// 2. Track points < 4 or > bgWidth * 10 => fail
	if len(trackList) < 4 || len(trackList) > bgWidth*10 {
		return model.CheckFail[interface{}]("滑动轨迹采样异常，请重试")
	}

	// 3. First point x,y near origin [-25, 25]
	first := trackList[0]
	if first.X > 25 || first.X < -25 || first.Y > 25 || first.Y < -25 {
		return model.CheckFail[interface{}]("起点坐标异常，请重试")
	}

	check4SameY := 0
	check7OverWidth := 0

	for i := 1; i < len(trackList); i++ {
		t := trackList[i]
		if t.Y == first.Y {
			check4SameY++
		}
		if t.X >= float64(bgWidth) {
			check7OverWidth++
		}
		// 5. Jump distance between consecutive points > 100 => fail
		prev := trackList[i-1]
		if math.Abs(t.X-prev.X) > 100 || math.Abs(t.Y-prev.Y) > 100 {
			return model.CheckFail[interface{}]("滑动轨迹突变跳跃，请重试")
		}
	}

	// 4. Perfectly straight Y line or x over width too many times => fail
	if (len(trackList) > 12 && check4SameY >= len(trackList)-1) || check7OverWidth > 200 {
		return model.CheckFail[interface{}]("检测到非人为机械滑动，请重试")
	}

	// 5. 抖动与曲率特征分析 (人体工程学垂直离散特征)
	// 真人手滑必定存在自然的生理微颤抖与曲率变化，而脚本往往生成绝对死板的完全平直线条
	var yVariance float64
	var totalMoveDist float64
	for i := 1; i < len(trackList); i++ {
		dy := trackList[i].Y - first.Y
		yVariance += dy * dy
		dx := trackList[i].X - trackList[i-1].X
		dyStep := trackList[i].Y - trackList[i-1].Y
		totalMoveDist += math.Sqrt(dx*dx + dyStep*dyStep)
	}
	yVariance /= float64(len(trackList) - 1)
	// 如果滑动距离较长且点数较多，但全程没有任何微小的 Y 轴自然抖动 (方差极其接近 0)
	if len(trackList) > 20 && totalMoveDist > 80 && yVariance < 0.001 {
		return model.CheckFail[interface{}]("轨迹曲率异常，检测到自动化机械滑动")
	}

	// 6. 人体工程学加速度与末端减速特征检测
	// 真人操作通常呈现“启动加速 -> 中段平稳移动 -> 临近目标减速微调”的生理曲线
	splitPos := int(float64(len(trackList)) * 0.7)
	if splitPos > 4 && splitPos < len(trackList)-2 {
		splitTrack := trackList[splitPos-1]
		startAvgSpeed := splitTrack.X / math.Max(1, splitTrack.T)
		lastTrack := trackList[len(trackList)-1]
		endDuration := math.Max(1, lastTrack.T-splitTrack.T)
		endAvgSpeed := (lastTrack.X - splitTrack.X) / endDuration

		// 检查极速恒定匀速脚本特征：前段与后段速度差值比例如果完全为 0 (绝对机械匀速)
		if len(trackList) > 30 && math.Abs(startAvgSpeed-endAvgSpeed) < 0.0001 && startAvgSpeed > 0 {
			return model.CheckFail[interface{}]("滑动加速度特征异常，请重试")
		}
	}

	return model.Success[interface{}](true)
}

// validateClickBehavior 针对文字点选和图形点选的行为与时间特征进行反作弊检测
func (b *BasicTrackValidator) validateClickBehavior(track *model.ImageCaptchaTrack) model.ApiResponse[interface{}] {
	var clickTracks []model.Track
	for _, t := range track.TrackList {
		if t.Type == model.TrackTypeClick || t.Type == "" {
			clickTracks = append(clickTracks, t)
		}
	}
	if len(clickTracks) == 0 {
		return model.CheckFail[interface{}]("没有检测到点选操作")
	}

	// 1. 点选总耗时检测：生理上至少需要阅读和定位时间 (如 250ms)
	totalDuration := track.StopTime - track.StartTime
	if totalDuration < 250 {
		return model.CheckFail[interface{}]("点击速度过快，请重试")
	}

	// 2. 连续点击间隔检测：防范脚本毫秒级瞬间连击 (单次点击间隔不应小于 50ms)
	for i := 1; i < len(clickTracks); i++ {
		interval := clickTracks[i].T - clickTracks[i-1].T
		if interval < 50 {
			return model.CheckFail[interface{}]("点击间隔异常，请重试")
		}
	}

	return model.Success[interface{}](true)
}

package imageutil

import (
	"bytes"
	"captcha/pkg/model"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"math/rand"
	"os"
)

// LoadImage loads an image from file path
func LoadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}

// DecodeImageBytes decodes an image directly from byte slice
func DecodeImageBytes(b []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(b))
	return img, err
}

// LoadResourceImage loads image from Resource (supports Bytes, File path)
func LoadResourceImage(res *model.Resource) (image.Image, error) {
	if res == nil {
		return nil, os.ErrNotExist
	}
	if len(res.Bytes) > 0 {
		return DecodeImageBytes(res.Bytes)
	}
	if res.Data != "" {
		return LoadImage(res.Data)
	}
	return nil, os.ErrNotExist
}

// LoadRGBA ensures an image is image.RGBA
func LoadRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, img, b.Min, draw.Src)
	return dst
}

// CloneRGBA makes an independent copy of an RGBA image
func CloneRGBA(src *image.RGBA) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	copy(dst.Pix, src.Pix)
	return dst
}

// QuantizeToPaletted converts an RGBA image to a lightweight 256-color Paletted image
// retaining transparency, which reduces PNG file size by ~40% - 60%.
func QuantizeToPaletted(src image.Image, maxColors int) *image.Paletted {
	bounds := src.Bounds()
	if maxColors <= 0 || maxColors > 256 {
		maxColors = 256
	}

	// 1. Collect representative colors from src (sampling)
	colorMap := make(map[uint32]color.Color)
	var colors color.Palette

	// Always reserve index 0 for full transparent if image has alpha
	hasAlpha := false
	colors = append(colors, color.RGBA{0, 0, 0, 0})

	w := bounds.Dx()
	h := bounds.Dy()
	step := 1
	if w*h > 20000 {
		step = 2
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y += step {
		for x := bounds.Min.X; x < bounds.Max.X; x += step {
			c := src.At(x, y)
			_, _, _, a := c.RGBA()
			if a < 128 {
				hasAlpha = true
				continue
			}
			// Quantize 8-bit per channel to reduce palette space
			r, g, b, _ := c.RGBA()
			qr := uint8(r >> 11 << 3)
			qg := uint8(g >> 11 << 3)
			qb := uint8(b >> 11 << 3)
			key := uint32(qr)<<16 | uint32(qg)<<8 | uint32(qb)
			if _, exists := colorMap[key]; !exists {
				qc := color.RGBA{R: qr, G: qg, B: qb, A: 255}
				colorMap[key] = qc
				colors = append(colors, qc)
				if len(colors) >= maxColors {
					break
				}
			}
		}
		if len(colors) >= maxColors {
			break
		}
	}

	if !hasAlpha && len(colors) > 1 {
		colors = colors[1:] // remove transparent index if no alpha
	}

	palImg := image.NewPaletted(bounds, colors)
	draw.FloydSteinberg.Draw(palImg, bounds, src, bounds.Min)
	return palImg
}

// ToCompressedBase64PNG converts an image to a lightweight paletted PNG base64 string
func ToCompressedBase64PNG(img image.Image) (string, error) {
	pal := QuantizeToPaletted(img, 256)
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(&buf, pal); err != nil {
		// Fallback to normal PNG
		return ToBase64PNG(img)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ToBase64PNG encodes image to data:image/png;base64,...
func ToBase64PNG(img image.Image) (string, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// ToBase64JPEG encodes image to data:image/jpeg;base64,...
func ToBase64JPEG(img image.Image, quality int) (string, error) {
	if quality <= 0 {
		quality = 85
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return "", err
	}
	return "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// CutImage cuts out part of baseImage according to template mask (where mask alpha > 100)
// Exactly reproducing CaptchaImageUtils.cutImage:
// for y from 0 to bh: for x from 0 to bw:
// if mask.alpha > 100 => target.Set(x, y, base.At(xPos + x, yPos + y))
func CutImage(base *image.RGBA, mask image.Image, xPos, yPos int) *image.RGBA {
	bounds := mask.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	target := image.NewRGBA(image.Rect(0, 0, w, h))

	baseBounds := base.Bounds()

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := mask.At(bounds.Min.X+x, bounds.Min.Y+y)
			_, _, _, a := c.RGBA()
			alpha := a >> 8 // to 0..255
			if alpha > 100 {
				bx := xPos + x
				by := yPos + y
				if bx >= baseBounds.Min.X && bx < baseBounds.Max.X && by >= baseBounds.Min.Y && by < baseBounds.Max.Y {
					target.Set(x, y, base.At(bx, by))
				}
			}
		}
	}
	return target
}

// OverlayImage draws overlay on top of base at (x, y) with alpha blending
func OverlayImage(base *image.RGBA, overlay image.Image, x, y int) {
	ob := overlay.Bounds()
	r := image.Rect(x, y, x+ob.Dx(), y+ob.Dy())
	draw.Draw(base, r, overlay, ob.Min, draw.Over)
}

// RotateImage rotates image by degree around center (counter-clockwise or clockwise)
// degree in degrees (standard math: radians = degree * math.Pi / 180)
// Using bilinear interpolation
func RotateImage(src image.Image, degree float64) *image.RGBA {
	rad := degree * math.Pi / 180.0
	sin, cos := math.Sin(rad), math.Cos(rad)

	bounds := src.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0

	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			// Offset from center in dst
			xOff := float64(dx) - cx
			yOff := float64(dy) - cy

			// Invert rotation to find source coord
			sx := cos*xOff + sin*yOff + cx
			sy := -sin*xOff + cos*yOff + cy

			if sx >= 0 && sx < float64(w-1) && sy >= 0 && sy < float64(h-1) {
				// Bilinear interpolation
				x0 := int(sx)
				y0 := int(sy)
				x1 := x0 + 1
				y1 := y0 + 1

				fx := sx - float64(x0)
				fy := sy - float64(y0)

				c00 := colorRGBA(src.At(bounds.Min.X+x0, bounds.Min.Y+y0))
				c10 := colorRGBA(src.At(bounds.Min.X+x1, bounds.Min.Y+y0))
				c01 := colorRGBA(src.At(bounds.Min.X+x0, bounds.Min.Y+y1))
				c11 := colorRGBA(src.At(bounds.Min.X+x1, bounds.Min.Y+y1))

				r := bilerp(c00.R, c10.R, c01.R, c11.R, fx, fy)
				g := bilerp(c00.G, c10.G, c01.G, c11.G, fx, fy)
				b := bilerp(c00.B, c10.B, c01.B, c11.B, fx, fy)
				a := bilerp(c00.A, c10.A, c01.A, c11.A, fx, fy)

				dst.SetRGBA(dx, dy, color.RGBA{R: r, G: g, B: b, A: a})
			}
		}
	}
	return dst
}

func colorRGBA(c color.Color) color.RGBA {
	r, g, b, a := c.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}

func bilerp(v00, v10, v01, v11 uint8, fx, fy float64) uint8 {
	f00 := float64(v00)
	f10 := float64(v10)
	f01 := float64(v01)
	f11 := float64(v11)

	top := f00*(1-fx) + f10*fx
	bot := f01*(1-fx) + f11*fx
	val := top*(1-fy) + bot*fy
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return uint8(math.Round(val))
}

// CropCircle crops an image into a circle centered at (w/2, h/2) with radius r
func CropCircle(src image.Image, radius int) *image.RGBA {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if radius <= 0 {
		radius = w / 2
		if h/2 < radius {
			radius = h / 2
		}
	}
	dia := radius * 2
	dst := image.NewRGBA(image.Rect(0, 0, dia, dia))
	cx := float64(w) / 2.0
	cy := float64(h) / 2.0
	r2 := float64(radius * radius)

	for y := 0; y < dia; y++ {
		for x := 0; x < dia; x++ {
			// Center relative coord in circle
			dx := float64(x - radius)
			dy := float64(y - radius)
			dist2 := dx*dx + dy*dy

			if dist2 <= r2 {
				// Anti-alias smooth edge in 1 pixel boundary
				alphaFactor := 1.0
				if dist2 > float64((radius-1)*(radius-1)) {
					alphaFactor = (float64(radius) - math.Sqrt(dist2))
					if alphaFactor < 0 {
						alphaFactor = 0
					}
				}

				srcX := int(cx + dx)
				srcY := int(cy + dy)
				if srcX >= bounds.Min.X && srcX < bounds.Max.X && srcY >= bounds.Min.Y && srcY < bounds.Max.Y {
					c := colorRGBA(src.At(srcX, srcY))
					c.A = uint8(float64(c.A) * alphaFactor)
					dst.SetRGBA(x, y, c)
				}
			}
		}
	}
	return dst
}

// CenterOverlayAndRotateImage rotates coverImage and overlays it in center of baseImage
func CenterOverlayAndRotateImage(base *image.RGBA, cover image.Image, degree float64) {
	rotated := RotateImage(cover, degree)
	bw := base.Bounds().Dx()
	bh := base.Bounds().Dy()
	cw := rotated.Bounds().Dx()
	ch := rotated.Bounds().Dy()
	OverlayImage(base, rotated, bw/2-cw/2, bh/2-ch/2)
}

// SplitImage splits an image into 2 parts horizontally or vertically
// if horizontal (direction=true): start is [0..height-pos], end is [height-pos..height]
// as in Java:
// startImageHeight = img.getHeight() - pos
// startImg = img.getSubimage(0, 0, width, startImageHeight)
// endImg = img.getSubimage(0, startImageHeight, width, pos)
func SplitImage(pos int, direction bool, img *image.RGBA) (*image.RGBA, *image.RGBA) {
	b := img.Bounds()
	w := b.Dx()
	h := b.Dy()

	var startImg, endImg *image.RGBA
	if direction {
		// Horizontal split
		startH := h - pos
		endH := pos
		startImg = image.NewRGBA(image.Rect(0, 0, w, startH))
		endImg = image.NewRGBA(image.Rect(0, 0, w, endH))
		draw.Draw(startImg, startImg.Bounds(), img, image.Point{0, 0}, draw.Src)
		draw.Draw(endImg, endImg.Bounds(), img, image.Point{0, startH}, draw.Src)
	} else {
		// Vertical split
		startW := pos
		endW := w - pos
		startImg = image.NewRGBA(image.Rect(0, 0, startW, h))
		endImg = image.NewRGBA(image.Rect(0, 0, endW, h))
		draw.Draw(startImg, startImg.Bounds(), img, image.Point{0, 0}, draw.Src)
		draw.Draw(endImg, endImg.Bounds(), img, image.Point{startW, 0}, draw.Src)
	}
	return startImg, endImg
}

// ConcatImage concatenates images along horizontal (direction=true) or vertical (direction=false) axis
func ConcatImage(direction bool, width, height int, imgs ...*image.RGBA) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	pos := 0
	for _, img := range imgs {
		ib := img.Bounds()
		var r image.Rectangle
		if direction {
			r = image.Rect(pos, 0, pos+ib.Dx(), ib.Dy())
			draw.Draw(dst, r, img, ib.Min, draw.Src)
			pos += ib.Dx()
		} else {
			r = image.Rect(0, pos, ib.Dx(), pos+ib.Dy())
			draw.Draw(dst, r, img, ib.Min, draw.Src)
			pos += ib.Dy()
		}
	}
	return dst
}

// DrawNoiseLines draws random lines on image
func DrawNoiseLines(img *image.RGBA, count int, col color.Color) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for i := 0; i < count; i++ {
		c := col
		if c == nil {
			c = RandomColor()
		}
		x1, y1 := rand.Intn(w), rand.Intn(h)
		x2, y2 := rand.Intn(w), rand.Intn(h)
		DrawLine(img, x1, y1, x2, y2, c)
	}
}

// DrawNoisePoints draws random circles/dots on image
func DrawNoisePoints(img *image.RGBA, count int, col color.Color) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	for i := 0; i < count; i++ {
		c := col
		if c == nil {
			c = RandomColor()
		}
		cx, cy := rand.Intn(w), rand.Intn(h)
		radius := 2 + rand.Intn(4)
		DrawCircle(img, cx, cy, radius, c)
	}
}

// DrawLine draws Bresenham line
func DrawLine(img *image.RGBA, x0, y0, x1, y1 int, col color.Color) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	sx, sy := 1, 1
	if x0 >= x1 {
		sx = -1
	}
	if y0 >= y1 {
		sy = -1
	}
	err := dx - dy

	for {
		img.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

// DrawCircle draws outline of circle
func DrawCircle(img *image.RGBA, cx, cy, r int, col color.Color) {
	x := r
	y := 0
	err := 0

	for x >= y {
		img.Set(cx+x, cy+y, col)
		img.Set(cx+y, cy+x, col)
		img.Set(cx-y, cy+x, col)
		img.Set(cx-x, cy+y, col)
		img.Set(cx-x, cy-y, col)
		img.Set(cx-y, cy-x, col)
		img.Set(cx+y, cy-x, col)
		img.Set(cx+x, cy-y, col)

		y++
		err += 1 + 2*y
		if 2*(err-x)+1 > 0 {
			x--
			err += 1 - 2*x
		}
	}
}

// DrawFilledCircle draws a filled circle
func DrawFilledCircle(img *image.RGBA, cx, cy, r int, col color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.Set(cx+x, cy+y, col)
			}
		}
	}
}

// DrawShapeIcon draws various geometric/symbol shapes onto an RGBA canvas of size (w, h)
func DrawShapeIcon(shapeType string, w, h int, col color.Color) *image.RGBA {
	canvas := LoadRGBA(image.NewRGBA(image.Rect(0, 0, w, h)))
	cx, cy := w/2, h/2
	r := int(float64(w) * 0.38)
	if h < w {
		r = int(float64(h) * 0.38)
	}

	switch shapeType {
	case "circle":
		DrawFilledCircle(canvas, cx, cy, r, col)
	case "square":
		half := int(float64(r) * 0.88)
		for y := cy - half; y <= cy+half; y++ {
			for x := cx - half; x <= cx+half; x++ {
				canvas.Set(x, y, col)
			}
		}
	case "triangle":
		// 正三角形
		hTri := int(float64(r) * 1.6)
		topY := cy - hTri/2
		botY := topY + hTri
		for y := topY; y <= botY; y++ {
			prog := float64(y-topY) / float64(hTri)
			halfW := int(prog * float64(r) * 1.1)
			for x := cx - halfW; x <= cx+halfW; x++ {
				canvas.Set(x, y, col)
			}
		}
	case "diamond":
		// 菱形
		for y := -r; y <= r; y++ {
			maxSpan := r - int(math.Abs(float64(y)))
			for x := -maxSpan; x <= maxSpan; x++ {
				canvas.Set(cx+x, cy+y, col)
			}
		}
	case "star":
		// 8角闪耀星标 (8-Point Sparkle Star)
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				fx := float64(x)
				fy := float64(y)
				fr := float64(r)
				cond1 := (fx*fx)/(fr*fr*0.25*0.25)+(fy*fy)/(fr*fr*1.0*1.0) <= 1.0
				cond2 := (fx*fx)/(fr*fr*1.0*1.0)+(fy*fy)/(fr*fr*0.25*0.25) <= 1.0
				d1 := fx + fy
				d2 := fx - fy
				cond3 := (d1*d1)/(fr*fr*0.95*0.95)+(d2*d2)/(fr*fr*0.30*0.30) <= 1.0
				cond4 := (d2*d2)/(fr*fr*0.95*0.95)+(d1*d1)/(fr*fr*0.30*0.30) <= 1.0
				if cond1 || cond2 || cond3 || cond4 {
					canvas.Set(cx+x, cy+y, col)
				}
			}
		}
	case "flower":
		// 五瓣梅花 (Flower / Blossom)
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				dist := math.Hypot(float64(x), float64(y))
				inPetal := false
				for k := 0; k < 5; k++ {
					ang := float64(k)*2.0*math.Pi/5.0 - math.Pi/2.0
					px := float64(r) * 0.48 * math.Cos(ang)
					py := float64(r) * 0.48 * math.Sin(ang)
					if math.Hypot(float64(x)-px, float64(y)-py) <= float64(r)*0.38 {
						inPetal = true
						break
					}
				}
				if inPetal && dist > float64(r)*0.22 {
					canvas.Set(cx+x, cy+y, col)
				}
			}
		}
	case "ring":
		// 圆环 / 甜甜圈 (Ring / Donut)
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				d2 := x*x + y*y
				if d2 <= r*r && d2 >= int(float64(r*r)*0.28) {
					canvas.Set(cx+x, cy+y, col)
				}
			}
		}
	case "moon":
		// 月牙 / 弯月 (Crescent Moon)
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				d1 := x*x + y*y
				offsetD2 := math.Pow(float64(x)-float64(r)*0.45, 2) + math.Pow(float64(y)-float64(r)*0.10, 2)
				if d1 <= r*r && offsetD2 > math.Pow(float64(r)*0.80, 2) {
					canvas.Set(cx+x, cy+y, col)
				}
			}
		}
	case "heart":
		// 爱心
		for py := -r; py <= r; py++ {
			for px := -r; px <= r; px++ {
				x := float64(px) / (float64(r) * 0.85)
				y := float64(-py)/(float64(r)*0.85) + 0.2
				// Standard heart curve: (x^2 + y^2 - 1)^3 - x^2 * y^3 <= 0
				f := math.Pow(x*x+y*y-1.0, 3) - x*x*y*y*y
				if f <= 0 {
					canvas.Set(cx+px, cy+py, col)
				}
			}
		}
	case "cross":
		// 十字形
		thick := int(float64(r) * 0.45)
		lenHalf := int(float64(r) * 0.95)
		for y := cy - lenHalf; y <= cy+lenHalf; y++ {
			for x := cx - thick; x <= cx+thick; x++ {
				canvas.Set(x, y, col)
			}
		}
		for y := cy - thick; y <= cy+thick; y++ {
			for x := cx - lenHalf; x <= cx+lenHalf; x++ {
				canvas.Set(x, y, col)
			}
		}
	default:
		DrawFilledCircle(canvas, cx, cy, r, col)
	}
	return canvas
}

// RandomShapeType returns a random shape type
func RandomShapeType() string {
	types := []string{"circle", "square", "triangle", "diamond", "star", "flower", "ring", "moon", "heart", "cross"}
	return types[rand.Intn(len(types))]
}

// DistinctShapes returns n unique shapes
func DistinctShapes(n int) []string {
	all := []string{"circle", "square", "triangle", "diamond", "star", "flower", "ring", "moon", "heart", "cross"}
	rand.Shuffle(len(all), func(i, j int) {
		all[i], all[j] = all[j], all[i]
	})
	if n > len(all) {
		n = len(all)
	}
	return all[:n]
}

// RandomColor generates a pleasant random RGB color
func RandomColor() color.RGBA {
	return color.RGBA{
		R: uint8(rand.Intn(200)),
		G: uint8(rand.Intn(200)),
		B: uint8(rand.Intn(200)),
		A: 255,
	}
}

package imageutil

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"os"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

var defaultFont *opentype.Font
var fontPool []*opentype.Font

// LoadFont loads a truetype / opentype font from disk
func LoadFont(path string) (*opentype.Font, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseFontBytes(b)
}

// ParseFontBytes parses opentype/truetype font bytes
func ParseFontBytes(b []byte) (*opentype.Font, error) {
	f, err := opentype.Parse(b)
	if err != nil {
		// In case of collection (TTC), try parse collection
		col, err2 := opentype.ParseCollection(b)
		if err2 == nil && col.NumFonts() > 0 {
			return col.Font(0)
		}
		return nil, err
	}
	return f, nil
}

// AddFontToPool adds a font to the available font pool for random selection
func AddFontToPool(f *opentype.Font) {
	if f != nil {
		fontPool = append(fontPool, f)
		if defaultFont == nil {
			defaultFont = f
		}
	}
}

// SetDefaultFont sets the global font fallback
func SetDefaultFont(f *opentype.Font) {
	defaultFont = f
	AddFontToPool(f)
}

// GetDefaultFont returns the default fallback font
func GetDefaultFont() *opentype.Font {
	return defaultFont
}

// GetRandomFont returns a random font from the pool
func GetRandomFont() *opentype.Font {
	if len(fontPool) > 0 {
		return fontPool[rand.Intn(len(fontPool))]
	}
	return defaultFont
}

// GetDistinctFont returns a font distinct from other, if possible
func GetDistinctFont(other *opentype.Font) *opentype.Font {
	if len(fontPool) <= 1 {
		return defaultFont
	}
	for i := 0; i < 10; i++ {
		candidate := fontPool[rand.Intn(len(fontPool))]
		if candidate != other {
			return candidate
		}
	}
	return defaultFont
}

// DrawWordImg renders a single Chinese character or word rotated into an RGBA image
func DrawWordImg(word string, fnt *opentype.Font, fontSize float64, col color.Color, width, height int, deg float64) (*image.RGBA, error) {
	if fnt == nil {
		fnt = defaultFont
	}
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	defer face.Close()

	srcImg := image.NewRGBA(image.Rect(0, 0, width, height))

	// Measure text
	bounds, _ := font.BoundString(face, word)
	strW := (bounds.Max.X - bounds.Min.X).Ceil()
	strH := (bounds.Max.Y - bounds.Min.Y).Ceil()

	dotX := (width - strW) / 2
	dotY := (height-strH)/2 + strH

	d := &font.Drawer{
		Dst:  srcImg,
		Src:  image.NewUniform(col),
		Face: face,
		Dot:  fixed.P(dotX, dotY),
	}
	d.DrawString(word)

	if deg != 0 {
		srcImg = RotateImage(srcImg, deg)
	}

	// 增加微小波浪扭曲与微扰 (Sine wave distortion / Warp)，破坏标准字形结构，有效对抗 OCR 提取
	return WarpSinImage(srcImg, 2.5, 18.0), nil
}

// WarpSinImage applies a subtle horizontal & vertical sine wave distortion to the image
func WarpSinImage(src *image.RGBA, amplitude, period float64) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w, h))

	phaseX := rand.Float64() * 2 * math.Pi
	phaseY := rand.Float64() * 2 * math.Pi

	for y := 0; y < h; y++ {
		// x 轴微波偏移
		xOffset := amplitude * math.Sin(2*math.Pi*float64(y)/period+phaseX)
		for x := 0; x < w; x++ {
			// y 轴微波偏移
			yOffset := (amplitude * 0.6) * math.Cos(2*math.Pi*float64(x)/period+phaseY)

			srcX := float64(x) + xOffset
			srcY := float64(y) + yOffset

			// 双线性采样
			x0 := int(math.Floor(srcX))
			y0 := int(math.Floor(srcY))
			x1 := x0 + 1
			y1 := y0 + 1

			if x0 >= 0 && x1 < w && y0 >= 0 && y1 < h {
				wx := srcX - float64(x0)
				wy := srcY - float64(y0)

				c00 := src.RGBAAt(x0, y0)
				c10 := src.RGBAAt(x1, y0)
				c01 := src.RGBAAt(x0, y1)
				c11 := src.RGBAAt(x1, y1)

				r := (1-wx)*(1-wy)*float64(c00.R) + wx*(1-wy)*float64(c10.R) + (1-wx)*wy*float64(c01.R) + wx*wy*float64(c11.R)
				g := (1-wx)*(1-wy)*float64(c00.G) + wx*(1-wy)*float64(c10.G) + (1-wx)*wy*float64(c01.G) + wx*wy*float64(c11.G)
				bVal := (1-wx)*(1-wy)*float64(c00.B) + wx*(1-wy)*float64(c10.B) + (1-wx)*wy*float64(c01.B) + wx*wy*float64(c11.B)
				a := (1-wx)*(1-wy)*float64(c00.A) + wx*(1-wy)*float64(c10.A) + (1-wx)*wy*float64(c01.A) + wx*wy*float64(c11.A)

				dst.SetRGBA(x, y, color.RGBA{
					R: uint8(math.Round(r)),
					G: uint8(math.Round(g)),
					B: uint8(math.Round(bVal)),
					A: uint8(math.Round(a)),
				})
			} else if x0 >= 0 && x0 < w && y0 >= 0 && y0 < h {
				dst.SetRGBA(x, y, src.RGBAAt(x0, y0))
			}
		}
	}
	return dst
}

// DrawTipImage creates a prompt/tip banner containing target words with noise lines and points
func DrawTipImage(text string, fnt *opentype.Font, fontSize float64) (*image.RGBA, error) {
	if fnt == nil {
		fnt = defaultFont
	}
	face, err := opentype.NewFace(fnt, &opentype.FaceOptions{
		Size:    fontSize,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return nil, err
	}
	defer face.Close()

	bounds, _ := font.BoundString(face, text)
	textW := (bounds.Max.X - bounds.Min.X).Ceil()
	textH := (bounds.Max.Y - bounds.Min.Y).Ceil()

	width := textW + 30
	height := int(fontSize) + 20

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	// Fill subtle light background
	draw.Draw(img, img.Bounds(), &image.Uniform{color.RGBA{240, 240, 240, 255}}, image.Point{}, draw.Src)

	// Draw noise lines
	DrawNoiseLines(img, 3, nil)
	DrawNoisePoints(img, 8, nil)

	dotX := 15
	dotY := (height-textH)/2 + textH

	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.RGBA{30, 30, 30, 255}),
		Face: face,
		Dot:  fixed.P(dotX, dotY),
	}
	d.DrawString(text)

	return img, nil
}

// RandomChineseChar returns a random common Chinese character (700+ 笔画常用汉字)
var commonChars = []rune("的一是在不了有和人这中大为上个国我以要他时来用生到作地于出就分对成会可主发年动同工也能下过子说产种面而方后多定行学法所民得经十三之进着等部度家电力里如水化高自二理起小物现实加量都两体制机当使点从业本去把性好应开它天正公向合结重见明问情各先并回意文立由关长间知物外月道想给总清真题门目特反建任直党放向保活每正应名平代变安次走其相正打向结向求百条利身关题海展转便指果代强色算题解保交结表风解意向通全走认提向意向文保条通明建海向提向平回正真解向求百打条色利利身解海名相题平提保回放全正意建明代指各水解公提保打解色算身展向全认水代向指求意向平身各明海条建解正题回放提展利各解名算身向真意打平展解水代题向认正回展求保提指各名解平解全身解公打各题正海解水建向平认展求指各名平解回身打题解正水平展公求全认各指海各平解打身题解建水展求公各指平各身打题解回全展求公各指平各身打题解建水展求")

func RandomChineseChar() string {
	idx := rand.Intn(len(commonChars))
	return string(commonChars[idx])
}

// RandomIdiomOrPoemPhrase returns a random 4-character idiom or poetry phrase
var idiomPhrases = []string{
	"如日中天", "风和日丽", "一马当先", "锦绣山河",
	"海阔天空", "天朗气清", "春风化雨", "光明磊落",
	"厚积薄发", "宁静致远", "志存高远", "乘风破浪",
	"生龙活虎", "万里晴空", "紫气东来", "万象更新",
	"星月交辉", "云淡风轻", "金石为开", "百花齐放",
}

func RandomIdiomOrPoemPhrase() string {
	return idiomPhrases[rand.Intn(len(idiomPhrases))]
}


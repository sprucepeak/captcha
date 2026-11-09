package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// GenerateDynamicKey derives a 32-byte AES key from captchaID and creation timestamp
func GenerateDynamicKey(captchaID string, timestamp int64) []byte {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%s_salt_#2026_%d", captchaID, timestamp)))
	return h.Sum(nil)
}

// Encrypt encrypts plain bytes using AES-GCM with dynamic key
func Encrypt(plaintext []byte, key []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts base64 encoded ciphertext using AES-GCM
func Decrypt(cipherBase64 string, key []byte) ([]byte, error) {
	data, err := base64.StdEncoding.DecodeString(cipherBase64)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// ClientEnvInfo captures browser anti-bot fingerprints
type ClientEnvInfo struct {
	Webdriver   bool    `json:"webdriver"`
	Platform    string  `json:"platform"`
	UserAgent   string  `json:"userAgent"`
	Language    string  `json:"language"`
	ScreenWidth int     `json:"screenWidth"`
	ScreenHeight int    `json:"screenHeight"`
	ColorDepth  int     `json:"colorDepth"`
	DevicePixelRatio float64 `json:"devicePixelRatio"`
	CanvasHash  string  `json:"canvasHash"`
	Timezone    int     `json:"timezone"`
}

// VerifyClientEnv checks if the client is an automated headless browser (Puppeteer, Selenium, Playwright)
func VerifyClientEnv(env *ClientEnvInfo) (bool, string) {
	if env == nil {
		// If no env is supplied in secure mode
		return true, ""
	}
	if env.Webdriver {
		return false, "环境异常：检测到自动化测试驱动 (WebDriver)"
	}
	if env.ScreenWidth <= 0 || env.ScreenHeight <= 0 {
		return false, "环境异常：无效的显示环境分辨率"
	}
	return true, ""
}

// DecryptTrackPayload decrypts and parses payload JSON
func DecryptTrackPayload[T any](cipherBase64 string, captchaID string, timestamp int64) (*T, error) {
	key := GenerateDynamicKey(captchaID, timestamp)
	raw, err := Decrypt(cipherBase64, key)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %w", err)
	}
	var res T
	if err := json.Unmarshal(raw, &res); err != nil {
		return nil, fmt.Errorf("json unmarshal failed: %w", err)
	}
	return &res, nil
}

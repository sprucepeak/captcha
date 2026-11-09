package security

import (
	"captcha/pkg/model"
	"testing"
)

func TestDynamicEncryptDecrypt(t *testing.T) {
	captchaID := "SLIDER_123456789"
	timestamp := int64(1700000000000)

	type Packet struct {
		Track model.ImageCaptchaTrack `json:"track"`
		Env   *ClientEnvInfo          `json:"env"`
	}

	pkt := Packet{
		Track: model.ImageCaptchaTrack{
			BgImageWidth:  600,
			BgImageHeight: 360,
			StartTime:     timestamp,
			StopTime:      timestamp + 500,
			TrackList: []model.Track{
				{X: 0, Y: 0, Type: "down", T: 0},
				{X: 100, Y: 1, Type: "move", T: 200},
				{X: 250, Y: 0, Type: "up", T: 500},
			},
		},
		Env: &ClientEnvInfo{
			Webdriver:   false,
			Platform:    "Win32",
			ScreenWidth: 1920,
			ScreenHeight: 1080,
		},
	}

	key := GenerateDynamicKey(captchaID, timestamp)
	raw, err := Encrypt([]byte(`{"track":{"bgImageWidth":600,"bgImageHeight":360},"env":{"webdriver":false,"screenWidth":1920,"screenHeight":1080}}`), key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}

	decrypted, err := DecryptTrackPayload[Packet](raw, captchaID, timestamp)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}

	if decrypted.Track.BgImageWidth != 600 {
		t.Fatalf("unexpected bg width: %d", decrypted.Track.BgImageWidth)
	}

	if ok, _ := VerifyClientEnv(decrypted.Env); !ok {
		t.Fatalf("env check should pass")
	}

	// Test webdriver rejection
	pkt.Env.Webdriver = true
	if ok, reason := VerifyClientEnv(pkt.Env); ok {
		t.Fatalf("webdriver should be rejected, got ok")
	} else {
		t.Logf("Webdriver rejected successfully: %s", reason)
	}
}


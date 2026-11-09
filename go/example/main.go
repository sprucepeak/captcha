package main

import (
	"fmt"
	"log"
	"net/http"

	"captcha"
)

func main() {
	// 一行代码使用内置资源初始化 SDK（无需依赖本地文件系统）
	app := captcha.NewDefault()

	// 快速生成测试
	res := app.GenerateCaptcha(&captcha.GenerateParam{Type: captcha.TypeSlider})
	if res.Success {
		fmt.Printf("✅ Slider Captcha generated successfully! ID: %s, Background size: %dx%d\n",
			res.Data.ID, res.Data.BackgroundImageWidth, res.Data.BackgroundImageHeight)
	} else {
		fmt.Printf("❌ Failed to generate captcha: %s\n", res.Msg)
	}

	// 旋转/角度验证码生成测试
	rotateRes := app.GenerateCaptcha(&captcha.GenerateParam{Type: captcha.TypeRotate})
	if rotateRes.Success {
		fmt.Printf("✅ Rotate (旋转图片) Captcha generated successfully! ID: %s, Degree: %v\n",
			rotateRes.Data.ID, rotateRes.Data.Data["degree"])
	} else {
		fmt.Printf("❌ Failed to generate rotate captcha: %s\n", rotateRes.Msg)
	}

	degreeRes := app.GenerateCaptcha(&captcha.GenerateParam{Type: captcha.TypeRotateDegree})
	if degreeRes.Success {
		fmt.Printf("✅ RotateDegree (角度验证) Captcha generated successfully! ID: %s, Degree: %v\n",
			degreeRes.Data.ID, degreeRes.Data.Data["degree"])
	} else {
		fmt.Printf("❌ Failed to generate rotate degree captcha: %s\n", degreeRes.Msg)
	}

	mux := http.NewServeMux()
	// 快捷注册 HTTP 路由: /captcha/gen 与 /captcha/check
	captcha.RegisterRoutes(mux, app)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`
			<!DOCTYPE html>
			<html>
			<head>
				<meta charset="utf-8">
				<title>Captcha Go SDK Demo</title>
				<style>
					body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; padding: 24px; background: #f0f2f5; color: #333; }
					.card { background: #fff; padding: 20px; border-radius: 8px; box-shadow: 0 2px 8px rgba(0,0,0,0.1); max-width: 650px; margin-bottom: 20px; }
					a { color: #1890ff; text-decoration: none; }
					a:hover { text-decoration: underline; }
					.btn { display: inline-block; padding: 8px 16px; background: #1890ff; color: #fff; border-radius: 4px; margin-right: 8px; margin-top: 8px; }
					#preview { margin-top: 20px; text-align: center; }
					#preview img { max-width: 100%; border-radius: 6px; box-shadow: 0 2px 8px rgba(0,0,0,0.15); }
				</style>
			</head>
			<body>
				<div class="card">
					<h2>Captcha Go SDK 在线演示与测试</h2>
					<p>点击测试各类型验证码生成接口：</p>
					<div>
						<a class="btn" href="javascript:loadCaptcha('SLIDER')">滑块拼图 (SLIDER)</a>
						<a class="btn" href="javascript:loadCaptcha('SLIDER_V2')">滑块验证V2 (SLIDER_V2)</a>
						<a class="btn" href="javascript:loadCaptcha('ROTATE')">旋转图片 (ROTATE)</a>
						<a class="btn" href="javascript:loadCaptcha('ROTATE_DEGREE')">角度验证 (ROTATE_DEGREE)</a>
						<a class="btn" href="javascript:loadCaptcha('CONCAT')">滑动还原 (CONCAT)</a>
						<a class="btn" href="javascript:loadCaptcha('WORD_IMAGE_CLICK')">文字点选 (WORD_CLICK)</a>
					</div>
					<div id="preview">
						<p style="color: #999;">点击上方按钮预览生成的验证码底图与模板...</p>
					</div>
				</div>
				<script>
					async function loadCaptcha(type) {
						const prev = document.getElementById('preview');
						prev.innerHTML = '<p>正在生成 ' + type + ' 验证码...</p>';
						try {
							const res = await fetch('/captcha/gen?type=' + type);
							const data = await res.json();
							if (data.success) {
								let html = '<div style="position:relative;display:inline-block;border-radius:6px;overflow:hidden;">';
								html += '<img src="' + data.data.backgroundImage + '" style="display:block;width:340px;">';
								if (data.data.templateImage) {
									if (type === 'ROTATE') {
										html += '<div style="position:absolute;top:0;left:0;width:100%;height:100%;display:flex;align-items:center;justify-content:center;">';
										html += '<img src="' + data.data.templateImage + '" style="height:100%;">';
										html += '</div>';
									} else if (type === 'SLIDER') {
										html += '<img src="' + data.data.templateImage + '" style="position:absolute;top:0;left:0;height:100%;">';
									}
								}
								html += '</div>';
								if (type === 'ROTATE_DEGREE') {
									html += '<div style="margin-top:8px;color:#1890ff;font-size:13px;">提示：拖动滑块使整张图片角度摆正</div>';
								}
								if (data.data.type === 'WORD_IMAGE_CLICK' && data.data.templateImage) {
									html += '<div style="margin-top:10px;"><b>点选提示词：</b><img src="' + data.data.templateImage + '" style="vertical-align:middle;height:28px;"></div>';
								}
								html += '<p style="font-size:12px;color:#888;margin-top:8px;">ID: ' + data.data.id + '</p>';
								prev.innerHTML = html;
							} else {
								prev.innerHTML = '<p style="color:red;">生成失败: ' + data.msg + '</p>';
							}
						} catch (e) {
							prev.innerHTML = '<p style="color:red;">请求出错: ' + e.message + '</p>';
						}
					}
				</script>
			</body>
			</html>
		`))
	})

	addr := ":8080"
	fmt.Printf("Server listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}


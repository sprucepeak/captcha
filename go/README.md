# Captcha Go SDK (行为验证码 Go 语言 SDK)

本项目是全功能行为验证码的 Go 语言 SDK 实现。支持 **滑块验证码 (SLIDER)**、**滑块验证V2 (SLIDER_V2)**、**旋转验证码 (ROTATE)**、**角度验证 (ROTATE_DEGREE)**、**还原拼接验证码 (CONCAT)** 和 **文字点选验证码 (WORD_IMAGE_CLICK)**。

---

## 特性亮点

1. **开箱即用 (零外部文件依赖)**：
   内置嵌入了默认背景图、模板切片与中文字体库（通过 `//go:embed`），引入包后**一行代码即可启动**，无需部署复杂的静态素材目录。
2. **多项目友好**：
   顶层包直接暴露所有核心类型与函数，同时保留子包架构，方便第三方项目直接以标准 Go 模块方式引入。
3. **极简标准库**：
   除文字字形解析外（使用官方扩展 `golang.org/x/image`），全部基于 Go 语言原生标准库实现（`image`、`sync`、`net/http`、`crypto/rand` 等），轻量无臃肿依赖。
4. **内置轨迹反爬防刷校验**：
   包含完整的滑动时间（<300ms 拦截）、采样点数量、起止跳跃异常、机械匀速等轨迹检测算法。

---

## SDK 目录设计

```
go/
├── captcha.go                  # 顶层 SDK 统一入口 (对外暴露快捷构造器与核心类型别名)
├── go.mod                      # 模块定义
├── go.sum
├── resources/                  # 内置静态资源 (含 embed.go 打包嵌入)
│   ├── embed.go
│   ├── resource/1.jpg
│   └── template/
│       ├── fonts/SIMSUN.TTC
│       ├── rotate_1/
│       ├── slider_1/
│       └── slider_2/
├── pkg/                        # 内部核心组件
│   ├── application/            # 核心业务门面与 TACBuilder
│   ├── generator/              # 4 种类型验证码生成器
│   ├── imageutil/              # 图像切片、抠图、旋转、字体渲染算法
│   ├── model/                  # 数据模型与统一 API 响应
│   ├── server/                 # 原生 net/http 路由处理器
│   ├── store/                  # 素材管理与内存 TTL 缓存
│   └── validator/              # 基础容错校验与行为轨迹算法校验
├── example/
│   └── main.go                 # HTTP Web 服务集成演示
└── test/
    ├── captcha_test.go         # 基础单元测试
    └── sdk_test.go             # SDK 顶层接口集成测试
```

---

## 快速使用

### 1. 其他项目引入方式

若作为本地依赖或通过 `replace` 引入：

```go
// go.mod
module myapp

go 1.22

require captcha v0.0.0
replace captcha => /path/to/captcha/go
```

### 2. 方式一：一行代码初始化（推荐，使用内置资源）

```go
package main

import (
	"fmt"
	"captcha"
)

func main() {
	// 使用内置资源初始化，开箱即用，无需配置任何素材路径
	app := captcha.NewDefault()

	// 1. 生成滑块验证码
	res := app.GenerateCaptcha(&captcha.GenerateParam{
		Type: captcha.TypeSlider, // 可选: TypeSlider, TypeRotate, TypeConcat, TypeWordImageClick
	})
	if res.Success {
		fmt.Printf("生成成功, ID: %s, 背景图 Base64 长度: %d\n", res.Data.ID, len(res.Data.BackgroundImage))
	}

	// 2. 快捷校验百分比 (例如滑动到 0.45 位置)
	matched := app.MatchingPercentage(res.Data.ID, 0.45)
	fmt.Printf("校验结果: %v\n", matched)
}
```

### 3. 方式二：自定义配置（Builder 模式）

```go
package main

import (
	"time"
	"captcha"
)

func main() {
	app := captcha.Builder().
		AddBuiltinResources().                // 加载内置模板与字体
		SetDefaultTolerant(0.02).             // 设置默认容错率 (2%)
		SetExpire(5 * time.Minute).           // 验证码有效期
		Build()

	// 使用 app 进行验证码生成与核销...
}
```

### 4. 方式三：快速接入 Web 服务 (`net/http`)

SDK 提供了原生 HTTP 路由处理器：

```go
package main

import (
	"net/http"
	"captcha"
)

func main() {
	app := captcha.NewDefault()
	mux := http.NewServeMux()

	// 快捷挂载 /captcha/gen 与 /captcha/check 接口
	captcha.RegisterRoutes(mux, app)

	http.ListenAndServe(":8080", mux)
}
```

**HTTP 接口说明：**

- **生成验证码**：`GET /captcha/gen?type=SLIDER`
  - 返回验证码 ID、Base64 背景图、Base64 滑块/提示图等信息。
- **校验验证码**：`POST /captcha/check`
  - 请求体：
    ```json
    {
      "id": "SLIDER_xxxxxx",
      "track": {
        "bgImageWidth": 600,
        "bgImageHeight": 360,
        "startTime": 1699999000,
        "stopTime": 1699999800,
        "trackList": [
          { "x": 0, "y": 0, "t": 10, "type": "down" },
          { "x": 30, "y": 2, "t": 200, "type": "move" },
          { "x": 120, "y": 4, "t": 800, "type": "up" }
        ]
      }
    }
    ```

---

## 运行与测试

在 `go` 目录下运行全部测试：

```bash
go test -v ./test
```

启动示例服务：

```bash
go run ./example/main.go
```

访问 `http://localhost:8080` 即可在浏览器直接体验接口。

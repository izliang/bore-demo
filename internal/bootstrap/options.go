// Package bootstrap 启动选项和常量定义。
//
// 此处可以放置：
//   - 启动参数的结构体定义（如命令行 flag）
//   - 默认常量（如默认端口、超时时间等）
//   - Option 模式配置（如 WithPort、WithTimeout 等）
//
// 当前模板为空，新项目按需扩展。
package bootstrap

import "context"

type initParam struct {
	env    bool
	config bool
}

type InitParamFn func(*initParam)

func WithEnv(env bool) InitParamFn {
	return func(p *initParam) {
		p.env = env
	}
}

func WithConfig(config bool) InitParamFn {
	return func(p *initParam) {
		p.config = config
	}
}

func ShouldInit(ctx context.Context, opts ...InitParamFn) {
	params := &initParam{}
	for _, fn := range opts {
		fn(params)
	}
	if params.env {
		initEnv(ctx)
	}
	if params.config {
		initConfig(ctx)
	}
}

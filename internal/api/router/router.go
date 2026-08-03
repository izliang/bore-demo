// Package router 负责 HTTP 路由注册和中间件挂载。
//
// 路由设计约定：
//   - 使用 cyberhuma 框架（基于 Huma v2），自动生成 OpenAPI 文档和参数校验
//   - 路由按业务模块分组，每组使用 api.Group() 创建子路由
//   - Handler 签名统一为 func(ctx context.Context, req *I) (*O, e.ServiceError)
//   - 健康检查、指标等基础端点注册在根路由上
//
// 新项目复刻指南：
//   - 复制现有的 Group 结构，替换业务模块名
//   - 每个业务模块一个 Group，内部按操作类型注册 GET/POST/PUT/DELETE
//   - 文档暴露策略通过 WithHumaDocs 控制（生产环境建议 DocConditional）
package router

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore-demo/internal/api/admin"
	"github.com/izliang/bore-demo/internal/api/controller"
	"github.com/izliang/bore-demo/internal/resource"
	"github.com/izliang/bore/e"
	"github.com/izliang/bore/web/middleware"
	"github.com/izliang/bore/web/router"
	"github.com/izliang/bore/web/router/cyberhuma"
)

// SetupRoutes 在给定的 Gin Engine 上注册所有路由和中间件。
//
// 注册顺序：
//  1. 全局中间件（Access 日志、HTTP 指标等，来自 resource.GinMiddlewares）
//  2. 基础端点（/health）
//  3. API v1 业务路由（/api/v1/...）
//
// 参数 r 通常由 main.go 中的 gin.New() 创建。
func SetupRoutes(r *gin.Engine) {

	// ============================================================
	// 注册全局中间件
	// 中间件在 bootstrap 各 init 阶段按顺序追加到 resource.GinMiddlewares
	// ============================================================
	for _, ginMiddleware := range resource.GinMiddlewares {
		r.Use(ginMiddleware)
	}

	// 语言检测中间件（供 amis 页面 i18n 渲染使用）。
	r.Use(middleware.LanguageMiddleware())

	// ============================================================
	// 创建 API Router（基于 cyberhuma，自动 OpenAPI 文档）
	//
	// WithHumaTitle：设置 OpenAPI 文档标题和版本
	// WithHumaDocs：配置文档暴露策略
	//   - Exposure: DocAlways  → 始终暴露（开发/测试环境）
	//   - Exposure: DocNever   → 始终隐藏（生产安全）
	//   - Exposure: DocConditional → 根据条件判断
	//   - UI: DocUISwagger      → 使用 Swagger UI
	//   - UI: DocUIRedoc        → 使用 Redoc UI
	//   - UI: DocUINone         → 不渲染 UI，仅提供 openapi.json
	// ============================================================
	apiRouter := cyberhuma.New(r, cyberhuma.WithHumaTitle("bore-demo", "1.0.0"),
		cyberhuma.WithHumaDocs(router.DocConfig{
			Exposure: router.DocAlways,
			UI:       router.DocUISwagger,
		}),
	)

	// ============================================================
	// 基础端点
	// ============================================================

	// /health —— K8s 健康检查探针。
	cyberhuma.GET(apiRouter, "/health", healthCheck)

	// ============================================================
	// AMIS 管理后台（登录页 /admin/* 页面 / 登录登出 API）
	// ============================================================
	adminMW, err := admin.Setup(r)
	if err != nil {
		panic("setup amis admin: " + err.Error())
	}

	// ============================================================
	// API v1 业务路由
	// ============================================================

	apiV1 := apiRouter.Group("/api/v1")
	{
		// /api/v1/user —— 用户模块
		userApi := apiV1.Group("/user")

		// GET /api/v1/user/get —— 根据 ID 获取用户信息。
		// cyberhuma 自动处理：
		//   - 根据 v1.GetUserRequest 的 struct tag 校验 query 参数
		//   - 生成 OpenAPI 文档
		//   - 参数校验失败时返回 422 + 友好错误信息
		cyberhuma.GET(userApi, "/get", controller.GetUser)

		// ────────────────────────────────────────────────────
		// /api/v1/auth —— 管理后台鉴权信息（登录态）
		// ────────────────────────────────────────────────────
		authApi := apiV1.Group("/auth", cyberhuma.WithGroupGinMiddleware(adminMW.RequireLogin()))
		cyberhuma.GET(authApi, "/info", controller.GetAuthInfo, cyberhuma.WithTags("auth"))

		// ────────────────────────────────────────────────────
		// /api/v1/admin —— 管理员管理（登录态 + 接口级权限）
		// 权限在 controller 入口通过 admin.CheckPerm 声明，便于审计。
		// ────────────────────────────────────────────────────
		adminApi := apiV1.Group("/admin", cyberhuma.WithGroupGinMiddleware(adminMW.RequireLogin()))
		cyberhuma.GET(adminApi, "/list", controller.ListAdmins, cyberhuma.WithTags("admin"))
		cyberhuma.GET(adminApi, "/get", controller.GetAdmin, cyberhuma.WithTags("admin"))
		cyberhuma.POST(adminApi, "/create", controller.CreateAdmin, cyberhuma.WithTags("admin"))
		cyberhuma.PUT(adminApi, "/update", controller.UpdateAdmin, cyberhuma.WithTags("admin"))
		cyberhuma.DELETE(adminApi, "/delete", controller.DeleteAdmin, cyberhuma.WithTags("admin"))
		cyberhuma.POST(adminApi, "/batch-delete", controller.BatchDeleteAdmins, cyberhuma.WithTags("admin"))
		cyberhuma.GET(adminApi, "/stats", controller.AdminStats, cyberhuma.WithTags("admin"))

		// ────────────────────────────────────────────────────
		// /api/v1/role —— 角色管理（登录态 + 接口级权限）
		// ────────────────────────────────────────────────────
		roleApi := apiV1.Group("/role", cyberhuma.WithGroupGinMiddleware(adminMW.RequireLogin()))
		cyberhuma.GET(roleApi, "/list", controller.ListRoles, cyberhuma.WithTags("role"))
		cyberhuma.GET(roleApi, "/get", controller.GetRole, cyberhuma.WithTags("role"))
		cyberhuma.GET(roleApi, "/options", controller.ListRoleOptions, cyberhuma.WithTags("role"))
		cyberhuma.POST(roleApi, "/create", controller.CreateRole, cyberhuma.WithTags("role"))
		cyberhuma.PUT(roleApi, "/update", controller.UpdateRole, cyberhuma.WithTags("role"))
		cyberhuma.DELETE(roleApi, "/delete", controller.DeleteRole, cyberhuma.WithTags("role"))

		// ────────────────────────────────────────────────────
		// /api/v1/menu —— 菜单/权限管理（登录态 + 接口级权限）
		// ────────────────────────────────────────────────────
		menuApi := apiV1.Group("/menu", cyberhuma.WithGroupGinMiddleware(adminMW.RequireLogin()))
		cyberhuma.GET(menuApi, "/tree", controller.MenuTree, cyberhuma.WithTags("menu"))
		cyberhuma.GET(menuApi, "/get", controller.GetMenu, cyberhuma.WithTags("menu"))
		cyberhuma.POST(menuApi, "/create", controller.CreateMenu, cyberhuma.WithTags("menu"))
		cyberhuma.PUT(menuApi, "/update", controller.UpdateMenu, cyberhuma.WithTags("menu"))
		cyberhuma.DELETE(menuApi, "/delete", controller.DeleteMenu, cyberhuma.WithTags("menu"))
	}
}

// HealthResponse 健康检查响应结构体。
type HealthResponse struct {
	// State 服务状态（ok / degraded / unavailable）。
	State string `json:"status"`

	// Message 附加说明信息。
	Message string `json:"message"`

	// Timestamp 检查时间戳（Unix 秒）。
	Timestamp int64 `json:"timestamp"`

	// Checks 各依赖组件的健康检查结果。
	// key 为组件名（如 "database"、"redis"），value 为检查详情。
	Checks map[string]any `json:"checks"`
}

// healthCheck 健康检查处理函数。
//
// 当前为最简实现，仅返回 ok 状态。
// 生产环境应检查数据库连接、消息队列连接等关键依赖的状态。
//
// 扩展方式：
//
//	checks := map[string]any{}
//	if err := resource.DB.Raw("SELECT 1").Error; err != nil {
//	    checks["database"] = "unhealthy"
//	    return &HealthResponse{State: "degraded", Checks: checks}, nil
//	}
//	checks["database"] = "healthy"
func healthCheck(context.Context, *struct{}) (*HealthResponse, e.ServiceError) {
	return &HealthResponse{
		State:     "ok",
		Message:   "health check completed",
		Timestamp: time.Now().Unix(),
		Checks:    map[string]any{},
	}, nil
}

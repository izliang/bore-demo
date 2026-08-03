package admin

import (
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore-demo/internal/resource"
	"github.com/izliang/bore/config"
	"github.com/izliang/bore/web/amis"
)

// pages 管理后台页面清单（引擎路径 → JSON 文件）。
var pages = []struct{ path, file string }{
	{path: "/login", file: "login.json"},
	{path: "/admin/dashboard", file: "dashboard.json"},
	{path: "/admin/admins", file: "admins.json"},
	{path: "/admin/roles", file: "roles.json"},
	{path: "/admin/menus", file: "menus.json"},
	{path: "/admin/settings", file: "settings.json"},
}

// Setup 装配 amis 管理后台：
//  1. 创建 amis Engine（CDN、i18n、品牌样式）
//  2. 注册登录页与全部管理页面
//  3. 一键接入登录/登出 API（SetupAuth）
//  4. 挂载受保护页面路由（RequireAuth + BindGin）
//
// 返回 API 权限中间件，供业务路由组挂载 RequireLogin。
func Setup(r *gin.Engine) (*Middleware, error) {
	if resource.I18nBundle == nil {
		return nil, fmt.Errorf("admin setup: resource.I18nBundle is nil, initI18n must run first")
	}

	// ── 1. amis Engine ────────────────────────────────────────
	engine := amis.New(
		amis.WithCDN("6.13.0", amis.ThemeCXD),
		amis.WithTitle("bore-demo Admin"),
		amis.WithExtraCSS(extraCSS),
		amis.WithI18n(amis.I18nConfig{
			Bundle:     resource.I18nBundle,
			StrictMode: true,
		}),
	)

	// ── 2. 页面注册 ───────────────────────────────────────────
	rootDir, err := config.GetProjectRootDir()
	if err != nil {
		return nil, fmt.Errorf("admin setup: get project root: %w", err)
	}
	pagesDir := filepath.Join(rootDir, "internal", "api", "admin", "pages")
	for _, p := range pages {
		if err := engine.RegisterJSONFile(p.path, filepath.Join(pagesDir, p.file)); err != nil {
			return nil, fmt.Errorf("admin setup: register page %s: %w", p.path, err)
		}
	}

	// ── 3. 登录鉴权 ───────────────────────────────────────────
	cfg := amis.DefaultAuthConfig()
	cfg.HomePath = "/admin/dashboard"
	msgs := amis.AuthMessagesZH()
	cfg.Messages = &msgs
	engine.SetupAuth(r, cfg, Authenticator)

	// 用自定义登录页覆盖默认 Schema（SetupAuth 之后再注册）。
	if err := engine.RegisterJSONFile("/login", filepath.Join(pagesDir, "login.json")); err != nil {
		return nil, fmt.Errorf("admin setup: register login page: %w", err)
	}

	// ── 4. 受保护页面路由 ─────────────────────────────────────
	protected := r.Group("/admin", cfg.RequireAuth())
	engine.BindGin(protected)

	return NewMiddleware(cfg.CookieName, permSession), nil
}

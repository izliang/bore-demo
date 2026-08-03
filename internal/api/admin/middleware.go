package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	authsvc "github.com/izliang/bore-demo/internal/model/service/auth"
	"github.com/izliang/bore-demo/internal/resource"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/e"
)

// tokenPrefix 会话 token 前缀，完整 token 形如 "admin-123"。
// demo 采用固定格式 token（Cookie 写入），生产建议替换为 Redis 会话/JWT。
const tokenPrefix = "admin-"

// Middleware 提供 API 层的登录态与权限校验能力。
type Middleware struct {
	cookieName string
	session    *Session
	// loader 加载管理员权限信息（缓存未命中时调用），可注入替换便于测试。
	loader func(c *gin.Context, adminID uint64) (*model.AdminPerms, e.ServiceError)
}

// NewMiddleware 创建 API 权限中间件。
func NewMiddleware(cookieName string, session *Session) *Middleware {
	m := &Middleware{
		cookieName: cookieName,
		session:    session,
	}
	m.loader = defaultPermsLoader
	return m
}

// RequireLogin 校验登录态并预热权限缓存。
//
// 通过后将管理员信息写入 gin/request context，供 controller 的
// CtxAdminID / CheckPerm 读取。失败统一返回 401 JSON（amis 原生格式）。
func (m *Middleware) RequireLogin() gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(m.cookieName)
		if err != nil {
			m.abortUnauthorized(c)
			return
		}
		adminID, ok := parseAdminToken(token)
		if !ok {
			m.abortUnauthorized(c)
			return
		}

		perms, ok := m.session.Get(adminID)
		if !ok {
			loaded, svcErr := m.loader(c, adminID)
			if svcErr != nil {
				m.abortUnauthorized(c)
				return
			}
			m.session.Set(adminID, *loaded)
			perms = *loaded
		}

		withAdminInfo(c, &adminInfo{
			AdminID: adminID,
			Roles:   perms.Roles,
			Perms:   perms.Perms,
			IsSuper: perms.IsSuper,
		})
		c.Next()
	}
}

func (m *Middleware) abortUnauthorized(c *gin.Context) {
	c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
		"status": 401,
		"msg":    "未登录或登录已过期",
		"data":   nil,
	})
}

// parseAdminToken 解析 "admin-{id}" 形式的会话 token。
func parseAdminToken(token string) (uint64, bool) {
	if !strings.HasPrefix(token, tokenPrefix) {
		return 0, false
	}
	id, err := strconv.ParseUint(strings.TrimPrefix(token, tokenPrefix), 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	return id, true
}

// defaultPermsLoader 从数据库加载管理员权限（缓存未命中时）。
func defaultPermsLoader(c *gin.Context, adminID uint64) (*model.AdminPerms, e.ServiceError) {
	atx := appctx.New(
		appctx.WithContext(c.Request.Context()),
		appctx.WithLogger(resource.Logger),
		appctx.WithGORM(resource.DB),
	)
	return authsvc.New(
		dao.NewAdminDao(atx),
		dao.NewRoleDao(atx),
		dao.NewMenuDao(atx),
	).GetPerms(adminID)
}

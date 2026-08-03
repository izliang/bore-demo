// Package admin 登录态中间件单元测试。
package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore/e"
)

const testCookieName = "bore-admin-token"

// setupGin 构建一个挂载 RequireLogin 的最小路由，handler 返回管理员 ID。
func setupGin(m *Middleware, t *testing.T) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/protected", m.RequireLogin(), func(c *gin.Context) {
		id, _ := CtxAdminID(c.Request.Context())
		c.JSON(http.StatusOK, gin.H{"admin_id": id})
	})
	return r
}

func doRequest(r *gin.Engine, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/protected", nil)
	for _, c := range cookies {
		if c != nil {
			req.AddCookie(c)
		}
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestRequireLogin(t *testing.T) {
	tests := []struct {
		name       string
		cookie     *http.Cookie
		loader     func(c *gin.Context, adminID uint64) (*model.AdminPerms, e.ServiceError)
		wantStatus int
	}{
		{
			name:       "失败_未携带Cookie",
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:       "失败_token格式非法",
			cookie:     &http.Cookie{Name: testCookieName, Value: "user-1"},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "失败_加载权限失败",
			cookie: &http.Cookie{Name: testCookieName, Value: "admin-1"},
			loader: func(*gin.Context, uint64) (*model.AdminPerms, e.ServiceError) {
				return nil, e.New("A0007", "x").Build()
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name:   "成功_缓存未命中走加载器",
			cookie: &http.Cookie{Name: testCookieName, Value: "admin-1"},
			loader: func(_ *gin.Context, adminID uint64) (*model.AdminPerms, e.ServiceError) {
				return &model.AdminPerms{Roles: []string{"admin"}, Perms: []string{"admin:list"}}, nil
			},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := NewMiddleware(testCookieName, NewSession(time.Minute))
			if tt.loader != nil {
				m.loader = tt.loader
			}
			r := setupGin(m, t)

			w := doRequest(r, tt.cookie)
			if w.Code != tt.wantStatus {
				t.Errorf("期望状态码 %d，实际 %d（body=%s）", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantStatus == http.StatusOK && w.Body.String() != `{"admin_id":1}` {
				t.Errorf("响应异常: %s", w.Body.String())
			}
		})
	}
}

func TestRequireLogin_Cache(t *testing.T) {
	session := NewSession(time.Minute)
	m := NewMiddleware(testCookieName, session)

	loadCount := 0
	m.loader = func(*gin.Context, uint64) (*model.AdminPerms, e.ServiceError) {
		loadCount++
		return &model.AdminPerms{Roles: []string{"admin"}, Perms: []string{"admin:list"}}, nil
	}
	r := setupGin(m, t)
	cookie := &http.Cookie{Name: testCookieName, Value: "admin-1"}

	// 第一次：缓存未命中，加载器调用一次并写入缓存。
	doRequest(r, cookie)
	if loadCount != 1 {
		t.Fatalf("期望加载器调用 1 次，实际 %d", loadCount)
	}
	if _, ok := session.Get(1); !ok {
		t.Fatal("期望权限已写入缓存")
	}

	// 第二次：缓存命中，加载器不再调用。
	doRequest(r, cookie)
	if loadCount != 1 {
		t.Errorf("缓存命中时加载器不应再调用，实际 %d 次", loadCount)
	}
}

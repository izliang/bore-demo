// Package admin 权限上下文与权限点校验单元测试。
package admin

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCtxAdminIDAndIsSuper(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/admin/admins", nil)
	withAdminInfo(c, &adminInfo{AdminID: 7, IsSuper: true, Roles: []string{"super_admin"}})

	id, ok := CtxAdminID(c.Request.Context())
	if !ok || id != 7 {
		t.Errorf("CtxAdminID 异常: id=%d ok=%v", id, ok)
	}
	if !CtxIsSuper(c.Request.Context()) {
		t.Error("期望 IsSuper=true，实际 false")
	}

	if _, ok := CtxAdminID(context.Background()); ok {
		t.Error("空 context 不应返回管理员 ID")
	}
	if CtxIsSuper(context.Background()) {
		t.Error("空 context 不应判定为超管")
	}
}

func TestCheckPerm(t *testing.T) {
	gin.SetMode(gin.TestMode)

	newCtx := func(perms []string, isSuper bool) context.Context {
		c, _ := gin.CreateTestContext(httptest.NewRecorder())
		c.Request = httptest.NewRequest("GET", "/admin/admins", nil)
		withAdminInfo(c, &adminInfo{AdminID: 1, Perms: perms, IsSuper: isSuper})
		return c.Request.Context()
	}

	tests := []struct {
		name     string
		ctx      context.Context
		perms    []string
		wantCode string
		wantErr  bool
	}{
		{
			name:     "失败_未登录",
			ctx:      context.Background(),
			perms:    []string{"admin:list"},
			wantCode: "A0007",
			wantErr:  true,
		},
		{
			name:    "成功_超管放行",
			ctx:     newCtx(nil, true),
			perms:   []string{"admin:list"},
			wantErr: false,
		},
		{
			name:    "成功_命中任一权限",
			ctx:     newCtx([]string{"role:list", "admin:list"}, false),
			perms:   []string{"admin:list"},
			wantErr: false,
		},
		{
			name:     "失败_缺少权限",
			ctx:      newCtx([]string{"role:list"}, false),
			perms:    []string{"admin:list"},
			wantCode: "A0007",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := CheckPerm(tt.ctx, tt.perms...)
			if tt.wantErr {
				if err == nil || err.GetCode() != tt.wantCode {
					t.Fatalf("期望错误码 %s，实际 %v", tt.wantCode, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("期望放行，实际拒绝: %+v", err)
			}
		})
	}
}

func TestParseAdminToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  uint64
		ok    bool
	}{
		{name: "成功_合法token", token: "admin-123", want: 123, ok: true},
		{name: "失败_前缀错误", token: "user-123"},
		{name: "失败_非数字", token: "admin-abc"},
		{name: "失败_空ID", token: "admin-"},
		{name: "失败_ID为0", token: "admin-0"},
		{name: "失败_负数", token: "admin--1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := parseAdminToken(tt.token)
			if ok != tt.ok || (tt.ok && id != tt.want) {
				t.Errorf("parseAdminToken(%q) = (%d, %v)，期望 (%d, %v)", tt.token, id, ok, tt.want, tt.ok)
			}
		})
	}
}

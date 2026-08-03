package admin

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/izliang/bore/e"
)

// 业务错误码（模块 auth，与 admin 模块共用 A 段）。
var (
	// ErrUnauthorized 登录态无效（未登录/管理员不存在或已禁用）。
	ErrUnauthorized = e.New("A0007", "登录状态无效或已过期").Build()
	// ErrForbidden 已登录但缺少所需权限。
	ErrForbidden = e.New("A0007", "无权限执行该操作").Build()
)

// ginCtxKey 是 gin/request context 中管理员信息的键。
// 使用 string 类型键避免与第三方库冲突。
type ctxKey string

const adminInfoKey ctxKey = "rbac:admin:info"

// adminInfo 当前登录管理员的运行时信息。
type adminInfo struct {
	AdminID uint64
	Roles   []string
	Perms   []string
	IsSuper bool
}

// withAdminInfo 将当前登录管理员信息同时写入 gin context 与 request context。
// request context 使 cyberhuma handler（接收 context.Context）也能读取。
func withAdminInfo(c *gin.Context, info *adminInfo) {
	c.Set(string(adminInfoKey), info)
	c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), adminInfoKey, info))
}

// CtxAdminID 从请求 context 读取当前登录管理员 ID。
func CtxAdminID(ctx context.Context) (uint64, bool) {
	info, ok := ctxAdminInfo(ctx)
	if !ok {
		return 0, false
	}
	return info.AdminID, true
}

// CtxIsSuper 判断当前登录管理员是否为超级管理员。
func CtxIsSuper(ctx context.Context) bool {
	info, ok := ctxAdminInfo(ctx)
	return ok && info.IsSuper
}

// CheckPerm 校验当前登录管理员是否拥有 perms 中任意一个权限。
//
// 供 controller 在接口入口调用（权限声明集中在 controller 顶部，便于审计）：
//
//	if err := adminapi.CheckPerm(ctx, "admin:add"); err != nil {
//	    return nil, err
//	}
//
// 超级管理员直接放行。
func CheckPerm(ctx context.Context, perms ...string) e.ServiceError {
	info, ok := ctxAdminInfo(ctx)
	if !ok {
		return ErrUnauthorized
	}
	if info.IsSuper {
		return nil
	}
	for _, want := range perms {
		if containsString(info.Perms, want) {
			return nil
		}
	}
	return ErrForbidden
}

func ctxAdminInfo(ctx context.Context) (*adminInfo, bool) {
	v, ok := ctx.Value(adminInfoKey).(*adminInfo)
	return v, ok
}

// containsString 判断切片是否包含目标字符串。
func containsString(list []string, target string) bool {
	for _, s := range list {
		if s == target {
			return true
		}
	}
	return false
}

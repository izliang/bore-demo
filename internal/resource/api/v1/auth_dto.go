package v1

// ============================================================
// 鉴权信息（Auth）
// ============================================================

// AuthInfoResponse 当前登录管理员信息响应（GET /api/v1/auth/info）。
//
// 供 amis 页面 initApi 使用：roles/perms 注入页面 data 后，
// 导航与按钮通过 visibleOn 表达式控制显隐。
type AuthInfoResponse struct {
	ID       uint64   `json:"id"`
	Username string   `json:"username"`
	Nickname string   `json:"nickname"`
	Avatar   string   `json:"avatar"`
	Roles    []string `json:"roles"`
	Perms    []string `json:"perms"`
}

// Package v1 定义 API v1 的 HTTP 请求/响应 DTO（Data Transfer Object）。
//
// 本文件为 RBAC 管理后台相关 DTO：管理员 CRUD、统计、鉴权信息。
// 约定详见 get_user.go 包注释。
package v1

import "time"

// ============================================================
// 管理员（Admin）
// ============================================================

// ListAdminsRequest 管理员分页列表请求（GET /api/v1/admin/list）。
type ListAdminsRequest struct {
	// Keyword 关键字，模糊匹配用户名/昵称。
	Keyword string `query:"keyword" doc:"关键字（用户名/昵称）"`
	// Page 页码，从 1 开始。
	Page int `query:"page" default:"1" minimum:"1" doc:"页码"`
	// PerPage 每页条数。
	PerPage int `query:"perPage" default:"10" minimum:"1" maximum:"100" doc:"每页条数"`
}

// RoleBrief 管理员列表中的角色摘要。
type RoleBrief struct {
	ID       uint64 `json:"id"`
	RoleName string `json:"role_name"`
	RoleKey  string `json:"role_key"`
}

// AdminItem 管理员响应项。
type AdminItem struct {
	ID          uint64      `json:"id"`
	Username    string      `json:"username"`
	Nickname    string      `json:"nickname"`
	Avatar      string      `json:"avatar"`
	Email       string      `json:"email"`
	Phone       string      `json:"phone"`
	Status      int8        `json:"status"`
	Roles       []RoleBrief `json:"roles"`
	RoleIDs     []uint64    `json:"role_ids"`
	LastLoginAt *time.Time  `json:"last_login_at"`
	CreatedAt   time.Time   `json:"created_at"`
}

// AdminListResponse 管理员分页列表响应（amis 读取 data.items / data.total）。
type AdminListResponse struct {
	Items []*AdminItem `json:"items"`
	Total int64        `json:"total"`
}

// GetAdminRequest 管理员详情请求（GET /api/v1/admin/get）。
type GetAdminRequest struct {
	ID uint64 `query:"id" required:"true" minimum:"1" doc:"管理员 ID"`
}

// CreateAdminRequest 新增管理员请求（POST /api/v1/admin/create）。
type CreateAdminRequest struct {
	Username string   `json:"username" required:"true" minLength:"1" maxLength:"64" doc:"登录用户名"`
	Password string   `json:"password" required:"true" minLength:"6" maxLength:"64" doc:"密码"`
	Nickname string   `json:"nickname" maxLength:"64" doc:"显示名称"`
	Avatar   string   `json:"avatar" maxLength:"255" doc:"头像 URL"`
	Email    string   `json:"email" maxLength:"128" doc:"邮箱"`
	Phone    string   `json:"phone" maxLength:"32" doc:"手机号"`
	Status   int8     `json:"status" default:"1" doc:"状态（1 启用 / 0 禁用）"`
	RoleIDs  []uint64 `json:"role_ids" doc:"角色 ID 列表"`
}

// UpdateAdminRequest 更新管理员请求（PUT /api/v1/admin/update）。
type UpdateAdminRequest struct {
	ID       uint64   `json:"id" required:"true" minimum:"1" doc:"管理员 ID"`
	Username string   `json:"username" required:"true" minLength:"1" maxLength:"64" doc:"登录用户名"`
	Password string   `json:"password" maxLength:"64" doc:"新密码（留空不修改）"`
	Nickname string   `json:"nickname" maxLength:"64" doc:"显示名称"`
	Avatar   string   `json:"avatar" maxLength:"255" doc:"头像 URL"`
	Email    string   `json:"email" maxLength:"128" doc:"邮箱"`
	Phone    string   `json:"phone" maxLength:"32" doc:"手机号"`
	Status   int8     `json:"status" default:"1" doc:"状态（1 启用 / 0 禁用）"`
	RoleIDs  []uint64 `json:"role_ids" doc:"角色 ID 列表"`
}

// DeleteAdminRequest 删除管理员请求（DELETE /api/v1/admin/delete?id=1）。
type DeleteAdminRequest struct {
	ID uint64 `query:"id" required:"true" minimum:"1" doc:"管理员 ID"`
}

// BatchDeleteAdminsRequest 批量删除管理员请求（POST /api/v1/admin/batch-delete）。
type BatchDeleteAdminsRequest struct {
	IDs []uint64 `json:"ids" required:"true" minItems:"1" doc:"管理员 ID 列表"`
}

// AdminStatsResponse 仪表盘统计响应（GET /api/v1/admin/stats）。
type AdminStatsResponse struct {
	Admins int64 `json:"admins" doc:"管理员总数"`
	Roles  int64 `json:"roles" doc:"角色总数"`
	Menus  int64 `json:"menus" doc:"菜单/权限总数"`
}

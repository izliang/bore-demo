package v1

import "time"

// ============================================================
// 角色（Role）
// ============================================================

// ListRolesRequest 角色分页列表请求（GET /api/v1/role/list）。
type ListRolesRequest struct {
	Keyword string `query:"keyword" doc:"关键字（角色名/角色标识）"`
	Page    int    `query:"page" default:"1" minimum:"1" doc:"页码"`
	PerPage int    `query:"perPage" default:"10" minimum:"1" maximum:"100" doc:"每页条数"`
}

// RoleItem 角色响应项。
type RoleItem struct {
	ID        uint64    `json:"id"`
	RoleName  string    `json:"role_name"`
	RoleKey   string    `json:"role_key"`
	RoleSort  int       `json:"role_sort"`
	Status    int8      `json:"status"`
	Remark    string    `json:"remark"`
	MenuIDs   []uint64  `json:"menu_ids,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// RoleListResponse 角色分页列表响应。
type RoleListResponse struct {
	Items []*RoleItem `json:"items"`
	Total int64       `json:"total"`
}

// RoleOption 角色下拉选项（admin 表单 / 权限分配用）。
type RoleOption struct {
	ID       uint64 `json:"id"`
	RoleName string `json:"role_name"`
	RoleKey  string `json:"role_key"`
}

// RoleOptionsResponse 角色下拉选项响应（data 为数组）。
type RoleOptionsResponse []*RoleOption

// GetRoleRequest 角色详情请求（GET /api/v1/role/get）。
type GetRoleRequest struct {
	ID uint64 `query:"id" required:"true" minimum:"1" doc:"角色 ID"`
}

// CreateRoleRequest 新增角色请求（POST /api/v1/role/create）。
type CreateRoleRequest struct {
	RoleName string   `json:"role_name" required:"true" minLength:"1" maxLength:"64" doc:"角色名称"`
	RoleKey  string   `json:"role_key" required:"true" minLength:"1" maxLength:"64" pattern:"^[a-z][a-z0-9_]*$" doc:"角色标识（创建后不可修改）"`
	RoleSort int      `json:"role_sort" default:"0" doc:"显示顺序"`
	Status   int8     `json:"status" default:"1" doc:"状态（1 正常 / 0 停用）"`
	Remark   string   `json:"remark" maxLength:"255" doc:"备注"`
	MenuIDs  []uint64 `json:"menu_ids" doc:"菜单/按钮权限 ID 列表"`
}

// UpdateRoleRequest 更新角色请求（PUT /api/v1/role/update）。
type UpdateRoleRequest struct {
	ID       uint64   `json:"id" required:"true" minimum:"1" doc:"角色 ID"`
	RoleName string   `json:"role_name" required:"true" minLength:"1" maxLength:"64" doc:"角色名称"`
	RoleSort int      `json:"role_sort" default:"0" doc:"显示顺序"`
	Status   int8     `json:"status" default:"1" doc:"状态（1 正常 / 0 停用）"`
	Remark   string   `json:"remark" maxLength:"255" doc:"备注"`
	MenuIDs  []uint64 `json:"menu_ids" doc:"菜单/按钮权限 ID 列表"`
}

// DeleteRoleRequest 删除角色请求（DELETE /api/v1/role/delete?id=1）。
type DeleteRoleRequest struct {
	ID uint64 `query:"id" required:"true" minimum:"1" doc:"角色 ID"`
}

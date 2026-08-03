package model

import "time"

// Role 角色领域实体。
//
// 对应若依 sys_role：角色通过 role_menus 关联菜单/按钮权限。
type Role struct {
	// ID 角色唯一标识。
	ID uint64
	// RoleName 角色显示名称（如 超级管理员）。
	RoleName string
	// RoleKey 角色权限标识（如 super_admin），唯一且不可修改。
	RoleKey string
	// RoleSort 显示顺序，越小越靠前。
	RoleSort int
	// DataScope 数据权限范围，预留（'1' 全部数据）。
	DataScope string
	// Status 角色状态（1 正常 / 0 停用）。
	Status int8
	// Remark 备注。
	Remark string
	// MenuIDs 关联的菜单/按钮权限 ID（分配权限场景使用）。
	MenuIDs []uint64
	// CreatedAt 创建时间。
	CreatedAt time.Time
	// UpdatedAt 更新时间。
	UpdatedAt time.Time
}

// CreateRoleParam 新增角色的领域入参。
type CreateRoleParam struct {
	RoleName string
	RoleKey  string
	RoleSort int
	Status   int8
	Remark   string
	MenuIDs  []uint64
}

// UpdateRoleParam 更新角色的领域入参。
// RoleKey 创建后不可修改（代码与种子数据依赖它）。
type UpdateRoleParam struct {
	ID       uint64
	RoleName string
	RoleSort int
	Status   int8
	Remark   string
	MenuIDs  []uint64
}

// RoleQueryParam 角色分页查询入参。
type RoleQueryParam struct {
	Keyword string
	Page    int
	PerPage int
}

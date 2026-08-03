// Package model 定义领域模型（Domain Model）。
//
// 本文件定义 RBAC 核心实体之一：管理员（Admin）及其关联的角色/权限信息。
// 遵循本仓库约定：领域模型不包含 ORM/JSON 标签，纯业务语义。
package model

import "time"

// AdminStatus 管理员账号状态。
type AdminStatus int8

const (
	// AdminStatusDisabled 账号禁用，禁止登录。
	AdminStatusDisabled AdminStatus = 0
	// AdminStatusEnabled 账号启用。
	AdminStatusEnabled AdminStatus = 1
)

// 内置角色标识（role_key）。
// super_admin 对应若依的超级管理员角色，拥有全部权限。
const (
	RoleKeySuperAdmin = "super_admin"
	RoleKeyAdmin      = "admin"
)

// Admin 管理员领域实体。
//
// 设计要点：
//   - PasswordHash 只存 bcrypt 哈希，任何对外输出必须剔除该字段
//   - Roles 在列表/详情场景由 Service 层填充，登录场景可为空
type Admin struct {
	// ID 管理员唯一标识。
	ID uint64
	// Username 登录用户名（唯一）。
	Username string
	// PasswordHash 密码 bcrypt 哈希，严禁对外输出。
	PasswordHash string
	// Nickname 显示名称。
	Nickname string
	// Avatar 头像 URL。
	Avatar string
	// Email 邮箱。
	Email string
	// Phone 手机号。
	Phone string
	// Status 账号状态（启用/禁用）。
	Status AdminStatus
	// LastLoginAt 最近一次登录时间。
	LastLoginAt *time.Time
	// Roles 关联的角色（由 Service 层填充）。
	Roles []Role
	// CreatedAt 创建时间。
	CreatedAt time.Time
	// UpdatedAt 更新时间。
	UpdatedAt time.Time
}

// AdminPerms 管理员的权限聚合结果，供鉴权与 /auth/info 使用。
type AdminPerms struct {
	// Roles 角色标识集合（role_key）。
	Roles []string
	// Perms 权限标识集合（如 admin:list）。
	Perms []string
	// IsSuper 是否为超级管理员（role_key = super_admin）。
	IsSuper bool
}

// CreateAdminParam 新增管理员的领域入参。
type CreateAdminParam struct {
	Username string
	Password string
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Status   AdminStatus
	RoleIDs  []uint64
}

// UpdateAdminParam 更新管理员的领域入参。
// Password 为空表示不修改密码。
type UpdateAdminParam struct {
	ID       uint64
	Username string
	Password string
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Status   AdminStatus
	RoleIDs  []uint64
}

// AdminQueryParam 管理员分页查询入参。
type AdminQueryParam struct {
	Keyword string
	Page    int
	PerPage int
}

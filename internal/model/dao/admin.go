// Package dao 是数据访问层（Data Access Object Layer）。
//
// 本文件定义 RBAC 的管理员（Admin）实体、管理员-角色关联实体，
// 以及 AdminDao 接口与实现。设计约定见 user.go 包注释。
package dao

import (
	"time"

	"github.com/izliang/bore/appctx"
	"gorm.io/gorm"
)

// ============================================================
// Admin —— 管理员表实体
// ============================================================

// Admin 管理员表实体（对应数据库 admins 表）。
//
// 与若依 sys_user 对应，角色通过 admin_roles 多对多关联。
// PasswordHash 使用 bcrypt 哈希，JSON 序列化时隐藏（json:"-"）。
type Admin struct {
	// ID 主键，自增。
	ID uint64 `gorm:"column:id;primary_key;" json:"id"`
	// Username 登录用户名，唯一。
	Username string `gorm:"column:username;size:64;uniqueIndex;not null" json:"username"`
	// PasswordHash 密码 bcrypt 哈希。
	PasswordHash string `gorm:"column:password_hash;size:255;not null" json:"-"`
	// Nickname 显示名称。
	Nickname string `gorm:"column:nickname;size:64;not null;default:''" json:"nickname"`
	// Avatar 头像 URL。
	Avatar string `gorm:"column:avatar;size:255;not null;default:''" json:"avatar"`
	// Email 邮箱。
	Email string `gorm:"column:email;size:128;not null;default:''" json:"email"`
	// Phone 手机号。
	Phone string `gorm:"column:phone;size:32;not null;default:''" json:"phone"`
	// Status 账号状态（1 启用 / 0 禁用）。
	Status int8 `gorm:"column:status;not null;default:1" json:"status"`
	// LastLoginAt 最近一次登录时间。
	LastLoginAt *time.Time `gorm:"column:last_login_at" json:"last_login_at"`
	// CreatedAt 创建时间。
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// UpdatedAt 更新时间。
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定 GORM 使用的数据库表名。
func (Admin) TableName() string {
	return "admins"
}

// ============================================================
// AdminRole —— 管理员-角色关联实体
// ============================================================

// AdminRole 管理员与角色的多对多关联（对应若依 sys_user_role）。
type AdminRole struct {
	// AdminID 管理员 ID。
	AdminID uint64 `gorm:"column:admin_id;primary_key" json:"admin_id"`
	// RoleID 角色 ID。
	RoleID uint64 `gorm:"column:role_id;primary_key" json:"role_id"`
}

// TableName 指定 GORM 使用的数据库表名。
func (AdminRole) TableName() string {
	return "admin_roles"
}

// ============================================================
// AdminDao —— 管理员数据访问接口
// ============================================================

// AdminDao 定义管理员聚合的所有数据访问能力。
type AdminDao interface {
	// GetAdmin 根据主键查询管理员。
	GetAdmin(id uint64) (*Admin, error)
	// GetAdminByUsername 根据登录用户名查询管理员（登录用）。
	GetAdminByUsername(username string) (*Admin, error)
	// CreateAdmin 新增管理员。
	CreateAdmin(a *Admin) error
	// UpdateAdmin 更新管理员基本信息。
	UpdateAdmin(a *Admin) error
	// DeleteAdmin 删除管理员及其角色关联。
	DeleteAdmin(id uint64) error
	// ListAdmins 关键字分页查询管理员，返回列表与总条数。
	ListAdmins(keyword string, offset, limit int) ([]*Admin, int64, error)
	// CountAdmins 统计管理员总数。
	CountAdmins() (int64, error)
	// TouchLastLogin 更新最近登录时间。
	TouchLastLogin(id uint64, t time.Time) error
	// ReplaceAdminRoles 全量替换管理员的角色关联。
	ReplaceAdminRoles(adminID uint64, roleIDs []uint64) error
	// GetRoleIDsByAdminID 查询管理员关联的角色 ID 列表。
	GetRoleIDsByAdminID(adminID uint64) ([]uint64, error)
	// CountOtherSuperAdmins 统计除 excludeID 外仍启用且绑定 super_admin 角色的管理员数量。
	// 用于"至少保留一个超级管理员"保护。
	CountOtherSuperAdmins(excludeID uint64) (int64, error)
}

// adminDao 是 AdminDao 接口的私有实现。
type adminDao struct {
	atx appctx.AppContext
}

// NewAdminDao 创建 AdminDao 实例。
func NewAdminDao(atx appctx.AppContext) AdminDao {
	return &adminDao{atx: atx}
}

func (d *adminDao) GetAdmin(id uint64) (*Admin, error) {
	a := &Admin{}
	if err := d.atx.GORM().First(a, id).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (d *adminDao) GetAdminByUsername(username string) (*Admin, error) {
	a := &Admin{}
	if err := d.atx.GORM().Where("username = ?", username).First(a).Error; err != nil {
		return nil, err
	}
	return a, nil
}

func (d *adminDao) CreateAdmin(a *Admin) error {
	return d.atx.GORM().Create(a).Error
}

func (d *adminDao) UpdateAdmin(a *Admin) error {
	return d.atx.GORM().Model(a).Select(
		"username", "password_hash", "nickname", "avatar", "email", "phone", "status",
	).Updates(a).Error
}

// DeleteAdmin 在事务中删除管理员及其角色关联，保证数据一致性。
func (d *adminDao) DeleteAdmin(id uint64) error {
	return d.atx.GORM().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_id = ?", id).Delete(&AdminRole{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Admin{}, id).Error
	})
}

// ListAdmins 关键字模糊匹配 username / nickname，按 ID 倒序分页。
func (d *adminDao) ListAdmins(keyword string, offset, limit int) ([]*Admin, int64, error) {
	db := d.atx.GORM().Model(&Admin{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("username LIKE ? OR nickname LIKE ?", like, like)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*Admin
	if err := db.Order("id DESC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *adminDao) CountAdmins() (int64, error) {
	var total int64
	if err := d.atx.GORM().Model(&Admin{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (d *adminDao) TouchLastLogin(id uint64, t time.Time) error {
	return d.atx.GORM().Model(&Admin{}).Where("id = ?", id).
		Update("last_login_at", t).Error
}

// ReplaceAdminRoles 在事务中先删除旧关联，再批量写入新关联。
func (d *adminDao) ReplaceAdminRoles(adminID uint64, roleIDs []uint64) error {
	roleIDs = uniqueUint64(roleIDs)
	return d.atx.GORM().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("admin_id = ?", adminID).Delete(&AdminRole{}).Error; err != nil {
			return err
		}
		if len(roleIDs) == 0 {
			return nil
		}
		rows := make([]AdminRole, 0, len(roleIDs))
		for _, rid := range roleIDs {
			rows = append(rows, AdminRole{AdminID: adminID, RoleID: rid})
		}
		return tx.Create(&rows).Error
	})
}

func (d *adminDao) GetRoleIDsByAdminID(adminID uint64) ([]uint64, error) {
	var ids []uint64
	if err := d.atx.GORM().Model(&AdminRole{}).
		Where("admin_id = ?", adminID).
		Pluck("role_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (d *adminDao) CountOtherSuperAdmins(excludeID uint64) (int64, error) {
	var total int64
	err := d.atx.GORM().Table("admins a").
		Joins("JOIN admin_roles ar ON ar.admin_id = a.id").
		Joins("JOIN roles r ON r.id = ar.role_id").
		Where("r.role_key = ? AND r.status = 1 AND a.status = 1 AND a.id <> ?", securitySuperRoleKey, excludeID).
		Count(&total).Error
	return total, err
}

// uniqueUint64 去重并保持顺序。
func uniqueUint64(in []uint64) []uint64 {
	seen := make(map[uint64]struct{}, len(in))
	out := make([]uint64, 0, len(in))
	for _, v := range in {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

// securitySuperRoleKey 避免与 model 包循环依赖，本地定义超管角色标识。
const securitySuperRoleKey = "super_admin"

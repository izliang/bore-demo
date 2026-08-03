package dao

import (
	"time"

	"github.com/izliang/bore/appctx"
	"gorm.io/gorm"
)

// ============================================================
// Role —— 角色表实体
// ============================================================

// Role 角色表实体（对应数据库 roles 表，若依 sys_role）。
//
// DataScope 数据权限范围预留（'1' 全部数据），本期不实现部门级数据隔离。
type Role struct {
	// ID 主键，自增。
	ID uint64 `gorm:"column:id;primary_key;" json:"id"`
	// RoleName 角色显示名称。
	RoleName string `gorm:"column:role_name;size:64;not null" json:"role_name"`
	// RoleKey 角色权限标识，唯一且不可修改。
	RoleKey string `gorm:"column:role_key;size:64;uniqueIndex;not null" json:"role_key"`
	// RoleSort 显示顺序。
	RoleSort int `gorm:"column:role_sort;not null;default:0" json:"role_sort"`
	// DataScope 数据权限范围（预留，默认 '1' 全部数据）。
	DataScope string `gorm:"column:data_scope;size:1;not null;default:'1'" json:"data_scope"`
	// Status 角色状态（1 正常 / 0 停用）。
	Status int8 `gorm:"column:status;not null;default:1" json:"status"`
	// Remark 备注。
	Remark string `gorm:"column:remark;size:255;not null;default:''" json:"remark"`
	// CreatedAt 创建时间。
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// UpdatedAt 更新时间。
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定 GORM 使用的数据库表名。
func (Role) TableName() string {
	return "roles"
}

// ============================================================
// RoleMenu —— 角色-菜单权限关联实体
// ============================================================

// RoleMenu 角色与菜单/按钮权限的多对多关联（对应若依 sys_role_menu）。
type RoleMenu struct {
	// RoleID 角色 ID。
	RoleID uint64 `gorm:"column:role_id;primary_key" json:"role_id"`
	// MenuID 菜单 ID。
	MenuID uint64 `gorm:"column:menu_id;primary_key" json:"menu_id"`
}

// TableName 指定 GORM 使用的数据库表名。
func (RoleMenu) TableName() string {
	return "role_menus"
}

// ============================================================
// RoleDao —— 角色数据访问接口
// ============================================================

// RoleDao 定义角色聚合的所有数据访问能力。
type RoleDao interface {
	// GetRole 根据主键查询角色。
	GetRole(id uint64) (*Role, error)
	// GetRoleByKey 根据角色标识查询角色。
	GetRoleByKey(key string) (*Role, error)
	// CreateRole 新增角色。
	CreateRole(r *Role) error
	// UpdateRole 更新角色基本信息（不含 role_key）。
	UpdateRole(r *Role) error
	// DeleteRole 删除角色及其菜单权限关联。
	DeleteRole(id uint64) error
	// ListRoles 关键字分页查询角色，返回列表与总条数。
	ListRoles(keyword string, offset, limit int) ([]*Role, int64, error)
	// ListEnabledRoles 查询全部启用角色（下拉选择用）。
	ListEnabledRoles() ([]*Role, error)
	// ReplaceRoleMenus 全量替换角色的菜单权限关联。
	ReplaceRoleMenus(roleID uint64, menuIDs []uint64) error
	// GetMenuIDsByRoleID 查询角色关联的菜单 ID 列表。
	GetMenuIDsByRoleID(roleID uint64) ([]uint64, error)
	// CountAdminRefs 统计绑定该角色的管理员数量。
	CountAdminRefs(roleID uint64) (int64, error)
	// GetRoleKeysByAdminID 查询管理员绑定的启用角色标识集合。
	GetRoleKeysByAdminID(adminID uint64) ([]string, error)
	// CountRoles 统计角色总数。
	CountRoles() (int64, error)
}

// roleDao 是 RoleDao 接口的私有实现。
type roleDao struct {
	atx appctx.AppContext
}

// NewRoleDao 创建 RoleDao 实例。
func NewRoleDao(atx appctx.AppContext) RoleDao {
	return &roleDao{atx: atx}
}

func (d *roleDao) GetRole(id uint64) (*Role, error) {
	r := &Role{}
	if err := d.atx.GORM().First(r, id).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func (d *roleDao) GetRoleByKey(key string) (*Role, error) {
	r := &Role{}
	if err := d.atx.GORM().Where("role_key = ?", key).First(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func (d *roleDao) CreateRole(r *Role) error {
	return d.atx.GORM().Create(r).Error
}

func (d *roleDao) UpdateRole(r *Role) error {
	return d.atx.GORM().Model(r).Select(
		"role_name", "role_sort", "status", "remark",
	).Updates(r).Error
}

// DeleteRole 在事务中删除角色及其菜单权限关联。
func (d *roleDao) DeleteRole(id uint64) error {
	return d.atx.GORM().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", id).Delete(&RoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Role{}, id).Error
	})
}

func (d *roleDao) ListRoles(keyword string, offset, limit int) ([]*Role, int64, error) {
	db := d.atx.GORM().Model(&Role{})
	if keyword != "" {
		db = db.Where("role_name LIKE ? OR role_key LIKE ?", "%"+keyword+"%", "%"+keyword+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []*Role
	if err := db.Order("role_sort ASC, id ASC").Offset(offset).Limit(limit).Find(&list).Error; err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

func (d *roleDao) ListEnabledRoles() ([]*Role, error) {
	var list []*Role
	if err := d.atx.GORM().Where("status = 1").Order("role_sort ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// ReplaceRoleMenus 在事务中先删除旧关联，再批量写入新关联。
func (d *roleDao) ReplaceRoleMenus(roleID uint64, menuIDs []uint64) error {
	menuIDs = uniqueUint64(menuIDs)
	return d.atx.GORM().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("role_id = ?", roleID).Delete(&RoleMenu{}).Error; err != nil {
			return err
		}
		if len(menuIDs) == 0 {
			return nil
		}
		rows := make([]RoleMenu, 0, len(menuIDs))
		for _, mid := range menuIDs {
			rows = append(rows, RoleMenu{RoleID: roleID, MenuID: mid})
		}
		return tx.Create(&rows).Error
	})
}

func (d *roleDao) GetMenuIDsByRoleID(roleID uint64) ([]uint64, error) {
	var ids []uint64
	if err := d.atx.GORM().Model(&RoleMenu{}).
		Where("role_id = ?", roleID).
		Pluck("menu_id", &ids).Error; err != nil {
		return nil, err
	}
	return ids, nil
}

func (d *roleDao) CountAdminRefs(roleID uint64) (int64, error) {
	var total int64
	if err := d.atx.GORM().Model(&AdminRole{}).
		Where("role_id = ?", roleID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (d *roleDao) CountRoles() (int64, error) {
	var total int64
	if err := d.atx.GORM().Model(&Role{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

func (d *roleDao) GetRoleKeysByAdminID(adminID uint64) ([]string, error) {
	var keys []string
	err := d.atx.GORM().Model(&Role{}).
		Joins("JOIN admin_roles ar ON ar.role_id = roles.id").
		Where("ar.admin_id = ? AND roles.status = 1", adminID).
		Distinct().Pluck("roles.role_key", &keys).Error
	return keys, err
}

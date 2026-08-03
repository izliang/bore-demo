package dao

import (
	"time"

	"github.com/izliang/bore/appctx"
	"gorm.io/gorm"
)

// ============================================================
// Menu —— 菜单/按钮权限表实体
// ============================================================

// Menu 菜单/按钮权限表实体（对应数据库 menus 表，若依 sys_menu）。
//
// MenuType：M 目录 / C 菜单 / F 按钮。按钮节点通过 perms 承载接口级权限标识。
type Menu struct {
	// ID 主键，自增。
	ID uint64 `gorm:"column:id;primary_key;" json:"id"`
	// ParentID 父菜单 ID，0 表示顶级。
	ParentID uint64 `gorm:"column:parent_id;not null;default:0;index" json:"parent_id"`
	// MenuName 菜单名称。
	MenuName string `gorm:"column:menu_name;size:64;not null" json:"menu_name"`
	// MenuType 菜单类型（M/C/F）。
	MenuType string `gorm:"column:menu_type;size:1;not null;default:'C'" json:"menu_type"`
	// Path 页面路由路径。
	Path string `gorm:"column:path;size:200;not null;default:''" json:"path"`
	// Perms 权限标识（如 admin:list）。
	Perms string `gorm:"column:perms;size:100;not null;default:''" json:"perms"`
	// Icon 图标类名。
	Icon string `gorm:"column:icon;size:100;not null;default:''" json:"icon"`
	// OrderNum 显示顺序。
	OrderNum int `gorm:"column:order_num;not null;default:0" json:"order_num"`
	// Visible 是否显示（1 显示 / 0 隐藏）。
	Visible int8 `gorm:"column:visible;not null;default:1" json:"visible"`
	// CreatedAt 创建时间。
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	// UpdatedAt 更新时间。
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
}

// TableName 指定 GORM 使用的数据库表名。
func (Menu) TableName() string {
	return "menus"
}

// ============================================================
// MenuDao —— 菜单数据访问接口
// ============================================================

// MenuDao 定义菜单/权限聚合的所有数据访问能力。
type MenuDao interface {
	// GetMenu 根据主键查询菜单。
	GetMenu(id uint64) (*Menu, error)
	// CreateMenu 新增菜单/按钮。
	CreateMenu(m *Menu) error
	// UpdateMenu 更新菜单/按钮。
	UpdateMenu(m *Menu) error
	// DeleteMenu 删除菜单及其角色关联（角色菜单关联由事务保证一致）。
	DeleteMenu(id uint64) error
	// ListMenus 查询全部菜单，按父 ID、排序号排列。
	ListMenus() ([]*Menu, error)
	// ListChildren 查询指定父 ID 的直接子菜单。
	ListChildren(parentID uint64) ([]*Menu, error)
	// CountMenus 统计菜单总数。
	CountMenus() (int64, error)
	// GetPermsByAdminID 聚合管理员（启用角色）拥有的全部权限标识。
	GetPermsByAdminID(adminID uint64) ([]string, error)
	// ListAllPerms 查询全部非空权限标识（超管信息展示用）。
	ListAllPerms() ([]string, error)
}

// menuDao 是 MenuDao 接口的私有实现。
type menuDao struct {
	atx appctx.AppContext
}

// NewMenuDao 创建 MenuDao 实例。
func NewMenuDao(atx appctx.AppContext) MenuDao {
	return &menuDao{atx: atx}
}

func (d *menuDao) GetMenu(id uint64) (*Menu, error) {
	m := &Menu{}
	if err := d.atx.GORM().First(m, id).Error; err != nil {
		return nil, err
	}
	return m, nil
}

func (d *menuDao) CreateMenu(m *Menu) error {
	return d.atx.GORM().Create(m).Error
}

func (d *menuDao) UpdateMenu(m *Menu) error {
	return d.atx.GORM().Model(m).Select(
		"parent_id", "menu_name", "menu_type", "path", "perms", "icon", "order_num", "visible",
	).Updates(m).Error
}

// DeleteMenu 在事务中删除菜单及其角色关联，保证数据一致性。
func (d *menuDao) DeleteMenu(id uint64) error {
	return d.atx.GORM().Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("menu_id = ?", id).Delete(&RoleMenu{}).Error; err != nil {
			return err
		}
		return tx.Delete(&Menu{}, id).Error
	})
}

func (d *menuDao) ListMenus() ([]*Menu, error) {
	var list []*Menu
	if err := d.atx.GORM().Order("parent_id ASC, order_num ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *menuDao) ListChildren(parentID uint64) ([]*Menu, error) {
	var list []*Menu
	if err := d.atx.GORM().Where("parent_id = ?", parentID).
		Order("order_num ASC, id ASC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (d *menuDao) CountMenus() (int64, error) {
	var total int64
	if err := d.atx.GORM().Model(&Menu{}).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetPermsByAdminID 通过 管理员→角色→菜单 三级 JOIN 聚合去重后的权限标识。
// 仅统计启用角色（status = 1）下的权限，禁用角色的权限不生效。
func (d *menuDao) ListAllPerms() ([]string, error) {
	var perms []string
	if err := d.atx.GORM().Model(&Menu{}).
		Where("perms <> ''").
		Distinct().Pluck("perms", &perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (d *menuDao) GetPermsByAdminID(adminID uint64) ([]string, error) {
	var perms []string
	err := d.atx.GORM().Model(&Menu{}).
		Joins("JOIN role_menus rm ON rm.menu_id = menus.id").
		Joins("JOIN roles r ON r.id = rm.role_id AND r.status = 1").
		Joins("JOIN admin_roles ar ON ar.role_id = r.id").
		Where("ar.admin_id = ? AND menus.perms <> ''", adminID).
		Distinct().Pluck("menus.perms", &perms).Error
	return perms, err
}

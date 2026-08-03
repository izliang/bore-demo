package model

import "time"

// 菜单类型（对应若依 sys_menu.menu_type）。
const (
	// MenuTypeDir 目录（顶级分组，无实际页面）。
	MenuTypeDir = "M"
	// MenuTypeMenu 菜单（实际页面）。
	MenuTypeMenu = "C"
	// MenuTypeButton 按钮（页面内的操作权限）。
	MenuTypeButton = "F"
)

// Menu 菜单/按钮权限领域实体。
//
// 对应若依 sys_menu：通过 parent_id 组成树，perms 为权限标识
// （如 admin:list），按钮节点（F）承载具体的接口级权限。
type Menu struct {
	// ID 菜单唯一标识。
	ID uint64
	// ParentID 父菜单 ID，0 表示顶级。
	ParentID uint64
	// MenuName 菜单名称。
	MenuName string
	// MenuType 菜单类型（M 目录 / C 菜单 / F 按钮）。
	MenuType string
	// Path 页面路由路径（目录/菜单使用）。
	Path string
	// Perms 权限标识（如 admin:list）。
	Perms string
	// Icon 图标类名。
	Icon string
	// OrderNum 显示顺序。
	OrderNum int
	// Visible 是否显示（1 显示 / 0 隐藏）。
	Visible int8
	// Children 子节点（树形结构，由 Service 层组装）。
	Children []*Menu
	// CreatedAt 创建时间。
	CreatedAt time.Time
	// UpdatedAt 更新时间。
	UpdatedAt time.Time
}

// CreateMenuParam 新增菜单/按钮的领域入参。
type CreateMenuParam struct {
	ParentID uint64
	MenuName string
	MenuType string
	Path     string
	Perms    string
	Icon     string
	OrderNum int
	Visible  int8
}

// UpdateMenuParam 更新菜单/按钮的领域入参。
type UpdateMenuParam struct {
	ID       uint64
	ParentID uint64
	MenuName string
	MenuType string
	Path     string
	Perms    string
	Icon     string
	OrderNum int
	Visible  int8
}

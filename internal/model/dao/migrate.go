package dao

import (
	"fmt"

	"github.com/izliang/bore-demo/internal/security"
	"github.com/izliang/bore/appctx"
	"gorm.io/gorm"
)

// ============================================================
// Migrate —— 数据库表结构迁移
// ============================================================

// Migrate 自动创建/更新 RBAC 全部数据表。
//
// 幂等：AutoMigrate 只会新增缺失的列/索引，不会删除数据。
// 接入点：bootstrap.initAdminDB（initDB 之后调用）。
func Migrate(atx appctx.AppContext) error {
	if err := atx.GORM().AutoMigrate(&Admin{}, &Role{}, &Menu{}, &AdminRole{}, &RoleMenu{}); err != nil {
		return fmt.Errorf("migrate rbac tables: %w", err)
	}
	return nil
}

// ============================================================
// Seed —— 种子数据
// ============================================================

// Seed 幂等初始化 RBAC 种子数据：
//  1. 角色：super_admin（超级管理员）、admin（普通管理员）
//  2. 菜单：仪表盘、管理员/角色/菜单管理（含按钮权限）、系统设置
//  3. 默认管理员：admin / admin123，绑定 super_admin 角色
//
// 仅在对应表为空时写入，重复启动不会产生脏数据。
func Seed(atx appctx.AppContext) error {
	g := atx.GORM()

	// ── 1. 角色种子 ──────────────────────────────────────────
	var roleCount int64
	if err := g.Model(&Role{}).Count(&roleCount).Error; err != nil {
		return fmt.Errorf("count roles: %w", err)
	}
	if roleCount == 0 {
		roles := []Role{
			{RoleName: "超级管理员", RoleKey: "super_admin", RoleSort: 1, Status: 1, DataScope: "1", Remark: "拥有全部权限"},
			{RoleName: "普通管理员", RoleKey: "admin", RoleSort: 2, Status: 1, DataScope: "1", Remark: "基础权限"},
		}
		if err := g.Create(&roles).Error; err != nil {
			return fmt.Errorf("seed roles: %w", err)
		}
	}

	// ── 2. 菜单种子 ──────────────────────────────────────────
	var menuCount int64
	if err := g.Model(&Menu{}).Count(&menuCount).Error; err != nil {
		return fmt.Errorf("count menus: %w", err)
	}
	if menuCount == 0 {
		menuIDs, err := seedMenus(g)
		if err != nil {
			return err
		}
		// 超级管理员绑定全部菜单权限（保证 auth/info 可展示全量权限）。
		super, err := getRoleByKey(g, "super_admin")
		if err != nil {
			return fmt.Errorf("get super_admin role: %w", err)
		}
		if err := bindRoleMenus(g, super.ID, menuIDs); err != nil {
			return err
		}
	}

	// ── 3. 默认管理员种子 ────────────────────────────────────
	var adminCount int64
	if err := g.Model(&Admin{}).Count(&adminCount).Error; err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if adminCount == 0 {
		hash, err := security.HashPassword("admin123")
		if err != nil {
			return fmt.Errorf("hash default admin password: %w", err)
		}
		super, err := getRoleByKey(g, "super_admin")
		if err != nil {
			return fmt.Errorf("get super_admin role: %w", err)
		}
		admin := Admin{
			Username:     "admin",
			PasswordHash: hash,
			Nickname:     "系统管理员",
			Status:       1,
		}
		if err := g.Create(&admin).Error; err != nil {
			return fmt.Errorf("seed default admin: %w", err)
		}
		if err := g.Create(&AdminRole{AdminID: admin.ID, RoleID: super.ID}).Error; err != nil {
			return fmt.Errorf("bind default admin role: %w", err)
		}
	}
	return nil
}

// menuSeedNode 菜单种子节点（parent 为父节点在当前列表中的索引，-1 表示顶级）。
type menuSeedNode struct {
	parent int
	menu   Menu
}

// seedMenus 创建默认菜单树，返回全部菜单 ID（顺序与定义一致）。
func seedMenus(g *gorm.DB) ([]uint64, error) {
	nodes := []menuSeedNode{
		{parent: -1, menu: Menu{MenuName: "仪表盘", MenuType: "C", Path: "/admin/dashboard", Perms: "dashboard:view", Icon: "fa fa-dashboard", OrderNum: 1}},
		{parent: -1, menu: Menu{MenuName: "系统管理", MenuType: "M", Path: "/admin", Icon: "fa fa-gear", OrderNum: 2}},
		{parent: 1, menu: Menu{MenuName: "管理员管理", MenuType: "C", Path: "/admin/admins", Perms: "admin:list", Icon: "fa fa-users", OrderNum: 1}},
		{parent: 2, menu: Menu{MenuName: "管理员查询", MenuType: "F", Perms: "admin:query", OrderNum: 1}},
		{parent: 2, menu: Menu{MenuName: "管理员新增", MenuType: "F", Perms: "admin:add", OrderNum: 2}},
		{parent: 2, menu: Menu{MenuName: "管理员修改", MenuType: "F", Perms: "admin:edit", OrderNum: 3}},
		{parent: 2, menu: Menu{MenuName: "管理员删除", MenuType: "F", Perms: "admin:remove", OrderNum: 4}},
		{parent: 1, menu: Menu{MenuName: "角色管理", MenuType: "C", Path: "/admin/roles", Perms: "role:list", Icon: "fa fa-user-circle", OrderNum: 2}},
		{parent: 7, menu: Menu{MenuName: "角色查询", MenuType: "F", Perms: "role:query", OrderNum: 1}},
		{parent: 7, menu: Menu{MenuName: "角色新增", MenuType: "F", Perms: "role:add", OrderNum: 2}},
		{parent: 7, menu: Menu{MenuName: "角色修改", MenuType: "F", Perms: "role:edit", OrderNum: 3}},
		{parent: 7, menu: Menu{MenuName: "角色删除", MenuType: "F", Perms: "role:remove", OrderNum: 4}},
		{parent: 7, menu: Menu{MenuName: "分配权限", MenuType: "F", Perms: "role:assign", OrderNum: 5}},
		{parent: 1, menu: Menu{MenuName: "菜单管理", MenuType: "C", Path: "/admin/menus", Perms: "menu:list", Icon: "fa fa-list", OrderNum: 3}},
		{parent: 13, menu: Menu{MenuName: "菜单查询", MenuType: "F", Perms: "menu:query", OrderNum: 1}},
		{parent: 13, menu: Menu{MenuName: "菜单新增", MenuType: "F", Perms: "menu:add", OrderNum: 2}},
		{parent: 13, menu: Menu{MenuName: "菜单修改", MenuType: "F", Perms: "menu:edit", OrderNum: 3}},
		{parent: 13, menu: Menu{MenuName: "菜单删除", MenuType: "F", Perms: "menu:remove", OrderNum: 4}},
		{parent: -1, menu: Menu{MenuName: "系统设置", MenuType: "C", Path: "/admin/settings", Perms: "settings:view", Icon: "fa fa-cog", OrderNum: 3}},
	}

	ids := make([]uint64, 0, len(nodes))
	for _, node := range nodes {
		m := node.menu
		if node.parent >= 0 {
			m.ParentID = ids[node.parent]
		}
		if err := g.Create(&m).Error; err != nil {
			return nil, fmt.Errorf("seed menu %q: %w", m.MenuName, err)
		}
		ids = append(ids, m.ID)
	}
	return ids, nil
}

func getRoleByKey(g *gorm.DB, key string) (*Role, error) {
	r := &Role{}
	if err := g.Where("role_key = ?", key).First(r).Error; err != nil {
		return nil, err
	}
	return r, nil
}

func bindRoleMenus(g *gorm.DB, roleID uint64, menuIDs []uint64) error {
	rows := make([]RoleMenu, 0, len(menuIDs))
	for _, mid := range menuIDs {
		rows = append(rows, RoleMenu{RoleID: roleID, MenuID: mid})
	}
	return g.Create(&rows).Error
}

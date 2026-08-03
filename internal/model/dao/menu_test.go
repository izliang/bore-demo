// Package dao 菜单 DAO 层单元测试。
//
// 测试策略：使用 sqlmock 模拟 MySQL，验证 GORM SQL 生成、事务与权限聚合。
// 风格与 admin_test.go 保持一致（table-driven + 正则匹配 SQL）。
package dao

import (
	"database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/tu"
)

// menuColumns 是 menus 表的查询列，与实体字段一一对应。
var menuColumns = []string{
	"id", "parent_id", "menu_name", "menu_type", "path",
	"perms", "icon", "order_num", "visible", "created_at", "updated_at",
}

func menuRow(id uint64, now time.Time) []driver.Value {
	return []driver.Value{id, 0, "系统管理", "M", "/system", "system:list", "icon", 1, 1, now, now}
}

func Test_GetMenu(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	now := time.Now()
	sqlMock.ExpectQuery("^SELECT \\* FROM `menus` WHERE `menus`.`id` = \\? ORDER BY `menus`.`id` LIMIT \\?$").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows(menuColumns).AddRow(menuRow(1, now)...))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	got, err := NewMenuDao(atx).GetMenu(1)
	tu.AssertWantError(t, err, false)
	if got == nil || got.ID != 1 || got.Perms != "system:list" {
		t.Errorf("GetMenu 结果异常: %+v", got)
	}
}

func Test_CreateMenu(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^INSERT INTO `menus`").
		WithArgs(0, "系统管理", "M", "/system", "system:list", "icon", 1, 1, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	m := &Menu{MenuName: "系统管理", MenuType: "M", Path: "/system", Perms: "system:list", Icon: "icon", OrderNum: 1, Visible: 1}
	if err := NewMenuDao(atx).CreateMenu(m); err != nil {
		t.Fatalf("CreateMenu 失败: %v", err)
	}
}

func Test_UpdateMenu(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^UPDATE `menus` SET `parent_id`=\\?,`menu_name`=\\?,`menu_type`=\\?,`path`=\\?,`perms`=\\?,`icon`=\\?,`order_num`=\\?,`visible`=\\?,`updated_at`=\\? WHERE `id` = \\?$").
		WithArgs(0, "用户管理", "C", "/user", "user:list", "", 2, 1, sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	m := &Menu{ID: 1, MenuName: "用户管理", MenuType: "C", Path: "/user", Perms: "user:list", OrderNum: 2, Visible: 1}
	if err := NewMenuDao(atx).UpdateMenu(m); err != nil {
		t.Fatalf("UpdateMenu 失败: %v", err)
	}
}

func Test_DeleteMenu(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^DELETE FROM `role_menus` WHERE menu_id = \\?$").
		WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 3))
	sqlMock.ExpectExec("^DELETE FROM `menus` WHERE `menus`.`id` = \\?$").
		WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	if err := NewMenuDao(atx).DeleteMenu(1); err != nil {
		t.Fatalf("DeleteMenu 失败: %v", err)
	}
}

func Test_ListMenusAndChildren(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	now := time.Now()
	sqlMock.ExpectQuery("^SELECT \\* FROM `menus` ORDER BY parent_id ASC, order_num ASC, id ASC$").
		WillReturnRows(sqlmock.NewRows(menuColumns).AddRow(menuRow(1, now)...))
	sqlMock.ExpectQuery("^SELECT \\* FROM `menus` WHERE parent_id = \\? ORDER BY order_num ASC, id ASC$").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows(menuColumns).AddRow(menuRow(2, now)...))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	menuDao := NewMenuDao(atx)
	all, err := menuDao.ListMenus()
	tu.AssertWantError(t, err, false)
	if len(all) != 1 {
		t.Errorf("ListMenus 结果异常: %+v", all)
	}
	children, err := menuDao.ListChildren(1)
	tu.AssertWantError(t, err, false)
	if len(children) != 1 || children[0].ID != 2 {
		t.Errorf("ListChildren 结果异常: %+v", children)
	}
}

func Test_CountMenus(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectQuery("^SELECT count\\(\\*\\) FROM `menus`$").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(18))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	total, err := NewMenuDao(atx).CountMenus()
	tu.AssertWantError(t, err, false)
	if total != 18 {
		t.Errorf("CountMenus 结果异常: %d", total)
	}
}

func Test_ListAllPerms(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectQuery("^SELECT DISTINCT `perms` FROM `menus` WHERE perms <> ''$").
		WillReturnRows(sqlmock.NewRows([]string{"perms"}).AddRow("admin:list").AddRow("role:list"))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	perms, err := NewMenuDao(atx).ListAllPerms()
	tu.AssertWantError(t, err, false)
	if len(perms) != 2 || perms[0] != "admin:list" {
		t.Errorf("ListAllPerms 结果异常: %v", perms)
	}
}

func Test_GetPermsByAdminID(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectQuery("^SELECT DISTINCT `menus`.`perms` FROM `menus` JOIN role_menus rm ON rm.menu_id = menus.id JOIN roles r ON r.id = rm.role_id AND r.status = 1 JOIN admin_roles ar ON ar.role_id = r.id WHERE ar.admin_id = \\? AND menus.perms <> ''$").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"perms"}).AddRow("user:list"))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	perms, err := NewMenuDao(atx).GetPermsByAdminID(1)
	tu.AssertWantError(t, err, false)
	if len(perms) != 1 || perms[0] != "user:list" {
		t.Errorf("GetPermsByAdminID 结果异常: %v", perms)
	}
}

// Package dao 角色 DAO 层单元测试。
//
// 测试策略：使用 sqlmock 模拟 MySQL，验证 GORM SQL 生成、事务与数据映射。
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

// roleColumns 是 roles 表的查询列，与实体字段一一对应。
var roleColumns = []string{
	"id", "role_name", "role_key", "role_sort", "data_scope",
	"status", "remark", "created_at", "updated_at",
}

func roleRow(id uint64, key string, now time.Time) []driver.Value {
	return []driver.Value{id, "角色" + key, key, 1, "1", 1, "", now, now}
}

func Test_GetRole(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	now := time.Now()
	sqlMock.ExpectQuery("^SELECT \\* FROM `roles` WHERE `roles`.`id` = \\? ORDER BY `roles`.`id` LIMIT \\?$").
		WithArgs(1, 1).
		WillReturnRows(sqlmock.NewRows(roleColumns).AddRow(roleRow(1, "ops", now)...))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	got, err := NewRoleDao(atx).GetRole(1)
	tu.AssertWantError(t, err, false)
	if got == nil || got.ID != 1 || got.RoleKey != "ops" {
		t.Errorf("GetRole 结果异常: %+v", got)
	}
}

func Test_GetRoleByKey(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	now := time.Now()
	sqlMock.ExpectQuery("^SELECT \\* FROM `roles` WHERE role_key = \\? ORDER BY `roles`.`id` LIMIT \\?$").
		WithArgs("ops", 1).
		WillReturnRows(sqlmock.NewRows(roleColumns).AddRow(roleRow(1, "ops", now)...))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	got, err := NewRoleDao(atx).GetRoleByKey("ops")
	tu.AssertWantError(t, err, false)
	if got == nil || got.RoleKey != "ops" {
		t.Errorf("GetRoleByKey 结果异常: %+v", got)
	}
}

func Test_CreateRole(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^INSERT INTO `roles`").
		WithArgs("运维", "ops", 1, "1", 1, "测试", sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	r := &Role{RoleName: "运维", RoleKey: "ops", RoleSort: 1, DataScope: "1", Status: 1, Remark: "测试"}
	if err := NewRoleDao(atx).CreateRole(r); err != nil {
		t.Fatalf("CreateRole 失败: %v", err)
	}
}

func Test_UpdateRole(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^UPDATE `roles` SET `role_name`=\\?,`role_sort`=\\?,`status`=\\?,`remark`=\\?,`updated_at`=\\? WHERE `id` = \\?$").
		WithArgs("运维", 2, 0, "备注", sqlmock.AnyArg(), 1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	r := &Role{ID: 1, RoleName: "运维", RoleKey: "ops", RoleSort: 2, Status: 0, Remark: "备注"}
	if err := NewRoleDao(atx).UpdateRole(r); err != nil {
		t.Fatalf("UpdateRole 失败: %v", err)
	}
}

func Test_DeleteRole(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^DELETE FROM `role_menus` WHERE role_id = \\?$").
		WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 2))
	sqlMock.ExpectExec("^DELETE FROM `roles` WHERE `roles`.`id` = \\?$").
		WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	if err := NewRoleDao(atx).DeleteRole(1); err != nil {
		t.Fatalf("DeleteRole 失败: %v", err)
	}
}

func Test_ListRoles(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	now := time.Now()
	sqlMock.ExpectQuery("^SELECT count\\(\\*\\) FROM `roles` WHERE role_name LIKE \\? OR role_key LIKE \\?$").
		WithArgs("%ops%", "%ops%").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
	sqlMock.ExpectQuery("^SELECT \\* FROM `roles` WHERE role_name LIKE \\? OR role_key LIKE \\? ORDER BY role_sort ASC, id ASC LIMIT \\? OFFSET \\?$").
		WithArgs("%ops%", "%ops%", 10, 10).
		WillReturnRows(sqlmock.NewRows(roleColumns).AddRow(roleRow(1, "ops", now)...))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	list, total, err := NewRoleDao(atx).ListRoles("ops", 10, 10)
	tu.AssertWantError(t, err, false)
	if total != 1 || len(list) != 1 || list[0].RoleKey != "ops" {
		t.Errorf("ListRoles 结果异常: total=%d list=%+v", total, list)
	}
}

func Test_ListEnabledRoles(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	now := time.Now()
	sqlMock.ExpectQuery("^SELECT \\* FROM `roles` WHERE status = 1 ORDER BY role_sort ASC, id ASC$").
		WillReturnRows(sqlmock.NewRows(roleColumns).AddRow(roleRow(1, "ops", now)...))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	list, err := NewRoleDao(atx).ListEnabledRoles()
	tu.AssertWantError(t, err, false)
	if len(list) != 1 {
		t.Errorf("ListEnabledRoles 结果异常: %+v", list)
	}
}

func Test_ReplaceRoleMenus(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^DELETE FROM `role_menus` WHERE role_id = \\?$").
		WithArgs(1).WillReturnResult(sqlmock.NewResult(0, 2))
	sqlMock.ExpectExec("^INSERT INTO `role_menus` \\(`role_id`,`menu_id`\\) VALUES \\(\\?,\\?\\),\\(\\?,\\?\\)$").
		WithArgs(1, 2, 1, 3).
		WillReturnResult(sqlmock.NewResult(0, 2))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	if err := NewRoleDao(atx).ReplaceRoleMenus(1, []uint64{2, 3, 2}); err != nil {
		t.Fatalf("ReplaceRoleMenus 失败: %v", err)
	}
}

func Test_GetMenuIDsByRoleID(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectQuery("^SELECT `menu_id` FROM `role_menus` WHERE role_id = \\?$").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"menu_id"}).AddRow(2).AddRow(3))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	ids, err := NewRoleDao(atx).GetMenuIDsByRoleID(1)
	tu.AssertWantError(t, err, false)
	if len(ids) != 2 || ids[0] != 2 || ids[1] != 3 {
		t.Errorf("GetMenuIDsByRoleID 结果异常: %v", ids)
	}
}

func Test_CountAdminRefsAndRoles(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectQuery("^SELECT count\\(\\*\\) FROM `admin_roles` WHERE role_id = \\?$").
		WithArgs(1).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	sqlMock.ExpectQuery("^SELECT count\\(\\*\\) FROM `roles`$").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(3))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	roleDao := NewRoleDao(atx)
	refs, err := roleDao.CountAdminRefs(1)
	tu.AssertWantError(t, err, false)
	if refs != 2 {
		t.Errorf("CountAdminRefs 结果异常: %d", refs)
	}
	total, err := roleDao.CountRoles()
	tu.AssertWantError(t, err, false)
	if total != 3 {
		t.Errorf("CountRoles 结果异常: %d", total)
	}
}

func Test_GetRoleKeysByAdminID(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectQuery("^SELECT DISTINCT `roles`.`role_key` FROM `roles` JOIN admin_roles ar ON ar.role_id = roles.id WHERE ar.admin_id = \\? AND roles.status = 1$").
		WithArgs(1).
		WillReturnRows(sqlmock.NewRows([]string{"role_key"}).AddRow("super_admin"))

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	keys, err := NewRoleDao(atx).GetRoleKeysByAdminID(1)
	tu.AssertWantError(t, err, false)
	if len(keys) != 1 || keys[0] != "super_admin" {
		t.Errorf("GetRoleKeysByAdminID 结果异常: %v", keys)
	}
}

// Package dao 管理员 DAO 层单元测试。
//
// 测试策略：使用 sqlmock 模拟 MySQL，验证 GORM SQL 生成与数据映射。
// 风格与 user_test.go 保持一致（table-driven + 正则匹配 SQL）。
package dao

import (
	"database/sql/driver"
	"reflect"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/tu"
)

// adminColumns 是 admins 表的查询列，与实体字段一一对应。
var adminColumns = []string{
	"id", "username", "password_hash", "nickname", "avatar",
	"email", "phone", "status", "last_login_at", "created_at", "updated_at",
}

func adminRow(now time.Time) []driver.Value {
	return []driver.Value{
		1, "admin", "hash", "系统管理员", "", "", "", 1, nil, now, now,
	}
}

func Test_GetAdminByUsername(t *testing.T) {
	cases := []struct {
		name     string
		username string
		rows     *sqlmock.Rows
		want     *Admin
		wantErr  bool
	}{
		{
			name:     "正常查询_返回管理员",
			username: "admin",
			rows: func() *sqlmock.Rows {
				r := sqlmock.NewRows(adminColumns)
				r.AddRow(adminRow(time.Now())...)
				return r
			}(),
			want: &Admin{
				ID:           1,
				Username:     "admin",
				PasswordHash: "hash",
				Nickname:     "系统管理员",
				Status:       1,
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gormDB, sqlMock, err := appctx.NewSQLMock()
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %+v", err)
			}
			defer func() { _ = appctx.CloseSQLMock(gormDB) }()

			sqlMock.ExpectQuery("^SELECT \\* FROM `admins` WHERE username = \\? ORDER BY `admins`.`id` LIMIT \\?$").
				WithArgs(c.username, 1).
				WillReturnRows(c.rows)

			atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
			got, err := NewAdminDao(atx).GetAdminByUsername(c.username)

			tu.AssertWantError(t, err, c.wantErr)
			if got != nil {
				got.CreatedAt = time.Time{}
				got.UpdatedAt = time.Time{}
			}
			if !reflect.DeepEqual(got, c.want) {
				t.Errorf("\n期望: %+v\n实际: %+v", c.want, got)
			}
		})
	}
}

func Test_ListAdmins(t *testing.T) {
	cases := []struct {
		name    string
		keyword string
		offset  int
		limit   int
		mock    func(mock sqlmock.Sqlmock)
		wantN   int64
		wantLen int
	}{
		{
			name:    "关键字分页查询",
			keyword: "admin",
			offset:  10,
			limit:   10,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT count\\(\\*\\) FROM `admins` WHERE username LIKE \\? OR nickname LIKE \\?$").
					WithArgs("%admin%", "%admin%").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

				rows := sqlmock.NewRows(adminColumns)
				rows.AddRow(adminRow(time.Now())...)
				mock.ExpectQuery("^SELECT \\* FROM `admins` WHERE username LIKE \\? OR nickname LIKE \\? ORDER BY id DESC LIMIT \\? OFFSET \\?$").
					WithArgs("%admin%", "%admin%", 10, 10).
					WillReturnRows(rows)
			},
			wantN:   1,
			wantLen: 1,
		},
		{
			name:   "无关键字全量分页",
			offset: 10,
			limit:  10,
			mock: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery("^SELECT count\\(\\*\\) FROM `admins`$").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery("^SELECT \\* FROM `admins` ORDER BY id DESC LIMIT \\? OFFSET \\?$").
					WithArgs(10, 10).
					WillReturnRows(sqlmock.NewRows(adminColumns))
			},
			wantN:   0,
			wantLen: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			gormDB, sqlMock, err := appctx.NewSQLMock()
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %+v", err)
			}
			defer func() { _ = appctx.CloseSQLMock(gormDB) }()

			c.mock(sqlMock)
			atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
			list, total, err := NewAdminDao(atx).ListAdmins(c.keyword, c.offset, c.limit)

			tu.AssertWantError(t, err, false)
			for _, a := range list {
				a.CreatedAt = time.Time{}
				a.UpdatedAt = time.Time{}
			}
			if total != c.wantN || len(list) != c.wantLen {
				t.Errorf("\n期望 total=%d len=%d\n实际 total=%d len=%d", c.wantN, c.wantLen, total, len(list))
			}
		})
	}
}

func Test_ReplaceAdminRoles(t *testing.T) {
	gormDB, sqlMock, err := appctx.NewSQLMock()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %+v", err)
	}
	defer func() { _ = appctx.CloseSQLMock(gormDB) }()

	sqlMock.ExpectBegin()
	sqlMock.ExpectExec("^DELETE FROM `admin_roles` WHERE admin_id = \\?$").
		WithArgs(1).
		WillReturnResult(sqlmock.NewResult(0, 1))
	sqlMock.ExpectExec("^INSERT INTO `admin_roles` \\(`admin_id`,`role_id`\\) VALUES \\(\\?,\\?\\),\\(\\?,\\?\\)$").
		WithArgs(1, 2, 1, 3).
		WillReturnResult(sqlmock.NewResult(0, 2))
	sqlMock.ExpectCommit()

	atx := appctx.NewMock(appctx.WithMockGORM(gormDB))
	if err := NewAdminDao(atx).ReplaceAdminRoles(1, []uint64{2, 3, 2}); err != nil {
		t.Fatalf("ReplaceAdminRoles 失败: %v", err)
	}
}

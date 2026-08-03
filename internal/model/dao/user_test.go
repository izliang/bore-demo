// Package dao 用户 DAO 层单元测试。
//
// 测试策略：
//   - 使用 sqlmock 模拟 MySQL 数据库，不依赖真实数据库连接
//   - 使用 appctx.NewSQLMock() 创建 mock GORM 实例
//   - 使用 appctx.NewMock() 创建 mock AppContext
//   - 使用 table-driven tests 风格，便于扩展
//
// appctx.NewSQLMock() 的工作机制：
//  1. 创建一个 sqlmock.Sqlmock 实例
//  2. 通过 sqlmock 打开一个 mock *sql.DB
//  3. 用 mock DB 创建 GORM 实例（gorm.Open）
//  4. 返回 GORM 实例和 sqlmock 供测试使用
//
// 新项目复刻指南（编写 DAO 测试的标准模板）：
//  1. 定义测试用例结构体（name、输入、期望输出、mock SQL）
//  2. 在 mock 函数中设置期望的 SQL 语句和返回的 mock 行
//  3. 创建 mock GORM 和 AppContext
//  4. 通过构造函数创建 DAO 实例
//  5. 调用被测方法并断言结果
//  6. 用 defer 关闭 sqlmock（自动验证所有期望的 SQL 是否被调用）
package dao

import (
	"reflect"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/tu"
)

// Test_GetUser 测试 DAO 的 GetUser 方法。
//
// 测试内容：
//   - 验证 GORM SQL 生成与预期一致
//   - 验证 JSON 列的序列化/反序列化
//   - 验证返回值与预期一致
func Test_GetUser(t *testing.T) {
	// table-driven tests 用例定义。
	cases := []struct {
		// name 用例名称。
		name string

		// id 查询参数。
		id uint64

		// u 期望的返回值。
		u *User

		// wantErr 是否期望返回错误。
		wantErr bool

		// mock 设置 sqlmock 期望的 SQL 和返回值。
		// 参数 mock 是 sqlmock.Sqlmock 实例。
		mock func(mock sqlmock.Sqlmock)
	}{
		{
			name:    "正常查询_返回用户记录",
			id:      1,
			wantErr: false,
			u: &User{
				ID:   1,
				Name: "test",
				Meta: UserMeta{
					AvatarURL: "https://www.aaa.com/pic.png",
				},
			},
			// 设置 mock SQL 期望：
			//   - SQL 语句使用正则匹配，确保 GORM 生成的 SQL 正确
			//   - NewRows 定义返回的列和行数据
			//   - meta 列是 JSON 字符串，测试 GORM 的 JSON 反序列化
			mock: func(mock sqlmock.Sqlmock) {
				rows := sqlmock.NewRows([]string{"id", "name", "meta"}).
					AddRow(1, "test", `{"avatar_url":"https://www.aaa.com/pic.png"}`)
				mock.ExpectQuery("^SELECT \\* FROM `user` WHERE `user`.`id` = \\? ORDER BY `user`.`id` LIMIT \\?$").
					WithArgs(1, 1).
					WillReturnRows(rows)
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// ============================================================
			// 1. 创建 sqlmock 和 mock GORM 实例。
			// ============================================================
			gormDB, sqlMock, err := appctx.NewSQLMock()
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %+v", err)
			}

			// 确保测试结束时关闭 mock 连接，同时验证所有期望的 SQL 都被调用。
			defer func() {
				_ = appctx.CloseSQLMock(gormDB)
			}()

			// ============================================================
			// 2. 设置 mock SQL 期望。
			// ============================================================
			c.mock(sqlMock)

			// ============================================================
			// 3. 创建 mock AppContext 并注入 mock GORM。
			// ============================================================
			atx := appctx.NewMock(appctx.WithMockGORM(gormDB))

			// ============================================================
			// 4. 创建 DAO 实例并调用被测方法。
			// ============================================================
			dt := New(atx)
			u, err := dt.GetUser(c.id)

			// ============================================================
			// 5. 断言结果。
			// ============================================================
			tu.AssertWantError(t, err, c.wantErr)
			if !reflect.DeepEqual(u, c.u) {
				t.Errorf("\n期望: %+v\n实际: %+v", c.u, u)
			}
		})
	}
}

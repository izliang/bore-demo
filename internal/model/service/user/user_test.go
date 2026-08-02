// Package user 用户 Service 层单元测试。
//
// 测试策略：
//   - 通过 mockDao 实现 dao.Dao 接口，完全隔离数据库依赖
//   - 测试 Service 层的 DAO Entity → Domain Model 映射逻辑
//   - 测试错误传递路径（DAO 错误 → Service 错误）
//
// 不需要连接真实数据库，不需要 sqlmock，
// 因为 DAO 接口通过构造函数注入，可以在测试中完全替换。
//
// 新项目复刻指南：
//   - 每个 Service 方法至少覆盖 "成功" 和 "失败" 两条路径
//   - mock 结构体定义在测试文件内部，保持简洁
//   - 使用 table-driven tests 风格，便于扩展测试用例
package user

import (
	"errors"
	"reflect"
	"testing"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore/tu"
)

// mockDao 是 dao.Dao 接口的测试替身。
//
// 通过预设返回值（user 和 err）来控制测试场景：
//   - 正常场景：user 非 nil，err 为 nil
//   - 异常场景：user 为 nil，err 非 nil（模拟 DB 异常或记录不存在）
type mockDao struct {
	// user 预设的 DAO Entity 返回值。
	user *dao.User

	// err 预设的错误返回值。
	err error
}

// GetUser 实现 dao.Dao 接口，返回预设的 mock 数据。
func (m *mockDao) GetUser(_ uint64) (*dao.User, error) {
	return m.user, m.err
}

// TestGetUser 测试 Service.GetUser 方法。
//
// 测试用例覆盖：
//  1. 成功路径：验证 DAO Entity → Domain Model 的字段映射是否正确
//  2. 失败路径：验证 DAO 错误是否正确传递到 Service 层
//
// 扩展建议：
//   - 增加 "用户不存在" 的专项测试（区分 gorm.ErrRecordNotFound）
//   - 增加 "部分字段为空" 的边界测试
//   - 使用 e.ServiceError 的 GetCode() 验证错误码是否正确
func TestGetUser(t *testing.T) {
	tests := []struct {
		// name 测试用例名称，用于标识和定位失败用例。
		name string

		// param 传入 Service 的领域入参。
		param *model.GetUserParam

		// mock 预设的 mock DAO 实例。
		mock *mockDao

		// want 期望的领域模型返回值（仅成功场景使用）。
		want *model.User

		// wantErr 是否期望返回错误。
		wantErr bool
	}{
		{
			name:  "成功_DAO实体正确映射为领域模型",
			param: &model.GetUserParam{ID: 1},
			mock: &mockDao{
				user: &dao.User{
					ID:   1,
					Name: "Alice",
					Meta: dao.UserMeta{AvatarURL: "https://cdn.example.com/avatar.png"},
				},
			},
			want: &model.User{
				ID:     1,
				Name:   "Alice",
				Avatar: "https://cdn.example.com/avatar.png",
			},
			wantErr: false,
		},
		{
			name:  "失败_DAO返回错误时Service应传递错误",
			param: &model.GetUserParam{ID: 999},
			mock: &mockDao{
				err: errors.New("record not found"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 通过构造函数注入 mock DAO，完全隔离数据库。
			svc := New(tt.mock)

			// 执行被测方法。
			got, err := svc.GetUser(tt.param)

			// 断言want error
			tu.AssertWantError(t, err, tt.wantErr)

			// 验证返回值。
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("\n期望: %+v\n实际: %+v", tt.want, got)
			}
		})
	}
}

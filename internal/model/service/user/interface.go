// Package user 是用户业务逻辑层（Service Layer）。
//
// Service 层是分层架构的核心枢纽，承载应用的所有业务逻辑：
//   - 接收领域入参（model.GetUserParam），产出领域模型（model.User）
//   - 与任何传输协议（HTTP、gRPC、MQ）完全解耦
//   - 通过接口依赖 DAO 层，支持 mock 测试
//
// 接口设计原则：
//   - 每个 Service 对应一个业务聚合（如 User、Order、Product）
//   - 接口方法按业务能力命名（CRUD → GetUser、CreateUser、UpdateUser、DeleteUser）
//   - 入参和出参使用领域模型，不使用 HTTP DTO
//   - 错误统一使用 e.ServiceError 类型，包含业务错误码
//
// 新项目复刻指南：
//   - 在 internal/model/service/ 下创建新的业务包（如 order/、product/）
//   - 每个包包含 interface.go（接口定义）+ 实现文件（如 order.go）
//   - 接口命名：Service，实现命名：小写包名（如 order、product）
//   - 构造函数的依赖通过参数注入，不在内部创建（便于 mock）
package user

import (
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore/e"
)

// Service 定义用户业务逻辑的契约接口。
//
// 实现者：user（本包内的私有 struct）
// 调用者：Controller 层
//
// 每个方法对应一个业务能力，命名遵循 "动词 + 名词" 的惯例。
type Service interface {
	// GetUser 根据入参获取用户领域模型。
	//
	// 参数 param 包含查询条件（当前为 ID，未来可扩展为复合条件）。
	// 返回的 *model.User 是纯粹的领域模型，不包含 ORM 标签。
	// 如果用户不存在或查询失败，返回 e.ServiceError。
	GetUser(param *model.GetUserParam) (*model.User, e.ServiceError)
}

// user 是 Service 接口的私有实现。
//
// 私有 struct 确保外部只能通过 New 构造函数创建实例，
// 且只能以 Service 接口的形式使用，保证依赖倒置原则。
type user struct {
	// dao 数据访问层接口。
	// 通过构造函数注入，便于单元测试时替换为 mock 实现。
	dao dao.Dao
}

// New 创建用户 Service 实例。
//
// 参数 d 是 DAO 接口的实现（真实 DB 操作或 mock）。
// 返回 Service 接口，调用方无需知道具体实现类型。
//
// 为什么不在 New 内部创建 DAO？
//
//	为了让 Service 层的单元测试可以使用 mock DAO，
//	而不依赖真实的数据库连接。这是一种手动的依赖注入。
//
// 使用示例：
//
//	// 生产代码（Controller）
//	d := dao.New(atx)
//	svc := user.New(d)
//	user, err := svc.GetUser(&model.GetUserParam{ID: 1})
//
//	// 测试代码
//	mock := &mockDao{user: &dao.User{ID: 1, Name: "test"}}
//	svc := user.New(mock)
func New(d dao.Dao) Service {
	return &user{dao: d}
}

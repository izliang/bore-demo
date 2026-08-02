// Package dao 是数据访问层（Data Access Object Layer）。
//
// DAO 层是唯一与数据库直接交互的层，职责明确：
//   - 封装所有 GORM/SQL 操作，对外暴露面向业务的方法
//   - 通过接口暴露，Service 层只依赖接口而非具体实现
//   - 返回 DAO Entity（带 GORM 标签的结构体），不返回领域模型
//
// 接口设计约定：
//   - 接口名：Dao（大写，意为 Data Access Object 的抽象）
//   - 方法命名：动词 + 名词，如 GetUser、CreateUser、ListUsers
//   - 参数尽量简单（基础类型或简单 struct），避免传递复杂对象
//   - 错误返回原生 error（GORM 的 error），由 Service 层包装为 ServiceError
//
// 为什么 DAO 返回的是带 GORM 标签的实体而不是领域模型？
//  1. DAO 层关注持久化，返回的实体反映数据库表结构
//  2. 领域模型应该是纯业务语义的，Service 层负责 Entity → Domain 的转换
//  3. 当数据库表结构变更时，只需修改 DAO Entity 和 Service 的映射代码
//
// 新项目复刻指南：
//   - 每个数据库表对应一个 Entity struct 文件（如 user.go、order.go）
//   - interface.go 定义一个统一的 Dao 接口，包含所有数据访问方法
//   - 如果项目很大，可以拆分为多个接口（UserDao、OrderDao），分别实现
//   - 注意文件名 interface.go 可能在 IDE 中被隐藏，建议保留此命名以明确意图
package dao

import "github.com/izliang/bore/appctx"

// Dao 数据访问层接口。
//
// 当前只有一个方法，实际项目中会包含完整的 CRUD：
//
//	type Dao interface {
//	    GetUser(id uint64) (*User, error)
//	    CreateUser(user *User) error
//	    UpdateUser(user *User) error
//	    DeleteUser(id uint64) error
//	    ListUsers(offset, limit int) ([]*User, int64, error)
//	}
type Dao interface {
	// GetUser 根据主键 ID 查询用户。
	//
	// 参数 id 是用户表的主键。
	// 如果记录不存在，返回 gorm.ErrRecordNotFound。
	// 返回 *User（带 GORM 标签的实体），由 Service 层转换为领域模型。
	GetUser(id uint64) (*User, error)
}

// dao 是 Dao 接口的私有实现。
//
// 持有 AppContext 以获取数据库连接（通过 atx.GORM()）。
// AppContext 是请求级别的，因此 DAO 实例也是请求级别的，
// 保证并发安全且数据库操作可在同一事务上下文中执行。
type dao struct {
	// atx 请求上下文，携带 GORM 实例、Logger 等基础设施。
	atx appctx.AppContext
}

// New 创建 DAO 实例。
//
// 参数 atx 由 Controller 层创建并传入，每次请求一个独立实例。
// 设计意图：不依赖全局变量，便于测试时替换为 sqlmock 的 GORM 实例。
//
// 使用示例：
//
//	atx := appctx.New(appctx.WithGORM(gormDB), ...)
//	d := dao.New(atx)
//	user, err := d.GetUser(1)
func New(atx appctx.AppContext) Dao {
	return &dao{
		atx: atx,
	}
}

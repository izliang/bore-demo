// Package model 定义领域模型（Domain Model）。
//
// 领域模型是业务逻辑的核心数据结构，具有以下特征：
//   - 不包含任何 ORM 标签（与数据库表结构解耦）
//   - 不包含任何 HTTP 序列化标签（与传输协议解耦）
//   - 字段命名使用业务术语，直观反映业务含义
//   - 由 Service 层产出，由 Controller 层消费
//
// 领域模型 vs DAO Entity vs HTTP DTO：
//
//	┌─────────────┐      ┌─────────────┐      ┌─────────────┐      ┌─────────────┐
//	│ HTTP DTO    │ ───→ │ Domain Param│ ───→ │ Service     │ ───→ │ DAO Entity  │
//	│ (v1 pkg)    │      │ (model pkg) │      │             │      │ (dao pkg)   │
//	│             │ ←─── │ Domain Model│ ←─── │             │ ←─── │             │
//	└─────────────┘      └─────────────┘      └─────────────┘      └─────────────┘
//	  传输层关注            业务层关注            业务逻辑编排           持久层关注
//	  json/query tag        纯业务语义            无框架依赖              gorm tag
//
// 新项目复刻指南：
//   - 每个业务聚合根一个文件（如 user.go、order.go）
//   - 入参（*Param）和出参（Entity）定义在同一个文件中
//   - 复杂实体可以拆分为多个 struct（如 Order、OrderItem、OrderAddress）
//   - 如果字段很多且与 DAO Entity 完全一致，可以考虑直接用 DAO Entity 作为领域模型
//     （这是务实的选择，本 demo 为了演示解耦采用了独立定义）
package model

// ============================================================
// 用户领域模型
// ============================================================

// GetUserParam 获取用户的输入参数（领域层）。
//
// 与 v1.GetUserRequest 的区别：
//   - 无 json/query/form 等 HTTP 序列化标签
//   - 无 required/minimum 等校验标签（校验应在 Controller 层由框架完成）
//   - 字段名使用业务命名而非 API 契约命名
//
// 当需要新增调用来源（如 gRPC、MQ Consumer）时，只需新增对应的
// DTO → GetUserParam 转换代码，Service 层无需任何修改。
type GetUserParam struct {
	// ID 用户唯一标识。
	ID uint64
}

// User 用户领域实体。
//
// 设计要点：
//   - 无 GORM 标签：与数据库表结构解耦，表结构变更不影响业务层
//   - 无 JSON 标签：与 HTTP 响应格式解耦，API 格式变更不影响业务层
//   - 字段精简：只包含业务所需的字段，不暴露数据库内部字段
//
// 当前包含最基础的三个字段。实际项目中可能包含：
//
//	type User struct {
//	    ID        uint64
//	    Name      string
//	    Avatar    string
//	    Email     string
//	    Phone     string
//	    Status    UserStatus  // 枚举类型，而非 raw int
//	    CreatedAt time.Time
//	    UpdatedAt time.Time
//	}
type User struct {
	// ID 用户唯一标识。
	ID uint64

	// Name 用户显示名称。
	Name string

	// Avatar 用户头像完整 URL。
	// 注意：DAO 层存储的是 Meta.AvatarURL，Service 层在此处展开为顶层字段。
	Avatar string
}

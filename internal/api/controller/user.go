// Package controller 是 HTTP 控制器层（Controller Layer）。
//
// 控制器层的职责：
//  1. 接收 cyberhuma 框架解析并校验后的 HTTP 请求 DTO
//  2. 构建请求级别的 AppContext（携带 Logger、DB 等基础设施）
//  3. 将 HTTP DTO 转换为领域层入参（DTO → Domain Param）
//  4. 调用 Service 层的业务逻辑
//  5. 将领域层返回的领域模型转换为 HTTP 响应 DTO（Domain Model → DTO）
//  6. 处理错误并返回适当的 HTTP 响应（由 cyberhuma 框架统一封装）
//
// 控制器层不应该包含：
//   - 业务逻辑（属于 Service 层）
//   - 数据库操作（属于 DAO 层）
//   - 复杂的数据转换（应抽取到独立的 converter 或 assembler）
//
// 设计理念 —— 为什么控制器不直接依赖 Service 单例？
//
//	每次请求在控制器中动态构建依赖链（AppContext → DAO → Service），
//	目的是支持单元测试时完全替换任何一层的实现。
//	例如：测试控制器时可以注入 mock DAO，测试 Service 时可以注入 mock DAO。
//
// 新项目复刻指南：
//   - 每个业务模块一个 controller 文件
//   - Handler 签名必须匹配 cyberhuma 的 TypedHandler[I, O] 泛型约束
//   - DTO → Domain 转换代码在此处编写，Service 层只处理领域对象
package controller

import (
	"context"

	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore-demo/internal/model/dao"
	"github.com/izliang/bore-demo/internal/model/service/user"
	"github.com/izliang/bore-demo/internal/resource"
	v1 "github.com/izliang/bore-demo/internal/resource/api/v1"
	"github.com/izliang/bore/appctx"
	"github.com/izliang/bore/e"
)

// GetUser 处理 GET /api/v1/user/get 请求。
//
// 链路追踪：
//
//	HTTP Request（query: ?id=1）
//	  │  cyberhuma 自动校验：required + minimum → 校验失败直接返回 422
//	  ▼
//	Controller.GetUser
//	  ├─ [1] 构建请求级 AppContext（每次请求独立的 DI 容器）
//	  ├─ [2] 创建 DAO 实例（从 AppContext 获取 DB）
//	  ├─ [3] 创建 Service 实例（注入 DAO）
//	  ├─ [4] HTTP DTO → Domain Param 转换
//	  ├─ [5] 调用 Service.GetUser(param)
//	  │      ├─ DAO.GetUser(id) → GORM First()
//	  │      └─ DAO Entity → Domain Model 转换
//	  ├─ [6] Domain Model → HTTP DTO 转换
//	  └─ 返回响应（由 cyberhuma 包装为统一 Envelope 格式）
func GetUser(ctx context.Context, req *v1.GetUserRequest) (*v1.GetUserResponse, e.ServiceError) {
	// ============================================================
	// 步骤1：构建请求级别的 AppContext
	//
	// AppContext 是请求范围的依赖注入容器，携带：
	//   - Context：请求上下文（超时控制、取消信号、TraceID 等元数据）
	//   - Logger：结构化日志实例
	//   - GORM：数据库连接
	//
	// 每次请求独立创建，保证并发安全且便于测试时替换。
	// ============================================================
	atx := appctx.New(
		appctx.WithContext(ctx),
		appctx.WithLogger(resource.Logger),
		appctx.WithGORM(resource.DB),
	)

	// ============================================================
	// 步骤2-3：构建依赖链
	//
	// AppContext → DAO → Service
	// 每层通过构造函数注入下一层依赖，形成清晰的依赖方向。
	// ============================================================
	d := dao.New(atx)
	svc := user.New(d)

	// ============================================================
	// 步骤4：HTTP 请求 DTO → 领域层入参
	//
	// 这是控制器层的核心职责之一：将传输层的数据结构转换为
	// 领域层所需的数据结构，实现传输层与业务逻辑的解耦。
	//
	// 对于复杂场景（如多个 DTO 字段组合为一个领域参数），
	// 可以抽取独立的 assembler 函数。
	// ============================================================
	param := &model.GetUserParam{ID: req.ID}

	// ============================================================
	// 步骤5：调用 Service 层执行业务逻辑
	//
	// Service 层返回领域模型（model.User）和业务错误（e.ServiceError）。
	// 业务错误由 cyberhuma 框架自动包装为统一响应格式：
	//   { "code": <错误码>, "message": "<错误信息>" }
	// ============================================================
	u, svcErr := svc.GetUser(param)
	if svcErr != nil {
		return nil, svcErr
	}

	// ============================================================
	// 步骤6：领域模型 → HTTP 响应 DTO
	//
	// 与步骤4对应：领域模型关注业务含义，响应 DTO 关注 API 契约。
	// 两者的字段可能不同（如字段名、结构、包含/排除某些字段）。
	// ============================================================
	return &v1.GetUserResponse{
		ID:     u.ID,
		Name:   u.Name,
		Avatar: u.Avatar,
	}, nil
}

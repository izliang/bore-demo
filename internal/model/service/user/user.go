package user

import (
	"github.com/izliang/bore-demo/internal/model"
	"github.com/izliang/bore/e"
)

// GetUser 实现 Service 接口：根据参数获取用户领域模型。
//
// 处理流程：
//  1. 调用 DAO 层查询数据库
//  2. 处理查询错误（如记录不存在、数据库异常）
//  3. 将 DAO Entity 转换为领域模型（Domain Model）
//
// DAO Entity → Domain Model 映射说明：
//
//		dao.User.ID          → model.User.ID     （直接映射）
//		dao.User.Name        → model.User.Name   （直接映射）
//		dao.User.Meta.AvatarURL → model.User.Avatar （嵌套字段展开）
//
//		这种映射体现了 Service 层的核心价值：
//	  - 数据库的存储结构（Meta JSON 字段）对上层透明
//	  - 业务层看到的是扁平、直观的字段结构
//	  - 未来数据库结构调整（如将 AvatarURL 提升为列）只需修改此映射
//
// 错误处理策略：
//   - DAO 返回的 error 通过 e.Wrap() 包装为 ServiceError
//   - e.Wrap() 自动提取底层错误类型（如 gorm.ErrRecordNotFound），
//     映射为统一的业务错误码，便于上层统一处理
func (u *user) GetUser(param *model.GetUserParam) (*model.User, e.ServiceError) {
	// ============================================================
	// 1. 调用 DAO 层查询数据库。
	// ============================================================
	us, err := u.dao.GetUser(param.ID)
	if err != nil {
		// 将底层错误包装为标准 ServiceError。
		// e.Wrap 会根据错误类型自动映射为合适的业务错误码。
		return nil, e.Wrap(err, "", "get user failed")
	}

	// ============================================================
	// 2. DAO Entity → Domain Model 转换。
	//
	// 这是业务逻辑的一部分：数据库存储的是 Meta JSON 嵌套结构，
	// 但业务层关心的是扁平的 Avatar 字段。
	//
	// 扩展点：
	//   - 如果头像 URL 需要加 CDN 前缀，在此处理
	//   - 如果需要根据用户状态计算展示名称，在此处理
	//   - 如果需要补全关联数据（如部门名称），调用其他 DAO
	// ============================================================
	return &model.User{
		ID:     us.ID,
		Name:   us.Name,
		Avatar: us.Meta.AvatarURL,
	}, nil
}

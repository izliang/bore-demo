// Package v1 定义 API v1 的 HTTP 请求/响应 DTO（Data Transfer Object）。
//
// DTO 属于表示层（Presentation Layer），是 HTTP 协议与业务逻辑之间的桥梁：
//   - 请求 DTO：承载 HTTP 请求参数（query、path、header、body），由框架自动解析和校验
//   - 响应 DTO：定义 HTTP 响应的数据结构，由框架自动序列化为 JSON
//
// DTO vs Domain Model：
//
//	DTO                          Domain Model
//	─────────────────────────     ─────────────────────
//	包含 json/query/form tag      无传输层标签
//	包含 required/minimum 等      无校验标签
//	校验标签（框架自动校验）
//	字段名面向 API 契约            字段名面向业务语义
//	可能包含多个接口的聚合字段      一个模型对应一个业务概念
//	结构随 API 版本变化            结构相对稳定
//
// 校验标签说明（cyberhuma / Huma v2）：
//   - required:"true"    → 参数必填，缺省时返回 422
//   - minimum:"1"        → 数值最小值，小于时返回 422
//   - maximum:"100"      → 数值最大值
//   - minLength:"1"      → 字符串最小长度
//   - maxLength:"255"    → 字符串最大长度
//   - pattern:"^[a-z]+$" → 正则表达式校验
//   - enum:"a,b,c"       → 枚举值校验
//   - doc:"说明"          → OpenAPI 文档中的字段描述
//
// 新项目复刻指南：
//   - 每个 API 端点一个文件（如 get_user.go、create_user.go）
//   - 命名规则：{操作}_{资源}.go（如 get_order.go、list_products.go）
//   - 请求/响应结构体放在同一个文件中（如果结构简单）
//   - 复杂响应可以拆分为独立的 response 文件
package v1

// GetUserRequest HTTP 请求 DTO（GET /api/v1/user/get）。
//
// 字段来源：query 参数（?id=1）或 form 参数。
// cyberhuma 框架根据 struct tag 自动解析和校验以下来源：
//   - query:"id"    → URL Query 参数（?id=1）
//   - path:"id"     → URL 路径参数（/users/{id}）
//   - header:"X-ID" → HTTP Header
//   - form:"id"     → Form 表单
//   - json:"id"     → JSON Body
//
// 校验规则（框架自动执行，校验失败返回 422 + 友好错误提示）：
//   - required:"true"  → id 参数必须提供
//   - minimum:"1"      → id 值必须 ≥ 1
type GetUserRequest struct {
	// ID 要查询的用户主键 ID。
	//
	// 约束：
	//   - 必填（required:"true"）
	//   - 最小值为 1（minimum:"1"），避免 id=0 导致 GORM 查询到第一条记录
	//
	// 多来源 tag 说明：
	//   - json:"id"   → 支持 JSON Body 传递（POST 场景）
	//   - query:"id"  → 支持 URL 查询参数传递（GET 场景）
	//   - form:"id"   → 支持表单传递
	//   - doc:"User ID" → OpenAPI 文档中的字段描述
	ID uint64 `json:"id" query:"id" form:"id" required:"true" minimum:"1" doc:"User ID"`
}

// GetUserResponse HTTP 响应 DTO。
//
// 响应结构设计约定：
//   - 每个字段有 json tag，控制 JSON 序列化的 key 名
//   - 字段名使用 camelCase（JSON 惯例）
//   - 不包含内部字段（如 password_hash、deleted_at）
//   - 如果字段可能为空，使用指针类型（*string）以区分 "空值" 和 "不返回"
//
// 实际响应由 cyberhuma 框架包装为统一 Envelope 格式：
//
//	{
//	  "code": 0,
//	  "message": "success",
//	  "data": {
//	    "id": 1,
//	    "name": "Alice",
//	    "avatar": "https://cdn.example.com/avatar.png"
//	  }
//	}
type GetUserResponse struct {
	// ID 用户唯一标识。
	ID uint64 `json:"id"`

	// Name 用户显示名称。
	Name string `json:"name"`

	// Avatar 用户头像完整 URL。
	Avatar string `json:"avatar"`
}

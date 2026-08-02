package dao

// ============================================================
// User —— 用户表的数据访问实体（DAO Entity）
//
// DAO Entity 的设计约定：
//   - 每个 struct 对应一张数据库表，TableName() 返回表名
//   - 字段使用 gorm tag 定义列映射、约束和序列化方式
//   - 字段使用 json tag 定义 JSON 序列化格式（用于日志/调试输出）
//   - 嵌套结构体使用 gorm serializer:json 存储为 JSON 列
//
// DAO Entity 与领域模型（model.User）的区别：
//
//	DAO Entity (dao.User)          Domain Model (model.User)
//	─────────────────────────       ────────────────────────
//	包含 gorm tag                   无任何 ORM 标签
//	Name 映射到 name 列             Name 是纯业务字段
//	Meta 是嵌套 JSON 结构体         Avatar 是展开的顶层字段
//	TableName() 定义表名            无 TableName()
//	在 DAO 层内部使用                在 Service/Controller 层之间传递
//
// 新项目复刻指南：
//   - 使用 gorm tag 定义数据库映射
//   - serializer:json 用于存储 JSON 对象（MySQL 5.7+ JSON 类型）
//   - TableName() 方法必须实现，否则 GORM 使用默认蛇形命名
//   - 如果表名与 struct 名不一致，在 TableName() 中显式返回
// ============================================================

// User 用户表实体（对应数据库 users 表）。
type User struct {
	// ID 主键，自增。
	// gorm tag: primary_key
	ID uint64 `gorm:"column:id;primary_key;" json:"id"`

	// Name 用户名称。
	// gorm tag: NOT NULL 确保数据库层面不为空。
	Name string `gorm:"column:name;NOT NULL" json:"name"`

	// Meta 用户扩展元数据（JSON 列）。
	// gorm tag: serializer:json 将 Go struct 自动序列化为 JSON 存入数据库。
	// 可以存储头像 URL、偏好设置等扩展字段，避免频繁修改表结构。
	Meta UserMeta `gorm:"column:meta;serializer:json" json:"meta"`
}

// UserMeta 用户扩展元数据的子结构，作为 JSON 存储在 meta 列中。
//
// 使用 JSON 列的好处：
//   - 扩展字段无需 DDL 变更（如新增偏好设置、社交链接等）
//   - GORM 的 serializer:json 自动处理序列化/反序列化
//   - MySQL 支持 JSON 列的索引（虚拟列 + 索引）
//
// 使用 JSON 列的注意事项：
//   - 不要将高频查询条件放在 JSON 列中（无法高效索引）
//   - JSON 列中的字段不参与数据库约束（如唯一性）
//   - 适合存储"附属信息"而非"核心业务字段"
type UserMeta struct {
	// AvatarURL 用户头像地址。
	AvatarURL string `json:"avatar_url"`
}

// TableName 指定 GORM 使用的数据库表名。
//
// 如果不实现此方法，GORM 默认使用蛇形复数：struct User → 表名 "users"。
// 此处显式定义为 "users" 以保持明确。
func (User) TableName() string {
	return "user"
}

// ============================================================
// GetUser —— 数据访问方法
// ============================================================

// GetUser 根据主键 ID 查询用户记录。
//
// 实现细节：
//   - 使用 GORM 的 First() 方法，等价于 SELECT * FROM users WHERE id = ? LIMIT 1
//   - 如果记录不存在，GORM 返回 gorm.ErrRecordNotFound
//   - 如果数据库连接异常或 SQL 执行失败，返回对应的 error
//
// 性能考虑：
//   - First() 按主键查询，利用主键索引，性能最优
//   - 如果查询频繁，建议在 Service 层引入缓存（Redis）
//
// 错误处理：
//   - 不在 DAO 层做错误分类（如区分 "不存在" vs "数据库异常"）
//   - 将原始 error 直接返回，由 Service 层统一包装和分类
//   - 这样 Service 层可以根据业务需求决定如何处理不同类型的错误
func (d *dao) GetUser(id uint64) (*User, error) {
	u := &User{}

	// GORM First() 按主键查询，自动使用主键索引。
	// 等价 SQL：SELECT * FROM `users` WHERE `users`.`id` = ? ORDER BY `users`.`id` LIMIT 1
	if err := d.atx.GORM().First(u, id).Error; err != nil {
		return nil, err
	}
	return u, nil
}

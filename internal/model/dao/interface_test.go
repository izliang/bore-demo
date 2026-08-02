// Package dao DAO 接口的编译期验证。
//
// 此文件用于验证 mock 实现是否满足 dao.Dao 接口。
// 在编写 mock 时（如 Service 层测试的 mockDao），
// 可以在此处添加编译期断言确保接口一致性。
//
// 示例：
//
//	var _ Dao = (*mockDao)(nil)  // 编译期检查 mockDao 实现了 Dao
//
// 当前为空，新项目按需添加。
package dao

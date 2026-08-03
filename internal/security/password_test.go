// Package security 密码工具单元测试。
package security

import "testing"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if hash == "admin123" || len(hash) < 50 {
		t.Fatalf("哈希结果异常: %q", hash)
	}

	if !CheckPassword(hash, "admin123") {
		t.Error("正确密码应校验通过")
	}
	if CheckPassword(hash, "wrong") {
		t.Error("错误密码不应校验通过")
	}
	if CheckPassword("not-a-hash", "admin123") {
		t.Error("非法哈希不应校验通过")
	}
}

func TestHashPassword_Unique(t *testing.T) {
	// bcrypt 带随机盐，同一明文两次哈希结果不同。
	h1, _ := HashPassword("admin123")
	h2, _ := HashPassword("admin123")
	if h1 == h2 {
		t.Error("期望两次哈希结果不同（随机盐）")
	}
}

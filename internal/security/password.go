// Package security 提供密码安全相关的通用能力。
//
// 当前仅封装 bcrypt 的哈希与校验，后续可扩展令牌签名等能力。
package security

import "golang.org/x/crypto/bcrypt"

// bcryptCost 是 bcrypt 的计算成本。
// 数值越大越安全、越慢；12 是当前安全性与性能的平衡点。
const bcryptCost = 12

// HashPassword 将明文密码哈希为 bcrypt 字符串。
// 返回的哈希可直接存入数据库 password_hash 列。
func HashPassword(plain string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验明文密码是否与哈希匹配。
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

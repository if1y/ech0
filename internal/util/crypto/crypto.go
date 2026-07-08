// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package util

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// 密码哈希算法标识，持久化在 user_local_auth.password_algo 列，登录时据此分派校验。
const (
	AlgoMD5    = "md5"    // 历史遗留：裸 MD5 无盐，仅用于存量校验，登录成功后惰性升级为 bcrypt
	AlgoBcrypt = "bcrypt" // 当前算法：自带盐、自描述哈希串
)

// MaxPasswordBytes 是 bcrypt 接受的口令字节上限——超过 72 字节 GenerateFromPassword 会返回
// ErrPasswordTooLong。写入侧据此前置校验，避免把 bcrypt 的裸英文错误抛给用户。
const MaxPasswordBytes = 72

// MD5Encrypt 对内容进行 MD5 编码。
//
// 已废弃用于新密码——仅保留给存量 AlgoMD5 口令的校验（见 CheckPassword）。
func MD5Encrypt(text string) string {
	hash := md5.New()
	hash.Write([]byte(text))
	hashInBytes := hash.Sum(nil)
	return hex.EncodeToString(hashInBytes)
}

// HashPassword 用 bcrypt（默认 cost，自带随机盐）对明文口令做哈希，返回自描述的哈希串。
// 返回的哈希应连同 AlgoBcrypt 一起持久化到 user_local_auth。
func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// CheckPassword 校验明文口令是否匹配存储的哈希，按 algo 分派：
//   - AlgoBcrypt：bcrypt 常数时间比对；
//   - 其它（AlgoMD5 或空串等存量值）：退化为裸 MD5 等值比对。
//
// 校验通过且 algo != AlgoBcrypt 时，调用方应惰性升级为 bcrypt。
func CheckPassword(algo, storedHash, plain string) bool {
	if algo == AlgoBcrypt {
		return bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(plain)) == nil
	}
	return MD5Encrypt(plain) == storedHash
}

// randomCharset 是 GenerateRandomString 使用的字符集（62 个字符）。
const randomCharset = "abcdefghijklmnopqrstuvwxyz" +
	"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// GenerateRandomString 使用密码学安全随机源（crypto/rand）生成指定长度的随机字符串。
//
// 该函数被用于生成 OAuth state、一次性 OAuth 兑换码、token JTI 等**安全敏感**的不可预测值，
// 因此必须使用 crypto/rand 而非 math/rand —— 后者可被预测，会让 CSRF state / 一次性码被伪造。
//
// 为避免取模偏置（256 不是 62 的整数倍），对落在不完整区间的随机字节做拒绝采样。
// length <= 0 时返回空串。crypto/rand 读取失败属系统级灾难，直接 panic，绝不退化为可预测值。
func GenerateRandomString(length int) string {
	if length <= 0 {
		return ""
	}

	// 拒绝采样上界：丢弃会引入取模偏置的高位字节（256 % 62 = 8 → 丢弃 248..255）。
	const limit = 256 - (256 % len(randomCharset))

	out := make([]byte, length)
	buf := make([]byte, length)
	for filled := 0; filled < length; {
		if _, err := rand.Read(buf); err != nil {
			panic("crypto: secure random source unavailable: " + err.Error())
		}
		for _, v := range buf {
			if int(v) >= limit {
				continue
			}
			out[filled] = randomCharset[int(v)%len(randomCharset)]
			filled++
			if filled == length {
				break
			}
		}
	}
	return string(out)
}

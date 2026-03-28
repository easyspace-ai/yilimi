package utils

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"

	"github.com/google/uuid"
)

// GenerateID 生成20位随机ID（类似SiYuan的ID格式）
func GenerateID() string {
	// 使用时间戳前缀 + 随机字符串
	timestamp := time.Now().Format("20060102150405")

	// 生成7位随机字符串
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	randomPart := make([]byte, 7)

	for i := range randomPart {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		randomPart[i] = charset[n.Int64()]
	}

	return timestamp + "-" + string(randomPart)
}

// GenerateUUID 生成标准UUID
func GenerateUUID() string {
	return uuid.New().String()
}

// GenerateShortID 生成短ID（8位）
func GenerateShortID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	id := make([]byte, 8)

	for i := range id {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		id[i] = charset[n.Int64()]
	}

	return string(id)
}

// ValidateID 验证ID格式是否正确
func ValidateID(id string) bool {
	if len(id) != 21 { // 格式: 20060102150405-abcdefg
		return false
	}

	if id[14] != '-' {
		return false
	}

	// 验证时间戳部分
	timestamp := id[:14]
	_, err := time.Parse("20060102150405", timestamp)
	if err != nil {
		return false
	}

	// 验证随机部分
	randomPart := id[15:]
	if len(randomPart) != 7 {
		return false
	}

	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	for _, char := range randomPart {
		found := false
		for _, validChar := range charset {
			if char == validChar {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// ExtractTimestampFromID 从ID中提取时间戳
func ExtractTimestampFromID(id string) (time.Time, error) {
	if !ValidateID(id) {
		return time.Time{}, fmt.Errorf("无效的ID格式: %s", id)
	}

	timestamp := id[:14]
	return time.Parse("20060102150405", timestamp)
}

// GenerateRefID 生成引用ID
func GenerateRefID() string {
	// 引用ID使用UUID格式
	return GenerateUUID()
}

// GenerateBlockID 生成块ID（别名，保持一致性）
func GenerateBlockID() string {
	return GenerateID()
}

// GenerateNotebookID 生成笔记本ID（别名，保持一致性）
func GenerateNotebookID() string {
	return GenerateID()
}

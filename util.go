package main

import (
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/chacha20poly1305"
	tele "gopkg.in/telebot.v3"
	"os"
	"regexp"
	"time"
)

const (
	DateFormat       = "2006-01-02"
	TimeFormat       = "15:04"
	TimeSecondFormat = "15:04:05"
	DateTimeFormat   = "2006-01-02 15:04:05"
)

func IsCommand(message *tele.Message) bool {
	for _, entity := range message.Entities {
		if entity.Type == tele.EntityCommand {
			return true
		}
	}
	return false
}

func ReplaceForMarkdownV2(input string) string {
	if input == "" {
		return input
	}

	re := regexp.MustCompile("[_*\\[\\]()~`>#+=\\-|{}.!]")
	return re.ReplaceAllString(input, `\$0`)
}

func GetDuration(date time.Time) string {
	now := time.Now()

	// 计算时间差
	duration := date.Sub(now)
	diffDays := int(duration.Hours()) / 24
	diffHours := int(duration.Hours()) % 24
	diffMinutes := int(duration.Minutes()) % 60

	if diffDays != 0 {
		return fmt.Sprintf("%d天%d小时%d分钟", diffDays, diffHours, diffMinutes)
	} else if diffHours != 0 {
		return fmt.Sprintf("%d小时%d分钟", diffHours, diffMinutes)
	} else {
		return fmt.Sprintf("%d分钟", diffMinutes)
	}
}

// 全局加密实例
var chachaCipher cipher.AEAD

func InitChacha20() {
	encryptionKey := os.Getenv("KEY")
	if len(encryptionKey) == 0 {
		panic("加密密钥不可以为空")
	}

	keyBytes := []byte(encryptionKey)
	var err error
	chachaCipher, err = chacha20poly1305.New(keyBytes)
	if err != nil {
		panic("初始化加密实例失败：" + err.Error())
	}
}

func encryptString(plaintext string) (string, error) {
	// 将输入字符串转换为字节切片
	plaintextBytes := []byte(plaintext)

	// 随机生成 12 字节的 nonce
	nonce := make([]byte, chacha20poly1305.NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}

	// 使用全局加密实例进行加密
	ciphertext := chachaCipher.Seal(nil, nonce, plaintextBytes, nil)

	// 将 nonce 和加密后的数据合并
	ciphertextWithNonce := append(nonce, ciphertext...)

	// 将加密后的字节切片转换为 base64 编码的字符串
	ciphertextBase64 := base64.StdEncoding.EncodeToString(ciphertextWithNonce)

	return ciphertextBase64, nil
}

func decryptString(ciphertextBase64 string) (string, error) {
	// 将输入字符串转换为字节切片
	ciphertextWithNonce, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}

	// 提取 nonce
	nonce := ciphertextWithNonce[:chacha20poly1305.NonceSize]
	ciphertext := ciphertextWithNonce[chacha20poly1305.NonceSize:]

	// 使用全局加密实例进行解密
	decryptedText, err := chachaCipher.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(decryptedText), nil
}

func calculateTraffic(byteSize int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)

	switch {
	case byteSize >= gb:
		return fmt.Sprintf("%s GB", decimal.NewFromInt(byteSize).Div(decimal.NewFromInt(gb)).Round(2).String())
	case byteSize >= mb:
		return fmt.Sprintf("%s MB", decimal.NewFromInt(byteSize).Div(decimal.NewFromInt(mb)).Round(2).String())
	case byteSize >= kb:
		return fmt.Sprintf("%s KB", decimal.NewFromInt(byteSize).Div(decimal.NewFromInt(kb)).Round(2).String())
	default:
		return fmt.Sprintf("%d Bytes", byteSize)
	}
}

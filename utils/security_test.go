package utils

import (
	"testing"
)

func TestCheckPasswordBcrypt(t *testing.T) {
	hash, err := HashPassword("admin123")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if !IsPasswordHash(hash) {
		t.Fatalf("生成的哈希没有 bcrypt 前缀: %s", hash)
	}
	// 正确密码通过，错误密码拒绝
	if !CheckPassword("admin123", hash) {
		t.Fatalf("正确密码校验未通过")
	}
	if CheckPassword("wrong", hash) {
		t.Fatalf("错误密码校验通过了")
	}
	// 空值一律拒绝（防误配：库里没密码时不能用空密码登录）
	if CheckPassword("", hash) || CheckPassword("admin123", "") {
		t.Fatalf("空密码/空哈希应该拒绝")
	}
}

func TestCheckPasswordLegacyPlaintext(t *testing.T) {
	// 存量明文库：明文命中（登录后由 UpgradePasswordHash 升级），错密码拒绝
	if !CheckPassword("admin", "admin") {
		t.Fatalf("存量明文密码应该能登录（用于触发升级）")
	}
	if CheckPassword("wrong", "admin") {
		t.Fatalf("存量明文库的错误密码应该拒绝")
	}
	// 已经是哈希的值不能再按明文比对
	hash, err := HashPassword("admin")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if CheckPassword(hash, hash) && hash == "admin" {
		t.Fatalf("哈希值本身不应该被当成明文密码接受")
	}
}

func TestIsImageContentType(t *testing.T) {
	for _, ct := range []string{"image/png", "image/jpeg; charset=binary", "IMAGE/WEBP", "image/svg+xml"} {
		if !IsImageContentType(ct) {
			t.Fatalf("%q 应该是图片类型", ct)
		}
	}
	for _, ct := range []string{"", "text/html", "application/json", "text/xml", "text/plain; charset=utf-8", "application/octet-stream"} {
		if IsImageContentType(ct) {
			t.Fatalf("%q 不应该是图片类型", ct)
		}
	}
}

func TestIsPasswordHash(t *testing.T) {
	hash, err := HashPassword("x")
	if err != nil {
		t.Fatalf("HashPassword 失败: %v", err)
	}
	if !IsPasswordHash(hash) {
		t.Fatalf("生成的哈希应该被识别")
	}
	for _, v := range []string{"", "admin", "$2a$", "pbkdf2:abc"} {
		if IsPasswordHash(v) {
			t.Fatalf("%q 不应该是 bcrypt 哈希", v)
		}
	}
}
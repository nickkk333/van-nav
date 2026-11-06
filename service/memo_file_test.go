package service

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

// 回归：长文件名里含多个点时，SafeImageFileName 截断到 50 字符后末尾可能正好是一个点，
// 再拼接扩展名会形成 ".."，被文件读取校验当成路径穿越，导致「存得下但读不出」、图片预览 404
func TestMemoFileNameWithManyDots(t *testing.T) {
	original := "Screenshot_2026-11-06-10-07-26-695_com.ss.android.ugc.aweme.131d57397b.jpg"
	ext := strings.ToLower(filepath.Ext(original))
	base := SafeImageFileName(strings.TrimSuffix(original, filepath.Ext(original)), "file")
	if strings.HasSuffix(base, ".") || strings.Contains(base, "..") {
		t.Fatalf("清洗后的 base 不应以点结尾或含连续点: %q", base)
	}
	name := fmt.Sprintf("%s%d_%s_%s%s", memoFilePrefix, 123, "abcd", base, ext)
	if strings.Contains(name, "..") {
		t.Fatalf("生成的附件名不应含连续点: %q", name)
	}
	if !isSafeMemoFileName(name) {
		t.Fatalf("生成的附件名应能通过读取校验: %q", name)
	}
	if !IsMemoImageName(name) {
		t.Fatalf("jpg 附件应被识别为图片并内联显示: %q", name)
	}
}

// 放宽 ".." 校验后：历史坏文件名可读取，但路径穿越仍必须被拒绝
func TestMemoFileNameRejectsTraversal(t *testing.T) {
	legacy := "memo_123_abcd_Screenshot_2026-11-06-10-07-26-695_com.ss.android..jpg"
	if !isSafeMemoFileName(legacy) {
		t.Fatalf("历史附件（含 '..' 但无路径分隔符）应可读取: %q", legacy)
	}
	bad := []string{
		"memo_../../etc/passwd",
		"memo_xxx/../../x",
		`memo_xxx\..\..\x`,
		"../memo_x.jpg",
		"",
	}
	for _, name := range bad {
		if isSafeMemoFileName(name) {
			t.Fatalf("路径穿越 / 非法文件名应被拒绝: %q", name)
		}
	}
}

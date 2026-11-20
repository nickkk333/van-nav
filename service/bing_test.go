package service

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveBingWallpaperDailyCopy(t *testing.T) {
	// 壁纸目录是相对路径 ./data，切到临时目录后测试不会碰到真实数据
	t.Chdir(t.TempDir())

	path, err := saveBingWallpaperDailyCopy([]byte("day one"), ".jpg")
	if err != nil {
		t.Fatalf("保存每日壁纸副本失败: %v", err)
	}
	if want := filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+".jpg"); path != want {
		t.Fatalf("副本路径 = %q, want %q", path, want)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "day one" {
		t.Fatalf("副本内容 = %q, err = %v", data, err)
	}

	// 旧扩展名的副本应被清理，同前缀的普通文件不受影响
	stale := filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+".png")
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	keep := filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+"说明.txt")
	if err := os.WriteFile(keep, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := saveBingWallpaperDailyCopy([]byte("day two"), ".jpg"); err != nil {
		t.Fatalf("覆盖每日壁纸副本失败: %v", err)
	}
	if data, err := os.ReadFile(path); err != nil || string(data) != "day two" {
		t.Fatalf("覆盖后副本内容 = %q, err = %v", data, err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("旧扩展名副本未被清理: %v", err)
	}
	if _, err := os.Stat(keep); err != nil {
		t.Fatalf("同前缀普通文件被误删: %v", err)
	}
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("临时文件未清理: %v", err)
	}
}

func TestEnsureBingWallpaperDailyCopy(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.MkdirAll(BingWallpaperDir, 0o755); err != nil {
		t.Fatal(err)
	}

	source := filepath.Join(BingWallpaperDir, BingWallpaperBaseName+"-20261120-测试.jpg")
	if err := os.WriteFile(source, []byte("wallpaper"), 0o644); err != nil {
		t.Fatal(err)
	}
	ensureBingWallpaperDailyCopy(source)

	target := filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+".jpg")
	if data, err := os.ReadFile(target); err != nil || string(data) != "wallpaper" {
		t.Fatalf("每日壁纸副本内容 = %q, err = %v", data, err)
	}
}

package service

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

const (
	// UploadDir 上传图片的存放目录（在 data 目录下，docker 部署时挂载 ./data 即可持久化）
	UploadDir = "./data/images"
	// UploadUrlPrefix 上传图片的访问前缀，需与 main.go 中的路由保持一致
	UploadUrlPrefix = "/api/uploadedImage/"
	// MaxUploadSize 单张图片大小上限：5MB
	MaxUploadSize = 5 << 20
	// uploadNamePrefix 上传文件名前缀，用于识别哪些文件是后台上传的
	uploadNamePrefix = "upload_"
	// uploadExtTips 允许上传的图片格式提示
	uploadExtTips = "png / jpg / jpeg / webp / gif / svg / ico"
)

// allowedUploadExt 允许上传的图片后缀
var allowedUploadExt = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".svg":  true,
	".ico":  true,
}

// SaveUploadedImage 保存上传的图片，返回可直接访问的 url
func SaveUploadedImage(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Size == 0 {
		return "", fmt.Errorf("请选择要上传的图片")
	}
	if file.Size > MaxUploadSize {
		return "", fmt.Errorf("图片大小不能超过 %d MB", MaxUploadSize>>20)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedUploadExt[ext] {
		return "", fmt.Errorf("不支持的图片格式 %s，仅支持 %s", ext, uploadExtTips)
	}
	utils.PathExistsOrCreate(UploadDir)
	name := fmt.Sprintf("%s%d%s", uploadNamePrefix, time.Now().UnixNano(), ext)
	if err := copyUploadedFile(file, filepath.Join(UploadDir, name)); err != nil {
		logger.LogError("保存上传图片失败: %s", err)
		return "", fmt.Errorf("保存图片失败")
	}
	logger.LogInfo("图片上传成功: %s", name)
	return UploadUrlPrefix + name, nil
}

// copyUploadedFile 把上传的文件写入目标路径
func copyUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, src)
	return err
}

// GetUploadedImagePath 校验并返回本地图片的磁盘路径（后台上传的图片、搜索引擎 logo 等）
func GetUploadedImagePath(name string) (string, bool) {
	if LocalImageName(UploadUrlPrefix+name) == "" {
		return "", false
	}
	path := filepath.Join(UploadDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

// LocalImageName 从 url 中取出本地图片文件名（上传的图片、按名称保存的 logo 等），不合法时返回空字符串
func LocalImageName(url string) string {
	if !strings.HasPrefix(url, UploadUrlPrefix) {
		return ""
	}
	name := strings.TrimPrefix(url, UploadUrlPrefix)
	if !isSafeImageFileName(name) {
		return ""
	}
	return name
}

// LocalImageExist 判断 url 是否指向本机实际存在的本地图片（上传的图片、按名称保存的 logo 等）
// 换机器导入备份后，本地图片文件并不存在，需要用这个判断决定是否重新获取图片
func LocalImageExist(url string) bool {
	name := LocalImageName(url)
	if name == "" {
		return false
	}
	_, ok := GetUploadedImagePath(name)
	return ok
}

// isSafeImageFileName 校验文件名：只能是 UploadDir 下的文件名，且是允许的图片格式，避免路径穿越
func isSafeImageFileName(name string) bool {
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	return allowedUploadExt[strings.ToLower(filepath.Ext(name))]
}

// UploadedImageName 从 url 中取出后台上传的图片文件名，非上传图片返回空字符串（用于清理上传图片）
func UploadedImageName(url string) string {
	name := LocalImageName(url)
	if name == "" || !strings.HasPrefix(name, uploadNamePrefix) {
		return ""
	}
	return name
}

// RemoveUploadedImage 删除后台上传的图片，外链地址不做处理
func RemoveUploadedImage(url string) {
	name := UploadedImageName(url)
	if name == "" {
		return
	}
	if err := os.Remove(filepath.Join(UploadDir, name)); err != nil && !os.IsNotExist(err) {
		logger.LogError("删除上传图片失败: %s", err)
	}
}

// RemoveLocalImage 删除本地保存的图片（上传的图片、搜索引擎 logo 等），外链地址不做处理
func RemoveLocalImage(url string) {
	name := LocalImageName(url)
	if name == "" {
		return
	}
	if err := os.Remove(filepath.Join(UploadDir, name)); err != nil && !os.IsNotExist(err) {
		logger.LogError("删除本地图片失败: %s", err)
	}
}

// SaveSearchEngineLogo 把搜索引擎的 logo 外链下载保存到本地，文件名使用搜索引擎名称，返回可访问 url
// 传入的 logo 不是外链（内置图标文件名或已保存到本地的地址）时不做处理，返回空字符串
func SaveSearchEngineLogo(engineName string, logoUrl string) (string, error) {
	logoUrl = strings.TrimSpace(logoUrl)
	if logoUrl == "" {
		return "", fmt.Errorf("logo 地址不能为空")
	}
	if !strings.HasPrefix(logoUrl, "http://") && !strings.HasPrefix(logoUrl, "https://") {
		return "", nil
	}
	data, ext, err := utils.DownloadImage(logoUrl)
	if err != nil {
		logger.LogError("下载搜索引擎 logo 失败: %s", err)
		return "", fmt.Errorf("下载 logo 失败：%s", err)
	}
	if int64(len(data)) > MaxUploadSize {
		return "", fmt.Errorf("图片大小不能超过 %d MB", MaxUploadSize>>20)
	}
	utils.PathExistsOrCreate(UploadDir)
	name := SafeImageFileName(engineName) + ext
	if err := os.WriteFile(filepath.Join(UploadDir, name), data, 0o644); err != nil {
		logger.LogError("保存搜索引擎 logo 失败: %s", err)
		return "", fmt.Errorf("保存 logo 失败")
	}
	logger.LogInfo("搜索引擎 logo 已保存到本地: %s", name)
	return UploadUrlPrefix + name, nil
}

// SafeImageFileName 把搜索引擎名称清洗成安全的文件名（去掉路径分隔符、Windows 非法字符与首尾空白）
func SafeImageFileName(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 || strings.ContainsRune(`/\:*?"<>|`, r) {
			return -1
		}
		return r
	}, strings.TrimSpace(name))
	cleaned = strings.Trim(cleaned, " .")
	// 连续的点会被图片访问校验当成路径穿越（含 ".."），折叠成单个点，保证「存得下也读得出」
	for strings.Contains(cleaned, "..") {
		cleaned = strings.ReplaceAll(cleaned, "..", ".")
	}
	cleaned = strings.Trim(cleaned, " .")
	if cleaned == "" {
		return "engine"
	}
	// 文件名不宜过长（多数文件系统单个文件名上限 255 字节）
	if runes := []rune(cleaned); len(runes) > 50 {
		cleaned = string(runes[:50])
	}
	return cleaned
}

// CleanupReplacedUploadedImages 设置更新后清理被替换掉的上传图片，避免无用文件堆积
func CleanupReplacedUploadedImages(oldSetting types.Setting, newSetting types.Setting) {
	inUse := []string{
		newSetting.BackgroundImage,
		newSetting.Favicon,
		newSetting.Logo192,
		newSetting.Logo512,
	}
	for _, url := range []string{
		oldSetting.BackgroundImage,
		oldSetting.Favicon,
		oldSetting.Logo192,
		oldSetting.Logo512,
	} {
		if url == "" || utils.In(url, inUse) {
			continue
		}
		RemoveUploadedImage(url)
	}
}

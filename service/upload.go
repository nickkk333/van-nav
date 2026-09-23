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

// GetUploadedImagePath 校验并返回上传图片的磁盘路径
func GetUploadedImagePath(name string) (string, bool) {
	if UploadedImageName(UploadUrlPrefix+name) == "" {
		return "", false
	}
	path := filepath.Join(UploadDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

// UploadedImageName 从 url 中取出上传图片的文件名，非本地上传的图片返回空字符串
func UploadedImageName(url string) string {
	if !strings.HasPrefix(url, UploadUrlPrefix) {
		return ""
	}
	name := strings.TrimPrefix(url, UploadUrlPrefix)
	// 只允许 UploadDir 下的、带固定前缀的图片文件名，避免路径穿越
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) {
		return ""
	}
	if !strings.HasPrefix(name, uploadNamePrefix) {
		return ""
	}
	if !allowedUploadExt[strings.ToLower(filepath.Ext(name))] {
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

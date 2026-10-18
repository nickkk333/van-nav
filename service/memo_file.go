package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/utils"
)

// ==================== 便签附件（上传文件 / 粘贴图片） ====================

const (
	// MemoFileDir 便签附件的存放目录（在 data 目录下，docker 部署时挂载 ./data 即可持久化）
	MemoFileDir = "./data/memo-files"
	// MemoFileUrlPrefix 便签附件的访问前缀，需与 main.go 中的路由保持一致
	MemoFileUrlPrefix = "/api/memoFile/"
	// MaxMemoFileSize 单个附件的大小上限：20MB
	MaxMemoFileSize = 20 << 20
	// memoFilePrefix 附件在磁盘上的文件名前缀，也用于识别哪些文件是便签上传的
	memoFilePrefix = "memo_"
	// memoFileExtTips 允许上传的文件格式提示
	memoFileExtTips = "图片（png/jpg/jpeg/webp/gif/bmp）、pdf、word、excel、ppt、" +
		"txt/md/csv/json/xml/yaml/log 与 zip/rar/7z/tar/gz 压缩包"
)

// allowedMemoFileExt 便签允许上传的文件后缀。
// 刻意不含 .svg / .html / .js 这类会被浏览器当页面执行脚本的格式：
// 附件地址是公开只读的（见 main.go），打开链接时不能有脚本执行风险
var allowedMemoFileExt = map[string]bool{
	// 图片：在便签里内联显示
	".png": true, ".jpg": true, ".jpeg": true, ".webp": true, ".gif": true, ".bmp": true,
	// 常见文档
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".ppt": true, ".pptx": true, ".csv": true, ".txt": true, ".md": true,
	".json": true, ".xml": true, ".yaml": true, ".yml": true, ".log": true,
	// 压缩包
	".zip": true, ".rar": true, ".7z": true, ".tar": true, ".gz": true,
}

// MemoFileResult 便签附件上传结果
type MemoFileResult struct {
	// Name 附件在服务端保存的文件名（附件地址的最后一段）
	Name string
	// Url 可直接访问的附件地址
	Url string
	// Original 上传时的原始文件名（前台在附件卡片上显示）
	Original string
	// Size 文件大小（字节）
	Size int64
	// IsImage 是否图片：前台据此决定内联显示图片还是显示「图标 + 文件名」卡片
	IsImage bool
}

// IsMemoImageName 判断便签附件文件名是不是内联显示的图片
func IsMemoImageName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".gif", ".bmp":
		return true
	}
	return false
}

// SaveMemoFile 保存便签上传的附件（图片或文档），返回保存后的文件名与原始文件名
func SaveMemoFile(file *multipart.FileHeader) (MemoFileResult, error) {
	ext, err := validateMemoFile(file)
	if err != nil {
		return MemoFileResult{}, err
	}
	original := memoFileDisplayName(file.Filename)
	base := SafeImageFileName(strings.TrimSuffix(original, filepath.Ext(original)), "file")
	// 文件名带时间戳与随机串：既避免重名覆盖，也让公开的附件地址无法被枚举
	name := fmt.Sprintf("%s%d_%s_%s%s", memoFilePrefix, time.Now().UnixNano(), memoFileRandom(), base, ext)
	utils.PathExistsOrCreate(MemoFileDir)
	if err := copyUploadedFile(file, filepath.Join(MemoFileDir, name)); err != nil {
		logger.LogError("保存便签附件失败: %s", err)
		return MemoFileResult{}, fmt.Errorf("保存文件失败")
	}
	logger.LogInfo("便签附件上传成功: %s (%d 字节)", name, file.Size)
	return MemoFileResult{
		Name:     name,
		Url:      MemoFileUrlPrefix + name,
		Original: original,
		Size:     file.Size,
		IsImage:  IsMemoImageName(name),
	}, nil
}

// validateMemoFile 校验上传的附件（非空、大小、后缀），返回小写后缀
func validateMemoFile(file *multipart.FileHeader) (string, error) {
	if file == nil || file.Size == 0 {
		return "", fmt.Errorf("请选择要上传的文件")
	}
	if file.Size > MaxMemoFileSize {
		return "", fmt.Errorf("文件大小不能超过 %d MB", MaxMemoFileSize>>20)
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedMemoFileExt[ext] {
		return "", fmt.Errorf("不支持的文件格式 %s，仅支持 %s", ext, memoFileExtTips)
	}
	return ext, nil
}

// GetMemoFilePath 校验并返回便签附件的磁盘路径
func GetMemoFilePath(name string) (string, bool) {
	if !isSafeMemoFileName(name) {
		return "", false
	}
	path := filepath.Join(MemoFileDir, name)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return "", false
	}
	return path, true
}

// MemoFileDownloadName 取附件下载时建议使用的文件名：
// 磁盘名形如 memo_<时间戳>_<随机串>_<原始名>，这里去掉前两段还原成原始名
func MemoFileDownloadName(name string) string {
	if !strings.HasPrefix(name, memoFilePrefix) {
		return name
	}
	rest := strings.TrimPrefix(name, memoFilePrefix)
	for i := 0; i < 2; i++ {
		idx := strings.Index(rest, "_")
		if idx < 0 {
			return name
		}
		rest = rest[idx+1:]
	}
	if rest == "" {
		return name
	}
	return rest
}

// isSafeMemoFileName 校验文件名：只能是 MemoFileDir 下的文件、后缀在允许范围内，避免路径穿越
func isSafeMemoFileName(name string) bool {
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) || strings.Contains(name, "..") {
		return false
	}
	if !strings.HasPrefix(name, memoFilePrefix) {
		return false
	}
	return allowedMemoFileExt[strings.ToLower(filepath.Ext(name))]
}

// memoFileDisplayName 取出用于展示的原始文件名：去掉路径与控制字符，超长时截断
func memoFileDisplayName(raw string) string {
	// 有些浏览器会把整个路径当文件名传上来
	name := strings.ReplaceAll(raw, "\\", "/")
	if idx := strings.LastIndex(name, "/"); idx >= 0 {
		name = name[idx+1:]
	}
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if runes := []rune(name); len(runes) > 120 {
		name = string(runes[:120])
	}
	if name == "" {
		return "附件"
	}
	return name
}

// memoFileRandom 生成 4 字节随机数的十六进制串（8 位），让附件地址不可枚举
func memoFileRandom() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		logger.LogError("生成附件随机名失败: %s", err)
		return fmt.Sprintf("%08x", uint32(time.Now().UnixNano()))
	}
	return hex.EncodeToString(b)
}

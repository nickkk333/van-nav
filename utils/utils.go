package utils

import (
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// CheckErr 记录错误并把 err 原样返回：
// 调用方可以继续按老写法当语句用，也可以 `if err := ...; utils.CheckErr(err) { return err }` 向上传播
func CheckErr(err error) error {
	if err != nil {
		logger.LogError("捕获到错误：%s, 堆栈信息：%s", err, string(debug.Stack()))
	}
	return err
}

// CheckTxErr 记录错误并回滚事务，同时把 err 返回给调用方：
// 回滚后调用方必须直接 return，不能再继续用这个事务（老代码只回滚不中断，后续语句全在已回滚的事务上执行）
func CheckTxErr(err error, tx *sql.Tx) error {
	if err != nil {
		logger.LogError("出现事务异常，回滚事务: %s, 堆栈信息：%s", err, string(debug.Stack()))
		if tx != nil {
			if err2 := tx.Rollback(); err2 != nil {
				CheckErr(err2)
			}
		}
	}
	return err
}

func In(target string, str_array []string) bool {
	for _, element := range str_array {
		if target == element {
			return true
		}
	}
	return false
}

// imageHTTPClient 抓图用的 HTTP 客户端：保持 TLS 证书校验（默认 Transport 即校验）。
// 之前全局 InsecureSkipVerify 会让中间人可替换图片内容；
// 目标站点证书异常时走调用方的空结果/兜底逻辑，不再静默跳过校验。
func imageHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func GetImgBase64FromUrl(url string) string {
	imgUrl := url
	//获取远端图片
	req, err := http.NewRequest("GET", imgUrl, nil)
	if err != nil {
		logger.LogError("图片地址不合法，跳过缓存：%s", err)
		return ""
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36")
	client := imageHTTPClient()
	res, err := client.Do(req)
	if err != nil {
		logger.LogError("下载远程图片失败，跳过缓存：%s", err)
		return ""
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		logger.LogError("下载远程图片失败（状态码 %d），跳过缓存：%s", res.StatusCode, imgUrl)
		return ""
	}

	// 读取获取的[]byte数据（限制大小，避免异常大图占满内存）
	data, err := io.ReadAll(io.LimitReader(res.Body, downloadImageLimit))
	if err != nil {
		logger.LogError("读取远程图片失败，跳过缓存：%s", err)
		return ""
	}
	if len(data) == 0 {
		return ""
	}

	imageBase64 := base64.StdEncoding.EncodeToString(data)
	return imageBase64
}

// GetSuffixFromUrl 取 url 路径末尾的后缀（含点），例如 https://a.b/x.png -> .png
// 老实现 url[strings.LastIndex(url, "."):] 在 LastIndex 返回 -1 时直接 panic，
// 且会把域名当成后缀（https://example.com -> ".com"），这里只认路径部分的后缀
func GetSuffixFromUrl(url string) string {
	// 去掉查询串与 hash，否则 .png?v=1 会被当成整个后缀
	path := url
	if i := strings.IndexAny(path, "?#"); i >= 0 {
		path = path[:i]
	}
	ext := filepath.Ext(path)
	// 只保留常见的图片后缀，域名（.com/.cn）之类不算
	switch strings.ToLower(ext) {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico", ".bmp":
		return ext
	}
	return ""
}

// downloadImageLimit 单张下载图片的大小上限，避免异常大图占满内存
const downloadImageLimit = 8 << 20

// DownloadImage 下载图片，返回图片内容与扩展名（优先按 Content-Type 判断，其次按 url 后缀）
func DownloadImage(imgUrl string) ([]byte, string, error) {
	data, ext, _, err := DownloadImageWithType(imgUrl)
	return data, ext, err
}

// DownloadImageWithType 下载图片，返回图片内容、扩展名与响应的 Content-Type
// （扩展名优先按 Content-Type 判断，其次按 url 后缀）
func DownloadImageWithType(imgUrl string) ([]byte, string, string, error) {
	req, err := http.NewRequest("GET", imgUrl, nil)
	if err != nil {
		return nil, "", "", err
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36")
	client := imageHTTPClient()
	res, err := client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("下载图片失败，状态码 %d", res.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(res.Body, downloadImageLimit))
	if err != nil {
		return nil, "", "", err
	}
	if len(data) == 0 {
		return nil, "", "", fmt.Errorf("下载到的图片内容为空")
	}
	contentType := res.Header.Get("Content-Type")
	return data, imageExt(contentType, imgUrl), contentType, nil
}

// IsImageContentType 判断响应的 Content-Type 是否是图片
func IsImageContentType(contentType string) bool {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	return strings.HasPrefix(ct, "image/")
}


// imageExt 根据 Content-Type 或 url 后缀推断图片扩展名，无法判断时按 png 处理
func imageExt(contentType string, imgUrl string) string {
	ct := strings.ToLower(strings.TrimSpace(strings.Split(contentType, ";")[0]))
	switch ct {
	case "image/png":
		return ".png"
	case "image/jpeg", "image/jpg":
		return ".jpg"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "image/svg+xml":
		return ".svg"
	case "image/x-icon", "image/vnd.microsoft.icon", "image/ico":
		return ".ico"
	}
	suffix := strings.ToLower(filepath.Ext(strings.Split(imgUrl, "?")[0]))
	switch suffix {
	case ".png", ".jpg", ".jpeg", ".gif", ".webp", ".svg", ".ico":
		return suffix
	}
	return ".png"
}
func GetMIME(suffix string) string {
	var t string = "image/x-icon"
	if suffix == ".svg" {
		t = "image/svg+xml"
	}
	if suffix == ".png" {
		t = "image/png"
	}
	return t
}

func PathExistsOrCreate(path string) {
	_, err := os.Stat(path)
	if err == nil {
		return
	}
	os.Mkdir(path, os.ModePerm)
}

func GenerateId() int {
	// 生成一个随机 id
	id := int(time.Now().Unix())
	return id
}

// bcryptCost 密码哈希强度：10 是安全与性能的常用平衡点（单次约 60~100ms）
// 登录/改密是低频操作，这个耗时可接受；调太高会让登录接口变慢
const bcryptCost = 10

// HashPassword 对明文密码做 bcrypt 哈希，失败返回 error（调用方必须处理，不能存明文兜底）
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword 校验明文密码与哈希是否匹配
// 兼容老数据：库里还是明文（不以 $2a$ 开头）时按明文比对，命中后调用方负责升级为哈希
func CheckPassword(password, hash string) bool {
	if hash == "" || password == "" {
		return false
	}
	if !IsPasswordHash(hash) {
		return password == hash
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// IsPasswordHash 判断存量值是否已经是 bcrypt 哈希（长度 60 的标准哈希串）
// 只看前缀不够："$2a$" 本身不是合法哈希，必须再校验长度，避免把脏数据当成哈希
func IsPasswordHash(value string) bool {
	if len(value) != 60 {
		return false
	}
	return strings.HasPrefix(value, "$2a$") || strings.HasPrefix(value, "$2b$") || strings.HasPrefix(value, "$2y$")
}

func FilterHideTools(tools []types.Tool, cates []types.Catelog) []types.Tool {
	result := make([]types.Tool, 0)
	var hideCates []string
	// 提取出需要隐藏的分类
	for _, cate := range cates {
		if cate.Hide {
			hideCates = append(hideCates, cate.Name)
		}
	}
	// 过滤工具
	for _, tool := range tools {
		if !tool.Hide && !In(tool.Catelog, hideCates) {
			result = append(result, tool)
		}
	}
	return result
}

func FilterHideCates(cates []types.Catelog) []types.Catelog {
	result := make([]types.Catelog, 0)
	for _, cate := range cates {
		if !cate.Hide {
			result = append(result, cate)
		}
	}
	return result
}

package utils

import (
	"crypto/tls"
	"database/sql"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

func CheckErr(err error) {
	if err != nil {
		logger.LogError("捕获到错误：%s, 堆栈信息：%s", err, string(debug.Stack()))
	}
}

func CheckTxErr(err error, tx *sql.Tx) {
	if err != nil {
		logger.LogError("出现事务异常，回滚事务: %s, 堆栈信息：%s", err, string(debug.Stack()))
		err2 := tx.Rollback()
		CheckErr(err2)
	}
}

func In(target string, str_array []string) bool {
	for _, element := range str_array {
		if target == element {
			return true
		}
	}
	return false
}

// GetImgBase64FromUrl 下载远程图片并转成 base64，失败时返回空字符串
// 只用于把外链图片缓存到数据库，网络不通、地址失效等情况属于正常现象，
// 这里返回空字符串、由调用方跳过缓存即可，不能当成代码异常打印堆栈
func GetImgBase64FromUrl(url string) string {
	imgUrl := url
	//获取远端图片
	req, err := http.NewRequest("GET", imgUrl, nil)
	if err != nil {
		logger.LogError("图片地址不合法，跳过缓存：%s", err)
		return ""
	}
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36")
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
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

func GetSuffixFromUrl(url string) string {
	suffix := url[strings.LastIndex(url, "."):]
	return suffix
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
	client := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, "", "", err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, "", "", fmt.Errorf("下载图片失败，状态码 %d", res.StatusCode)
	}
	data, err := ioutil.ReadAll(io.LimitReader(res.Body, downloadImageLimit))
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

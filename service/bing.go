package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/utils"
)

const (
	// BingWallpaperDir 必应每日壁纸的存放目录（与 nav.db 同级，docker 部署挂载 ./data 即可持久化）
	BingWallpaperDir = "./data"
	// BingWallpaperBaseName 必应每日壁纸的文件名（不含扩展名，扩展名按下载到的图片格式决定）
	BingWallpaperBaseName = "必应壁纸"
	// bingWallpaperApi 必应每日壁纸信息接口，返回当天的壁纸信息
	bingWallpaperApi = "https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1&mkt=zh-CN"
	// bingWallpaperHost 必应图片的域名前缀（接口返回的图片地址是相对路径）
	bingWallpaperHost = "https://www.bing.com"
	// bingWallpaperUserAgent 请求必应接口使用的浏览器标识
	bingWallpaperUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36"
	// bingWallpaperTimeout 请求必应接口的超时时间
	bingWallpaperTimeout = 8 * time.Second
	// bingWallpaperRetryInterval 下载失败后的重试间隔，避免离线时每次请求都去访问必应
	bingWallpaperRetryInterval = 10 * time.Minute
)

var (
	// bingWallpaperMu 保证同一时间只有一个下载任务，避免并发重复下载
	bingWallpaperMu sync.Mutex
	// lastBingWallpaperTry 上一次尝试下载的时间
	lastBingWallpaperTry time.Time
)

// bingWallpaperInfo 必应接口返回的单张壁纸信息
type bingWallpaperInfo struct {
	// Url 标准尺寸壁纸地址（相对路径）
	Url string `json:"url"`
	// Urlbase 壁纸地址前缀（相对路径），加上尺寸后缀可以获取其他尺寸
	Urlbase string `json:"urlbase"`
	// Title 壁纸标题
	Title string `json:"title"`
}

// DownloadBingWallpaper 下载必应每日壁纸到 data 目录，保存为「必应壁纸 + 图片格式后缀」，前台通过 /api/bingWallpaper 读取
// 当天已经下载过、或距离上次失败尝试过近时直接跳过；下载失败只记录日志，不影响服务启动
func DownloadBingWallpaper() {
	bingWallpaperMu.Lock()
	defer bingWallpaperMu.Unlock()
	if BingWallpaperDownloadedToday() {
		logger.LogInfo("必应每日壁纸今天已经下载过，跳过下载")
		return
	}
	if time.Since(lastBingWallpaperTry) < bingWallpaperRetryInterval {
		return
	}
	lastBingWallpaperTry = time.Now()

	info, err := fetchBingWallpaperInfo()
	if err != nil {
		logger.LogError("获取必应每日壁纸信息失败: %s", err)
		return
	}
	for _, url := range bingWallpaperUrls(info) {
		logger.LogInfo("开始下载必应每日壁纸: %s", url)
		data, ext, err := utils.DownloadImage(url)
		if err != nil {
			logger.LogError("下载必应每日壁纸失败(%s): %s", url, err)
			continue
		}
		utils.PathExistsOrCreate(BingWallpaperDir)
		name := BingWallpaperBaseName + ext
		if err := os.WriteFile(filepath.Join(BingWallpaperDir, name), data, 0o644); err != nil {
			logger.LogError("保存必应每日壁纸失败: %s", err)
			return
		}
		// 图片格式可能变化，清掉上一次保存的文件，保证 data 目录里只有一张必应壁纸
		removeBingWallpaperExcept(name)
		logger.LogInfo("必应每日壁纸已保存到本地: %s %s", filepath.Join(BingWallpaperDir, name), info.Title)
		return
	}
}

// GetBingWallpaperPath 获取本地必应每日壁纸的磁盘路径，文件不存在时返回 false
func GetBingWallpaperPath() (string, bool) {
	files := bingWallpaperFiles()
	if len(files) == 0 {
		return "", false
	}
	return files[0], true
}

// BingWallpaperDownloadedToday 判断必应每日壁纸是否已经下载过（每天只需要下载一次）
func BingWallpaperDownloadedToday() bool {
	path, ok := GetBingWallpaperPath()
	if !ok {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	year, month, day := info.ModTime().Date()
	nowYear, nowMonth, nowDay := time.Now().Date()
	return year == nowYear && month == nowMonth && day == nowDay
}

// bingWallpaperFiles 列出 data 目录下所有必应壁纸文件
func bingWallpaperFiles() []string {
	matches, err := filepath.Glob(filepath.Join(BingWallpaperDir, BingWallpaperBaseName+".*"))
	if err != nil {
		return nil
	}
	files := make([]string, 0, len(matches))
	for _, path := range matches {
		info, err := os.Stat(path)
		if err != nil || info.IsDir() {
			continue
		}
		if !allowedUploadExt[strings.ToLower(filepath.Ext(path))] {
			continue
		}
		files = append(files, path)
	}
	return files
}

// removeBingWallpaperExcept 删除除 keep 之外的历史必应壁纸文件
func removeBingWallpaperExcept(keep string) {
	for _, path := range bingWallpaperFiles() {
		if filepath.Base(path) == keep {
			continue
		}
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			logger.LogError("删除旧的必应每日壁纸失败: %s", err)
		}
	}
}

// fetchBingWallpaperInfo 请求必应接口，获取当天壁纸的信息
func fetchBingWallpaperInfo() (bingWallpaperInfo, error) {
	req, err := http.NewRequest("GET", bingWallpaperApi, nil)
	if err != nil {
		return bingWallpaperInfo{}, err
	}
	req.Header.Add("User-Agent", bingWallpaperUserAgent)
	client := &http.Client{Timeout: bingWallpaperTimeout}
	res, err := client.Do(req)
	if err != nil {
		return bingWallpaperInfo{}, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return bingWallpaperInfo{}, fmt.Errorf("请求必应接口失败，状态码 %d", res.StatusCode)
	}
	var result struct {
		Images []bingWallpaperInfo `json:"images"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&result); err != nil {
		return bingWallpaperInfo{}, err
	}
	if len(result.Images) == 0 {
		return bingWallpaperInfo{}, fmt.Errorf("必应接口没有返回壁纸信息")
	}
	info := result.Images[0]
	if info.Url == "" && info.Urlbase == "" {
		return bingWallpaperInfo{}, fmt.Errorf("必应接口返回的壁纸地址为空")
	}
	return info, nil
}

// bingWallpaperUrls 必应壁纸的候选地址：优先 4K 大图，失败时回落到接口返回的标准尺寸图
func bingWallpaperUrls(info bingWallpaperInfo) []string {
	urls := make([]string, 0, 2)
	if info.Urlbase != "" {
		urls = append(urls, bingWallpaperHost+info.Urlbase+"_UHD.jpg")
	}
	if info.Url != "" {
		urls = append(urls, bingWallpaperHost+info.Url)
	}
	return urls
}

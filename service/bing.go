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
	// BingWallpaperDailyBaseName 每日壁纸副本的文件名（不含扩展名）：
	// 每次下载/更新后都会把最新壁纸复制成「每日壁纸.扩展名」，覆盖旧文件，方便其它程序固定读取同一个路径
	BingWallpaperDailyBaseName = "每日壁纸"
	// bingWallpaperApi 必应每日壁纸信息接口，返回当天的壁纸信息
	bingWallpaperApi = "https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1&mkt=zh-CN"
	// bingWallpaperHost 必应图片的域名前缀（接口返回的图片地址是相对路径）
	bingWallpaperHost = "https://www.bing.com"
	// bingWallpaperUserAgent 请求必应接口使用的浏览器标识
	bingWallpaperUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/100.0.4896.88 Safari/537.36"
	// bingWallpaperTimeout 请求必应接口的超时时间
	bingWallpaperTimeout = 8 * time.Second
	// bingWallpaperFetchHour 每天固定拉取壁纸的整点（本地时区，0-23）
	bingWallpaperFetchHour = 8
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

// bingWallpaperFileName 生成壁纸文件名（不含扩展名）：必应壁纸-YYYYMMDD-图片描述
// 例：必应壁纸-20261004-秋日晨曦中的黄山；描述清洗掉非法字符，为空时只保留日期段
// 格式与 BingWallpaperTitle 的解析保持一致，保证存下去能再读出来
func bingWallpaperFileName(title string) string {
	name := BingWallpaperBaseName + "-" + time.Now().Format("20060102")
	// 必应标题可能带 / : * ? 等文件名非法字符，复用统一的清洗逻辑（去非法字符、防路径穿越、限长）
	if desc := SafeImageFileName(title, ""); desc != "" {
		name += "-" + desc
	}
	return name
}

// DownloadBingWallpaper 下载必应每日壁纸到 data 目录
// 保存为「必应壁纸-YYYYMMDD-图片描述 + 图片格式后缀」，历史壁纸保留不删；
// 当天已经下载过、或距离上次失败尝试过近时直接跳过；下载失败只记录日志，不影响服务启动
func DownloadBingWallpaper() {
	bingWallpaperMu.Lock()
	defer bingWallpaperMu.Unlock()
	if BingWallpaperDownloadedToday() {
		// 升级后首次启动时，历史壁纸可能还没有固定文件名的副本，这里补一份
		if path, ok := GetBingWallpaperPath(); ok {
			ensureBingWallpaperDailyCopy(path)
		}
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
		name := bingWallpaperFileName(info.Title) + ext
		if err := os.WriteFile(filepath.Join(BingWallpaperDir, name), data, 0o644); err != nil {
			logger.LogError("保存必应每日壁纸失败: %s", err)
			return
		}
		// 壁纸按日期命名（必应壁纸-YYYYMMDD-描述），历史壁纸一律保留、不做清理
		logger.LogInfo("必应每日壁纸已保存到本地: %s %s", filepath.Join(BingWallpaperDir, name), info.Title)
		// 同时覆盖一份固定文件名的「每日壁纸.扩展名」，方便其它程序按固定路径读取当天壁纸
		if dailyPath, err := saveBingWallpaperDailyCopy(data, ext); err != nil {
			logger.LogError("保存每日壁纸副本失败: %s", err)
		} else {
			logger.LogInfo("每日壁纸副本已更新: %s", dailyPath)
		}
		return
	}
}

// saveBingWallpaperDailyCopy 把当天壁纸复制为固定文件「每日壁纸.扩展名」，覆盖旧文件。
// 下载到的图片格式变化时（例如 jpg 变成 png），顺带删除其它扩展名的旧副本，保证只保留一个。
func saveBingWallpaperDailyCopy(data []byte, ext string) (string, error) {
	utils.PathExistsOrCreate(BingWallpaperDir)
	path := filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+ext)
	// 先写临时文件再原子替换，避免其它程序正好读取时拿到只写了一半的图片
	tmp := path + ".tmp"
	defer os.Remove(tmp)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		return "", err
	}
	// 只清理「每日壁纸.图片后缀」这类副本，不碰用户放在 data 目录里的其它文件
	matches, err := filepath.Glob(filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+".*"))
	if err != nil {
		return path, nil
	}
	for _, old := range matches {
		if old == path {
			continue
		}
		info, err := os.Stat(old)
		if err != nil || info.IsDir() {
			continue
		}
		if allowedUploadExt[strings.ToLower(filepath.Ext(old))] {
			_ = os.Remove(old)
		}
	}
	return path, nil
}

// ensureBingWallpaperDailyCopy 检查固定文件「每日壁纸.扩展名」是否与最新壁纸一致：
// 副本不存在、或比最新壁纸旧时才复制，避免前台每次请求壁纸都重复写入整张大图。
func ensureBingWallpaperDailyCopy(path string) {
	ext := strings.ToLower(filepath.Ext(path))
	sourceInfo, err := os.Stat(path)
	if err != nil {
		return
	}
	target := filepath.Join(BingWallpaperDir, BingWallpaperDailyBaseName+ext)
	if targetInfo, err := os.Stat(target); err == nil &&
		!targetInfo.ModTime().Before(sourceInfo.ModTime()) &&
		targetInfo.Size() == sourceInfo.Size() {
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		logger.LogError("读取必应每日壁纸失败: %s", err)
		return
	}
	dailyPath, err := saveBingWallpaperDailyCopy(data, ext)
	if err != nil {
		logger.LogError("更新每日壁纸副本失败: %s", err)
		return
	}
	logger.LogInfo("每日壁纸副本已更新: %s", dailyPath)
}

// GetBingWallpaperPath 获取本地必应每日壁纸的磁盘路径，文件不存在时返回 false
// 历史壁纸按日期递增保留，因此这里取日期最新的那一张
func GetBingWallpaperPath() (string, bool) {
	files := bingWallpaperFiles()
	if len(files) == 0 {
		return "", false
	}
	newest := files[0]
	newestDate := bingWallpaperDateOf(newest)
	for _, path := range files[1:] {
		if date := bingWallpaperDateOf(path); date.After(newestDate) {
			newest = path
			newestDate = date
		}
	}
	return newest, true
}

// BingWallpaperTitle 从当前壁纸文件名解析图片描述，供前台搜索框 placeholder 显示
// 命名规则：必应壁纸-YYYYMMDD-图片描述.后缀（见 bingWallpaperFileName）
// 没有描述、或还是旧命名（必应壁纸.后缀）时返回空字符串
func BingWallpaperTitle() string {
	path, ok := GetBingWallpaperPath()
	if !ok {
		return ""
	}
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	rest, found := strings.CutPrefix(base, BingWallpaperBaseName+"-")
	if !found {
		return "" // 旧命名：必应壁纸
	}
	// 再剥掉 YYYYMMDD- 日期段（8 位数字 + 连字符），剩下的就是图片描述
	if len(rest) >= 9 && isASCIIDigits(rest[:8]) && rest[8] == '-' {
		return rest[9:]
	}
	return ""
}

// bingWallpaperDateOf 取壁纸文件对应的日期：优先从文件名里解析 YYYYMMDD，
// 兼容旧命名（必应壁纸.后缀）时回落到文件修改时间
func bingWallpaperDateOf(path string) time.Time {
	base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	rest, found := strings.CutPrefix(base, BingWallpaperBaseName+"-")
	if found && len(rest) >= 8 && isASCIIDigits(rest[:8]) {
		if parsed, err := time.Parse("20060102", rest[:8]); err == nil {
			return parsed
		}
	}
	if info, err := os.Stat(path); err == nil {
		return info.ModTime()
	}
	return time.Time{}
}

// isASCIIDigits 判断字符串是否全部是 ASCII 数字（用于识别文件名里的 YYYYMMDD 日期段）
func isASCIIDigits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return len(s) > 0
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
// 两个模式分别匹配旧命名（必应壁纸.jpg）与新命名（必应壁纸-YYYYMMDD-描述.jpg）；
// 固定副本「每日壁纸.扩展名」不参与匹配，避免被当成最新壁纸影响标题解析；
// 不用宽泛的「必应壁纸*」，避免误伤用户手工放进 data 目录的同前缀文件
func bingWallpaperFiles() []string {
	patterns := []string{
		filepath.Join(BingWallpaperDir, BingWallpaperBaseName+".*"),
		filepath.Join(BingWallpaperDir, BingWallpaperBaseName+"-*"),
	}
	files := make([]string, 0, 2)
	for _, pattern := range patterns {
		matches, err := filepath.Glob(pattern)
		if err != nil {
			continue
		}
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
	}
	return files
}

// EnsureBingWallpaper 供进程启动与前台「每天首次访问」时调用：
// 本地已经有壁纸图片就直接跳过（只保留最省的一次下载），
// 同时确保固定文件名的「每日壁纸.扩展名」副本存在且是最新的；
// 本地还没有任何壁纸时才发起下载。下载失败只记录日志，不影响页面返回
func EnsureBingWallpaper() {
	if path, ok := GetBingWallpaperPath(); ok {
		ensureBingWallpaperDailyCopy(path)
		return
	}
	DownloadBingWallpaper()
}

// StartBingWallpaperScheduler 每天在固定整点触发一次壁纸下载
// 与 EnsureTodayWallpaper 的首次访问兜底互为补充：
// 机器长期不重启且无人访问时，靠这里的定时器把当天壁纸拉下来；
// 是否真的下载由 DownloadBingWallpaper 内部判断，重复触发是安全的
func StartBingWallpaperScheduler() {
	hour := bingWallpaperFetchHour
	for {
		now := time.Now()
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}
		<-time.After(time.Until(next))
		DownloadBingWallpaper()
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

package service

import (
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/utils"
)

// gstaticFaviconSize gstatic 图标接口请求的图标尺寸（goscraper 抓不到图标时的兜底）
const gstaticFaviconSize = 128

// imgCacheDir 外链图标缓存目录（图片出库：内容存文件不存数据库，避免 base64 撑大 nav.db）
// 与 data/images 同级，docker 部署挂载 ./data 即可持久化
const imgCacheDir = "./data/imgcache"

// normalizeSiteUrl 补全站点地址：没有协议（例如 example.com）时补上 http://
// goscraper 抓取与 gstatic 接口都需要带协议、带域名的地址
func normalizeSiteUrl(rawUrl string) string {
	rawUrl = strings.TrimSpace(rawUrl)
	if rawUrl == "" {
		return ""
	}
	// 协议相对地址（//example.com）补上 http:
	if strings.HasPrefix(rawUrl, "//") {
		return "http:" + rawUrl
	}
	if parsed, err := url.Parse(rawUrl); err == nil && parsed.Hostname() != "" {
		return rawUrl
	}
	if strings.Contains(rawUrl, "://") {
		return rawUrl
	}
	return "http://" + rawUrl
}

// GstaticFaviconUrl 使用 gstatic 的接口按站点域名取图标地址，域名解析不出来时返回空字符串
func GstaticFaviconUrl(rawUrl string) string {
	parsed, err := url.Parse(normalizeSiteUrl(rawUrl))
	if err != nil || parsed.Hostname() == "" {
		return ""
	}
	return fmt.Sprintf(
		"https://t0.gstatic.cn/faviconV2?client=SOCIAL&type=FAVICON&fallback_opts=TYPE,SIZE,URL&url=http://%s&size=%d",
		parsed.Hostname(), gstaticFaviconSize)
}

// getIcon 抓取网站图标地址：先解析页面（goscraper），页面打不开或页面里没有图标时用 gstatic 兜底，
// 都拿不到才返回空字符串
func getIcon(rawUrl string) string {
	siteUrl := normalizeSiteUrl(rawUrl)
	if siteUrl == "" {
		logger.LogError("获取图标失败：网址为空")
		return ""
	}
	logger.LogInfo("getIcon: %s", siteUrl)
	document, ok := scrapeDocument(siteUrl)
	if !ok {
		logger.LogError("getIcon: 页面抓取失败，使用 gstatic 兜底: %s", siteUrl)
		return GstaticFaviconUrl(siteUrl)
	}
	icon := absoluteIconUrl(siteUrl, document.Preview.Link, document.Preview.Icon)
	if icon == "" {
		logger.LogInfo("getIcon: 页面里没有图标，使用 gstatic 兜底: %s", siteUrl)
		return GstaticFaviconUrl(siteUrl)
	}
	logger.LogInfo("getIcon: %s", icon)
	return icon
}

// imgCachePath 外链图标缓存的文件路径：url 的 sha256 作文件名（无需扩展名，类型按内容嗅探）
func imgCachePath(rawUrl string) string {
	sum := sha256.Sum256([]byte(rawUrl))
	return filepath.Join(imgCacheDir, hex.EncodeToString(sum[:]))
}

// writeImgCache 把图片内容写入文件缓存
func writeImgCache(rawUrl string, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("图片内容为空")
	}
	utils.PathExistsOrCreate(imgCacheDir)
	return os.WriteFile(imgCachePath(rawUrl), data, 0o644)
}

// GetCachedImg 读取外链图标的本地文件缓存，没有时返回 nil
// 缓存没有（外链没下载成功、图标地址失效等）时由调用方（GetLogoImgHandler）按 404 返回，
// 前台 <img> 会触发 error 事件并显示名称首字符；这里不能返回内置的灰圈占位图，
// 否则前台会以为图片加载成功，把占位图当成网站图标显示
func GetCachedImg(rawUrl string) []byte {
	if strings.TrimSpace(rawUrl) == "" {
		return nil
	}
	data, err := os.ReadFile(imgCachePath(rawUrl))
	if err != nil || len(data) == 0 {
		// 文件缓存没有时回落读老版本的 nav_img 表（迁移失败留下的行），读到就顺手搬到文件
		return readLegacyImgCache(rawUrl)
	}
	return data
}

// readLegacyImgCache 读老版本存在 nav_img 表里的 base64 图片（出库迁移未完成时的兜底）
// 读到内容就地搬到文件缓存，让后续读取直接走磁盘
func readLegacyImgCache(rawUrl string) []byte {
	if !database.TableExists("nav_img") {
		return nil
	}
	var value string
	err := database.DB.QueryRow(`SELECT value FROM nav_img WHERE url = ?;`, url.QueryEscape(rawUrl)).Scan(&value)
	if err != nil {
		if err != sql.ErrNoRows {
			utils.CheckErr(err)
		}
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		logger.LogError("老图片缓存解码失败（url=%s）: %s", rawUrl, err)
		return nil
	}
	if err := writeImgCache(rawUrl, data); err != nil {
		logger.LogError("老图片缓存搬移到文件失败（url=%s）: %s", rawUrl, err)
	}
	return data
}

// UpdateImg 下载外链图标并写入文件缓存（图片出库：不再写 nav_img 表）
// 已有缓存时跳过；本地保存的图片（/api/uploadedImage/）直接读磁盘，无需缓存
func UpdateImg(url1 string) {
	// logo 为空时不需要缓存（前台会用名称首字符占位）
	if strings.TrimSpace(url1) == "" {
		return
	}
	// 本地保存的图片直接读磁盘，不需要再缓存
	if LocalImageName(url1) != "" {
		return
	}
	if _, err := os.Stat(imgCachePath(url1)); err == nil {
		return // 已有缓存
	}
	data, _, err := utils.DownloadImage(url1)
	if err != nil {
		logger.LogError("下载图标缓存失败（%s）: %s", url1, err)
		return
	}
	if err := writeImgCache(url1, data); err != nil {
		logger.LogError("保存图标缓存失败（%s）: %s", url1, err)
	}
}

// RemoveCachedImgIfUnused 没有别的工具引用该图标网址时，删除它的文件缓存
// （删除工具时调用，等价于老实现里按 url 删 nav_img 行，且多了引用检查，避免误删共享缓存）
func RemoveCachedImgIfUnused(rawUrl string) {
	rawUrl = strings.TrimSpace(rawUrl)
	// 本地图片文件由 RemoveToolLogoIfUnused 负责，不在这里处理
	if rawUrl == "" || LocalImageName(rawUrl) != "" {
		return
	}
	var count int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM nav_table WHERE logo = ?;`, rawUrl).Scan(&count); err != nil {
		utils.CheckErr(err)
		return
	}
	if count > 0 {
		return // 还有别的工具在用这张缓存
	}
	if err := os.Remove(imgCachePath(rawUrl)); err != nil && !os.IsNotExist(err) {
		logger.LogError("删除图标缓存失败（%s）: %s", rawUrl, err)
	}
}

// CachedImgContentType 取文件缓存图片的 Content-Type：
// 优先按内容嗅探（http.DetectContentType），svg 这类纯文本嗅探不出 image/* 时按网址后缀兜底
func CachedImgContentType(rawUrl string, data []byte) string {
	if ct := http.DetectContentType(data); strings.HasPrefix(strings.ToLower(ct), "image/") {
		return ct
	}
	return utils.GetMIME(utils.GetSuffixFromUrl(strings.Split(rawUrl, "?")[0]))
}

// MigrateImgCacheToFiles 图片出库：把老版本存在 nav_img 表里的 base64 图片搬到文件缓存，
// 全部搬完后删表并 VACUUM 回收空间（base64 有 33% 膨胀，是 nav.db 变大的主要来源之一）；
// 有搬不动的（内容损坏等）就保留表，下次启动重试，读取侧见 GetCachedImg 的老表兜底
func MigrateImgCacheToFiles() {
	if !database.TableExists("nav_img") {
		return // 新装的库没有这张表
	}
	rows, err := database.DB.Query("SELECT url, value FROM nav_img;")
	if err != nil {
		utils.CheckErr(err)
		return
	}
	moved, total, okAll := 0, 0, true
	for rows.Next() {
		total++
		var storedUrl, value string
		if err := rows.Scan(&storedUrl, &value); err != nil {
			utils.CheckErr(err)
			okAll = false
			continue
		}
		// 老代码写库时 url 用 url.QueryEscape 转义过，这里原样匹配文件名
		raw := storedUrl
		if unescaped, err := url.QueryUnescape(storedUrl); err == nil && unescaped != "" {
			raw = unescaped
		}
		data, err := base64.StdEncoding.DecodeString(value)
		if err != nil {
			logger.LogError("老图片缓存内容损坏，跳过该行（url=%s）: %s", raw, err)
			okAll = false
			continue
		}
		if err := writeImgCache(raw, data); err != nil {
			logger.LogError("老图片缓存写入文件失败（url=%s）: %s", raw, err)
			okAll = false
			continue
		}
		moved++
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		utils.CheckErr(err)
		okAll = false
	}
	if !okAll {
		logger.LogInfo("图片缓存部分迁出（%d/%d 行），保留 nav_img 表，下次启动重试", moved, total)
		return
	}
	if _, err := database.DB.Exec("DROP TABLE nav_img;"); err != nil {
		utils.CheckErr(err)
		return
	}
	// 回收 base64 占用的空间（失败只记录，不影响功能）
	if _, err := database.DB.Exec("VACUUM;"); err != nil {
		utils.CheckErr(err)
	}
	logger.LogInfo("图片缓存已迁出 nav_img 表（%d 张），nav.db 空间已回收", moved)
}
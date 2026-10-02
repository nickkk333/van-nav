package service

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// gstaticFaviconSize gstatic 图标接口请求的图标尺寸（goscraper 抓不到图标时的兜底）
const gstaticFaviconSize = 128

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

// GetImgFromDB 从图片缓存表里取图标：外链图标在保存工具时已经下载并缓存到数据库
// 缓存里没有（外链没下载成功、图标地址失效等）时返回空结构体，Value 为空字符串，
// 由调用方（GetLogoImgHandler）按 404 返回：前台 <img> 会触发 error 事件并显示名称首字符，
// 这里不能返回内置的灰圈占位图，否则前台会以为图片加载成功，把占位图当成网站图标显示
func GetImgFromDB(url1 string) types.Img {
	urlEncoded := url.QueryEscape(url1)
	sql_get_img := `
		SELECT id,url,value FROM nav_img
		WHERE url=?;
		`
	rows, err := database.DB.Query(sql_get_img, urlEncoded)
	if err != nil {
		utils.CheckErr(err)
		return types.Img{}
	}
	defer rows.Close()
	var result types.Img
	for rows.Next() {
		err = rows.Scan(&result.Id, &result.Url, &result.Value)
		utils.CheckErr(err)
	}
	return result
}

func UpdateImg(url1 string) {
	// 除了更新工具本身之外，也要更新 img 表
	// 先看有没有，有的话就不管了，没有的话就创建
	// logo 为空时不需要缓存（前台会用名称首字符占位）
	if strings.TrimSpace(url1) == "" {
		return
	}
	// 本地保存的图片直接读磁盘，不需要再缓存到数据库
	if LocalImageName(url1) != "" {
		return
	}
	urlEncoded := url.QueryEscape(url1)
	base64ImgValue := utils.GetImgBase64FromUrl(url1)
	if base64ImgValue == "" {
		return
	}
	sql_get_img := `
		SELECT * FROM nav_img
		WHERE url = ?;
		`

	rows, err := database.DB.Query(sql_get_img, urlEncoded)
	utils.CheckErr(err)
	defer rows.Close()
	if !rows.Next() {
		sql_add_img := `
		INSERT INTO nav_img (url, value)
		VALUES (?, ?);
		`
		stmt, err := database.DB.Prepare(sql_add_img)
		utils.CheckErr(err)
		_, err = stmt.Exec(urlEncoded, base64ImgValue)
		utils.CheckErr(err)
	}
}

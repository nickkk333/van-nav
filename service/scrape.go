package service

import (
	"net/url"
	"strings"
	"time"

	"github.com/mereith/nav/goscraper"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// 抓取网址信息的最长等待时间，避免前端一直转圈
const urlInfoTimeout = 10 * time.Second

// GetUrlInfo 抓取网址的名称/标题/描述/图标，供后台添加工具时自动填充
func GetUrlInfo(rawUrl string) types.UrlInfoDto {
	ch := make(chan types.UrlInfoDto, 1)
	go func() {
		ch <- scrapeUrlInfo(rawUrl)
	}()
	select {
	case info := <-ch:
		return info
	case <-time.After(urlInfoTimeout):
		logger.LogError("GetUrlInfo 超时: %s", rawUrl)
		return types.UrlInfoDto{}
	}
}

func scrapeUrlInfo(rawUrl string) types.UrlInfoDto {
	document, err := goscraper.Scrape(rawUrl, 5)
	if err != nil {
		logger.LogError("GetUrlInfo 抓取失败: %s", err)
		return types.UrlInfoDto{}
	}
	preview := document.Preview
	return types.UrlInfoDto{
		Name:        strings.TrimSpace(preview.Name),
		Title:       strings.TrimSpace(preview.Title),
		Description: strings.TrimSpace(preview.Description),
		Logo:        absoluteIconUrl(rawUrl, preview.Link, preview.Icon),
	}
}

// absoluteIconUrl 把相对路径的图标补成绝对地址
func absoluteIconUrl(rawUrl string, link string, icon string) string {
	icon = strings.TrimSpace(icon)
	if icon == "" {
		return ""
	}
	if strings.HasPrefix(icon, "http://") || strings.HasPrefix(icon, "https://") {
		return icon
	}
	// 优先用 og:url（link），拿不到就用用户填的地址做基准
	base, err := url.Parse(strings.TrimSpace(link))
	if err != nil || base.Scheme == "" || base.Host == "" {
		base, err = url.Parse(strings.TrimSpace(rawUrl))
		if err != nil || base.Scheme == "" || base.Host == "" {
			return ""
		}
	}
	ref, err := url.Parse(icon)
	if err != nil {
		return ""
	}
	return base.ResolveReference(ref).String()
}

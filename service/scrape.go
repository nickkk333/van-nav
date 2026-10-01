package service

import (
	"net/url"
	"strings"
	"time"

	"github.com/mereith/nav/goscraper"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// 抓取页面的最长等待时间：避免前端一直转圈，也避免保存工具时因为目标站点不响应而卡住
const scrapeTimeout = 10 * time.Second

// scrapeDocument 带超时抓取页面（goscraper 自带的 http 客户端没有超时，需要自己兜一层）
// 第二个返回值为 false 表示抓取失败或超时
func scrapeDocument(rawUrl string) (*goscraper.Document, bool) {
	ch := make(chan *goscraper.Document, 1)
	go func() {
		document, err := goscraper.Scrape(rawUrl, 5)
		if err != nil {
			logger.LogError("抓取页面失败: %s", err)
			ch <- nil
			return
		}
		ch <- document
	}()
	select {
	case document := <-ch:
		return document, document != nil
	case <-time.After(scrapeTimeout):
		logger.LogError("抓取页面超时: %s", rawUrl)
		return nil, false
	}
}

// GetUrlInfo 抓取网址的名称/标题/描述/图标，供后台添加工具时自动填充
func GetUrlInfo(rawUrl string) types.UrlInfoDto {
	return scrapeUrlInfo(rawUrl)
}

func scrapeUrlInfo(rawUrl string) types.UrlInfoDto {
	// 网址没有协议时补上 http://，保证抓取与 gstatic 兜底都能正常工作
	siteUrl := normalizeSiteUrl(rawUrl)
	document, ok := scrapeDocument(siteUrl)
	if !ok {
		logger.LogError("GetUrlInfo 抓取失败: %s", siteUrl)
		// 抓取失败时用 gstatic 兜底，至少能把图标填上
		return types.UrlInfoDto{Logo: GstaticFaviconUrl(siteUrl)}
	}
	preview := document.Preview
	logo := absoluteIconUrl(siteUrl, preview.Link, preview.Icon)
	if logo == "" {
		// 页面能打开但没有图标，走 gstatic 兜底
		logo = GstaticFaviconUrl(siteUrl)
	}
	return types.UrlInfoDto{
		Name:        strings.TrimSpace(preview.Name),
		Title:       strings.TrimSpace(preview.Title),
		Description: strings.TrimSpace(preview.Description),
		Logo:        logo,
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

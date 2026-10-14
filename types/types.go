package types

import "encoding/json"

// 默认是 0
type Setting struct {
	Id                   int    `json:"id"`
	Favicon              string `json:"favicon"`
	Title                string `json:"title"`
	GovRecord            string `json:"govRecord"`
	Logo192              string `json:"logo192"`
	Logo512              string `json:"logo512"`
	HideAdmin            bool   `json:"hideAdmin"`
	HideGithub           bool   `json:"hideGithub"`
	// HideCalendar / HideMemo 首页日历、便签入口显隐（类似 Github 按钮的开关，默认显示）
	HideCalendar         bool   `json:"hideCalendar"`
	HideMemo             bool   `json:"hideMemo"`
	HideToggleJumpTarget bool   `json:"hideToggleJumpTarget"`
	JumpTargetBlank      bool   `json:"jumpTargetBlank"`
	// BackgroundImage 首页背景图地址：后台上传返回的 url 或外链地址，留空时前台使用必应每日壁纸作为背景
	BackgroundImage string `json:"backgroundImage"`
	// DefaultLogo 默认图标地址：后台上传返回的 url 或外链地址，相当于替换内置的 default.png
	// 工具/搜索引擎没有自己的图标时前台显示它，留空时前台用名称首字符占位
	DefaultLogo string `json:"defaultLogo"`
	// DefaultSearchEngine 默认搜索引擎 id：回车无匹配卡片或按 Ctrl+Enter 时使用；0 表示自动（第一个启用的引擎）
	DefaultSearchEngine int `json:"defaultSearchEngine"`
}

type Token struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Value    string `json:"value"`
	Disabled int    `json:"disabled"`
}

type User struct {
	Id       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
}
type Img struct {
	Id    int    `json:"id"`
	Url   string `json:"url"`
	Value string `json:"value"`
}

type Tool struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Url  string `json:"url"`
	// Logo 图标网址：保存可以下载到图片的 url 地址（抓取到的网站图标地址等）
	Logo string `json:"logo"`
	// LogoName 图标图片名：图片下载保存到 data 目录（data/images）后的文件名，前台优先用它读本地图片
	LogoName string `json:"logoName"`
	Catelog  string `json:"catelog"`
	Desc     string `json:"desc"`
	Sort     int    `json:"sort"`
	Hide     bool   `json:"hide"`
	Default  bool   `json:"default"`
}

type Catelog struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
	Hide bool   `json:"hide"`
	// Default 该分类下的工具是否展示在主页默认栏
	Default bool `json:"default"`
}

// 搜索引擎模型
type SearchEngine struct {
	Id         int    `json:"id"`
	Name       string `json:"name"`
	BaseUrl    string `json:"baseUrl"`
	QueryParam string `json:"queryParam"`
	Logo       string `json:"logo"`
	Sort       int    `json:"sort"`
	Enabled    bool   `json:"enabled"`
}

// 便签：单文档模式（id 固定为 1），多端通过后端同步，历史版本保留最近 N 条
type Memo struct {
	Id        int    `json:"id"`
	Content   string `json:"content"`
	Mode      string `json:"mode"`
	UpdatedAt int64  `json:"updatedAt"`
}

// MemoVersion 便签历史版本（本地误覆盖时可找回）
type MemoVersion struct {
	Id        int    `json:"id"`
	Content   string `json:"content"`
	Mode      string `json:"mode"`
	CreatedAt int64  `json:"createdAt"`
}

// CalendarEvent 日历事件（按日期挂载，日历上打点提示）
type CalendarEvent struct {
	Id        int    `json:"id"`
	Title     string `json:"title"`
	Date      string `json:"date"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Desc      string `json:"desc"`
	Color     string `json:"color"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

// BackupData 导入导出的备份数据：所有工具、分类、搜索引擎、便签与日历事件
// 图标只保存图标网址（工具的图片名与搜索引擎的图标不导出），导入后由服务端按网址自动重新获取
type BackupData struct {
	// Version 备份文件格式版本
	Version int `json:"version"`
	// ExportedAt 导出时间（RFC3339）
	ExportedAt     string          `json:"exportedAt"`
	Tools          []Tool          `json:"tools"`
	Catelogs       []Catelog       `json:"catelogs"`
	SearchEngines  []SearchEngine  `json:"searchEngines"`
	Memo           *Memo           `json:"memo"`
	MemoVersions   []MemoVersion   `json:"memoVersions"`
	// Todos 老备份（格式版本 3 及更早）里的待办数据：待办功能已移除，不导出也不导入，
	// 保留这个字段只是为了把老备份识别成「有内容的备份」而不是空文件
	Todos          json.RawMessage `json:"todos,omitempty"`
	CalendarEvents []CalendarEvent `json:"calendarEvents"`
}

// 网站配置模型
type SiteConfig struct {
	Id          int  `json:"id"`
	NoImageMode bool `json:"noImageMode"`
	CompactMode bool `json:"compactMode"`
	// CardsPerRow 首页每行展示的网站数量
	CardsPerRow int `json:"cardsPerRow"`
}

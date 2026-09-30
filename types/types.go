package types

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
	HideToggleJumpTarget bool   `json:"hideToggleJumpTarget"`
	JumpTargetBlank      bool   `json:"jumpTargetBlank"`
	// BackgroundImage 首页背景图地址：后台上传返回的 url 或外链地址，留空表示不显示背景图
	BackgroundImage string `json:"backgroundImage"`
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
	Id      int    `json:"id"`
	Name    string `json:"name"`
	Url     string `json:"url"`
	Logo    string `json:"logo"`
	Catelog string `json:"catelog"`
	Desc    string `json:"desc"`
	Sort    int    `json:"sort"`
	Hide    bool   `json:"hide"`
	Default bool   `json:"default"`
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

// BackupData 导入导出的备份数据：所有工具、分类、搜索引擎与 api token
// 图标只保存网址（不包含图片内容），导入后由服务端按网址自动重新获取
type BackupData struct {
	// Version 备份文件格式版本
	Version int `json:"version"`
	// ExportedAt 导出时间（RFC3339）
	ExportedAt    string         `json:"exportedAt"`
	Tools         []Tool         `json:"tools"`
	Catelogs      []Catelog      `json:"catelogs"`
	SearchEngines []SearchEngine `json:"searchEngines"`
	ApiTokens     []Token        `json:"apiTokens"`
}

// 网站配置模型
type SiteConfig struct {
	Id          int  `json:"id"`
	NoImageMode bool `json:"noImageMode"`
	CompactMode bool `json:"compactMode"`
	// CardsPerRow 首页每行展示的网站数量
	CardsPerRow int `json:"cardsPerRow"`
}

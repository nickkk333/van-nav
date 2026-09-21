package types

type ResUserDto struct {
	Name string `json:"name"`
}

type UpdateUserDto struct {
	Id       int64  `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
}

type LoginDto struct {
	Name     string `json:"name"`
	Password string `json:"password"`
}
type AddTokenDto struct {
	Name string `json:"name"`
}

type UpdateCatelogDto struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
	Sort int    `json:"sort"`
	Hide bool   `json:"hide"`
}

type AddCatelogDto struct {
	Name string `json:"name"`
	Sort int    `json:"sort"`
	Hide bool   `json:"hide"`
}
type UpdateToolDto struct {
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
type AddToolDto struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Logo    string `json:"logo"`
	Catelog string `json:"catelog"`
	Desc    string `json:"desc"`
	// Sort 新建工具的落点：-1（默认）或负数排到最后；0 或留空排到最前；正数插入到该序号位置
	// 落位后由后端把所有工具的排序值统一重排成从 1 开始依次递增
	Sort    int  `json:"sort"`
	Hide    bool `json:"hide"`
	Default bool `json:"default"`
}
type UpdateToolsSortDto struct {
	Id   int `json:"id"`
	Sort int `json:"sort"`
}

// UrlInfoDto 抓取网址得到的信息，用于后台添加工具时自动填充
type UrlInfoDto struct {
	Name        string `json:"name"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
}

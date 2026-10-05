package database

import (
	"database/sql"
	"path/filepath"

	_ "modernc.org/sqlite"

	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/utils"
)

var DB *sql.DB

func columnExists(tableName string, columnName string) bool {
	query := `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name=?`
	var count int
	err := DB.QueryRow(query, tableName, columnName).Scan(&count)
	if err != nil {
		return false
	}
	return count > 0
}

// TableExists 判断表是否存在（老库升级迁移时区分「表不存在」与查询错误）
func TableExists(tableName string) bool {
	var name string
	err := DB.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name = ?;`, tableName).Scan(&name)
	return err == nil && name != ""
}

func InitDB() {
	var err error
	utils.PathExistsOrCreate("./data")
	// 创建数据库
	dir := "./data"
	dbPath := filepath.Join(dir, "nav.db")
	// 添加连接参数
	dbPath = dbPath + "?_journal=WAL&_timeout=5000&_busy_timeout=5000&_txlock=immediate"
	DB, err = sql.Open("sqlite", dbPath)
	utils.CheckErr(err)
	// user 表
	sql_create_table := `
		CREATE TABLE IF NOT EXISTS nav_user (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			password TEXT
		);
		`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)
	// setting 表
	sql_create_table = `
	CREATE TABLE IF NOT EXISTS nav_setting (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		favicon TEXT,
		title TEXT,
		govRecord TEXT,
		logo192 TEXT,
		logo512 TEXT,
		hideAdmin BOOLEAN,
		hideGithub BOOLEAN,
		hideToggleJumpTarget BOOLEAN,
		jumpTargetBlank BOOLEAN,
		backgroundImage TEXT,
		defaultLogo TEXT
	);
	`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)
	// 检查并添加列
	if !columnExists("nav_setting", "logo192") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN logo192 TEXT;`)
	}
	if !columnExists("nav_setting", "logo512") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN logo512 TEXT;`)
	}
	if !columnExists("nav_setting", "govRecord") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN govRecord TEXT;`)
	}
	if !columnExists("nav_setting", "jumpTargetBlank") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN jumpTargetBlank BOOLEAN;`)
	}
	// 设置表表结构升级-20230628
	if !columnExists("nav_setting", "hideAdmin") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN hideAdmin BOOLEAN;`)
	}
	// 设置表表结构升级-20230627
	if !columnExists("nav_setting", "hideGithub") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN hideGithub BOOLEAN;`)
	}
	// 设置表表结构升级-20250624
	if !columnExists("nav_setting", "hideToggleJumpTarget") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN hideToggleJumpTarget BOOLEAN;`)
	}
	// 设置表表结构升级-20250923【首页背景图，存放后台上传返回的 url 或外链地址】
	if !columnExists("nav_setting", "backgroundImage") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN backgroundImage TEXT;`)
	}
	// 设置表表结构升级-20261002【默认图标，存放后台上传返回的 url 或外链地址，相当于替换内置的 default.png】
	// 工具/搜索引擎没有自己的图标时前台显示它，留空时前台用名称首字符占位
	if !columnExists("nav_setting", "defaultLogo") {
		DB.Exec(`ALTER TABLE nav_setting ADD COLUMN defaultLogo TEXT;`)
	}

	// 默认 tools 用的 表
	sql_create_table = `
		CREATE TABLE IF NOT EXISTS nav_table (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			'name' TEXT,
			url TEXT,
			logo TEXT,
			logo_name TEXT,
			catelog TEXT,
			'desc' TEXT,
			'default' BOOLEAN NOT NULL DEFAULT 0
		);
		`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)

	// tools数据表结构升级-20230327
	if !columnExists("nav_table", "sort") {
		DB.Exec(`ALTER TABLE nav_table ADD COLUMN sort INTEGER;`)
	}

	// tools数据表结构升级-20230627
	if !columnExists("nav_table", "hide") {
		DB.Exec(`ALTER TABLE nav_table ADD COLUMN hide BOOLEAN;`)
	}

	// tools数据表结构升级-20261001：图标拆成「图标网址（logo）+ 保存到 data 目录的图片名（logo_name）」
	if !columnExists("nav_table", "logo_name") {
		DB.Exec(`ALTER TABLE nav_table ADD COLUMN logo_name TEXT;`)
		// 老数据里 logo 直接存的是本地图片地址（/api/uploadedImage/xxx.png），把文件名补进 logo_name，
		// 前台会优先按 logo_name 读本地图片，logo 保持原值不动
		DB.Exec(`UPDATE nav_table
			SET logo_name = REPLACE(logo, '/api/uploadedImage/', '')
			WHERE logo LIKE '/api/uploadedImage/%' AND (logo_name IS NULL OR logo_name = '');`)
		// 其余老记录没有图片名，统一置成空字符串，避免出现 NULL
		DB.Exec(`UPDATE nav_table SET logo_name = '' WHERE logo_name IS NULL;`)
	}

	// 分类表
	sql_create_table = `
		CREATE TABLE IF NOT EXISTS nav_catelog (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT
		);
			`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)

	// 分类表表结构升级-20230327
	if !columnExists("nav_catelog", "sort") {
		DB.Exec(`ALTER TABLE nav_catelog ADD COLUMN sort INTEGER NOT NULL DEFAULT 0;`)
	}

	// 分类表表结构升级-20241219-【隐藏分类】
	if !columnExists("nav_catelog", "hide") {
		DB.Exec(`ALTER TABLE nav_catelog ADD COLUMN hide BOOLEAN;`)
	}

	// 分类表表结构升级-20250929-【分类默认栏开关】
	if !columnExists("nav_catelog", "default") {
		DB.Exec(`ALTER TABLE nav_catelog ADD COLUMN "default" BOOLEAN;`)
		// 用该分类下工具的默认状态初始化：没有工具或工具全部开启视为开启
		DB.Exec(`
			UPDATE nav_catelog SET "default" = (
				SELECT CASE
					WHEN COUNT(*) = 0 THEN 1
					WHEN SUM(CASE WHEN IFNULL(t."default", 0) = 1 THEN 1 ELSE 0 END) = COUNT(*) THEN 1
					ELSE 0
				END
				FROM nav_table t WHERE t.catelog = nav_catelog.name
			);
			`)
	}
	migration_2024_12_13() // 只涉及 nav_catelog 表，所以可以放在这里

	// api token 表
	sql_create_table = `
		CREATE TABLE IF NOT EXISTS nav_api_token (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT,
			value TEXT,
			disabled INTEGER
		);
		`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)
	// 搜索引擎表
	sql_create_table = `
		CREATE TABLE IF NOT EXISTS nav_search_engine (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			baseUrl TEXT NOT NULL,
			queryParam TEXT NOT NULL,
			logo TEXT,
			sort INTEGER NOT NULL DEFAULT 0,
			enabled BOOLEAN NOT NULL DEFAULT 1
		);
		`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)

	// 网站配置表
	sql_create_table = `
		CREATE TABLE IF NOT EXISTS nav_site_config (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			noImageMode BOOLEAN NOT NULL DEFAULT 0,
			compactMode BOOLEAN NOT NULL DEFAULT 0,
			cardsPerRow INTEGER NOT NULL DEFAULT 5
		);
		`
	_, err = DB.Exec(sql_create_table)
	utils.CheckErr(err)

	// 网站配置表结构升级 - 添加compactMode列
	if !columnExists("nav_site_config", "compactMode") {
		DB.Exec(`ALTER TABLE nav_site_config ADD COLUMN compactMode BOOLEAN NOT NULL DEFAULT 0;`)
	}

	// 网站配置表结构升级 - 添加cardsPerRow列（首页每行展示的网站数量）
	if !columnExists("nav_site_config", "cardsPerRow") {
		DB.Exec(`ALTER TABLE nav_site_config ADD COLUMN cardsPerRow INTEGER NOT NULL DEFAULT 5;`)
	}

	// 如果不存在，就初始化默认搜索引擎
	sql_get_search_engine := `
		SELECT COUNT(*) FROM nav_search_engine;
		`
	var searchEngineCount int
	err = DB.QueryRow(sql_get_search_engine).Scan(&searchEngineCount)
	utils.CheckErr(err)
	if searchEngineCount == 0 {
		// 初始化默认搜索引擎
		defaultEngines := []struct {
			name       string
			baseUrl    string
			queryParam string
			logo       string
			sort       int
		}{
			{"百度", "https://www.baidu.com/s", "wd", "/baidu.ico", 1},
			{"Bing", "https://cn.bing.com/search", "q", "/bing.ico", 2},
			{"Google", "https://www.google.com/search", "q", "/google.ico", 3},
		}

		sql_add_search_engine := `
			INSERT INTO nav_search_engine (name, baseUrl, queryParam, logo, sort, enabled)
			VALUES (?, ?, ?, ?, ?, ?);
			`
		stmt, err := DB.Prepare(sql_add_search_engine)
		utils.CheckErr(err)
		defer stmt.Close()

		for _, engine := range defaultEngines {
			_, err = stmt.Exec(engine.name, engine.baseUrl, engine.queryParam, engine.logo, engine.sort, true)
			utils.CheckErr(err)
		}
		logger.LogInfo("默认搜索引擎初始化成功")
	}

	// 如果不存在，就初始化用户
	sql_get_user := `
		SELECT * FROM nav_user;
		`
	rows, err := DB.Query(sql_get_user)
	if err != nil {
		// Query 失败时 rows 为 nil，老代码直接 rows.Next() 会 panic
		utils.CheckErr(err)
		return
	}
	if !rows.Next() {
		// 一次性插入改用 DB.Exec：老代码 Prepare 失败后 stmt 为 nil，再 Exec 会 panic
		if _, err = DB.Exec(`INSERT INTO nav_user (id, name, password) VALUES (?, ?, ?);`,
			utils.GenerateId(), "admin", "admin"); err != nil {
			utils.CheckErr(err)
		}
	}
	rows.Close()
	// 如果不存在设置，就初始化
	sql_get_setting := `
		SELECT * FROM nav_setting;
		`
	rows, err = DB.Query(sql_get_setting)
	if err != nil {
		utils.CheckErr(err)
		return
	}
	if !rows.Next() {
		if _, err = DB.Exec(`
			INSERT INTO nav_setting (favicon, title, govRecord, logo192, logo512, hideAdmin, hideGithub, hideToggleJumpTarget, jumpTargetBlank)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
			`, "favicon.ico", "Van Nav", "", "logo192.png", "logo512.png", false, false, false, true); err != nil {
			utils.CheckErr(err)
		}
	}
	rows.Close()

	// 如果不存在网站配置，就初始化
	sql_get_site_config := `
		SELECT * FROM nav_site_config;
		`
	rows, err = DB.Query(sql_get_site_config)
	if err != nil {
		utils.CheckErr(err)
		return
	}
	if !rows.Next() {
		// 5 为每行展示网站数量的默认值，与 service.DefaultCardsPerRow 保持一致
		if _, err = DB.Exec(`
			INSERT INTO nav_site_config (noImageMode, compactMode, cardsPerRow)
			VALUES (?, ?, ?);
			`, false, false, 5); err != nil {
			utils.CheckErr(err)
		}
	}
	rows.Close()

	// 常用查询补索引（IF NOT EXISTS，老库升级与每次启动执行都安全）：
	// - sort/id：首页与后台的 ORDER BY sort, id（分类重排也是按这个顺序遍历）
	// - catelog：分类改名/隐藏/默认开关时按分类名批量更新工具
	// - nav_api_token.value：JWT 中间件每次请求都按 value 查（鉴权热点）
	// - nav_user.name：登录按用户名查
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_nav_table_sort ON nav_table(sort, id);`,
		`CREATE INDEX IF NOT EXISTS idx_nav_table_catelog ON nav_table(catelog);`,
		`CREATE INDEX IF NOT EXISTS idx_nav_catelog_sort ON nav_catelog(sort, id);`,
		`CREATE INDEX IF NOT EXISTS idx_nav_api_token_value ON nav_api_token(value);`,
		`CREATE INDEX IF NOT EXISTS idx_nav_user_name ON nav_user(name);`,
	}
	for _, idx := range indexes {
		if _, err := DB.Exec(idx); err != nil {
			utils.CheckErr(err)
		}
	}
	// nav_img 是老版本存 base64 图片缓存的表，图片出库后会整表删掉（见 service.MigrateImgCacheToFiles），
	// 只在它还在时补索引，读取侧搬迁过程中要按 url 查
	if TableExists("nav_img") {
		if _, err := DB.Exec(`CREATE INDEX IF NOT EXISTS idx_nav_img_url ON nav_img(url);`); err != nil {
			utils.CheckErr(err)
		}
	}

	logger.LogInfo("数据库初始化成功💗")

	// 清理空分类记录 - 删除名称为空或只包含空白字符的分类
	cleanupEmptyCategories()
}

// cleanupEmptyCategories 清理空分类记录
func cleanupEmptyCategories() {
	// 删除名称为空或只包含空白字符的分类记录
	sql_cleanup := `
		DELETE FROM nav_catelog 
		WHERE name IS NULL OR name = '' OR TRIM(name) = '';
	`
	result, err := DB.Exec(sql_cleanup)
	if err != nil {
		logger.LogInfo("清理空分类记录时出错: %v", err)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err == nil && rowsAffected > 0 {
		logger.LogInfo("已清理 %d 条空分类记录", rowsAffected)
	}
}

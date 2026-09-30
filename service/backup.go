package service

import (
	"database/sql"
	"strings"
	"time"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// backupVersion 备份文件格式版本，格式变化时递增，便于后续做兼容处理
const backupVersion = 1

// ExportBackupData 导出所有工具、分类、搜索引擎与 api token
// 图标只导出网址（不导出图片内容），导入后由服务端按网址自动重新获取
func ExportBackupData() types.BackupData {
	engines, err := database.GetAllSearchEngines()
	if err != nil {
		utils.CheckErr(err)
		engines = make([]types.SearchEngine, 0)
	}
	return types.BackupData{
		Version:       backupVersion,
		ExportedAt:    time.Now().Format(time.RFC3339),
		Tools:         GetAllTool(),
		Catelogs:      GetAllCatelog(),
		SearchEngines: engines,
		ApiTokens:     GetApiTokens(),
	}
}

// ImportBackupData 导入备份数据（工具、分类、搜索引擎、api token）
// 同 id 的记录会被覆盖，没有出现在备份里的记录保持不动
// 导入完成后异步获取图片：备份里只存了图标网址，这里按网址重新缓存/下载
func ImportBackupData(data types.BackupData) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err = importCatelogs(tx, data.Catelogs); err != nil {
		return err
	}
	// 工具引用到但备份里没有的分类（手工编辑过的备份文件）补建记录，避免分类名对不上
	if err = ensureToolCatelogs(tx, data.Tools); err != nil {
		return err
	}
	if err = importTools(tx, data.Tools); err != nil {
		return err
	}
	if err = importSearchEngines(tx, data.SearchEngines); err != nil {
		return err
	}
	if err = importApiTokens(tx, data.ApiTokens); err != nil {
		return err
	}
	// 分类排序统一重排成从 1 开始依次递增，与后台分类管理的规则保持一致
	if err = renumberCatelogs(tx); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}

	logger.LogInfo("导入备份数据：工具 %d 条，分类 %d 条，搜索引擎 %d 条，api token %d 条",
		len(data.Tools), len(data.Catelogs), len(data.SearchEngines), len(data.ApiTokens))
	// 图片获取比较耗时，放到后台执行，避免阻塞接口
	go fetchImportedImages(data.Tools, data.SearchEngines)
	return nil
}

// importCatelogs 导入分类
// 工具是按分类名称归属的，所以同名分类直接合并到已有记录上，避免出现重复分类
func importCatelogs(tx *sql.Tx, catelogs []types.Catelog) error {
	for _, catelog := range catelogs {
		name := strings.TrimSpace(catelog.Name)
		if name == "" {
			continue
		}
		var existId int64
		err := tx.QueryRow(`SELECT id FROM nav_catelog WHERE name = ?;`, name).Scan(&existId)
		switch {
		case err == nil:
			if _, err = tx.Exec(`UPDATE nav_catelog SET sort = ?, "hide" = ?, "default" = ? WHERE id = ?;`,
				catelog.Sort, catelog.Hide, catelog.Default, existId); err != nil {
				return err
			}
		case err == sql.ErrNoRows:
			if _, err = tx.Exec(`INSERT INTO nav_catelog (name, sort, "hide", "default") VALUES (?, ?, ?, ?);`,
				name, catelog.Sort, catelog.Hide, catelog.Default); err != nil {
				return err
			}
		default:
			return err
		}
	}
	return nil
}

// importTools 导入工具：同 id 覆盖，没出现在备份里的工具保持不动
func importTools(tx *sql.Tx, tools []types.Tool) error {
	stmt, err := tx.Prepare(`
		INSERT INTO nav_table (id, name, url, logo, catelog, "desc", sort, "hide", "default")
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			url = excluded.url,
			logo = excluded.logo,
			catelog = excluded.catelog,
			"desc" = excluded."desc",
			sort = excluded.sort,
			"hide" = excluded."hide",
			"default" = excluded."default";
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, tool := range tools {
		name := strings.TrimSpace(tool.Name)
		if name == "" {
			continue
		}
		if tool.Id > 0 {
			if _, err = stmt.Exec(tool.Id, name, tool.Url, tool.Logo, tool.Catelog, tool.Desc,
				tool.Sort, tool.Hide, tool.Default); err != nil {
				return err
			}
			continue
		}
		// 手工编辑过的备份可能没有 id，按新增处理，交给自增主键分配
		if _, err = tx.Exec(`
			INSERT INTO nav_table (name, url, logo, catelog, "desc", sort, "hide", "default")
			VALUES (?, ?, ?, ?, ?, ?, ?, ?);
			`, name, tool.Url, tool.Logo, tool.Catelog, tool.Desc, tool.Sort, tool.Hide, tool.Default); err != nil {
			return err
		}
	}
	return nil
}

// importSearchEngines 导入搜索引擎：同 id 覆盖，没出现在备份里的保持不动
func importSearchEngines(tx *sql.Tx, engines []types.SearchEngine) error {
	stmt, err := tx.Prepare(`
		INSERT INTO nav_search_engine (id, name, baseUrl, queryParam, logo, sort, enabled)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			baseUrl = excluded.baseUrl,
			queryParam = excluded.queryParam,
			logo = excluded.logo,
			sort = excluded.sort,
			enabled = excluded.enabled;
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, engine := range engines {
		name := strings.TrimSpace(engine.Name)
		if name == "" {
			continue
		}
		if engine.Id > 0 {
			if _, err = stmt.Exec(engine.Id, name, engine.BaseUrl, engine.QueryParam, engine.Logo,
				engine.Sort, engine.Enabled); err != nil {
				return err
			}
			continue
		}
		if _, err = tx.Exec(`
			INSERT INTO nav_search_engine (name, baseUrl, queryParam, logo, sort, enabled)
			VALUES (?, ?, ?, ?, ?, ?);
			`, name, engine.BaseUrl, engine.QueryParam, engine.Logo, engine.Sort, engine.Enabled); err != nil {
			return err
		}
	}
	return nil
}

// importApiTokens 导入 api token：同 id 覆盖，没出现在备份里的保持不动
func importApiTokens(tx *sql.Tx, tokens []types.Token) error {
	stmt, err := tx.Prepare(`
		INSERT INTO nav_api_token (id, name, value, disabled)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			value = excluded.value,
			disabled = excluded.disabled;
		`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, token := range tokens {
		value := strings.TrimSpace(token.Value)
		if value == "" {
			continue
		}
		if token.Id > 0 {
			if _, err = stmt.Exec(token.Id, token.Name, value, token.Disabled); err != nil {
				return err
			}
			continue
		}
		if _, err = tx.Exec(`INSERT INTO nav_api_token (name, value, disabled) VALUES (?, ?, ?);`,
			token.Name, value, token.Disabled); err != nil {
			return err
		}
	}
	return nil
}

// ensureToolCatelogs 为工具里用到、但备份中没有的分类补建记录（追加到分类列表末尾）
func ensureToolCatelogs(tx *sql.Tx, tools []types.Tool) error {
	for _, tool := range tools {
		name := strings.TrimSpace(tool.Catelog)
		if name == "" {
			continue
		}
		if _, err := tx.Exec(`
			INSERT INTO nav_catelog (name, sort, "hide", "default")
			SELECT ?, (SELECT IFNULL(MAX(sort), 0) + 1 FROM nav_catelog), 0, 0
			WHERE NOT EXISTS (SELECT 1 FROM nav_catelog WHERE name = ?);
			`, name, name); err != nil {
			return err
		}
	}
	return nil
}

// ==================== 导入后自动获取图片 ====================

// fetchImportedImages 导入后获取图片：备份里只保存了图标网址，这里按网址把图片拿回来
func fetchImportedImages(tools []types.Tool, engines []types.SearchEngine) {
	for _, tool := range tools {
		refreshImportedToolLogo(tool)
	}
	for _, engine := range engines {
		refreshImportedSearchEngineLogo(engine)
	}
}

// refreshImportedToolLogo 按工具 logo 网址重新缓存图片
// 网址为空或本地图片在本机不存在（换机器导入）时，重新抓取网站图标
func refreshImportedToolLogo(tool types.Tool) {
	if tool.Id <= 0 {
		return
	}
	logo := strings.TrimSpace(tool.Logo)
	if logo == "" || needFetchLogo(logo) {
		url := strings.TrimSpace(tool.Url)
		if url == "" {
			return
		}
		LazyFetchLogo(url, int64(tool.Id))
		return
	}
	// 外链图片缓存到数据库，首页与后台直接读缓存
	if strings.HasPrefix(logo, "http://") || strings.HasPrefix(logo, "https://") {
		UpdateImg(logo)
	}
}

// refreshImportedSearchEngineLogo 按搜索引擎 logo 网址把外链图片下载保存到本地
// 网址为空或本地图片在本机不存在（换机器导入）时，重新抓取网站图标再保存
func refreshImportedSearchEngineLogo(engine types.SearchEngine) {
	if engine.Id <= 0 {
		return
	}
	logo := strings.TrimSpace(engine.Logo)
	if logo == "" || needFetchLogo(logo) {
		logo = fetchSearchEngineIcon(engine.BaseUrl)
	}
	if !strings.HasPrefix(logo, "http://") && !strings.HasPrefix(logo, "https://") {
		return
	}
	localUrl, err := SaveSearchEngineLogo(engine.Name, logo)
	if err != nil {
		logger.LogError("导入后获取搜索引擎 logo 失败: %s", err)
		return
	}
	if localUrl == "" {
		return
	}
	// 只更新 logo 字段，避免覆盖导入后又被改过的其他字段
	if _, err = database.DB.Exec(`UPDATE nav_search_engine SET logo = ? WHERE id = ?;`, localUrl, engine.Id); err != nil {
		logger.LogError("导入后更新搜索引擎 logo 失败: %s", err)
	}
}

// fetchSearchEngineIcon 抓取站点图标地址，抓不到时返回空字符串
func fetchSearchEngineIcon(baseUrl string) string {
	baseUrl = strings.TrimSpace(baseUrl)
	if !strings.HasPrefix(baseUrl, "http://") && !strings.HasPrefix(baseUrl, "https://") {
		return ""
	}
	return strings.TrimSpace(GetUrlInfo(baseUrl).Logo)
}

// needFetchLogo 判断 logo 是否是「本机并不存在的本地图片地址」，这种地址需要重新获取图片
func needFetchLogo(logo string) bool {
	return LocalImageName(logo) != "" && !LocalImageExist(logo)
}

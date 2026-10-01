package service

import (
	"database/sql"
	"strings"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// ImportTools 批量导入工具（兼容旧的 /admin/importTools 接口，只处理工具，其余保留原样）
func ImportTools(data []types.Tool) error {
	return ImportBackupData(types.BackupData{Tools: data})
}

// needRefreshToolLogo 保存工具（新增/修改/导入）时判断 logo 是否需要重新获取图标
// 需要重新获取的情况：
//   - logo 为空
//   - logo 是可下载的图片链接（http://、https:// 或 // 开头的外链）
//   - logo 指向本机保存的图片，但文件在本机并不存在（换机器导入备份）
//
// 用户上传保存到本机的图片、站内相对路径等其他地址不需要处理
func needRefreshToolLogo(logo string) bool {
	logo = strings.TrimSpace(logo)
	if logo == "" {
		return true
	}
	if strings.HasPrefix(logo, "//") || strings.HasPrefix(logo, "http://") || strings.HasPrefix(logo, "https://") {
		return true
	}
	return needFetchLogo(logo)
}

// NormalizeToolLogo 保存工具（新增/修改/导入）时统一处理 logo，需要重新获取时按下面的顺序兜底：
//  1. 先用 goscraper 抓取工具网址的网站图标
//  2. 抓不到再用 gstatic 的图标接口
//  3. 拿到图标地址就下载保存到 data/images（文件名使用工具名称），logo 改为本地地址，之后都从本地读取
//  4. 两步都拿不到、或图片下载失败时把 logo 置空，前台会显示默认图标 default.png
//
// 不需要重新获取的 logo（用户上传的本地图片等）原样返回；工具没有填写网址时也保留原值
func NormalizeToolLogo(name string, siteUrl string, logo string) string {
	logo = strings.TrimSpace(logo)
	if !needRefreshToolLogo(logo) {
		return logo
	}
	siteUrl = strings.TrimSpace(siteUrl)
	if siteUrl == "" {
		// 没有网址可抓，保留原值（外链仍可以通过后端代理显示，空值由前台显示默认图标）
		logger.LogError("工具 %s 没有填写网址，无法获取图标", name)
		return logo
	}
	iconUrl := getIcon(siteUrl)
	if iconUrl == "" {
		logger.LogError("工具 %s 未能获取到图标，logo 置空，前台显示默认图标", name)
		return ""
	}
	localUrl, err := SaveToolLogo(name, iconUrl)
	if err != nil {
		logger.LogError("工具 %s 的图标下载失败，logo 置空，前台显示默认图标: %s", name, err)
		return ""
	}
	if localUrl == "" {
		logger.LogError("工具 %s 的图标保存失败，logo 置空，前台显示默认图标: %s", name, iconUrl)
		return ""
	}
	logger.LogInfo("工具 %s 的 logo 已保存到本地: %s", name, localUrl)
	return localUrl
}

// UpdateTool 更新工具，同时更新图片表
func UpdateTool(data types.UpdateToolDto) {
	_, err := database.DB.Exec(`
		UPDATE nav_table
		SET name = ?, url = ?, logo = ?, catelog = ?, "desc" = ?, sort = ?, "hide" = ?, "default" = ?
		WHERE id = ?;
		`, data.Name, data.Url, data.Logo, data.Catelog, data.Desc, data.Sort, data.Hide, data.Default, data.Id)
	utils.CheckErr(err)
	UpdateImg(data.Logo)
}

// ResolveToolInsertIndex 计算新工具在排序中的插入位置（count 为插入前的工具总数）
// - sort < 0：排到最后
// - sort == 0（含留空）：排到最前
// - sort > 0：插入到该序号位置（从 1 开始，超出范围则排到最后）
func ResolveToolInsertIndex(sort int, count int) int {
	if sort < 0 {
		return count
	}
	if sort == 0 {
		return 0
	}
	index := sort - 1
	if index > count {
		index = count
	}
	return index
}

// AddTool 新增工具
// 排序规则：-1（默认）或负数排到最后；0 或留空排到最前；正数插入到该序号位置
// 落位完成后把所有工具的排序值重排成从 1 开始依次递增
func AddTool(data types.AddToolDto) (int64, error) {
	tx, err := database.DB.Begin()
	if err != nil {
		return 0, err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	// 先取出当前所有工具 id（按排序升序），用于计算新工具的落点
	rows, err := tx.Query(`
		SELECT id FROM nav_table ORDER BY sort, id;
		`)
	if err != nil {
		return 0, err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var eachId int64
		if err = rows.Scan(&eachId); err != nil {
			rows.Close()
			return 0, err
		}
		ids = append(ids, eachId)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return 0, err
	}

	res, err := tx.Exec(`
		INSERT INTO nav_table (name, url, logo, catelog, "desc", sort, "hide", "default")
		VALUES (?, ?, ?, ?, ?, ?, ?, ?);
		`, data.Name, data.Url, data.Logo, data.Catelog, data.Desc, data.Sort, data.Hide, data.Default)
	if err != nil {
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// 把新工具插到目标位置，再把所有排序值重写成 1 开始依次递增
	index := ResolveToolInsertIndex(data.Sort, len(ids))
	ordered := make([]int64, 0, len(ids)+1)
	ordered = append(ordered, ids[:index]...)
	ordered = append(ordered, id)
	ordered = append(ordered, ids[index:]...)

	stmt, err := tx.Prepare(`UPDATE nav_table SET sort = ? WHERE id = ?;`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()
	for i, eachId := range ordered {
		if _, err = stmt.Exec(i+1, eachId); err != nil {
			return 0, err
		}
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	logger.LogInfo("新增工具: %s", data.Name)

	if data.Logo != "" {
		UpdateImg(data.Logo)
	}
	return id, nil
}

// GetAllTool 获取全部工具（按排序升序）
func GetAllTool() []types.Tool {
	rows, err := database.DB.Query(`
		SELECT id, name, url, logo, catelog, "desc", sort, "hide", "default"
		FROM nav_table ORDER BY sort;
		`)
	if err != nil {
		utils.CheckErr(err)
		return []types.Tool{}
	}
	defer rows.Close()

	results := make([]types.Tool, 0)
	for rows.Next() {
		var tool types.Tool
		var (
			sortVal sql.NullInt64
			hide    sql.NullBool
			defVal  sql.NullBool
		)
		if err = rows.Scan(&tool.Id, &tool.Name, &tool.Url, &tool.Logo, &tool.Catelog, &tool.Desc, &sortVal, &hide, &defVal); err != nil {
			utils.CheckErr(err)
			continue
		}
		tool.Sort = int(sortVal.Int64)
		tool.Hide = hide.Bool
		tool.Default = defVal.Bool
		results = append(results, tool)
	}
	return results
}

// GetToolLogoUrlById 根据 id 获取工具的 logo 地址
func GetToolLogoUrlById(id int) string {
	var logo string
	err := database.DB.QueryRow(`SELECT logo FROM nav_table WHERE id = ?;`, id).Scan(&logo)
	if err != nil && err != sql.ErrNoRows {
		utils.CheckErr(err)
	}
	return logo
}

// RemoveToolLogoIfUnused 清理工具换掉或删除后的本地旧图片，还有别的工具在用同一张图片时不删
// 需要在更新/删除数据库记录之后调用，外链地址不做处理
func RemoveToolLogoIfUnused(logo string) {
	if LocalImageName(logo) == "" {
		return
	}
	var count int
	if err := database.DB.QueryRow(`SELECT COUNT(*) FROM nav_table WHERE logo = ?;`, logo).Scan(&count); err != nil {
		utils.CheckErr(err)
		return
	}
	if count > 0 {
		return
	}
	RemoveLocalImage(logo)
}

// UpdateToolsSort 批量更新工具排序
func UpdateToolsSort(updates []types.UpdateToolsSortDto) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE nav_table SET sort = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, update := range updates {
		if _, err = stmt.Exec(update.Sort, update.Id); err != nil {
			return err
		}
	}
	return tx.Commit()
}

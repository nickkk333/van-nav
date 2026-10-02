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

// isRemoteImageUrl 判断 logo 是否是可下载的图片外链（http://、https://、// 开头）
func isRemoteImageUrl(logo string) bool {
	return strings.HasPrefix(logo, "//") || strings.HasPrefix(logo, "http://") || strings.HasPrefix(logo, "https://")
}

// needRefreshToolLogo 保存工具（新增/修改/导入）时判断图标是否需要重新获取，需要重新获取的情况：
//   - logo_name 指向的图片在 data/images 里并不存在（还没有本地图片，或换机器导入备份）
//   - logo_name 为空，且 logo 为空 / 是可下载的图片外链 / 是指向本机但文件不存在的本地地址
//
// 本机已经有图片（logo_name 指向的文件存在，例如后台上传的图片）时保持不动
func needRefreshToolLogo(logo string, logoName string) bool {
	logoName = strings.TrimSpace(logoName)
	if logoName != "" {
		return !LocalImageNameExist(logoName)
	}
	return needRefreshToolLogoUrl(logo)
}

// needRefreshToolLogoUrl 只按 logo 网址判断图标是否需要重新获取
func needRefreshToolLogoUrl(logo string) bool {
	logo = strings.TrimSpace(logo)
	if logo == "" {
		return true
	}
	if isRemoteImageUrl(logo) {
		return true
	}
	return needFetchLogo(logo)
}

// ToolLogo 工具图标的处理结果：
// Logo 是图标网址（可以下载到图片的 url 地址），LogoName 是下载保存到 data 目录（data/images）的图片文件名
type ToolLogo struct {
	Logo     string `json:"logo"`
	LogoName string `json:"logoName"`
}

// ResolveToolLogo 保存工具（新增/修改/导入）时统一处理图标：
//  1. 本机已经有图片（logo_name 指向的文件存在）时保持不动，原样返回
//  2. 需要重新获取时先按工具网址抓取网站图标（goscraper），抓不到再用 gstatic 的图标接口兜底
//  3. 拿到图标网址就下载保存到 data/images（文件名使用工具名称），logo 存图标网址、logoName 存文件名
//  4. 都拿不到、或图片下载失败时把两个字段都置空，前台会显示默认图标 default.png
//
// 工具没有填写网址时保留原有的 logo 网址（图片名留空，前台按网址显示或退化为默认图标）
func ResolveToolLogo(name string, siteUrl string, logo string, logoName string) ToolLogo {
	logo = strings.TrimSpace(logo)
	logoName = strings.TrimSpace(logoName)
	// 图片名必须是合法的文件名，非法值（手填了路径等）直接丢掉，避免读到别的文件
	if logoName != "" && !isSafeImageFileName(logoName) {
		logger.LogError("工具 %s 的图标图片名不合法，已忽略: %s", name, logoName)
		logoName = ""
	}
	if !needRefreshToolLogo(logo, logoName) {
		return ToolLogo{Logo: logo, LogoName: logoName}
	}
	// 工具没有填写网址时没法按网址抓图标，保留原有的 logo 网址（图片名留空，前台按网址显示或退化为默认图标）
	if strings.TrimSpace(siteUrl) == "" {
		logger.LogError("工具 %s 没有填写网址，无法获取图标", name)
		return ToolLogo{Logo: logo}
	}
	return FetchToolLogoBySiteUrl(name, siteUrl)
}

// FetchToolLogoBySiteUrl 按工具网址获取图标并下载保存到本地：
// 先用 goscraper 抓网站图标，抓不到再用 gstatic 的图标接口兜底；拿到图标网址就下载保存到
// data 目录（data/images，文件名使用工具名称），返回图标网址与保存后的文件名
// 工具没有填写网址、抓不到图标、图片下载或保存失败时都返回空值（两个字段都置空，前台显示默认图标 default.png）
func FetchToolLogoBySiteUrl(name string, siteUrl string) ToolLogo {
	siteUrl = strings.TrimSpace(siteUrl)
	if siteUrl == "" {
		logger.LogError("工具 %s 没有填写网址，无法获取图标", name)
		return ToolLogo{}
	}
	iconUrl := getIcon(siteUrl)
	if iconUrl == "" {
		logger.LogError("工具 %s 未能获取到图标，图标置空，前台显示默认图标", name)
		return ToolLogo{}
	}
	savedName, err := SaveToolLogo(name, iconUrl)
	if err != nil {
		logger.LogError("工具 %s 的图标下载失败，图标置空，前台显示默认图标: %s", name, err)
		return ToolLogo{}
	}
	if savedName == "" {
		logger.LogError("工具 %s 的图标保存失败，图标置空，前台显示默认图标: %s", name, iconUrl)
		return ToolLogo{}
	}
	logger.LogInfo("工具 %s 的图标已保存到本地: %s（来源 %s）", name, savedName, iconUrl)
	return ToolLogo{Logo: iconUrl, LogoName: savedName}
}

// ResolveUpdatedToolLogo 更新工具时处理图标（表格里改 logo 网址后的规则）：
//  1. logo 网址没有改动：按 ResolveToolLogo 的规则处理（本机已经有这张图片就原样不动）
//  2. 改成图片外链（http://、https:// 开头）：先按填写的地址下载这张图片保存到 data/images，
//     保存成功时 logo 网址保持填写的地址
//  3. 按填写的地址拿不到图片（下载失败、地址不是图片）、或改成了空与其他内容：
//     改按工具网址走 FetchToolLogoBySiteUrl 获取网站图标（goscraper，抓不到用 gstatic 兜底）
//
// 获取成功时 logo 存拿到的图标网址、logoName 存保存的图片文件名；都拿不到时两个字段都置空（前台显示默认图标 default.png）
func ResolveUpdatedToolLogo(name string, siteUrl string, logo string, logoName string, oldLogo string) ToolLogo {
	logo = strings.TrimSpace(logo)
	// logo 网址没改动时按原有规则处理
	if logo == strings.TrimSpace(oldLogo) {
		return ResolveToolLogo(name, siteUrl, logo, logoName)
	}
	// 改成图片外链时先按填写的地址下载这张图片
	if isRemoteImageUrl(logo) {
		savedName, err := SaveToolLogoFromImageUrl(name, logo)
		if err != nil {
			logger.LogError("工具 %s 的图标下载失败，改按工具网址重新获取: %s", name, err)
		} else if savedName != "" {
			logger.LogInfo("工具 %s 的图标已保存到本地: %s（来源 %s）", name, savedName, logo)
			return ToolLogo{Logo: logo, LogoName: savedName}
		}
	}
	// 改成空或其他内容、或按填写的地址拿不到图片时：按工具网址重新获取网站图标
	return FetchToolLogoBySiteUrl(name, siteUrl)
}

// UpdateTool 更新工具，同时更新图片表
func UpdateTool(data types.UpdateToolDto) {
	_, err := database.DB.Exec(`
		UPDATE nav_table
		SET name = ?, url = ?, logo = ?, logo_name = ?, catelog = ?, "desc" = ?, sort = ?, "hide" = ?, "default" = ?
		WHERE id = ?;
		`, data.Name, data.Url, data.Logo, data.LogoName, data.Catelog, data.Desc, data.Sort, data.Hide, data.Default, data.Id)
	utils.CheckErr(err)
	// 图片已经下载到 data/images 时不需要再缓存到数据库，前台直接读本地图片
	if strings.TrimSpace(data.LogoName) == "" {
		UpdateImg(data.Logo)
	}
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
		INSERT INTO nav_table (name, url, logo, logo_name, catelog, "desc", sort, "hide", "default")
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
		`, data.Name, data.Url, data.Logo, data.LogoName, data.Catelog, data.Desc, data.Sort, data.Hide, data.Default)
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

	// 图片已经下载到 data/images 时不需要再缓存到数据库，前台直接读本地图片
	if strings.TrimSpace(data.LogoName) == "" && data.Logo != "" {
		UpdateImg(data.Logo)
	}
	return id, nil
}

// GetAllTool 获取全部工具（按排序升序）
func GetAllTool() []types.Tool {
	rows, err := database.DB.Query(`
		SELECT id, name, url, logo, logo_name, catelog, "desc", sort, "hide", "default"
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
			logoName sql.NullString
			sortVal  sql.NullInt64
			hide     sql.NullBool
			defVal   sql.NullBool
		)
		if err = rows.Scan(&tool.Id, &tool.Name, &tool.Url, &tool.Logo, &logoName, &tool.Catelog, &tool.Desc, &sortVal, &hide, &defVal); err != nil {
			utils.CheckErr(err)
			continue
		}
		tool.LogoName = logoName.String
		tool.Sort = int(sortVal.Int64)
		tool.Hide = hide.Bool
		tool.Default = defVal.Bool
		results = append(results, tool)
	}
	return results
}

// GetToolLogoById 根据 id 取出工具的图标网址与保存到本地的图片名（更新/删除前用于清理旧的本地图片）
func GetToolLogoById(id int) (string, string) {
	var (
		logo     sql.NullString
		logoName sql.NullString
	)
	err := database.DB.QueryRow(`SELECT logo, logo_name FROM nav_table WHERE id = ?;`, id).Scan(&logo, &logoName)
	if err != nil && err != sql.ErrNoRows {
		utils.CheckErr(err)
	}
	return logo.String, logoName.String
}

// GetToolNameAndLogoById 根据 id 取出工具名称与图标信息（图标网址、保存到本地的图片名）
// 给工具上传本地图标时用：名称用于生成图片文件名，原图标用于上传后清理被替换掉的旧图片
// 工具不存在时最后一个返回值为 false
func GetToolNameAndLogoById(id int) (string, string, string, bool) {
	var (
		name     sql.NullString
		logo     sql.NullString
		logoName sql.NullString
	)
	err := database.DB.QueryRow(`SELECT name, logo, logo_name FROM nav_table WHERE id = ?;`, id).
		Scan(&name, &logo, &logoName)
	if err == sql.ErrNoRows {
		return "", "", "", false
	}
	if err != nil {
		utils.CheckErr(err)
		return "", "", "", false
	}
	return name.String, logo.String, logoName.String, true
}

// UpdateToolLogo 只更新工具的图标（图标网址与保存到本地的图片名），其余字段不动
// 用于给工具上传本地图标：logo 存本地图片地址，logoName 存文件名
func UpdateToolLogo(id int, logo string, logoName string) error {
	_, err := database.DB.Exec(`UPDATE nav_table SET logo = ?, logo_name = ? WHERE id = ?;`, logo, logoName, id)
	utils.CheckErr(err)
	return err
}

// RemoveToolLogoIfUnused 清理工具换掉或删除后的本地旧图片，还有别的工具在用同一张图片时不删
// 需要在更新/删除数据库记录之后调用；图标网址等非本地图片不做处理
// logoName 为工具原来的图片名（logo_name），为空时兼容老数据从 logo 里取本地图片地址
func RemoveToolLogoIfUnused(logo string, logoName string) {
	name := strings.TrimSpace(logoName)
	if !isSafeImageFileName(name) {
		name = LocalImageName(logo)
	}
	if name == "" {
		return
	}
	// 还有别的工具在引用这张图片（logo_name 指向它，或老数据把本地地址存在 logo 里）时不删
	var count int
	if err := database.DB.QueryRow(
		`SELECT COUNT(*) FROM nav_table WHERE logo_name = ? OR logo = ?;`,
		name, UploadUrlPrefix+name).Scan(&count); err != nil {
		utils.CheckErr(err)
		return
	}
	if count > 0 {
		return
	}
	RemoveLocalImageByName(name)
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

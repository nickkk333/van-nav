package service

import (
	"database/sql"
	"strings"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// ImportTools 批量导入工具
func ImportTools(data []types.Tool) {
	var catelogs []string
	stmt, err := database.DB.Prepare(`
		INSERT INTO nav_table (id, name, catelog, url, logo, "desc", sort, "hide", "default")
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);
		`)
	if err != nil {
		utils.CheckErr(err)
		return
	}
	defer stmt.Close()

	for _, v := range data {
		// 过滤掉空分类，只收集有效的分类名称
		if strings.TrimSpace(v.Catelog) != "" && !utils.In(v.Catelog, catelogs) {
			catelogs = append(catelogs, v.Catelog)
		}
		_, err = stmt.Exec(v.Id, v.Name, v.Catelog, v.Url, v.Logo, v.Desc, v.Sort, v.Hide, v.Default)
		utils.CheckErr(err)
	}
	for _, catelog := range catelogs {
		AddCatelog(types.AddCatelogDto{Name: catelog})
	}
	// 转存所有图片，异步
	go func(data []types.Tool) {
		for _, v := range data {
			UpdateImg(v.Logo)
		}
	}(data)
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

// UpdateToolIcon 更新工具图标
func UpdateToolIcon(id int64, logo string) {
	_, err := database.DB.Exec(`UPDATE nav_table SET logo = ? WHERE id = ?;`, logo, id)
	utils.CheckErr(err)
	UpdateImg(logo)
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

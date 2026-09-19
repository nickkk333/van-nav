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
		INSERT INTO nav_table (id, name, catelog, url, logo, "desc")
		VALUES (?, ?, ?, ?, ?, ?);
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
		_, err = stmt.Exec(v.Id, v.Name, v.Catelog, v.Url, v.Logo, v.Desc)
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

// AddTool 新增工具
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

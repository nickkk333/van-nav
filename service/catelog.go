package service

import (
	"database/sql"
	"strings"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

func UpdateCatelog(data types.UpdateCatelogDto) {

	// 查询分类原名称、原有的隐藏状态与默认栏开关
	sql_select_old_catelog_name := `select name, hide, "default" from nav_catelog where id = ?;`
	var oldName string
	var oldHide sql.NullBool
	var oldDefault sql.NullBool
	err := database.DB.QueryRow(sql_select_old_catelog_name, data.Id).Scan(&oldName, &oldHide, &oldDefault)
	utils.CheckErr(err)

	// 开启事务
	tx, err := database.DB.Begin()
	utils.CheckErr(err)

	// 更新分类新名称
	sql_update_catelog := `
		UPDATE nav_catelog
		SET name = ?, sort = ?, hide = ?, "default" = ?
		WHERE id = ?;
		`
	stmt, err := tx.Prepare(sql_update_catelog)
	utils.CheckTxErr(err, tx)
	res, err := stmt.Exec(data.Name, data.Sort, data.Hide, data.Default, data.Id)
	utils.CheckTxErr(err, tx)
	_, err = res.RowsAffected()
	utils.CheckTxErr(err, tx)

	if oldName != data.Name {
		// 更新工具分类新名称
		sql_update_tools := `
		UPDATE nav_table
		SET catelog = ?
		WHERE catelog = ?;
		`
		stmt2, err := tx.Prepare(sql_update_tools)
		utils.CheckTxErr(err, tx)
		res2, err := stmt2.Exec(data.Name, oldName)
		utils.CheckTxErr(err, tx)
		_, err = res2.RowsAffected()
		utils.CheckTxErr(err, tx)
	}

	// 隐藏状态发生变化时，同步该分类下所有工具的隐藏状态
	// （改名已在上面完成，这里统一按新名称匹配）
	if oldHide.Bool != data.Hide {
		sql_update_tools_hide := `
			UPDATE nav_table
			SET "hide" = ?
			WHERE catelog = ?;
			`
		stmt3, err := tx.Prepare(sql_update_tools_hide)
		utils.CheckTxErr(err, tx)
		res3, err := stmt3.Exec(data.Hide, data.Name)
		utils.CheckTxErr(err, tx)
		_, err = res3.RowsAffected()
		utils.CheckTxErr(err, tx)
	}

	// 默认栏开关变化时，同步该分类下所有工具的默认状态
	if oldDefault.Bool != data.Default {
		sql_update_tools_default := `
			UPDATE nav_table
			SET "default" = ?
			WHERE catelog = ?;
			`
		stmt4, err := tx.Prepare(sql_update_tools_default)
		utils.CheckTxErr(err, tx)
		res4, err := stmt4.Exec(data.Default, data.Name)
		utils.CheckTxErr(err, tx)
		_, err = res4.RowsAffected()
		utils.CheckTxErr(err, tx)
	}

	// 修改后自动重排所有分类的排序值
	utils.CheckTxErr(renumberCatelogs(tx), tx)

	// 提交事务
	err = tx.Commit()
	utils.CheckErr(err)
}

// AddCatelog 新增分类
// 排序规则：-1（默认）或负数排到最后；0 或留空排到最前；正数插入到该序号位置
// 落位完成后把所有分类的排序值重排成从 1 开始依次递增
func AddCatelog(data types.AddCatelogDto) {
	// 检查分类名称是否为空，如果为空则不创建
	if data.Name == "" || strings.TrimSpace(data.Name) == "" {
		return
	}

	// 先检查重复不重复
	existCatelogs := GetAllCatelog()
	var existCatelogsArr []string
	for _, catelogDto := range existCatelogs {
		existCatelogsArr = append(existCatelogsArr, catelogDto.Name)
	}
	if utils.In(data.Name, existCatelogsArr) {
		return
	}

	tx, err := database.DB.Begin()
	if err != nil {
		utils.CheckErr(err)
		return
	}
	defer tx.Rollback()

	// 先取出当前所有分类 id（按排序升序），用于计算新分类的落点
	rows, err := tx.Query(`SELECT id FROM nav_catelog ORDER BY sort, id;`)
	if err != nil {
		utils.CheckErr(err)
		return
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var eachId int64
		if err = rows.Scan(&eachId); err != nil {
			utils.CheckErr(err)
			rows.Close()
			return
		}
		ids = append(ids, eachId)
	}
	rows.Close()

	res, err := tx.Exec(`
		INSERT INTO nav_catelog (name, sort, hide, "default")
		VALUES (?, ?, ?, ?);
		`, data.Name, data.Sort, data.Hide, data.Default)
	if err != nil {
		utils.CheckErr(err)
		return
	}

	id, err := res.LastInsertId()
	if err != nil {
		utils.CheckErr(err)
		return
	}

	// 把新分类插到目标位置，再把所有排序值重写成 1 开始依次递增
	index := ResolveToolInsertIndex(data.Sort, len(ids))
	ordered := make([]int64, 0, len(ids)+1)
	ordered = append(ordered, ids[:index]...)
	ordered = append(ordered, id)
	ordered = append(ordered, ids[index:]...)

	stmt, err := tx.Prepare(`UPDATE nav_catelog SET sort = ? WHERE id = ?;`)
	if err != nil {
		utils.CheckErr(err)
		return
	}
	defer stmt.Close()
	for i, eachId := range ordered {
		if _, err = stmt.Exec(i+1, eachId); err != nil {
			utils.CheckErr(err)
			return
		}
	}

	err = tx.Commit()
	utils.CheckErr(err)
}

// DeleteCatelog 删除分类，同时删除该分类下的所有工具
func DeleteCatelog(id string) error {
	// 查询分类名称，用于同步删除该分类下的工具
	var name string
	err := database.DB.QueryRow(`SELECT name FROM nav_catelog WHERE id = ?;`, id).Scan(&name)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 名称为空的分类不涉及工具，避免误删「未分类」的工具
	if strings.TrimSpace(name) != "" {
		if _, err = tx.Exec(`DELETE FROM nav_table WHERE catelog = ?;`, name); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(`DELETE FROM nav_catelog WHERE id = ?;`, id); err != nil {
		return err
	}
	// 删除后自动重排剩余分类的排序值
	if err = renumberCatelogs(tx); err != nil {
		return err
	}
	return tx.Commit()
}

// renumberCatelogs 按当前顺序把所有分类的排序值重排成从 1 开始依次递增
func renumberCatelogs(tx *sql.Tx) error {
	rows, err := tx.Query(`SELECT id FROM nav_catelog ORDER BY sort, id;`)
	if err != nil {
		return err
	}
	ids := make([]int64, 0)
	for rows.Next() {
		var eachId int64
		if err = rows.Scan(&eachId); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, eachId)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return err
	}

	stmt, err := tx.Prepare(`UPDATE nav_catelog SET sort = ? WHERE id = ?;`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, eachId := range ids {
		if _, err = stmt.Exec(i+1, eachId); err != nil {
			return err
		}
	}
	return nil
}

// UpdateCatelogsSort 批量更新分类排序
func UpdateCatelogsSort(updates []types.UpdateCatelogsSortDto) error {
	tx, err := database.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`UPDATE nav_catelog SET sort = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, update := range updates {
		if _, err = stmt.Exec(update.Sort, update.Id); err != nil {
			return err
		}
	}
	// 排序后自动把所有分类的排序值重排成从 1 开始依次递增
	if err = renumberCatelogs(tx); err != nil {
		return err
	}
	return tx.Commit()
}

func GetAllCatelog() []types.Catelog {
	sql_get_all := `
		SELECT id,name,sort,hide,"default" FROM nav_catelog order by sort, id;
	`
	results := make([]types.Catelog, 0)
	rows, err := database.DB.Query(sql_get_all)
	utils.CheckErr(err)
	for rows.Next() {
		var catelog types.Catelog
		var (
			sortVal    sql.NullInt64
			hide       sql.NullBool
			defaultVal sql.NullBool
		)
		err = rows.Scan(&catelog.Id, &catelog.Name, &sortVal, &hide, &defaultVal)
		utils.CheckErr(err)
		catelog.Sort = int(sortVal.Int64)
		catelog.Hide = hide.Bool
		catelog.Default = defaultVal.Bool
		results = append(results, catelog)
	}
	defer rows.Close()
	return results
}

// ToCatelogNames 把分类转成名称数组（前台标签使用）
func ToCatelogNames(catelogs []types.Catelog) []string {
	names := make([]string, 0, len(catelogs))
	for _, catelog := range catelogs {
		if strings.TrimSpace(catelog.Name) == "" {
			continue
		}
		names = append(names, catelog.Name)
	}
	return names
}

package service

import (
	"database/sql"
	"strings"
	"time"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// ==================== 便签 ====================

// MemoVersionLimit 便签历史版本保留条数（只保留最近 N 条，超出的在保存时删除）
const MemoVersionLimit = 20

// memoMode 便签模式归一化：只认 simple / rich，其余回落 simple
func memoMode(mode string) string {
	if mode == "rich" {
		return "rich"
	}
	return "simple"
}

// GetMemo 获取便签（单文档，id 固定为 1；没有记录时返回空文档）
func GetMemo() types.Memo {
	memo := types.Memo{Id: 1, Content: "", Mode: "simple", UpdatedAt: 0}
	var (
		content sql.NullString
		mode    sql.NullString
		updated sql.NullInt64
	)
	err := database.DB.QueryRow(`SELECT content, mode, updated_at FROM nav_memo WHERE id = 1;`).Scan(&content, &mode, &updated)
	if err == sql.ErrNoRows {
		return memo
	}
	if err != nil {
		utils.CheckErr(err)
		return memo
	}
	memo.Content = content.String
	memo.Mode = memoMode(mode.String)
	memo.UpdatedAt = updated.Int64
	return memo
}

// SaveMemo 保存便签（整体覆盖）：写入前把旧文档归档一条历史版本，保留最近 MemoVersionLimit 条
func SaveMemo(content string, mode string) types.Memo {
	mode = memoMode(strings.TrimSpace(mode))
	now := time.Now().UnixMilli()
	old := GetMemo()
	if old.Content != "" || old.Mode != "" {
		if _, err := database.DB.Exec(`INSERT INTO nav_memo_version (content, mode, created_at) VALUES (?, ?, ?);`,
			old.Content, old.Mode, now); err != nil {
			utils.CheckErr(err)
		} else {
			_, _ = database.DB.Exec(`DELETE FROM nav_memo_version WHERE id NOT IN (
				SELECT id FROM nav_memo_version ORDER BY created_at DESC, id DESC LIMIT ?);`, MemoVersionLimit)
		}
	}
	if _, err := database.DB.Exec(`INSERT INTO nav_memo (id, content, mode, updated_at) VALUES (1, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET content = excluded.content, mode = excluded.mode, updated_at = excluded.updated_at;`,
		content, mode, now); err != nil {
		utils.CheckErr(err)
	}
	return types.Memo{Id: 1, Content: content, Mode: mode, UpdatedAt: now}
}

// GetMemoVersions 获取便签历史版本（按创建时间倒序）
func GetMemoVersions() []types.MemoVersion {
	rows, err := database.DB.Query(`SELECT id, content, mode, created_at FROM nav_memo_version ORDER BY created_at DESC, id DESC LIMIT ?;`, MemoVersionLimit)
	if err != nil {
		utils.CheckErr(err)
		return []types.MemoVersion{}
	}
	defer rows.Close()
	versions := make([]types.MemoVersion, 0)
	for rows.Next() {
		var v types.MemoVersion
		var (
			content sql.NullString
			mode    sql.NullString
			created sql.NullInt64
		)
		if err := rows.Scan(&v.Id, &content, &mode, &created); err != nil {
			utils.CheckErr(err)
			continue
		}
		v.Content = content.String
		v.Mode = memoMode(mode.String)
		v.CreatedAt = created.Int64
		versions = append(versions, v)
	}
	return versions
}

// RestoreMemoVersion 按版本 id 恢复便签（恢复本身也会归档当前文档，可撤销）
func RestoreMemoVersion(id int) (types.Memo, bool) {
	var (
		content sql.NullString
		mode    sql.NullString
	)
	err := database.DB.QueryRow(`SELECT content, mode FROM nav_memo_version WHERE id = ?;`, id).Scan(&content, &mode)
	if err != nil {
		if err != sql.ErrNoRows {
			utils.CheckErr(err)
		}
		return types.Memo{}, false
	}
	return SaveMemo(content.String, mode.String), true
}

// DeleteMemoVersion 删除某条便签历史版本
func DeleteMemoVersion(id int) {
	_, err := database.DB.Exec(`DELETE FROM nav_memo_version WHERE id = ?;`, id)
	utils.CheckErr(err)
}

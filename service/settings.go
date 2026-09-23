package service

import (
	"database/sql"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// GetSetting 获取网站设置
func GetSetting() types.Setting {
	row := database.DB.QueryRow(`
		SELECT id, favicon, title, govRecord, logo192, logo512, hideAdmin, hideGithub, hideToggleJumpTarget, jumpTargetBlank, backgroundImage
		FROM nav_setting
		ORDER BY id ASC
		LIMIT 1;
		`)
	var (
		setting              types.Setting
		hideAdmin            sql.NullBool
		hideGithub           sql.NullBool
		hideToggleJumpTarget sql.NullBool
		jumpTargetBlank      sql.NullBool
		backgroundImage      sql.NullString
	)
	err := row.Scan(&setting.Id, &setting.Favicon, &setting.Title, &setting.GovRecord, &setting.Logo192, &setting.Logo512,
		&hideAdmin, &hideGithub, &hideToggleJumpTarget, &jumpTargetBlank, &backgroundImage)
	if err != nil {
		logger.LogError("获取配置失败: %s", err)
		return types.Setting{
			Id:                   1,
			Favicon:              "favicon.ico",
			Title:                "Van Nav",
			GovRecord:            "",
			Logo192:              "logo192.png",
			Logo512:              "logo512.png",
			HideAdmin:            false,
			HideGithub:           false,
			HideToggleJumpTarget: false,
			JumpTargetBlank:      true,
			BackgroundImage:      "",
		}
	}
	setting.HideAdmin = hideAdmin.Bool
	setting.HideGithub = hideGithub.Bool
	setting.HideToggleJumpTarget = hideToggleJumpTarget.Bool
	// 未设置时默认新标签页打开
	setting.JumpTargetBlank = true
	if jumpTargetBlank.Valid {
		setting.JumpTargetBlank = jumpTargetBlank.Bool
	}
	// 未设置时前台不展示背景图
	setting.BackgroundImage = backgroundImage.String
	return setting
}

// UpdateSetting 更新网站设置
func UpdateSetting(data types.Setting) error {
	_, err := database.DB.Exec(`
		UPDATE nav_setting
		SET favicon = ?, title = ?, govRecord = ?, logo192 = ?, logo512 = ?, hideAdmin = ?, hideGithub = ?, hideToggleJumpTarget = ?, jumpTargetBlank = ?, backgroundImage = ?
		WHERE id = (SELECT id FROM nav_setting ORDER BY id ASC LIMIT 1);
		`, data.Favicon, data.Title, data.GovRecord, data.Logo192, data.Logo512,
		data.HideAdmin, data.HideGithub, data.HideToggleJumpTarget, data.JumpTargetBlank, data.BackgroundImage)
	return err
}

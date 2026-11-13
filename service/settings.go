package service

import (
	"database/sql"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// 首页显示配置的默认值（与前端 utils/theme.ts 保持一致）
const (
	DefaultColorTheme       = "auto"
	DefaultCardShowDuration = 5
	MinCardShowDuration     = 1
	MaxCardShowDuration     = 600
)

// GetSetting 获取网站设置
func GetSetting() types.Setting {
	row := database.DB.QueryRow(`
		SELECT id, favicon, title, govRecord, logo192, logo512, hideAdmin, hideGithub, hideMemo, jumpTargetBlank, backgroundImage, defaultLogo, defaultSearchEngine, colorTheme, cardShowDuration
		FROM nav_setting
		ORDER BY id ASC
		LIMIT 1;
		`)
	var (
		setting             types.Setting
		hideAdmin           sql.NullBool
		hideGithub          sql.NullBool
		hideMemo            sql.NullBool
		jumpTargetBlank     sql.NullBool
		backgroundImage     sql.NullString
		defaultLogo         sql.NullString
		defaultSearchEngine sql.NullInt64
		colorTheme          sql.NullString
		cardShowDuration    sql.NullInt64
	)
	err := row.Scan(&setting.Id, &setting.Favicon, &setting.Title, &setting.GovRecord, &setting.Logo192, &setting.Logo512,
		&hideAdmin, &hideGithub, &hideMemo, &jumpTargetBlank, &backgroundImage, &defaultLogo, &defaultSearchEngine,
		&colorTheme, &cardShowDuration)
	if err != nil {
		logger.LogError("获取配置失败: %s", err)
		return types.Setting{
			Id:                  1,
			Favicon:             "favicon.ico",
			Title:               "Van Nav",
			GovRecord:           "",
			Logo192:             "logo192.png",
			Logo512:             "logo512.png",
			HideAdmin:           false,
			HideGithub:          false,
			HideMemo:            false,
			JumpTargetBlank:     true,
			BackgroundImage:     "",
			DefaultLogo:         "",
			DefaultSearchEngine: 0,
			ColorTheme:          DefaultColorTheme,
			CardShowDuration:    DefaultCardShowDuration,
		}
	}
	setting.HideAdmin = hideAdmin.Bool
	setting.HideGithub = hideGithub.Bool
	// 未设置时首页显示便签入口
	setting.HideMemo = hideMemo.Bool
	// 未设置时默认新标签页打开
	setting.JumpTargetBlank = true
	if jumpTargetBlank.Valid {
		setting.JumpTargetBlank = jumpTargetBlank.Bool
	}
	// 未设置时前台使用必应每日壁纸作为背景图
	setting.BackgroundImage = backgroundImage.String
	// 未设置默认图标时前台用名称首字符占位（LogoFallback）
	setting.DefaultLogo = defaultLogo.String
	// 未设置默认搜索引擎时回落到「自动」（第一个启用的引擎）
	setting.DefaultSearchEngine = 0
	if defaultSearchEngine.Valid {
		setting.DefaultSearchEngine = int(defaultSearchEngine.Int64)
	}
	// 未设置 / 非法配色时回落到自动主题
	setting.ColorTheme = DefaultColorTheme
	if colorTheme.Valid && (colorTheme.String == "auto" || colorTheme.String == "light" || colorTheme.String == "dark") {
		setting.ColorTheme = colorTheme.String
	}
	// 未设置 / 越界的持续显示时间回落到默认值
	setting.CardShowDuration = DefaultCardShowDuration
	if cardShowDuration.Valid && cardShowDuration.Int64 >= MinCardShowDuration && cardShowDuration.Int64 <= MaxCardShowDuration {
		setting.CardShowDuration = int(cardShowDuration.Int64)
	}
	return setting
}

// UpdateSetting 更新网站设置
func UpdateSetting(data types.Setting) error {
	_, err := database.DB.Exec(`
		UPDATE nav_setting
		SET favicon = ?, title = ?, govRecord = ?, logo192 = ?, logo512 = ?, hideAdmin = ?, hideGithub = ?, hideMemo = ?, jumpTargetBlank = ?, backgroundImage = ?, defaultLogo = ?, defaultSearchEngine = ?, colorTheme = ?, cardShowDuration = ?
		WHERE id = (SELECT id FROM nav_setting ORDER BY id ASC LIMIT 1);
		`, data.Favicon, data.Title, data.GovRecord, data.Logo192, data.Logo512,
		data.HideAdmin, data.HideGithub, data.HideMemo, data.JumpTargetBlank, data.BackgroundImage, data.DefaultLogo,
		data.DefaultSearchEngine, data.ColorTheme, data.CardShowDuration)
	return err
}

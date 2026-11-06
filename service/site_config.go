package service

import (
	"database/sql"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

const (
	// DefaultCardsPerRow 首页每行展示的网站数量默认值
	DefaultCardsPerRow = 5
	// MaxCardsPerRow 首页每行展示的网站数量上限
	MaxCardsPerRow = 12
)

// NormalizeCardsPerRow 校正每行展示数量，避免出现 0、负数或过大的值
func NormalizeCardsPerRow(value int) int {
	if value < 1 {
		return DefaultCardsPerRow
	}
	if value > MaxCardsPerRow {
		return MaxCardsPerRow
	}
	return value
}

// GetSiteConfig 获取站点展示配置
func GetSiteConfig() types.SiteConfig {
	row := database.DB.QueryRow(`
		SELECT id, noImageMode, compactMode, cardsPerRow, adminHiddenColumns
		FROM nav_site_config
		ORDER BY id ASC
		LIMIT 1;
		`)
	var (
		siteConfig         types.SiteConfig
		noImageMode        sql.NullBool
		compactMode        sql.NullBool
		cardsPerRow        sql.NullInt64
		adminHiddenColumns sql.NullString
	)
	if err := row.Scan(&siteConfig.Id, &noImageMode, &compactMode, &cardsPerRow, &adminHiddenColumns); err != nil {
		logger.LogError("获取网站配置失败: %s", err)
		return types.SiteConfig{Id: 1, NoImageMode: false, CompactMode: false, CardsPerRow: DefaultCardsPerRow}
	}
	siteConfig.NoImageMode = noImageMode.Bool
	siteConfig.CompactMode = compactMode.Bool
	siteConfig.CardsPerRow = NormalizeCardsPerRow(int(cardsPerRow.Int64))
	siteConfig.AdminHiddenColumns = adminHiddenColumns.String
	return siteConfig
}

// UpdateSiteConfig 更新站点展示配置。
// 注意：后台表格隐藏列 adminHiddenColumns 由 UpdateAdminHiddenColumns 单独维护，
// 避免系统设置表单整包提交时把列配置覆盖掉
func UpdateSiteConfig(data types.SiteConfig) error {
	_, err := database.DB.Exec(`
		UPDATE nav_site_config
		SET noImageMode = ?, compactMode = ?, cardsPerRow = ?
		WHERE id = (SELECT id FROM nav_site_config ORDER BY id ASC LIMIT 1);
		`, data.NoImageMode, data.CompactMode, NormalizeCardsPerRow(data.CardsPerRow))
	return err
}

// UpdateAdminHiddenColumns 保存后台表格隐藏列配置（JSON 字符串，已由 handler 校验合法性）。
// 存服务端全局单条记录，多设备端读取同一份配置
func UpdateAdminHiddenColumns(hidden string) error {
	_, err := database.DB.Exec(`
		UPDATE nav_site_config
		SET adminHiddenColumns = ?
		WHERE id = (SELECT id FROM nav_site_config ORDER BY id ASC LIMIT 1);
		`, hidden)
	return err
}

package service

import (
	"database/sql"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
)

// GetSiteConfig 获取站点展示配置
func GetSiteConfig() types.SiteConfig {
	row := database.DB.QueryRow(`
		SELECT id, noImageMode, compactMode
		FROM nav_site_config
		ORDER BY id ASC
		LIMIT 1;
		`)
	var (
		siteConfig  types.SiteConfig
		noImageMode sql.NullBool
		compactMode sql.NullBool
	)
	if err := row.Scan(&siteConfig.Id, &noImageMode, &compactMode); err != nil {
		logger.LogError("获取网站配置失败: %s", err)
		return types.SiteConfig{Id: 1, NoImageMode: false, CompactMode: false}
	}
	siteConfig.NoImageMode = noImageMode.Bool
	siteConfig.CompactMode = compactMode.Bool
	return siteConfig
}

// UpdateSiteConfig 更新站点展示配置
func UpdateSiteConfig(data types.SiteConfig) error {
	_, err := database.DB.Exec(`
		UPDATE nav_site_config
		SET noImageMode = ?, compactMode = ?
		WHERE id = (SELECT id FROM nav_site_config ORDER BY id ASC LIMIT 1);
		`, data.NoImageMode, data.CompactMode)
	return err
}

package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mereith/nav/service"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// ==================== 便签 ====================

// GetMemoHandler 获取便签（登录后可读：便签是私人数据，未登录返回 401 由前台引导登录）
func GetMemoHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    service.GetMemo(),
	})
}

// SaveMemoHandler 保存便签（整体覆盖，空内容也允许保存，即清空）
func SaveMemoHandler(c *gin.Context) {
	var data types.SaveMemoDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "便签已保存",
		"data":    service.SaveMemo(data.Content, data.Mode),
	})
}

// GetMemoVersionsHandler 获取便签历史版本
func GetMemoVersionsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    service.GetMemoVersions(),
	})
}

// RestoreMemoVersionHandler 按版本 id 恢复便签
func RestoreMemoVersionHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	memo, ok := service.RestoreMemoVersion(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "该版本不存在",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "已恢复该版本",
		"data":    memo,
	})
}

// DeleteMemoVersionHandler 删除某条便签历史版本
func DeleteMemoVersionHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	service.DeleteMemoVersion(id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "版本已删除",
	})
}

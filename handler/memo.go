package handler

import (
	"mime"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/mereith/nav/logger"
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

// ==================== 便签附件 ====================

// UploadMemoFileHandler 上传便签附件（图片 / 文档），表单字段名为 file。
// 返回可访问地址与原始文件名：图片在前台内联显示，其它文件显示「图标 + 文件名」卡片
func UploadMemoFileHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		logger.LogError("解析便签附件失败: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "请选择要上传的文件",
		})
		return
	}
	result, err := service.SaveMemoFile(fileHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "上传成功",
		"data": gin.H{
			"name":     result.Name,
			"url":      result.Url,
			"original": result.Original,
			"size":     result.Size,
			"isImage":  result.IsImage,
		},
	})
}

// GetMemoFileHandler 输出便签附件。
// 附件是用 <img>/<a> 直接引用的，浏览器不会带上 Authorization 头，所以这个只读接口不校验登录；
// 文件名带时间戳与随机串不可枚举（与 /api/uploadedImage/:name 一致）。
// 非图片附件按下载返回，避免被浏览器当页面执行
func GetMemoFileHandler(c *gin.Context) {
	path, ok := service.GetMemoFilePath(c.Param("name"))
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "文件不存在",
		})
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "no-cache")
	if !service.IsMemoImageName(c.Param("name")) {
		// 文件名里带中文等非 ASCII 字符时 FormatMediaType 会自动按 RFC 2231 编码
		c.Header("Content-Disposition", mime.FormatMediaType("attachment", map[string]string{
			"filename": service.MemoFileDownloadName(c.Param("name")),
		}))
	}
	c.File(path)
}

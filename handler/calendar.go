package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mereith/nav/service"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// ==================== 日历事件 ====================

// GetCalendarEventsHandler 按月份获取日历事件（month=YYYY-MM，不传返回全部）
func GetCalendarEventsHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    service.GetCalendarEvents(c.Query("month")),
	})
}

// AddCalendarEventHandler 新增日历事件（标题与日期必填）
func AddCalendarEventHandler(c *gin.Context) {
	var data types.SaveCalendarEventDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	if strings.TrimSpace(data.Title) == "" || strings.TrimSpace(data.Date) == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "标题与日期不能为空",
		})
		return
	}
	event, ok := service.AddCalendarEvent(data)
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "标题与日期不能为空",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "事件已添加",
		"data":    event,
	})
}

// UpdateCalendarEventHandler 更新日历事件
func UpdateCalendarEventHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	var data types.SaveCalendarEventDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	event, ok := service.UpdateCalendarEvent(id, data)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "该事件不存在或标题日期为空",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "事件已更新",
		"data":    event,
	})
}

// DeleteCalendarEventHandler 删除日历事件
func DeleteCalendarEventHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	if !service.DeleteCalendarEvent(id) {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "该事件不存在",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "事件已删除",
	})
}

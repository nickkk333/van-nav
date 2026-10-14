package service

import (
	"database/sql"
	"strings"
	"time"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

// calendarTime 时间归一化：只接受 HH:mm，空串表示全天
func calendarTime(t string) string {
	t = strings.TrimSpace(t)
	if t == "" {
		return ""
	}
	if _, err := time.Parse("15:04", t); err != nil {
		return ""
	}
	return t
}

// calendarColor 颜色归一化：只接受 #rrggbb，非法回落空串（前台用默认色）
func calendarColor(color string) string {
	color = strings.TrimSpace(color)
	if len(color) == 7 && strings.HasPrefix(color, "#") {
		valid := true
		for _, r := range color[1:] {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
				valid = false
				break
			}
		}
		if valid {
			return strings.ToLower(color)
		}
	}
	return ""
}

// calendarDate 日期归一化：只接受 YYYY-MM-DD
func calendarDate(date string) string {
	date = strings.TrimSpace(date)
	if date == "" {
		return ""
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return ""
	}
	return date
}

// GetCalendarEvents 按月份获取日历事件（month 为 YYYY-MM，空串返回全部）
func GetCalendarEvents(month string) []types.CalendarEvent {
	month = strings.TrimSpace(month)
	var rows *sql.Rows
	var err error
	if month == "" {
		rows, err = database.DB.Query(`SELECT id, title, date, start_time, end_time, "desc", color, done, created_at, updated_at
			FROM nav_calendar_event ORDER BY date, start_time, id;`)
	} else {
		if _, parseErr := time.Parse("2006-01", month); parseErr != nil {
			return []types.CalendarEvent{}
		}
		rows, err = database.DB.Query(`SELECT id, title, date, start_time, end_time, "desc", color, done, created_at, updated_at
			FROM nav_calendar_event WHERE substr(date, 1, 7) = ? ORDER BY date, start_time, id;`, month)
	}
	if err != nil {
		utils.CheckErr(err)
		return []types.CalendarEvent{}
	}
	defer rows.Close()
	events := make([]types.CalendarEvent, 0)
	for rows.Next() {
		var event types.CalendarEvent
		var (
			title     sql.NullString
			date      sql.NullString
			startTime sql.NullString
			endTime   sql.NullString
			desc      sql.NullString
			color     sql.NullString
			done      sql.NullBool
			created   sql.NullInt64
			updated   sql.NullInt64
		)
		if err := rows.Scan(&event.Id, &title, &date, &startTime, &endTime, &desc, &color, &done, &created, &updated); err != nil {
			utils.CheckErr(err)
			continue
		}
		event.Title = title.String
		event.Date = date.String
		event.StartTime = startTime.String
		event.EndTime = endTime.String
		event.Desc = desc.String
		event.Color = color.String
		event.Done = done.Bool
		event.CreatedAt = created.Int64
		event.UpdatedAt = updated.Int64
		events = append(events, event)
	}
	return events
}

// AddCalendarEvent 新增日历事件
func AddCalendarEvent(data types.SaveCalendarEventDto) (types.CalendarEvent, bool) {
	date := calendarDate(strings.TrimSpace(data.Date))
	if date == "" || strings.TrimSpace(data.Title) == "" {
		return types.CalendarEvent{}, false
	}
	startTime := calendarTime(data.StartTime)
	endTime := calendarTime(data.EndTime)
	if startTime != "" && endTime != "" && endTime < startTime {
		startTime, endTime = endTime, startTime
	}
	now := time.Now().UnixMilli()
	res, err := database.DB.Exec(`INSERT INTO nav_calendar_event (title, date, start_time, end_time, "desc", color, done, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);`,
		strings.TrimSpace(data.Title), date, startTime, endTime, strings.TrimSpace(data.Desc), calendarColor(data.Color), data.Done, now, now)
	if err != nil {
		utils.CheckErr(err)
		return types.CalendarEvent{}, false
	}
	id, _ := res.LastInsertId()
	return types.CalendarEvent{Id: int(id), Title: strings.TrimSpace(data.Title), Date: date,
		StartTime: startTime, EndTime: endTime, Desc: strings.TrimSpace(data.Desc),
		Color: calendarColor(data.Color), Done: data.Done, CreatedAt: now, UpdatedAt: now}, true
}

// UpdateCalendarEvent 更新日历事件
func UpdateCalendarEvent(id int, data types.SaveCalendarEventDto) (types.CalendarEvent, bool) {
	date := calendarDate(strings.TrimSpace(data.Date))
	if date == "" || strings.TrimSpace(data.Title) == "" {
		return types.CalendarEvent{}, false
	}
	startTime := calendarTime(data.StartTime)
	endTime := calendarTime(data.EndTime)
	if startTime != "" && endTime != "" && endTime < startTime {
		startTime, endTime = endTime, startTime
	}
	now := time.Now().UnixMilli()
	res, err := database.DB.Exec(`UPDATE nav_calendar_event
		SET title = ?, date = ?, start_time = ?, end_time = ?, "desc" = ?, color = ?, done = ?, updated_at = ?
		WHERE id = ?;`,
		strings.TrimSpace(data.Title), date, startTime, endTime, strings.TrimSpace(data.Desc), calendarColor(data.Color), data.Done, now, id)
	if err != nil {
		utils.CheckErr(err)
		return types.CalendarEvent{}, false
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.CalendarEvent{}, false
	}
	event := types.CalendarEvent{Id: id, Title: strings.TrimSpace(data.Title), Date: date,
		StartTime: startTime, EndTime: endTime, Desc: strings.TrimSpace(data.Desc),
		Color: calendarColor(data.Color), Done: data.Done, UpdatedAt: now}
	_ = database.DB.QueryRow(`SELECT created_at FROM nav_calendar_event WHERE id = ?;`, id).Scan(&event.CreatedAt)
	return event, true
}

// ToggleCalendarEventDone 打勾/取消打勾：已完成视为已结束，不再触发小红点提醒
func ToggleCalendarEventDone(id int, done bool) (types.CalendarEvent, bool) {
	now := time.Now().UnixMilli()
	res, err := database.DB.Exec(`UPDATE nav_calendar_event SET done = ?, updated_at = ? WHERE id = ?;`, done, now, id)
	if err != nil {
		utils.CheckErr(err)
		return types.CalendarEvent{}, false
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return types.CalendarEvent{}, false
	}
	events := GetCalendarEvents("")
	for _, e := range events {
		if e.Id == id {
			return e, true
		}
	}
	return types.CalendarEvent{Id: id}, true
}

// DeleteCalendarEvent 删除日历事件
func DeleteCalendarEvent(id int) bool {
	res, err := database.DB.Exec(`DELETE FROM nav_calendar_event WHERE id = ?;`, id)
	if err != nil {
		utils.CheckErr(err)
		return false
	}
	affected, _ := res.RowsAffected()
	return affected > 0
}

package service

// 导入备份时日历/便签的合并行为测试：
// 保留现有数据，备份内容追加进来，重复的跳过（不整表替换），且重复导入幂等。

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/types"
)

func TestImportBackupMergesProductivity(t *testing.T) {
	// InitDB 固定使用 ./data：切到独立临时目录里跑，避免污染仓库
	// （不用 t.TempDir()：Windows 上当前工作目录被进程占用，要先退出目录再删除）
	tmpDir := filepath.Join(os.TempDir(), fmt.Sprintf("van-nav-import-test-%d", os.Getpid()))
	if err := os.RemoveAll(tmpDir); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(tmpDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	database.InitDB()

	// 当前数据：便签有内容；日历两条事件（其中一条已打勾）
	SaveMemo("当前便签", "simple")
	meeting, ok := AddCalendarEvent(types.SaveCalendarEventDto{Title: "开会", Date: "2026-10-20", StartTime: "10:00", EndTime: "11:00", Desc: "周会"})
	if !ok {
		t.Fatal("seed AddCalendarEvent 失败")
	}
	if _, ok = ToggleCalendarEventDone(meeting.Id, true); !ok {
		t.Fatal("seed ToggleCalendarEventDone 失败")
	}
	if _, ok = AddCalendarEvent(types.SaveCalendarEventDto{Title: "保留事件", Date: "2026-10-21"}); !ok {
		t.Fatal("seed AddCalendarEvent 失败")
	}

	backup := types.BackupData{
		Version: 4,
		Memo:    &types.Memo{Id: 1, Content: "导入的便签", Mode: "simple", UpdatedAt: 1000},
		MemoVersions: []types.MemoVersion{
			{Content: "历史版本A", Mode: "simple", CreatedAt: 1000},
			{Content: "历史版本A", Mode: "simple", CreatedAt: 1000}, // 重复条目，应跳过
			{Content: "历史版本B", Mode: "simple", CreatedAt: 2000},
		},
		CalendarEvents: []types.CalendarEvent{
			// 与现有「开会」完全相同：应跳过（保留现有 id 与打勾状态）
			{Title: "开会", Date: "2026-10-20", StartTime: "10:00", EndTime: "11:00", Desc: "周会"},
			// 新事件：应追加
			{Title: "新事件", Date: "2026-10-22", StartTime: "09:00", EndTime: "09:30"},
			// 缺日期：应忽略
			{Title: "坏数据"},
		},
	}

	for round := 1; round <= 2; round++ { // 导入两轮，验证幂等（重复导入不翻倍）
		if err := ImportBackupData(backup); err != nil {
			t.Fatalf("第 %d 轮导入失败: %v", round, err)
		}

		// 便签：保留当前内容，不被导入内容覆盖
		if got := GetMemo().Content; got != "当前便签" {
			t.Fatalf("第 %d 轮后当前便签应保留为「当前便签」，实际 %q", round, got)
		}

		// 历史版本：现有版本 + 导入的便签内容 + 版本A/B 各一条（重复的 A 跳过）
		countByContent := map[string]int{}
		for _, v := range GetMemoVersions() {
			countByContent[v.Content]++
		}
		if countByContent["导入的便签"] != 1 {
			t.Fatalf("第 %d 轮后「导入的便签」应归档为 1 条历史版本，实际 %d", round, countByContent["导入的便签"])
		}
		if countByContent["历史版本A"] != 1 {
			t.Fatalf("第 %d 轮后「历史版本A」应只有 1 条（重复跳过），实际 %d", round, countByContent["历史版本A"])
		}
		if countByContent["历史版本B"] != 1 {
			t.Fatalf("第 %d 轮后「历史版本B」应只有 1 条，实际 %d", round, countByContent["历史版本B"])
		}

		// 日历事件：现有的 2 条 + 导入的 1 条新事件 = 3 条；坏数据被忽略
		events := GetCalendarEvents("")
		if len(events) != 3 {
			t.Fatalf("第 %d 轮后应有 3 条事件，实际 %d 条: %+v", round, len(events), events)
		}
		byTitle := map[string]types.CalendarEvent{}
		for _, e := range events {
			byTitle[e.Title] = e
		}
		if e, exists := byTitle["开会"]; !exists || e.Id != meeting.Id || !e.Done {
			t.Fatalf("第 %d 轮后「开会」应保留原记录（id=%d 且已打勾），实际 %+v", round, meeting.Id, e)
		}
		if _, exists := byTitle["新事件"]; !exists {
			t.Fatalf("第 %d 轮后应包含导入的新事件", round)
		}
	}

	// 收尾：关库、退出临时目录后清理（Windows 上 cwd 与未释放的句柄都会阻止删除）
	database.DB.SetMaxIdleConns(0)
	if err := database.DB.Close(); err != nil {
		t.Fatalf("关闭数据库失败: %v", err)
	}
	if err := os.Chdir(os.TempDir()); err != nil {
		t.Fatal(err)
	}
	var rmErr error
	for i := 0; i < 50; i++ {
		if rmErr = os.RemoveAll(tmpDir); rmErr == nil {
			break
		}
		runtime.GC()
		time.Sleep(100 * time.Millisecond)
	}
	if rmErr != nil {
		t.Fatalf("清理临时目录失败: %v", rmErr)
	}
}
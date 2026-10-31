package service

// 导入备份时便签的合并行为测试：
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

	// 当前数据：便签有内容
	SaveMemo("当前便签", "simple")

	backup := types.BackupData{
		Version: 5,
		Memo:    &types.Memo{Id: 1, Content: "导入的便签", Mode: "simple", UpdatedAt: 1000},
		MemoVersions: []types.MemoVersion{
			{Content: "历史版本A", Mode: "simple", CreatedAt: 1000},
			{Content: "历史版本A", Mode: "simple", CreatedAt: 1000}, // 重复条目，应跳过
			{Content: "历史版本B", Mode: "simple", CreatedAt: 2000},
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
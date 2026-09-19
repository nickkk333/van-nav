package main

import (
	"context"
	"embed"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/mereith/nav/database"
	"github.com/mereith/nav/handler"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/middleware"
)

const INDEX = "index.html"

// 构建信息，通过 -ldflags "-X main.version=..." 注入
var (
	version = "dev"
	commit  = "none"
)

//go:embed public
var fs embed.FS

// binaryFileSystem 把 embed.FS 包装成 http.FileSystem
type binaryFileSystem struct {
	fs   http.FileSystem
	root string
}

func (b *binaryFileSystem) Open(name string) (http.File, error) {
	return b.fs.Open(path.Join(b.root, name))
}

func (b *binaryFileSystem) Exists(prefix string, filepath string) bool {
	p := strings.TrimPrefix(filepath, prefix)
	if len(p) >= len(filepath) {
		return false
	}
	name := path.Join(b.root, p)
	if p == "" {
		name = path.Join(b.root, INDEX)
	}
	if _, err := b.fs.Open(name); err != nil {
		return false
	}
	return true
}

// BinaryFileSystem 创建内嵌静态文件系统
func BinaryFileSystem(data embed.FS, root string) *binaryFileSystem {
	return &binaryFileSystem{
		fs:   http.FS(data),
		root: root,
	}
}

var port = flag.String("port", "6412", "指定监听端口")
var addr = flag.String("addr", "0.0.0.0", "指定监听地址")

func main() {
	flag.Parse()

	database.InitDB()

	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()
	router.Use(gzip.Gzip(gzip.DefaultCompression, gzip.WithExcludedExtensions([]string{".png", ".jpg", ".jpeg", ".ico", ".svg"})))

	// 动态生成的 manifest
	router.GET("/manifest.json", handler.ManifastHanlder)
	// 内嵌前端静态资源（单页应用，找不到的路径回落到 index.html）
	router.Use(Serve("/", BinaryFileSystem(fs, "public")))

	api := router.Group("/api")
	{
		// 获取数据的路由
		api.GET("/", handler.GetAllHandler)
		api.POST("/login", handler.LoginHandler)
		api.GET("/logout", handler.LogoutHandler)
		api.GET("/img", handler.GetLogoImgHandler)

		// 获取启用的搜索引擎（公开接口）
		api.GET("/searchEngines", handler.GetEnabledSearchEnginesHandler)

		// 管理员用的
		admin := api.Group("/admin")
		admin.Use(middleware.JWTMiddleware())
		{
			admin.POST("/apiToken", handler.AddApiTokenHandler)
			admin.DELETE("/apiToken/:id", handler.DeleteApiTokenHandler)
			admin.GET("/all", handler.GetAdminAllDataHandler)

			admin.GET("/exportTools", handler.ExportToolsHandler)
			admin.POST("/importTools", handler.ImportToolsHandler)

			admin.PUT("/user", handler.UpdateUserHandler)
			admin.PUT("/setting", handler.UpdateSettingHandler)
			admin.PUT("/siteConfig", handler.UpdateSiteConfigHandler)

			admin.POST("/tool", handler.AddToolHandler)
			admin.DELETE("/tool/:id", handler.DeleteToolHandler)
			admin.PUT("/tool/:id", handler.UpdateToolHandler)
			admin.PUT("/tools/sort", handler.UpdateToolsSortHandler)

			admin.POST("/catelog", handler.AddCatelogHandler)
			admin.DELETE("/catelog/:id", handler.DeleteCatelogHandler)
			admin.PUT("/catelog/:id", handler.UpdateCatelogHandler)

			// 搜索引擎管理路由
			admin.GET("/searchEngine", handler.GetAllSearchEnginesHandler)
			admin.POST("/searchEngine", handler.AddSearchEngineHandler)
			admin.PUT("/searchEngine/:id", handler.UpdateSearchEngineHandler)
			admin.DELETE("/searchEngine/:id", handler.DeleteSearchEngineHandler)
			admin.PUT("/searchEngines/sort", handler.UpdateSearchEngineSortHandler)
		}
	}

	listen := fmt.Sprintf("%s:%s", *addr, *port)
	srv := &http.Server{
		Addr:         listen,
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.LogInfo("Van Nav %s (commit %s) 启动成功，网址: http://localhost:%s", version, commit, *port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.LogError("应用启动失败，错误: %s", err)
			os.Exit(1)
		}
	}()

	// 优雅退出
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	logger.LogInfo("收到退出信号，正在关闭服务...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.LogError("关闭服务失败: %s", err)
	}
	logger.LogInfo("服务已退出")
}

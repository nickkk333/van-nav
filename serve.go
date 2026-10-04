package main

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mereith/nav/logger"
)

type ServeFileSystem interface {
	http.FileSystem
	Exists(prefix string, path string) bool
}

func Serve(urlPrefix string, fs ServeFileSystem) gin.HandlerFunc {
	fileserver := http.FileServer(fs)
	if urlPrefix != "" {
		fileserver = http.StripPrefix(urlPrefix, fileserver)
	}
	return func(c *gin.Context) {
		if fs.Exists(urlPrefix, c.Request.URL.Path) {
			// 带 hash 的构建产物（/assets/index-*.js 等）内容不可变，给一年长缓存；
			// index.html 与其它入口保持不缓存，保证发版后立刻拿到新引用
			if strings.HasPrefix(c.Request.URL.Path, "/assets/") {
				c.Header("Cache-Control", "public, max-age=31536000, immutable")
			} else {
				c.Header("Cache-Control", "no-cache")
			}
			fileserver.ServeHTTP(c.Writer, c.Request)
			c.Abort()
		} else {
			path := c.Request.URL.Path
			// /api 开头的是接口（/api-token 页面除外，它是前端路由）；其它未知路径回落到 index.html
			pathHasAPI := strings.HasPrefix(path, "/api/") && !strings.HasPrefix(path, "/api-token")
			// pathHasAdmin := strings.Contains(path, "/admin")
			// pathHasLogin := strings.Contains(path, "/login")
			if pathHasAPI {
				return
			} else {
				file, err := fs.Open("index.html")
				if err != nil {
					logger.LogError("文件不存在: %s", c.Request.URL.Path)
					return
				}
				defer file.Close()
				// 把文件返回
				c.Header("Cache-Control", "no-cache")
				http.ServeContent(c.Writer, c.Request, "index.html", time.Now(), file)
				c.Abort()
			}

		}
	}
}

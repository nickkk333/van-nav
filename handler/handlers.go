package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/middleware"
	"github.com/mereith/nav/service"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

func ExportToolsHandler(c *gin.Context) {
	tools := service.GetAllTool()
	c.JSON(200, gin.H{
		"success": true,
		"message": "导出工具成功",
		"data":    tools,
	})
}

// ExportAllHandler 导出所有工具、分类、搜索引擎与 api token
// 图标只导出图标网址：工具的图片名（logoName）与搜索引擎的 logo 都不导出，导入后由服务端按网址重新获取图标
func ExportAllHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"message": "导出数据成功",
		"data":    service.ExportBackupData(),
	})
}

// ImportAllHandler 导入备份数据（工具、分类、搜索引擎与 api token）
// 同 id 的记录会被覆盖，未出现在备份里的记录保持不动；导入后服务端会自动获取图片
func ImportAllHandler(c *gin.Context) {
	var data types.BackupData
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	if len(data.Tools) == 0 && len(data.Catelogs) == 0 && len(data.SearchEngines) == 0 && len(data.ApiTokens) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "备份文件内容为空",
		})
		return
	}
	if err := service.ImportBackupData(data); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "导入数据成功",
		"data": gin.H{
			"tools":         len(data.Tools),
			"catelogs":      len(data.Catelogs),
			"searchEngines": len(data.SearchEngines),
			"apiTokens":     len(data.ApiTokens),
		},
	})
}

func ImportToolsHandler(c *gin.Context) {
	var tools []types.Tool
	err := c.ShouldBindJSON(&tools)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	// 导入所有工具（兼容旧接口，只处理工具）
	if err := service.ImportTools(tools); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "导入工具成功",
	})
}

func DeleteApiTokenHandler(c *gin.Context) {
	// 删除 Token
	id := c.Param("id")
	sql_delete_api_token := `
		UPDATE nav_api_token
		SET disabled = 1
		WHERE id = ?;
		`
	stmt, err := database.DB.Prepare(sql_delete_api_token)
	utils.CheckErr(err)
	res, err := stmt.Exec(id)
	utils.CheckErr(err)
	_, err = res.RowsAffected()
	utils.CheckErr(err)
	c.JSON(200, gin.H{
		"success": true,
		"message": "删除 API Token 成功",
	})
}

func AddApiTokenHandler(c *gin.Context) {
	var token types.AddTokenDto
	err := c.ShouldBindJSON(&token)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	newId := utils.GenerateId()
	var signedJwt string
	signedJwt, err = utils.SignJWTForAPI(token.Name, newId)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	service.AddApiTokenInDB(types.Token{
		Name:     token.Name,
		Value:    signedJwt,
		Id:       newId,
		Disabled: 0,
	})
	// 签名 jwt
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"id":    newId,
			"Value": signedJwt,
			"Name":  token.Name,
		},
		"message": "添加 Token 成功",
	})
}

func UpdateSettingHandler(c *gin.Context) {
	var data types.Setting
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	logger.LogInfo("更新配置: %+v", data)
	oldSetting := service.GetSetting()
	err := service.UpdateSetting(data)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	// 清理被替换掉的上传图片
	service.CleanupReplacedUploadedImages(oldSetting, data)
	c.JSON(200, gin.H{
		"success": true,
		"message": "更新配置成功",
	})
}

func UpdateUserHandler(c *gin.Context) {
	var data types.UpdateUserDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	if err := service.UpdateUser(data); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "更新用户成功",
	})
}

func UpdateSiteConfigHandler(c *gin.Context) {
	var data types.SiteConfig
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	logger.LogInfo("更新网站配置: %+v", data)
	err := service.UpdateSiteConfig(data)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "更新网站配置成功",
	})
}

func GetAllHandler(c *gin.Context) {
	tools := service.GetAllTool()
	// 获取全部数据
	catelogs := service.GetAllCatelog()
	if !utils.IsLogin(c) {
		// 过滤掉隐藏工具
		tools = utils.FilterHideTools(tools, catelogs)

		// 过滤掉隐藏分类
		catelogs = utils.FilterHideCates(catelogs)
	}
	setting := service.GetSetting()
	siteConfig := service.GetSiteConfig()
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"tools":      tools,
			"catelogs":   service.ToCatelogNames(catelogs),
			"setting":    setting,
			"siteConfig": siteConfig,
		},
	})
}

func GetLogoImgHandler(c *gin.Context) {
	url := c.Query("url")
	if url == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "URL参数不能为空",
		})
		return
	}
	// 只从图片缓存里取图标（外链图标在保存工具时下载到 data/imgcache）：缓存里没有就返回 404，
	// 前台 <img> 会触发 error 事件并用名称首字符占位（ui/src/components/LogoFallback.vue）；
	// 这里不能返回内置的灰圈占位图，否则前台会以为图片加载成功，把占位图当成网站图标显示
	// 内置图标文件名（例如 baidu.ico）随前端一起内嵌在 public 里，前端按静态资源路径直接请求，不走这个接口
	imgBuffer := service.GetCachedImg(url)
	if len(imgBuffer) == 0 {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "未找到图片",
		})
		return
	}
	// 按内容嗅探真实类型（缓存文件没有扩展名，老实现按 url 后缀猜，gstatic 这类无后缀地址会一律判成 x-icon）
	// 直接输出二进制数据，避免 string 转换导致的内存多分配
	c.Data(http.StatusOK, service.CachedImgContentType(url, imgBuffer), imgBuffer)
}

// UploadImageHandler 上传图片（背景图 / logo 等），返回可直接使用的 url
func UploadImageHandler(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		logger.LogError("解析上传文件失败: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "请选择要上传的图片",
		})
		return
	}
	url, err := service.SaveUploadedImage(fileHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "上传成功",
		"data": gin.H{
			"url": url,
		},
	})
}

// GetBingWallpaperHandler 输出本地保存的必应每日壁纸（前台未配置背景图时的默认背景）
func GetBingWallpaperHandler(c *gin.Context) {
	path, ok := service.GetBingWallpaperPath()
	if !ok {
		// 本地还没有壁纸（首次启动且后台下载未完成 / 之前下载失败）时，同步下载一次再返回
		service.DownloadBingWallpaper()
		path, ok = service.GetBingWallpaperPath()
	}
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "必应每日壁纸不存在",
		})
		return
	}
	// 壁纸每天都会更新（文件名带日期），因此不做长时间缓存
	c.Header("Cache-Control", "no-cache")
	c.File(path)
}

// GetBingWallpaperInfoHandler 返回必应每日壁纸的信息（图片描述），供前台搜索框 placeholder 显示
// 描述解析自壁纸文件名（命名规则见 service.bingWallpaperFileName）；本地还没有壁纸时先同步下载一次（与图片接口一致），
// 拿不到描述时 title 为空字符串，前台回落到默认 placeholder
func GetBingWallpaperInfoHandler(c *gin.Context) {
	if _, ok := service.GetBingWallpaperPath(); !ok {
		// 与 GetBingWallpaperHandler 相同的兜底：首次启动后台下载未完成时同步等一次，保证能拿到当天的描述
		service.DownloadBingWallpaper()
	}
	// 文件名带日期，每天都会变，不做缓存
	c.Header("Cache-Control", "no-cache")
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"title": service.BingWallpaperTitle(),
		},
	})
}

// GetUploadedImageHandler 输出后台上传的图片
func GetUploadedImageHandler(c *gin.Context) {
	path, ok := service.GetUploadedImagePath(c.Param("name"))
	if !ok {
		// 图片可能稍后才生成（保存工具时会重新下载），这里不做缓存，
		// 避免浏览器一直用这张“图片不存在”的响应
		c.Header("Cache-Control", "no-cache")
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "图片不存在",
		})
		return
	}
	// 上传的图片带唯一文件名，但工具 logo 是按名称保存的（重新下载会覆盖同名文件），
	// 因此不做长时间缓存：浏览器带 If-Modified-Since 回源校验，图片换了能立刻拿到新的，没换返回 304
	c.Header("Cache-Control", "no-cache")
	c.File(path)
}

func GetAdminAllDataHandler(c *gin.Context) {
	// 管理员获取全部数据，还有个用户名。
	tools := service.GetAllTool()
	catelogs := service.GetAllCatelog()
	setting := service.GetSetting()
	siteConfig := service.GetSiteConfig()
	tokens := service.GetApiTokens()
	userId, ok := c.Get("uid")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "不存在该用户！",
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"data": gin.H{
			"tools":      tools,
			"catelogs":   catelogs,
			"setting":    setting,
			"siteConfig": siteConfig,
			"user": gin.H{
				"name": c.GetString("username"),
				"id":   userId,
			},
			"tokens": tokens,
		},
	})
}

func LoginHandler(c *gin.Context) {
	var data types.LoginDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	user := service.GetUser(data.Name)
	// 用户名与密码错误返回同一文案，避免攻击者枚举用户名；失败计入限流
	if user.Name == "" || !utils.CheckPassword(data.Password, user.Password) {
		middleware.LoginFail(c)
		c.JSON(200, gin.H{
			"success":      false,
			"errorMessage": "用户名或密码错误",
		})
		return
	}
	middleware.LoginSuccess(c)
	// 存量明文密码在登录成功后原地升级为 bcrypt 哈希（见 service.UpgradePasswordHash）
	if !utils.IsPasswordHash(user.Password) {
		service.UpgradePasswordHash(user.Id, data.Password)
	}
	// 生成 token
	token, err := utils.SignJWT(user)
	utils.CheckErr(err)

	c.JSON(200, gin.H{
		"success": true,
		"message": "登录成功",
		"data": gin.H{
			"user":  user,
			"token": token,
		},
	})

}

// 退出登录
func LogoutHandler(c *gin.Context) {
	c.JSON(200, gin.H{
		"success": true,
		"message": "登出成功",
	})
}

func AddToolHandler(c *gin.Context) {
	// 添加工具
	var data types.AddToolDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	// 排序落点与全表排序值重排都由 service.AddTool 处理：
	// -1（默认）或负数排到最后；0 或留空排到最前；正数插入到该序号位置，最终排序值从 1 开始依次递增
	// 图标先按提交的值入库，不在这里同步抓取（抓图标要访问目标站点，最长 10s+，会把接口卡住）
	id, err := service.AddTool(data)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	// 工具已入库，图标交给后台抓：抓到后回写 nav_table，前台先用名称首字符占位
	fetching := service.StartToolLogoFetchAsync(int(id), data.Name, data.Url, data.Logo, data.LogoName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "添加成功",
		"data": gin.H{
			"id": id,
			// 图标还在后台抓，前台据此提示「图标稍后自动补上」
			"logoFetching": fetching,
		},
	})
}

func DeleteToolHandler(c *gin.Context) {
	// 删除工具
	id := c.Param("id")
	// 先取出图标网址与本地图片名，删除记录后就查不到了：图标缓存文件和本地保存的图片都要清理
	numberId, err := strconv.Atoi(id)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	logo, logoName := service.GetToolLogoById(numberId)
	if _, err := database.DB.Exec(`DELETE FROM nav_table WHERE id = ?;`, numberId); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": "删除失败",
		})
		return
	}
	// 删除该工具的外链图标文件缓存（还有别的工具在用同一网址时不删）
	service.RemoveCachedImgIfUnused(logo)
	// 同时删除该工具保存在本地的图标图片（还有别的工具在用时不删）
	service.RemoveToolLogoIfUnused(logo, logoName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "删除成功",
	})
}

// GetUrlInfoHandler 抓取网址信息，供后台添加工具时自动填充
func GetUrlInfoHandler(c *gin.Context) {
	target := strings.TrimSpace(c.Query("url"))
	if target == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "缺少 url 参数",
		})
		return
	}
	if !strings.HasPrefix(target, "http://") && !strings.HasPrefix(target, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "网址必须以 http:// 或 https:// 开头",
		})
		return
	}
	logger.LogInfo("获取网址信息: %s", target)
	c.JSON(200, gin.H{
		"success": true,
		"message": "获取网址信息成功",
		"data":    service.GetUrlInfo(target),
	})
}

func UpdateToolHandler(c *gin.Context) {
	// 更新工具
	var data types.UpdateToolDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	// 先取出旧的图标网址与本地图片名：用于判断 logo 网址有没有被改动（没改动时图标按原有规则处理）
	oldLogo, oldLogoName := service.GetToolLogoById(data.Id)
	// 处理图标：logo 网址被改成图片外链时先按填写的地址下载这张图片（下载不到则回退），
	// 改成空与其他内容时按工具网址抓取网站图标（抓不到用 gstatic 兜底），拿到图标就下载到 data/images：
	// logo 存图标网址、logoName 存图片名；都拿不到则都置空，前台显示默认图标 default.png
	resolved := service.ResolveUpdatedToolLogo(data.Name, data.Url, data.Logo, data.LogoName, oldLogo)
	data.Logo, data.LogoName = resolved.Logo, resolved.LogoName
	service.UpdateTool(data)
	// 图标换掉（含改名后文件名变化、或置空）时删除旧的本地图片
	if (oldLogoName != "" || oldLogo != "") && (oldLogoName != data.LogoName || oldLogo != data.Logo) {
		service.RemoveToolLogoIfUnused(oldLogo, oldLogoName)
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新成功",
	})
}

// UploadToolLogoHandler 给工具上传本地图标：图片按「工具名称 + 图片后缀」命名保存到 data 目录
// （data/images，与按网址自动下载的图标命名一致），并更新该工具的图标网址与 logo 图片名
func UploadToolLogoHandler(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		logger.LogError("解析上传文件失败: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "请选择要上传的图片",
		})
		return
	}
	// 图片名用工具名称生成，所以先取出名称；原图标用于上传后清理被替换掉的旧图片
	name, oldLogo, oldLogoName, ok := service.GetToolNameAndLogoById(id)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{
			"success":      false,
			"errorMessage": "工具不存在",
		})
		return
	}
	savedName, err := service.SaveToolLogoFromUpload(name, fileHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	logo := service.UploadUrlPrefix + savedName
	if err := service.UpdateToolLogo(id, logo, savedName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": "更新工具图标失败",
		})
		return
	}
	// 图标换掉（文件名变化）时删除旧的本地图片，还有别的工具在用时不删
	if oldLogoName != savedName || oldLogo != logo {
		service.RemoveToolLogoIfUnused(oldLogo, oldLogoName)
	}
	logger.LogInfo("工具 %s 的图标已上传到本地: %s", name, savedName)
	c.JSON(200, gin.H{
		"success": true,
		"message": "图片已保存到 data 目录",
		"data": gin.H{
			"name": savedName,
			"url":  logo,
		},
	})
}

func AddCatelogHandler(c *gin.Context) {
	// 添加分类
	var data types.AddCatelogDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	service.AddCatelog(data)

	c.JSON(200, gin.H{
		"success": true,
		"message": "增加分类成功",
	})
}

func DeleteCatelogHandler(c *gin.Context) {
	// 删除分类（同时删除该分类下的所有工具）
	id := c.Param("id")
	err := service.DeleteCatelog(id)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"message": "删除分类成功",
	})
}

func UpdateCatelogHandler(c *gin.Context) {
	// 更新分类
	var data types.UpdateCatelogDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	service.UpdateCatelog(data)

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新分类成功",
	})
}

func UpdateCatelogsSortHandler(c *gin.Context) {
	var updates []types.UpdateCatelogsSortDto
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	err := service.UpdateCatelogsSort(updates)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新排序成功",
	})
}

func ManifastHanlder(c *gin.Context) {

	setting := service.GetSetting()
	title := setting.Title

	var icons = []gin.H{}

	logo192 := setting.Logo192
	if logo192 == "" {
		logo192 = "logo192.png"
	}

	logo512 := setting.Logo512
	if logo512 == "" {
		logo512 = "logo512.png"
	}

	icons = append(icons, gin.H{
		"src":   logo192,
		"type":  "image/png",
		"sizes": "192x192",
	})
	icons = append(icons, gin.H{
		"src":   logo512,
		"type":  "image/png",
		"sizes": "512x512",
	})

	if title == "" {
		title = "Van nav"
	}
	c.JSON(200, gin.H{
		"short_name":       title,
		"name":             title,
		"icons":            icons,
		"start_url":        "/",
		"display":          "standalone",
		"scope":            "/",
		"theme_color":      "#000000",
		"background_color": "#ffffff",
	})
}

func UpdateToolsSortHandler(c *gin.Context) {
	var updates []types.UpdateToolsSortDto
	if err := c.ShouldBindJSON(&updates); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	err := service.UpdateToolsSort(updates)
	if err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新排序成功",
	})
}

// ==================== 搜索引擎相关处理函数 ====================

// 获取所有搜索引擎
func GetAllSearchEnginesHandler(c *gin.Context) {
	engines, err := database.GetAllSearchEngines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"data":    engines,
	})
}

// 获取启用的搜索引擎（用于前端搜索功能）
func GetEnabledSearchEnginesHandler(c *gin.Context) {
	engines, err := database.GetEnabledSearchEngines()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	c.JSON(200, gin.H{
		"success": true,
		"data":    engines,
	})
}

// 添加搜索引擎
func AddSearchEngineHandler(c *gin.Context) {
	var engine types.SearchEngine
	err := c.ShouldBindJSON(&engine)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	id, err := database.AddSearchEngine(engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "添加搜索引擎成功",
		"data": gin.H{
			"id": id,
		},
	})
}

// searchEngineLogoById 查询搜索引擎当前的 logo，用于更新/删除后清理本地图片
func searchEngineLogoById(id int) string {
	engines, err := database.GetAllSearchEngines()
	if err != nil {
		utils.CheckErr(err)
		return ""
	}
	for _, engine := range engines {
		if engine.Id == id {
			return engine.Logo
		}
	}
	return ""
}

// SaveSearchEngineLogoHandler 下载搜索引擎 logo 并保存到本地（文件名使用搜索引擎名称）
func SaveSearchEngineLogoHandler(c *gin.Context) {
	var data types.SaveSearchEngineLogoDto
	if err := c.ShouldBindJSON(&data); err != nil {
		utils.CheckErr(err)
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	localUrl, err := service.SaveSearchEngineLogo(data.Name, data.Logo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "logo 已保存到本地",
		"data": gin.H{
			"url": localUrl,
		},
	})
}

// 更新搜索引擎
func UpdateSearchEngineHandler(c *gin.Context) {
	var engine types.SearchEngine
	err := c.ShouldBindJSON(&engine)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	// 从URL参数获取ID
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}
	engine.Id = id

	oldLogo := searchEngineLogoById(id)
	err = database.UpdateSearchEngine(engine)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	// logo 换掉（含改名后文件名变化）时删除旧的本地图片
	if oldLogo != "" && oldLogo != engine.Logo {
		service.RemoveLocalImage(oldLogo)
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新搜索引擎成功",
	})
}

// 删除搜索引擎
func DeleteSearchEngineHandler(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": "无效的ID",
		})
		return
	}

	oldLogo := searchEngineLogoById(id)
	err = database.DeleteSearchEngine(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}
	// 同时删除该搜索引擎保存在本地的 logo 图片
	service.RemoveLocalImage(oldLogo)

	c.JSON(200, gin.H{
		"success": true,
		"message": "删除搜索引擎成功",
	})
}

// 更新搜索引擎排序
func UpdateSearchEngineSortHandler(c *gin.Context) {
	var sortData []struct {
		Id   int `json:"id"`
		Sort int `json:"sort"`
	}
	err := c.ShouldBindJSON(&sortData)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	err = database.UpdateSearchEngineSort(sortData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success":      false,
			"errorMessage": err.Error(),
		})
		return
	}

	c.JSON(200, gin.H{
		"success": true,
		"message": "更新排序成功",
	})
}

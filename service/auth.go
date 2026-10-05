package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/mereith/nav/database"
	"github.com/mereith/nav/logger"
	"github.com/mereith/nav/types"
	"github.com/mereith/nav/utils"
)

func GetApiTokens() []types.Token {
	sql_get_api_tokens := `
		SELECT id,name,value,disabled FROM nav_api_token WHERE disabled = 0;
		`
	results := make([]types.Token, 0)
	rows, err := database.DB.Query(sql_get_api_tokens)
	if err != nil {
		// Query 失败时 rows 为 nil，老代码直接 rows.Next() 会 panic
		utils.CheckErr(err)
		return results
	}
	defer rows.Close()
	for rows.Next() {
		var token types.Token
		err = rows.Scan(&token.Id, &token.Name, &token.Value, &token.Disabled)
		utils.CheckErr(err)
		results = append(results, token)
	}
	return results
}

func GetUser(name string) types.User {
	sql_get_user := `
		SELECT id,name,password FROM nav_user WHERE name = ?;
		`
	var user types.User
	row := database.DB.QueryRow(sql_get_user, name)
	err := row.Scan(&user.Id, &user.Name, &user.Password)
	// 用户不存在是正常的登录失败路径，不打错误堆栈，只在真正查询异常时记录
	if err != nil && err != sql.ErrNoRows {
		utils.CheckErr(err)
	}
	return user
}

// AddApiTokenInDB 写入 API Token，失败时返回 error（调用方回 500，不再静默吞掉）
func AddApiTokenInDB(data types.Token) error {
	sql_add_api_token := `
		INSERT INTO nav_api_token (id,name,value,disabled)
		VALUES (?,?,?,?);
		`
	// 一次性插入用 DB.Exec：老代码 Prepare 失败后 stmt 为 nil，再 Exec 会 panic
	_, err := database.DB.Exec(sql_add_api_token, data.Id, data.Name, data.Value, data.Disabled)
	return utils.CheckErr(err)
}

func UpdateUser(data types.UpdateUserDto) error {
	if strings.TrimSpace(data.Name) == "" {
		return fmt.Errorf("用户名不能为空")
	}
	// 后台表单要求必填密码：空密码拒绝落库，避免误操作把自己锁在外面
	if data.Password == "" {
		return fmt.Errorf("密码不能为空")
	}
	if len([]rune(data.Password)) < 4 {
		return fmt.Errorf("密码长度至少 4 位")
	}
	hash, err := utils.HashPassword(data.Password)
	if err != nil {
		logger.LogError("密码哈希失败: %s", err)
		return fmt.Errorf("密码处理失败")
	}
	sql_update_user := `
		UPDATE nav_user
		SET name = ?, password = ?
		WHERE id = ?;
		`
	stmt, err := database.DB.Prepare(sql_update_user)
	utils.CheckErr(err)
	res, err := stmt.Exec(data.Name, hash, data.Id)
	utils.CheckErr(err)
	_, err = res.RowsAffected()
	utils.CheckErr(err)
	return err
}

// UpgradePasswordHash 登录成功后调用：库里还是明文的老密码时，原地升级为 bcrypt 哈希
// 失败只记录日志（不影响本次登录），下次登录会再试
func UpgradePasswordHash(userId int, plainPassword string) {
	hash, err := utils.HashPassword(plainPassword)
	if err != nil {
		logger.LogError("登录后升级密码哈希失败: %s", err)
		return
	}
	if _, err := database.DB.Exec(`UPDATE nav_user SET password = ? WHERE id = ?;`, hash, userId); err != nil {
		logger.LogError("登录后升级密码哈希失败: %s", err)
	}
}

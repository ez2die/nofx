package main

import (
	"database/sql"
	"fmt"
	"log"
	"nofx/auth"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// 配置
	dbPath := "config.db"
	email := "pigfun@gmail.com"
	newPassword := "114477ez2die."

	// 检查数据库文件是否存在
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		log.Fatalf("❌ 数据库文件不存在: %s", dbPath)
	}

	// 打开数据库
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("❌ 打开数据库失败: %v", err)
	}
	defer db.Close()

	// 查找用户
	var userID string
	var userEmail string
	err = db.QueryRow("SELECT id, email FROM users WHERE email = ?", email).Scan(&userID, &userEmail)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Fatalf("❌ 用户不存在: %s", email)
		}
		log.Fatalf("❌ 查询用户失败: %v", err)
	}

	fmt.Printf("✅ 找到用户:\n")
	fmt.Printf("   - 用户ID: %s\n", userID)
	fmt.Printf("   - 邮箱: %s\n", userEmail)
	fmt.Println()

	// 生成密码哈希
	fmt.Println("🔐 正在生成密码哈希...")
	passwordHash, err := auth.HashPassword(newPassword)
	if err != nil {
		log.Fatalf("❌ 生成密码哈希失败: %v", err)
	}

	// 更新密码
	fmt.Println("🔄 正在更新数据库...")
	result, err := db.Exec(
		"UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?",
		passwordHash, userID,
	)
	if err != nil {
		log.Fatalf("❌ 更新密码失败: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Fatalf("❌ 获取影响行数失败: %v", err)
	}

	if rowsAffected == 0 {
		log.Fatalf("❌ 没有更新任何记录")
	}

	fmt.Println()
	fmt.Println("✅ 密码更新成功！")
	fmt.Printf("   - 用户ID: %s\n", userID)
	fmt.Printf("   - 邮箱: %s\n", userEmail)
	fmt.Printf("   - 新密码: %s\n", newPassword)
	fmt.Println()
	fmt.Println("🎉 完成！用户现在可以使用新密码登录了。")
}

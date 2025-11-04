#!/usr/bin/env python3
"""
快速密码重置脚本 - 一次性操作
将用户 pigfun@gmail.com 的密码重置为 114477ez2die.
"""

import sqlite3
import bcrypt
import sys

# 配置
DB_PATH = "config.db"
EMAIL = "pigfun@gmail.com"
NEW_PASSWORD = "114477ez2die."

def main():
    print("🔍 查找用户: {}".format(EMAIL))
    
    # 连接数据库
    try:
        conn = sqlite3.connect(DB_PATH)
        cursor = conn.cursor()
    except Exception as e:
        print("❌ 连接数据库失败: {}".format(e))
        sys.exit(1)
    
    # 查找用户
    cursor.execute("SELECT id, email FROM users WHERE email = ?", (EMAIL,))
    user = cursor.fetchone()
    
    if not user:
        print("❌ 用户不存在: {}".format(EMAIL))
        conn.close()
        sys.exit(1)
    
    user_id, user_email = user
    print("✅ 找到用户:")
    print("   - 用户ID: {}".format(user_id))
    print("   - 邮箱: {}".format(user_email))
    print()
    
    # 生成密码哈希
    print("🔐 正在生成密码哈希...")
    try:
        password_hash = bcrypt.hashpw(NEW_PASSWORD.encode('utf-8'), bcrypt.gensalt()).decode('utf-8')
    except Exception as e:
        print("❌ 生成密码哈希失败: {}".format(e))
        conn.close()
        sys.exit(1)
    
    # 更新密码
    print("🔄 正在更新数据库...")
    try:
        cursor.execute(
            "UPDATE users SET password_hash = ?, updated_at = datetime('now') WHERE id = ?",
            (password_hash, user_id)
        )
        conn.commit()
        
        if cursor.rowcount == 0:
            print("❌ 没有更新任何记录")
            conn.close()
            sys.exit(1)
    except Exception as e:
        print("❌ 更新密码失败: {}".format(e))
        conn.rollback()
        conn.close()
        sys.exit(1)
    
    conn.close()
    
    print()
    print("✅ 密码更新成功！")
    print("   - 用户ID: {}".format(user_id))
    print("   - 邮箱: {}".format(user_email))
    print("   - 新密码: {}".format(NEW_PASSWORD))
    print()
    print("🎉 完成！用户现在可以使用新密码登录了。")

if __name__ == "__main__":
    main()


#!/bin/bash
# 使用在线生成的 bcrypt 哈希重置密码
# 密码: 114477ez2die
# 使用 bcrypt-generator.com 生成哈希后替换下面的 HASH 变量

DB_PATH="config.db"
EMAIL="pigfun@gmail.com"
NEW_PASSWORD="114477ez2die"

echo "🔍 查找用户: $EMAIL"
USER_INFO=$(sqlite3 "$DB_PATH" "SELECT id, email FROM users WHERE email = '$EMAIL';" 2>/dev/null)

if [ -z "$USER_INFO" ]; then
    echo "❌ 用户不存在: $EMAIL"
    exit 1
fi

USER_ID=$(echo "$USER_INFO" | cut -d'|' -f1)
echo "✅ 找到用户: $USER_INFO"
echo ""
echo "📝 请访问: https://bcrypt-generator.com/"
echo "   输入密码: $NEW_PASSWORD"
echo "   Cost: 10"
echo "   复制生成的哈希值"
echo ""
read -p "请输入生成的 bcrypt 哈希值: " HASH

if [ -z "$HASH" ]; then
    echo "❌ 哈希值不能为空"
    exit 1
fi

echo ""
echo "🔄 更新密码..."
sqlite3 "$DB_PATH" "UPDATE users SET password_hash = '$HASH', updated_at = datetime('now') WHERE id = '$USER_ID';"

if [ $? -eq 0 ]; then
    echo "✅ 密码更新成功！"
    echo "   - 用户ID: $USER_ID"
    echo "   - 邮箱: $EMAIL"
    echo "   - 新密码: $NEW_PASSWORD"
    echo ""
    echo "🎉 完成！"
else
    echo "❌ 更新失败"
    exit 1
fi

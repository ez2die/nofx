#!/bin/bash
# 快速密码重置脚本
# 将用户 pigfun@gmail.com 的密码重置为 114477ez2die.

DB_PATH="config.db"
EMAIL="pigfun@gmail.com"
NEW_PASSWORD="114477ez2die."

echo "🔍 查找用户: $EMAIL"
USER_INFO=$(sqlite3 "$DB_PATH" "SELECT id, email FROM users WHERE email = '$EMAIL';" 2>/dev/null)

if [ -z "$USER_INFO" ]; then
    echo "❌ 用户不存在: $EMAIL"
    exit 1
fi

USER_ID=$(echo "$USER_INFO" | cut -d'|' -f1)
echo "✅ 找到用户:"
echo "   - 用户ID: $USER_ID"
echo "   - 邮箱: $EMAIL"
echo ""
echo "📝 请访问以下任一网站生成 bcrypt 哈希:"
echo "   1. https://bcrypt-generator.com/ (推荐)"
echo "   2. https://www.bcrypt.fr/"
echo "   3. https://bcrypt.online/"
echo ""
echo "   输入密码: $NEW_PASSWORD"
echo "   Cost/Rounds: 10 (默认)"
echo ""
read -p "请输入生成的 bcrypt 哈希值: " HASH

if [ -z "$HASH" ]; then
    echo "❌ 哈希值不能为空"
    exit 1
fi

# 验证哈希格式（bcrypt 哈希通常以 $2a$, $2b$, $2y$ 开头，长度约60字符）
if [[ ! "$HASH" =~ ^\$2[aby]\$ ]]; then
    echo "⚠️  警告: 哈希值格式可能不正确（bcrypt 哈希应以 \$2a\$、\$2b\$ 或 \$2y\$ 开头）"
    read -p "是否继续? (y/N): " CONFIRM
    if [ "$CONFIRM" != "y" ] && [ "$CONFIRM" != "Y" ]; then
        echo "❌ 已取消"
        exit 1
    fi
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
    echo "🎉 完成！用户现在可以使用新密码登录了。"
else
    echo "❌ 更新失败"
    exit 1
fi


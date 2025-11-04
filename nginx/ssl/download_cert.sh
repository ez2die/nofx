#!/bin/bash
# SSL证书下载脚本
# 使用方法: ./download_cert.sh <证书URL>

set -e

CERT_URL="${1}"
CERT_DIR="$(dirname "$0")"

if [ -z "$CERT_URL" ]; then
    echo "错误: 请提供证书下载URL"
    echo "使用方法: $0 <证书URL>"
    exit 1
fi

echo "正在从以下地址下载SSL证书:"
echo "$CERT_URL"
echo ""

# 下载证书文件
CERT_FILE="$CERT_DIR/certificate.pem"
curl -o "$CERT_FILE" "$CERT_URL"

# 检查下载的文件是否是有效的证书
if file "$CERT_FILE" | grep -q "PEM\|certificate\|ASCII text"; then
    # 检查是否是XML错误消息
    if head -1 "$CERT_FILE" | grep -q "<?xml"; then
        echo "错误: 下载的文件是XML错误消息，可能是URL已过期"
        echo "文件内容:"
        head -10 "$CERT_FILE"
        rm -f "$CERT_FILE"
        exit 1
    fi
    
    echo "证书下载成功: $CERT_FILE"
    echo ""
    echo "证书信息:"
    openssl x509 -in "$CERT_FILE" -noout -subject -issuer -dates 2>/dev/null || {
        echo "注意: 这可能是证书链或私钥文件，请检查文件内容"
        echo "文件前20行:"
        head -20 "$CERT_FILE"
    }
else
    echo "警告: 下载的文件可能不是标准PEM格式证书"
    echo "文件类型: $(file "$CERT_FILE")"
    echo "文件前20行:"
    head -20 "$CERT_FILE"
fi

echo ""
echo "证书已保存到: $CERT_FILE"
echo "请根据实际证书类型，将其重命名为:"
echo "  - certificate.crt (证书文件)"
echo "  - certificate.key (私钥文件)"
echo "  - certificate.pem (证书或证书链)"


#!/bin/bash
# 归档 Trader 02 和 04 的历史目录，保留当前使用的目录

set -e

BASE_DIR="/root/nofx"
ARCHIVE_DIR="${BASE_DIR}/decision_logs_archive"
DECISION_LOGS_DIR="${BASE_DIR}/decision_logs"

# 当前使用的目录（不归档）
CURRENT_TRADER02_DIR="decision_logs/hyperliquid_26f0a132-0026-4516-b5ef-59a2d0406dec_deepseek_1762352985"
CURRENT_TRADER04_DIR="decision_logs/hyperliquid_a5e99c3c-bef8-4b5b-a19a-01328b593128_deepseek_1762351525"

echo "=================================================================================="
echo "归档 Trader 02 和 04 的历史目录"
echo "=================================================================================="
echo ""
echo "保留的当前使用目录："
echo "  ✅ Trader 02: ${CURRENT_TRADER02_DIR}"
echo "  ✅ Trader 04: ${CURRENT_TRADER04_DIR}"
echo ""
echo "归档目标目录: ${ARCHIVE_DIR}"
echo ""

# 创建归档目录
mkdir -p "${ARCHIVE_DIR}"

# 归档 Trader 02 的历史目录
echo "归档 Trader 02 的历史目录..."
TRADER02_DIRS=$(find "${DECISION_LOGS_DIR}" -type d -name "*26f0a132*" | sort)
ARCHIVED_COUNT=0

for dir in ${TRADER02_DIRS}; do
    # 检查是否是当前使用的目录
    if [[ "${dir}" == "${BASE_DIR}/${CURRENT_TRADER02_DIR}" ]]; then
        echo "  ⏭️  跳过当前使用目录: $(basename ${dir})"
        continue
    fi
    
    # 归档目录
    dir_name=$(basename ${dir})
    archive_path="${ARCHIVE_DIR}/${dir_name}"
    
    if [ -d "${archive_path}" ]; then
        echo "  ⚠️  归档目录已存在，跳过: ${dir_name}"
        continue
    fi
    
    echo "  📦 归档: ${dir_name}"
    mv "${dir}" "${archive_path}"
    ARCHIVED_COUNT=$((ARCHIVED_COUNT + 1))
done

echo "  ✅ Trader 02 归档完成，共归档 ${ARCHIVED_COUNT} 个目录"
echo ""

# 归档 Trader 04 的历史目录
echo "归档 Trader 04 的历史目录..."
TRADER04_DIRS=$(find "${DECISION_LOGS_DIR}" -type d -name "*a5e99c3c*" | sort)
ARCHIVED_COUNT=0

for dir in ${TRADER04_DIRS}; do
    # 检查是否是当前使用的目录
    if [[ "${dir}" == "${BASE_DIR}/${CURRENT_TRADER04_DIR}" ]]; then
        echo "  ⏭️  跳过当前使用目录: $(basename ${dir})"
        continue
    fi
    
    # 归档目录
    dir_name=$(basename ${dir})
    archive_path="${ARCHIVE_DIR}/${dir_name}"
    
    if [ -d "${archive_path}" ]; then
        echo "  ⚠️  归档目录已存在，跳过: ${dir_name}"
        continue
    fi
    
    echo "  📦 归档: ${dir_name}"
    mv "${dir}" "${archive_path}"
    ARCHIVED_COUNT=$((ARCHIVED_COUNT + 1))
done

echo "  ✅ Trader 04 归档完成，共归档 ${ARCHIVED_COUNT} 个目录"
echo ""

# 显示归档结果
echo "=================================================================================="
echo "归档完成"
echo "=================================================================================="
echo ""
echo "归档目录: ${ARCHIVE_DIR}"
echo "归档的目录数量: $(find ${ARCHIVE_DIR} -type d -maxdepth 1 | wc -l)"
echo ""
echo "保留的当前使用目录："
echo "  ✅ Trader 02: ${CURRENT_TRADER02_DIR}"
echo "  ✅ Trader 04: ${CURRENT_TRADER04_DIR}"
echo ""


// Trader颜色配置 - 统一的颜色分配逻辑
// 用于 ComparisonChart 和 Leaderboard，确保颜色一致性
// 颜色选择原则：确保相邻颜色有足够的对比度，易于区分

export const TRADER_COLORS = [
  '#3b82f6', // blue-500 (更鲜明的蓝色)
  '#f59e0b', // amber-500 (橙色，与蓝色对比度高)
  '#10b981', // emerald-500 (绿色，与蓝色和橙色对比度高)
  '#ef4444', // red-500 (红色，与其他颜色对比度高)
  '#8b5cf6', // violet-500 (紫色，但放在后面避免与蓝色混淆)
  '#ec4899', // pink-500 (粉色)
  '#06b6d4', // cyan-500 (青色)
  '#f97316', // orange-500 (橙红色)
  '#84cc16', // lime-500 (黄绿色)
  '#6366f1', // indigo-500 (靛蓝色)
];

/**
 * 根据trader的索引位置获取颜色
 * @param traders - trader列表
 * @param traderId - 当前trader的ID
 * @returns 对应的颜色值
 */
export function getTraderColor(
  traders: Array<{ trader_id: string }>,
  traderId: string
): string {
  const traderIndex = traders.findIndex((t) => t.trader_id === traderId);
  if (traderIndex === -1) return TRADER_COLORS[0]; // 默认返回第一个颜色
  // 如果超出颜色池大小，循环使用
  return TRADER_COLORS[traderIndex % TRADER_COLORS.length];
}

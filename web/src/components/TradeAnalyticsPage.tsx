import React, { useState, useMemo } from 'react';
import { useTradeAnalytics } from '../hooks/useTradeAnalytics';
import { useTradePerformance } from '../hooks/useTradePerformance';
import type {
  TradeAnalyticsFilter,
  SymbolStatistics,
  TradePerformanceSymbolStats,
  TradePerformanceRecentTrade,
} from '../types';
import {
  BarChart3,
  TrendingUp,
  DollarSign,
  Target,
  AlertTriangle,
  Activity,
  PieChart,
  Clock,
  ArrowUpRight,
  ArrowDownRight,
} from 'lucide-react';

interface TradeAnalyticsPageProps {
  traderId: string;
}

export function TradeAnalyticsPage({ traderId }: TradeAnalyticsPageProps) {
  const [filters, setFilters] = useState<Partial<TradeAnalyticsFilter>>({});
  const [includePairs, setIncludePairs] = useState(false);

  const { data, isLoading, error, refresh } = useTradeAnalytics(
    traderId,
    filters,
    { includePairs, groupBy: 'day' },
  );

  const {
    performance,
    isLoading: performanceLoading,
    error: performanceError,
    refresh: refreshPerformance,
  } = useTradePerformance(traderId, {
    start_time: filters.start_time,
    end_time: filters.end_time,
  });

  const numberFormatter = useMemo(
    () => new Intl.NumberFormat(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 }),
    [],
  );

  const symbolPerformanceList = useMemo(() => {
    if (!performance?.symbol_stats) return [];
    return Object.values(performance.symbol_stats).sort(
      (a: TradePerformanceSymbolStats, b: TradePerformanceSymbolStats) => b.total_trades - a.total_trades,
    );
  }, [performance]);

  const recentPerformanceTrades: TradePerformanceRecentTrade[] = performance?.recent_trades ?? [];

  const bestSymbolStats =
    performance?.best_symbol && performance.symbol_stats
      ? performance.symbol_stats[performance.best_symbol]
      : undefined;
  const worstSymbolStats =
    performance?.worst_symbol && performance.symbol_stats
      ? performance.symbol_stats[performance.worst_symbol]
      : undefined;

  const formatDuration = (seconds: number): string => {
    if (seconds < 60) return `${seconds}秒`;
    if (seconds < 3600) return `${Math.floor(seconds / 60)}分钟`;
    if (seconds < 86400) return `${Math.floor(seconds / 3600)}小时`;
    return `${Math.floor(seconds / 86400)}天`;
  };

  const formatPnL = (value: number) => {
    const formatted = numberFormatter.format(Math.abs(value));
    return `${value >= 0 ? '+' : '-'}${formatted}`;
  };

  const handleRefresh = () => {
    refresh();
    refreshPerformance();
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-[#848E9C]">加载中...</div>
      </div>
    );
  }

  if (error) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-red-500">加载失败: {error.message}</div>
      </div>
    );
  }

  if (!data) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="text-[#848E9C]">暂无数据</div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      {/* 过滤器 */}
      <div className="binance-card p-4">
        <div className="flex flex-wrap items-center gap-4">
          <div className="flex items-center gap-2">
            <label className="text-sm text-[#848E9C]">币种:</label>
            <input
              type="text"
              placeholder="BTCUSDT"
              value={filters.symbol || ''}
              onChange={(e) => setFilters({ ...filters, symbol: e.target.value || undefined })}
              className="px-3 py-1 bg-[#1E2329] border border-[#2B3139] rounded text-sm text-[#EAECEF] focus:outline-none focus:border-[#F0B90B]"
            />
          </div>
          <div className="flex items-center gap-2">
            <label className="text-sm text-[#848E9C]">方向:</label>
            <select
              value={filters.side || ''}
              onChange={(e) => setFilters({ ...filters, side: e.target.value || undefined })}
              className="px-3 py-1 bg-[#1E2329] border border-[#2B3139] rounded text-sm text-[#EAECEF] focus:outline-none focus:border-[#F0B90B]"
            >
              <option value="">全部</option>
              <option value="long">做多</option>
              <option value="short">做空</option>
            </select>
          </div>
          <div className="flex items-center gap-2">
            <label className="text-sm text-[#848E9C]">开始时间:</label>
            <input
              type="datetime-local"
              value={filters.start_time ? filters.start_time.slice(0, 16) : ''}
              onChange={(e) =>
                setFilters({
                  ...filters,
                  start_time: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                })
              }
              className="px-3 py-1 bg-[#1E2329] border border-[#2B3139] rounded text-sm text-[#EAECEF] focus:outline-none focus:border-[#F0B90B]"
            />
          </div>
          <div className="flex items-center gap-2">
            <label className="text-sm text-[#848E9C]">结束时间:</label>
            <input
              type="datetime-local"
              value={filters.end_time ? filters.end_time.slice(0, 16) : ''}
              onChange={(e) =>
                setFilters({
                  ...filters,
                  end_time: e.target.value ? new Date(e.target.value).toISOString() : undefined,
                })
              }
              className="px-3 py-1 bg-[#1E2329] border border-[#2B3139] rounded text-sm text-[#EAECEF] focus:outline-none focus:border-[#F0B90B]"
            />
          </div>
          <label className="flex items-center gap-2 cursor-pointer">
            <input
              type="checkbox"
              checked={includePairs}
              onChange={(e) => setIncludePairs(e.target.checked)}
              className="w-4 h-4"
            />
            <span className="text-sm text-[#848E9C]">包含配对分析</span>
          </label>
          <button
            onClick={() => {
              setFilters({});
              setIncludePairs(false);
            }}
            className="px-4 py-1 bg-[#2B3139] hover:bg-[#3A4149] text-[#EAECEF] rounded text-sm transition-colors"
          >
            重置
          </button>
          <button
            onClick={handleRefresh}
            className="px-4 py-1 bg-[#F0B90B] hover:bg-[#D4A309] text-[#0B0E11] rounded text-sm font-semibold transition-colors"
          >
            刷新
          </button>
        </div>
      </div>

      {/* 交易表现（Trade History） */}
      <div className="binance-card p-6 space-y-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold text-[#EAECEF]">交易表现（Trade History）</h3>
            <p className="text-sm text-[#848E9C]">基于持久化 Trade History 的统计结果</p>
          </div>
          {performance?.generated_at && (
            <div className="text-xs text-[#848E9C]">
              更新于 {new Date(performance.generated_at).toLocaleString()}
            </div>
          )}
        </div>
        {performanceError && (
          <div className="text-sm text-red-500">交易表现加载失败: {performanceError.message}</div>
        )}
        {performanceLoading && !performance && (
          <div className="text-sm text-[#848E9C]">交易表现加载中...</div>
        )}
        {performance && (
          <div className="space-y-6">
            <div className="grid grid-cols-1 md:grid-cols-3 lg:grid-cols-5 gap-4">
              <StatCard
                title="总交易数"
                value={performance.total_trades}
                icon={<Activity className="w-5 h-5" />}
                color="#60a5fa"
              />
              <StatCard
                title="胜率"
                value={`${numberFormatter.format(performance.win_rate)}%`}
                icon={<Target className="w-5 h-5" />}
                color="#34d399"
              />
              <StatCard
                title="盈亏比"
                value={numberFormatter.format(performance.profit_factor)}
                icon={<BarChart3 className="w-5 h-5" />}
                color="#fbbf24"
              />
              <StatCard
                title="夏普比率"
                value={numberFormatter.format(performance.sharpe_ratio)}
                icon={<TrendingUp className="w-5 h-5" />}
                color="#a78bfa"
              />
              <StatCard
                title="总盈亏"
                value={`${performance.total_pnl >= 0 ? '+' : ''}${numberFormatter.format(performance.total_pnl)}`}
                icon={<DollarSign className="w-5 h-5" />}
                color={performance.total_pnl >= 0 ? '#0ECB81' : '#F6465D'}
              />
            </div>

            {(bestSymbolStats || worstSymbolStats) && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                {bestSymbolStats && (
                  <div className="p-4 bg-[#1E2329] rounded border border-[#2B3139]">
                    <div className="text-sm text-[#848E9C] mb-1">最佳币种</div>
                    <div className="text-2xl font-semibold text-[#0ECB81]">{performance.best_symbol}</div>
                    <div className="mt-2 text-sm text-[#EAECEF]">
                      胜率 {numberFormatter.format(bestSymbolStats.win_rate)}% · 盈亏 {formatPnL(bestSymbolStats.total_pn_l)}
                    </div>
                  </div>
                )}
                {worstSymbolStats && (
                  <div className="p-4 bg-[#1E2329] rounded border border-[#2B3139]">
                    <div className="text-sm text-[#848E9C] mb-1">最差币种</div>
                    <div className="text-2xl font-semibold text-[#F6465D]">{performance.worst_symbol}</div>
                    <div className="mt-2 text-sm text-[#EAECEF]">
                      胜率 {numberFormatter.format(worstSymbolStats.win_rate)}% · 盈亏 {formatPnL(worstSymbolStats.total_pn_l)}
                    </div>
                  </div>
                )}
              </div>
            )}

            {symbolPerformanceList.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h4 className="text-base font-semibold text-[#EAECEF]">币种表现</h4>
                  <span className="text-xs text-[#848E9C]">仅统计平仓交易</span>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-left text-[#848E9C] border-b border-[#2B3139]">
                        <th className="py-2 pr-4 font-normal">币种</th>
                        <th className="py-2 pr-4 font-normal">交易数</th>
                        <th className="py-2 pr-4 font-normal">胜率</th>
                        <th className="py-2 pr-4 font-normal">总盈亏</th>
                        <th className="py-2 pr-4 font-normal">平均盈亏</th>
                      </tr>
                    </thead>
                    <tbody>
                      {symbolPerformanceList.map((stat) => (
                        <tr key={stat.symbol} className="border-b border-[#2B3139] last:border-none">
                          <td className="py-2 pr-4 text-[#EAECEF] font-semibold">{stat.symbol}</td>
                          <td className="py-2 pr-4 text-[#EAECEF]">{stat.total_trades}</td>
                          <td className="py-2 pr-4 text-[#EAECEF]">
                            {numberFormatter.format(stat.win_rate)}%
                          </td>
                          <td
                            className={`py-2 pr-4 font-semibold ${
                              stat.total_pn_l >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                            }`}
                          >
                            {formatPnL(stat.total_pn_l)}
                          </td>
                          <td
                            className={`py-2 pr-4 font-semibold ${
                              stat.avg_pn_l >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                            }`}
                          >
                            {formatPnL(stat.avg_pn_l)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}

            {recentPerformanceTrades.length > 0 && (
              <div className="space-y-3">
                <div className="flex items-center justify-between">
                  <h4 className="text-base font-semibold text-[#EAECEF]">最近平仓交易</h4>
                  <span className="text-xs text-[#848E9C]">展示 {recentPerformanceTrades.length} 条</span>
                </div>
                <div className="overflow-x-auto">
                  <table className="w-full text-sm">
                    <thead>
                      <tr className="text-left text-[#848E9C] border-b border-[#2B3139]">
                        <th className="py-2 pr-4 font-normal">时间</th>
                        <th className="py-2 pr-4 font-normal">币种/方向</th>
                        <th className="py-2 pr-4 font-normal">数量</th>
                        <th className="py-2 pr-4 font-normal">价格</th>
                        <th className="py-2 pr-4 font-normal">盈亏</th>
                        <th className="py-2 pr-4 font-normal">手续费</th>
                      </tr>
                    </thead>
                    <tbody>
                      {recentPerformanceTrades.map((trade, idx) => (
                        <tr key={`${trade.symbol}-${trade.timestamp}-${idx}`} className="border-b border-[#2B3139] last:border-none">
                          <td className="py-2 pr-4 text-[#EAECEF]">
                            {new Date(trade.timestamp).toLocaleString()}
                          </td>
                          <td className="py-2 pr-4 text-[#EAECEF]">
                            {trade.symbol}{' '}
                            <span className="text-xs text-[#848E9C]">
                              {trade.side === 'long' ? '做多' : '做空'}
                            </span>
                          </td>
                          <td className="py-2 pr-4 text-[#EAECEF]">
                            {numberFormatter.format(trade.quantity)}
                          </td>
                          <td className="py-2 pr-4 text-[#EAECEF]">
                            {numberFormatter.format(trade.execution_price)}
                          </td>
                          <td
                            className={`py-2 pr-4 font-semibold ${
                              trade.pn_l >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                            }`}
                          >
                            {formatPnL(trade.pn_l)}
                          </td>
                          <td className="py-2 pr-4 text-[#EAECEF]">
                            {numberFormatter.format(trade.fee)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        )}
      </div>

      {/* 概览统计卡片 */}
      {data.overview && (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-5 gap-4">
          <StatCard
            title="总交易数"
            value={data.overview.total_trades}
            icon={<Activity className="w-5 h-5" />}
            color="#60a5fa"
          />
          <StatCard
            title="开仓数"
            value={data.overview.open_trades}
            icon={<ArrowUpRight className="w-5 h-5" />}
            color="#34d399"
          />
          <StatCard
            title="平仓数"
            value={data.overview.close_trades}
            icon={<ArrowDownRight className="w-5 h-5" />}
            color="#f87171"
          />
          <StatCard
            title="已完成"
            value={data.overview.completed_trades}
            icon={<Target className="w-5 h-5" />}
            color="#fbbf24"
          />
          <StatCard
            title="未平仓"
            value={data.overview.unclosed_trades}
            icon={<AlertTriangle className="w-5 h-5" />}
            color="#a78bfa"
          />
        </div>
      )}

      {/* 盈亏统计 */}
      {data.pnl_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <DollarSign className="w-5 h-5" style={{ color: '#F0B90B' }} />
            盈亏统计
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">总盈亏</div>
              <div
                className={`text-xl font-semibold ${
                  data.pnl_stats.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                }`}
              >
                {data.pnl_stats.total_pnl >= 0 ? '+' : ''}
                {numberFormatter.format(data.pnl_stats.total_pnl)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">总盈利</div>
              <div className="text-xl font-semibold text-[#0ECB81]">
                +{numberFormatter.format(data.pnl_stats.total_profit)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">总亏损</div>
              <div className="text-xl font-semibold text-[#F6465D]">
                -{numberFormatter.format(data.pnl_stats.total_loss)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">净盈亏</div>
              <div
                className={`text-xl font-semibold ${
                  data.pnl_stats.net_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                }`}
              >
                {data.pnl_stats.net_pnl >= 0 ? '+' : ''}
                {numberFormatter.format(data.pnl_stats.net_pnl)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平均盈亏</div>
              <div
                className={`text-lg font-semibold ${
                  data.pnl_stats.avg_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                }`}
              >
                {data.pnl_stats.avg_pnl >= 0 ? '+' : ''}
                {numberFormatter.format(data.pnl_stats.avg_pnl)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最大盈利</div>
              <div className="text-lg font-semibold text-[#0ECB81]">
                +{numberFormatter.format(data.pnl_stats.max_win)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最大亏损</div>
              <div className="text-lg font-semibold text-[#F6465D]">
                -{numberFormatter.format(data.pnl_stats.max_loss)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">盈亏比</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {numberFormatter.format(data.pnl_stats.profit_factor)}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 胜率统计 */}
      {data.win_rate_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <Target className="w-5 h-5" style={{ color: '#F0B90B' }} />
            胜率统计
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">胜率</div>
              <div className="text-2xl font-semibold text-[#0ECB81]">
                {numberFormatter.format(data.win_rate_stats.win_rate)}%
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">盈利交易</div>
              <div className="text-xl font-semibold text-[#0ECB81]">{data.win_rate_stats.winning_trades}</div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">亏损交易</div>
              <div className="text-xl font-semibold text-[#F6465D]">{data.win_rate_stats.losing_trades}</div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">持平交易</div>
              <div className="text-xl font-semibold text-[#848E9C]">{data.win_rate_stats.break_even_trades}</div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平均盈利</div>
              <div className="text-lg font-semibold text-[#0ECB81]">
                +{numberFormatter.format(data.win_rate_stats.avg_win)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平均亏损</div>
              <div className="text-lg font-semibold text-[#F6465D]">
                -{numberFormatter.format(data.win_rate_stats.avg_loss)}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 费用统计 */}
      {data.fee_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <DollarSign className="w-5 h-5" style={{ color: '#F0B90B' }} />
            费用统计
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">总费用</div>
              <div className="text-xl font-semibold text-[#EAECEF]">
                {numberFormatter.format(data.fee_stats.total_fees)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">开仓费用</div>
              <div className="text-lg font-semibold text-[#34d399]">
                {numberFormatter.format(data.fee_stats.open_fees)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平仓费用</div>
              <div className="text-lg font-semibold text-[#f87171]">
                {numberFormatter.format(data.fee_stats.close_fees)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平均费用</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {numberFormatter.format(data.fee_stats.avg_fee)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">费用占比</div>
              <div className="text-lg font-semibold text-[#F0B90B]">
                {numberFormatter.format(data.fee_stats.fee_ratio)}%
              </div>
            </div>
            {data.fee_stats.builder_fees > 0 && (
              <div>
                <div className="text-sm text-[#848E9C] mb-1">Builder费用</div>
                <div className="text-lg font-semibold text-[#a78bfa]">
                  {numberFormatter.format(data.fee_stats.builder_fees)}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 风险指标 */}
      {data.risk_metrics && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <AlertTriangle className="w-5 h-5" style={{ color: '#F0B90B' }} />
            风险指标
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最大回撤</div>
              <div className="text-xl font-semibold text-[#F6465D]">
                -{numberFormatter.format(data.risk_metrics.max_drawdown)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最大回撤%</div>
              <div className="text-xl font-semibold text-[#F6465D]">
                -{numberFormatter.format(data.risk_metrics.max_drawdown_percent)}%
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">波动率</div>
              <div className="text-xl font-semibold text-[#EAECEF]">
                {numberFormatter.format(data.risk_metrics.volatility)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">夏普比率</div>
              <div className="text-xl font-semibold text-[#EAECEF]">
                {numberFormatter.format(data.risk_metrics.sharpe_ratio)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">回撤持续时间</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {formatDuration(data.risk_metrics.drawdown_duration)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">恢复时间</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {formatDuration(data.risk_metrics.recovery_time)}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 方向统计 */}
      {data.direction_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <PieChart className="w-5 h-5" style={{ color: '#F0B90B' }} />
            方向统计
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <div className="text-sm text-[#848E9C] mb-3">做多统计</div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">交易数:</span>
                  <span className="text-[#EAECEF]">{data.direction_stats.long_stats.total_trades}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">总盈亏:</span>
                  <span
                    className={
                      data.direction_stats.long_stats.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                    }
                  >
                    {data.direction_stats.long_stats.total_pnl >= 0 ? '+' : ''}
                    {numberFormatter.format(data.direction_stats.long_stats.total_pnl)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">胜率:</span>
                  <span className="text-[#EAECEF]">
                    {numberFormatter.format(data.direction_stats.long_stats.win_rate)}%
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">平均盈亏:</span>
                  <span
                    className={
                      data.direction_stats.long_stats.avg_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                    }
                  >
                    {data.direction_stats.long_stats.avg_pnl >= 0 ? '+' : ''}
                    {numberFormatter.format(data.direction_stats.long_stats.avg_pnl)}
                  </span>
                </div>
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-3">做空统计</div>
              <div className="space-y-2">
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">交易数:</span>
                  <span className="text-[#EAECEF]">{data.direction_stats.short_stats.total_trades}</span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">总盈亏:</span>
                  <span
                    className={
                      data.direction_stats.short_stats.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                    }
                  >
                    {data.direction_stats.short_stats.total_pnl >= 0 ? '+' : ''}
                    {numberFormatter.format(data.direction_stats.short_stats.total_pnl)}
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">胜率:</span>
                  <span className="text-[#EAECEF]">
                    {numberFormatter.format(data.direction_stats.short_stats.win_rate)}%
                  </span>
                </div>
                <div className="flex justify-between">
                  <span className="text-[#848E9C]">平均盈亏:</span>
                  <span
                    className={
                      data.direction_stats.short_stats.avg_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                    }
                  >
                    {data.direction_stats.short_stats.avg_pnl >= 0 ? '+' : ''}
                    {numberFormatter.format(data.direction_stats.short_stats.avg_pnl)}
                  </span>
                </div>
              </div>
            </div>
          </div>
          {data.direction_stats.preference && (
            <div className="mt-4 pt-4 border-t border-[#2B3139]">
              <div className="text-sm text-[#848E9C] mb-2">方向偏好</div>
              <div className="flex items-center gap-4">
                <div className="text-lg font-semibold text-[#F0B90B]">
                  {data.direction_stats.preference.preferred_side === 'long' ? '做多' : '做空'}
                </div>
                <div className="flex-1">
                  <div className="flex justify-between text-sm mb-1">
                    <span className="text-[#848E9C]">做多占比</span>
                    <span className="text-[#EAECEF]">
                      {numberFormatter.format(data.direction_stats.preference.long_ratio)}%
                    </span>
                  </div>
                  <div className="flex justify-between text-sm">
                    <span className="text-[#848E9C]">做空占比</span>
                    <span className="text-[#EAECEF]">
                      {numberFormatter.format(data.direction_stats.preference.short_ratio)}%
                    </span>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      )}

      {/* 交易频率统计 */}
      {data.frequency_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <Clock className="w-5 h-5" style={{ color: '#F0B90B' }} />
            交易频率统计
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">日均交易数</div>
              <div className="text-xl font-semibold text-[#EAECEF]">
                {numberFormatter.format(data.frequency_stats.daily_average_trades)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平均交易间隔</div>
              <div className="text-xl font-semibold text-[#EAECEF]">
                {formatDuration(data.frequency_stats.avg_trade_interval)}
              </div>
            </div>
            {data.frequency_stats.most_active_hours && data.frequency_stats.most_active_hours.length > 0 && (
              <div>
                <div className="text-sm text-[#848E9C] mb-2">最活跃时段</div>
                <div className="flex flex-wrap gap-2">
                  {data.frequency_stats.most_active_hours.map((hour: number) => (
                    <span
                      key={hour}
                      className="px-3 py-1 bg-[#1E2329] border border-[#2B3139] rounded text-sm text-[#EAECEF]"
                    >
                      {hour}:00
                    </span>
                  ))}
                </div>
              </div>
            )}
            {data.frequency_stats.most_active_days && data.frequency_stats.most_active_days.length > 0 && (
              <div>
                <div className="text-sm text-[#848E9C] mb-2">最活跃日期</div>
                <div className="flex flex-wrap gap-2">
                  {data.frequency_stats.most_active_days.map((day: string) => (
                    <span
                      key={day}
                      className="px-3 py-1 bg-[#1E2329] border border-[#2B3139] rounded text-sm text-[#EAECEF]"
                    >
                      {day}
                    </span>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 币种统计 */}
      {data.symbol_stats && Object.keys(data.symbol_stats).length > 0 && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <BarChart3 className="w-5 h-5" style={{ color: '#F0B90B' }} />
            币种统计
          </h3>
          <div className="overflow-x-auto">
            <table className="min-w-full text-sm">
              <thead>
                <tr className="border-b border-[#2B3139]">
                  <th className="text-left py-2 px-4 text-[#848E9C]">币种</th>
                  <th className="text-right py-2 px-4 text-[#848E9C]">交易数</th>
                  <th className="text-right py-2 px-4 text-[#848E9C]">总盈亏</th>
                  <th className="text-right py-2 px-4 text-[#848E9C]">胜率</th>
                  <th className="text-right py-2 px-4 text-[#848E9C]">平均盈亏</th>
                  <th className="text-right py-2 px-4 text-[#848E9C]">最大盈利</th>
                  <th className="text-right py-2 px-4 text-[#848E9C]">最大亏损</th>
                </tr>
              </thead>
              <tbody>
                {Object.values(data.symbol_stats)
                  .sort((a: SymbolStatistics, b: SymbolStatistics) => b.total_pnl - a.total_pnl)
                  .map((stat: SymbolStatistics) => (
                    <tr key={stat.symbol} className="border-b border-[#2B3139] hover:bg-[#1E2329]">
                      <td className="py-2 px-4 text-[#EAECEF] font-medium">{stat.symbol}</td>
                      <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.completed_trades}</td>
                      <td
                        className={`py-2 px-4 text-right font-semibold ${
                          stat.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                        }`}
                      >
                        {stat.total_pnl >= 0 ? '+' : ''}
                        {numberFormatter.format(stat.total_pnl)}
                      </td>
                      <td className="py-2 px-4 text-right text-[#EAECEF]">
                        {numberFormatter.format(stat.win_rate)}%
                      </td>
                      <td
                        className={`py-2 px-4 text-right ${
                          stat.avg_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                        }`}
                      >
                        {stat.avg_pnl >= 0 ? '+' : ''}
                        {numberFormatter.format(stat.avg_pnl)}
                      </td>
                      <td className="py-2 px-4 text-right text-[#0ECB81]">
                        +{numberFormatter.format(stat.max_win)}
                      </td>
                      <td className="py-2 px-4 text-right text-[#F6465D]">
                        -{numberFormatter.format(stat.max_loss)}
                      </td>
                    </tr>
                  ))}
              </tbody>
            </table>
          </div>
        </div>
      )}

      {/* 连续统计 */}
      {data.streak_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <TrendingUp className="w-5 h-5" style={{ color: '#F0B90B' }} />
            连续统计
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-4 gap-4">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最长连胜</div>
              <div className="text-xl font-semibold text-[#0ECB81]">{data.streak_stats.longest_winning_streak}</div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最长连亏</div>
              <div className="text-xl font-semibold text-[#F6465D]">{data.streak_stats.longest_losing_streak}</div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">当前连续</div>
              <div
                className={`text-xl font-semibold ${
                  data.streak_stats.current_streak_type === 'winning' ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                }`}
              >
                {Math.abs(data.streak_stats.current_streak)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">当前类型</div>
              <div
                className={`text-lg font-semibold ${
                  data.streak_stats.current_streak_type === 'winning' ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                }`}
              >
                {data.streak_stats.current_streak_type === 'winning' ? '连胜' : '连亏'}
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 交易类型统计 */}
      {data.action_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <Activity className="w-5 h-5" style={{ color: '#F0B90B' }} />
            交易类型统计
          </h3>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
            {/* 开仓统计 */}
            <div>
              <h4 className="text-base font-semibold mb-3 text-[#34d399]">开仓统计</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-sm text-[#848E9C] mb-1">交易数</div>
                  <div className="text-lg font-semibold text-[#EAECEF]">
                    {data.action_stats.open_stats.total_trades}
                  </div>
                </div>
                <div>
                  <div className="text-sm text-[#848E9C] mb-1">总费用</div>
                  <div className="text-lg font-semibold text-[#EAECEF]">
                    {numberFormatter.format(data.action_stats.open_stats.total_fees)}
                  </div>
                </div>
              </div>
            </div>
            {/* 平仓统计 */}
            <div>
              <h4 className="text-base font-semibold mb-3 text-[#f87171]">平仓统计</h4>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <div className="text-sm text-[#848E9C] mb-1">交易数</div>
                  <div className="text-lg font-semibold text-[#EAECEF]">
                    {data.action_stats.close_stats.total_trades}
                  </div>
                </div>
                <div>
                  <div className="text-sm text-[#848E9C] mb-1">总费用</div>
                  <div className="text-lg font-semibold text-[#EAECEF]">
                    {numberFormatter.format(data.action_stats.close_stats.total_fees)}
                  </div>
                </div>
                <div>
                  <div className="text-sm text-[#848E9C] mb-1">总盈亏</div>
                  <div
                    className={`text-lg font-semibold ${
                      data.action_stats.close_stats.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                    }`}
                  >
                    {data.action_stats.close_stats.total_pnl >= 0 ? '+' : ''}
                    {numberFormatter.format(data.action_stats.close_stats.total_pnl)}
                  </div>
                </div>
                <div>
                  <div className="text-sm text-[#848E9C] mb-1">平均盈亏</div>
                  <div
                    className={`text-lg font-semibold ${
                      data.action_stats.close_stats.avg_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                    }`}
                  >
                    {data.action_stats.close_stats.avg_pnl >= 0 ? '+' : ''}
                    {numberFormatter.format(data.action_stats.close_stats.avg_pnl)}
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* 时间序列统计 */}
      {data.time_series_stats && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <Clock className="w-5 h-5" style={{ color: '#F0B90B' }} />
            时间序列统计
          </h3>
          <div className="space-y-6">
            {/* 每日统计 */}
            {data.time_series_stats.daily_stats && data.time_series_stats.daily_stats.length > 0 && (
              <div>
                <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">每日统计</h4>
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead>
                      <tr className="border-b border-[#2B3139]">
                        <th className="text-left py-2 px-4 text-[#848E9C]">日期</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">交易数</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">已完成</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">总盈亏</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">总费用</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">胜率</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.time_series_stats.daily_stats.map((stat) => (
                        <tr key={stat.date} className="border-b border-[#2B3139] hover:bg-[#1E2329]">
                          <td className="py-2 px-4 text-[#EAECEF]">{stat.date}</td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.total_trades}</td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.completed_trades}</td>
                          <td
                            className={`py-2 px-4 text-right font-semibold ${
                              stat.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                            }`}
                          >
                            {stat.total_pnl >= 0 ? '+' : ''}
                            {numberFormatter.format(stat.total_pnl)}
                          </td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">
                            {numberFormatter.format(stat.total_fees)}
                          </td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">
                            {numberFormatter.format(stat.win_rate)}%
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            {/* 每周统计 */}
            {data.time_series_stats.weekly_stats && data.time_series_stats.weekly_stats.length > 0 && (
              <div>
                <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">每周统计</h4>
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead>
                      <tr className="border-b border-[#2B3139]">
                        <th className="text-left py-2 px-4 text-[#848E9C]">周</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">交易数</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">已完成</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">总盈亏</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">总费用</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">胜率</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.time_series_stats.weekly_stats.map((stat) => (
                        <tr key={stat.week} className="border-b border-[#2B3139] hover:bg-[#1E2329]">
                          <td className="py-2 px-4 text-[#EAECEF]">{stat.week}</td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.total_trades}</td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.completed_trades}</td>
                          <td
                            className={`py-2 px-4 text-right font-semibold ${
                              stat.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                            }`}
                          >
                            {stat.total_pnl >= 0 ? '+' : ''}
                            {numberFormatter.format(stat.total_pnl)}
                          </td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">
                            {numberFormatter.format(stat.total_fees)}
                          </td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">
                            {numberFormatter.format(stat.win_rate)}%
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
            {/* 每月统计 */}
            {data.time_series_stats.monthly_stats && data.time_series_stats.monthly_stats.length > 0 && (
              <div>
                <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">每月统计</h4>
                <div className="overflow-x-auto">
                  <table className="min-w-full text-sm">
                    <thead>
                      <tr className="border-b border-[#2B3139]">
                        <th className="text-left py-2 px-4 text-[#848E9C]">月份</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">交易数</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">已完成</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">总盈亏</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">总费用</th>
                        <th className="text-right py-2 px-4 text-[#848E9C]">胜率</th>
                      </tr>
                    </thead>
                    <tbody>
                      {data.time_series_stats.monthly_stats.map((stat) => (
                        <tr key={stat.month} className="border-b border-[#2B3139] hover:bg-[#1E2329]">
                          <td className="py-2 px-4 text-[#EAECEF]">{stat.month}</td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.total_trades}</td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">{stat.completed_trades}</td>
                          <td
                            className={`py-2 px-4 text-right font-semibold ${
                              stat.total_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                            }`}
                          >
                            {stat.total_pnl >= 0 ? '+' : ''}
                            {numberFormatter.format(stat.total_pnl)}
                          </td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">
                            {numberFormatter.format(stat.total_fees)}
                          </td>
                          <td className="py-2 px-4 text-right text-[#EAECEF]">
                            {numberFormatter.format(stat.win_rate)}%
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              </div>
            )}
          </div>
        </div>
      )}

      {/* 配对统计 */}
      {data.pair_stats && includePairs && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <PieChart className="w-5 h-5" style={{ color: '#F0B90B' }} />
            交易配对统计
          </h3>
          <div className="grid grid-cols-2 md:grid-cols-3 gap-4 mb-6">
            <div>
              <div className="text-sm text-[#848E9C] mb-1">配对数量</div>
              <div className="text-xl font-semibold text-[#EAECEF]">{data.pair_stats.total_pairs}</div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">配对成功率</div>
              <div className="text-xl font-semibold text-[#0ECB81]">
                {numberFormatter.format(data.pair_stats.pair_success_rate)}%
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">平均持仓时间</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {formatDuration(data.pair_stats.avg_holding_time)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最短持仓时间</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {formatDuration(data.pair_stats.min_holding_time)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">最长持仓时间</div>
              <div className="text-lg font-semibold text-[#EAECEF]">
                {formatDuration(data.pair_stats.max_holding_time)}
              </div>
            </div>
            <div>
              <div className="text-sm text-[#848E9C] mb-1">未配对记录</div>
              <div className="text-lg font-semibold text-[#F6465D]">{data.pair_stats.unpaired_trades}</div>
            </div>
          </div>
          {/* 配对详情 */}
          {data.pair_stats.pairs && data.pair_stats.pairs.length > 0 && (
            <div>
              <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">配对详情</h4>
              <div className="overflow-x-auto">
                <table className="min-w-full text-sm">
                  <thead>
                    <tr className="border-b border-[#2B3139]">
                      <th className="text-left py-2 px-4 text-[#848E9C]">币种</th>
                      <th className="text-left py-2 px-4 text-[#848E9C]">方向</th>
                      <th className="text-right py-2 px-4 text-[#848E9C]">数量</th>
                      <th className="text-right py-2 px-4 text-[#848E9C]">持仓时间</th>
                      <th className="text-right py-2 px-4 text-[#848E9C]">盈亏</th>
                      <th className="text-right py-2 px-4 text-[#848E9C]">开仓费用</th>
                      <th className="text-right py-2 px-4 text-[#848E9C]">平仓费用</th>
                    </tr>
                  </thead>
                  <tbody>
                    {data.pair_stats.pairs.slice(0, 20).map((pair, idx) => (
                      <tr key={idx} className="border-b border-[#2B3139] hover:bg-[#1E2329]">
                        <td className="py-2 px-4 text-[#EAECEF] font-medium">
                          {pair.open_record.symbol}
                        </td>
                        <td className="py-2 px-4 text-[#EAECEF]">
                          <span
                            className={`px-2 py-1 rounded text-xs font-semibold ${
                              pair.open_record.side === 'long'
                                ? 'bg-emerald-500/10 text-emerald-400 border border-emerald-500/30'
                                : 'bg-red-500/10 text-red-400 border border-red-500/30'
                            }`}
                          >
                            {pair.open_record.side.toUpperCase()}
                          </span>
                        </td>
                        <td className="py-2 px-4 text-right text-[#EAECEF]">
                          {numberFormatter.format(pair.matched_qty)}
                        </td>
                        <td className="py-2 px-4 text-right text-[#EAECEF]">
                          {formatDuration(pair.holding_time)}
                        </td>
                        <td
                          className={`py-2 px-4 text-right font-semibold ${
                            pair.pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                          }`}
                        >
                          {pair.pnl >= 0 ? '+' : ''}
                          {numberFormatter.format(pair.pnl)}
                        </td>
                        <td className="py-2 px-4 text-right text-[#EAECEF]">
                          {numberFormatter.format(pair.open_fee)}
                        </td>
                        <td className="py-2 px-4 text-right text-[#EAECEF]">
                          {numberFormatter.format(pair.close_fee)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
                {data.pair_stats.pairs.length > 20 && (
                  <div className="mt-4 text-sm text-[#848E9C] text-center">
                    显示前 20 条，共 {data.pair_stats.pairs.length} 条配对记录
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
      )}

      {/* 趋势分析 */}
      {data.trend_analysis && (
        <div className="binance-card p-6">
          <h3 className="text-lg font-semibold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
            <TrendingUp className="w-5 h-5" style={{ color: '#F0B90B' }} />
            趋势分析
          </h3>
          <div className="space-y-6">
            {/* 累计盈亏曲线 */}
            {data.trend_analysis.cumulative_pnl_series &&
              data.trend_analysis.cumulative_pnl_series.length > 0 && (
                <div>
                  <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">累计盈亏曲线</h4>
                  <div className="bg-[#0B0E11] p-4 rounded border border-[#2B3139]">
                    <div className="text-sm text-[#848E9C] mb-2">
                      数据点数量: {data.trend_analysis.cumulative_pnl_series.length}
                    </div>
                    <div className="space-y-2 max-h-64 overflow-y-auto">
                      {data.trend_analysis.cumulative_pnl_series
                        .slice(-20)
                        .map((point, idx) => (
                          <div
                            key={idx}
                            className="flex items-center justify-between text-sm border-b border-[#2B3139] pb-2"
                          >
                            <span className="text-[#848E9C]">
                              {new Date(point.timestamp).toLocaleString()}
                            </span>
                            <span
                              className={`font-semibold ${
                                point.cumulative_pnl >= 0 ? 'text-[#0ECB81]' : 'text-[#F6465D]'
                              }`}
                            >
                              {point.cumulative_pnl >= 0 ? '+' : ''}
                              {numberFormatter.format(point.cumulative_pnl)}
                            </span>
                          </div>
                        ))}
                    </div>
                    {data.trend_analysis.cumulative_pnl_series.length > 20 && (
                      <div className="mt-2 text-xs text-[#848E9C] text-center">
                        显示最近 20 个数据点，共 {data.trend_analysis.cumulative_pnl_series.length} 个
                      </div>
                    )}
                  </div>
                </div>
              )}
            {/* 盈亏分布 */}
            {data.trend_analysis.pnl_distribution &&
              data.trend_analysis.pnl_distribution.length > 0 && (
                <div>
                  <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">盈亏分布</h4>
                  <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                    {data.trend_analysis.pnl_distribution.map((bin, idx) => (
                      <div
                        key={idx}
                        className="bg-[#0B0E11] p-3 rounded border border-[#2B3139] text-center"
                      >
                        <div className="text-xs text-[#848E9C] mb-1">{bin.range}</div>
                        <div className="text-lg font-semibold text-[#EAECEF]">{bin.count}</div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            {/* 交易频率趋势 */}
            {data.trend_analysis.trade_frequency_trend &&
              data.trend_analysis.trade_frequency_trend.length > 0 && (
                <div>
                  <h4 className="text-base font-semibold mb-3 text-[#EAECEF]">交易频率趋势</h4>
                  <div className="bg-[#0B0E11] p-4 rounded border border-[#2B3139]">
                    <div className="text-sm text-[#848E9C] mb-2">
                      数据点数量: {data.trend_analysis.trade_frequency_trend.length}
                    </div>
                    <div className="space-y-2 max-h-64 overflow-y-auto">
                      {data.trend_analysis.trade_frequency_trend.slice(-20).map((point, idx) => (
                        <div
                          key={idx}
                          className="flex items-center justify-between text-sm border-b border-[#2B3139] pb-2"
                        >
                          <span className="text-[#848E9C]">
                            {new Date(point.timestamp).toLocaleString()}
                          </span>
                          <span className="font-semibold text-[#EAECEF]">{point.trade_count} 笔</span>
                        </div>
                      ))}
                    </div>
                    {data.trend_analysis.trade_frequency_trend.length > 20 && (
                      <div className="mt-2 text-xs text-[#848E9C] text-center">
                        显示最近 20 个数据点，共 {data.trend_analysis.trade_frequency_trend.length} 个
                      </div>
                    )}
                  </div>
                </div>
              )}
          </div>
        </div>
      )}
    </div>
  );
}

interface StatCardProps {
  title: string;
  value: number | string;
  icon: React.ReactNode;
  color: string;
}

function StatCard({ title, value, icon, color }: StatCardProps) {
  return (
    <div className="binance-card p-4">
      <div className="flex items-center justify-between mb-2">
        <div className="text-sm text-[#848E9C]">{title}</div>
        <div style={{ color }}>{icon}</div>
      </div>
      <div className="text-2xl font-semibold text-[#EAECEF]">{value}</div>
    </div>
  );
}


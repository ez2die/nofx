// 系统状态
export interface SystemStatus {
  is_running: boolean;
  start_time: string;
  runtime_minutes: number;
  call_count: number;
  initial_balance: number;
  scan_interval: string;
  stop_until: string;
  last_reset_time: string;
  ai_provider: string;
}

// 账户信息
export interface AccountInfo {
  total_equity: number;
  available_balance: number;
  total_pnl: number;
  total_pnl_pct: number;
  total_unrealized_pnl: number;
  margin_used: number;
  margin_used_pct: number;
  position_count: number;
  initial_balance: number;
  daily_pnl: number;
}

// 持仓信息
export interface Position {
  symbol: string;
  side: string;
  entry_price: number;
  mark_price: number;
  quantity: number;
  leverage: number;
  unrealized_pnl: number;
  unrealized_pnl_pct: number;
  liquidation_price: number;
  margin_used: number;
}

// 决策动作
export interface DecisionAction {
  action: string;
  symbol: string;
  quantity: number;
  leverage: number;
  price: number;
  order_id: number;
  timestamp: string;
  success: boolean;
  error: string;
}

// 决策记录
export interface DecisionRecord {
  timestamp: string;
  cycle_number: number;
  input_prompt: string;
  cot_trace: string;
  decision_json: string;
  account_state: {
    total_balance: number;
    available_balance: number;
    total_unrealized_profit: number;
    position_count: number;
    margin_used_pct: number;
  };
  positions: Array<{
    symbol: string;
    side: string;
    position_amt: number;
    entry_price: number;
    mark_price: number;
    unrealized_profit: number;
    leverage: number;
    liquidation_price: number;
  }>;
  candidate_coins: string[];
  decisions: DecisionAction[];
  execution_log: string[];
  success: boolean;
  error_message: string;
}

// 统计信息
export interface Statistics {
  total_cycles: number;
  successful_cycles: number;
  failed_cycles: number;
  total_open_positions: number;
  total_close_positions: number;
}

// AI Trading 相关类型
export interface TraderInfo {
  trader_id: string;
  trader_name: string;
  ai_model: string;
  exchange_id?: string;
  is_running?: boolean;
  custom_prompt?: string;
}

export interface AIModel {
  id: string;
  name: string;
  provider: string;
  enabled: boolean;
  apiKey?: string;
  customApiUrl?: string;
}

export interface Exchange {
  id: string;
  name: string;
  type: 'cex' | 'dex';
  enabled: boolean;
  apiKey?: string;
  secretKey?: string;
  testnet?: boolean;
  hyperliquidWalletAddr?: string;
}

export interface CreateTraderRequest {
  name: string;
  ai_model_id: string;
  exchange_id: string;
  initial_balance: number;
  scan_interval_minutes?: number;
  btc_eth_leverage?: number;
  altcoin_leverage?: number;
  trading_symbols?: string;
  custom_prompt?: string;
  override_base_prompt?: boolean;
  system_prompt_template?: string;
  is_cross_margin?: boolean;
  use_coin_pool?: boolean;
  use_oi_top?: boolean;
}

export interface UpdateModelConfigRequest {
  models: Record<
    string,
    {
      enabled: boolean;
      api_key: string;
      custom_api_url?: string;
      custom_model_name?: string;
    }
  >;
}

export interface UpdateExchangeConfigRequest {
  exchanges: Record<
    string,
    {
      enabled: boolean;
      api_key: string;
      secret_key: string;
      testnet?: boolean;
      hyperliquid_wallet_addr?: string;
    }
  >;
}

// Competition
export interface CompetitionTraderData {
  trader_id: string;
  trader_name: string;
  ai_model: string;
  exchange: string;
  total_equity: number;
  total_pnl: number;
  total_pnl_pct: number;
  position_count: number;
  margin_used_pct: number;
  is_running: boolean;
}

export interface CompetitionData {
  traders: CompetitionTraderData[];
  count: number;
}

export interface TraderConfigData {
  trader_id?: string;
  trader_name: string;
  ai_model: string;
  exchange_id: string;
  btc_eth_leverage: number;
  altcoin_leverage: number;
  trading_symbols: string;
  custom_prompt: string;
  override_base_prompt: boolean;
  system_prompt_template?: string;
  is_cross_margin: boolean;
  use_coin_pool: boolean;
  use_oi_top: boolean;
  initial_balance: number;
  scan_interval_minutes: number;
  is_running: boolean;
}

// Trade Performance（trade history 扩展统计）
export interface TradePerformanceStats {
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_pnl: number;
  avg_win: number;
  avg_loss: number;
  profit_factor: number;
  total_fees: number;
}

export interface TradePerformanceSymbolStats {
  symbol: string;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_pn_l: number;
  avg_pn_l: number;
}

export interface TradePerformanceRecentTrade {
  symbol: string;
  action: string;
  side: string;
  quantity: number;
  signed_quantity?: number;
  execution_price: number;
  pn_l: number;
  fee: number;
  timestamp: string;
  exchange_timestamp?: string;
}

export interface TradePerformance extends TradePerformanceStats {
  sharpe_ratio: number;
  recent_trades: TradePerformanceRecentTrade[];
  symbol_stats: Record<string, TradePerformanceSymbolStats>;
  best_symbol: string;
  worst_symbol: string;
  generated_at: string;
}

// Trade Analytics 类型定义
export interface TradeAnalyticsFilter {
  trader_id: string;
  symbol?: string;
  side?: string;
  action?: string;
  start_time?: string;
  end_time?: string;
  group_by?: string;
  include_pairs?: boolean;
}

export interface TradeOverview {
  total_trades: number;
  open_trades: number;
  close_trades: number;
  completed_trades: number;
  unclosed_trades: number;
}

export interface PnLStatistics {
  total_pnl: number;
  total_profit: number;
  total_loss: number;
  net_pnl: number;
  avg_pnl: number;
  max_win: number;
  max_loss: number;
  profit_factor: number;
}

export interface WinRateStatistics {
  win_rate: number;
  winning_trades: number;
  losing_trades: number;
  break_even_trades: number;
  avg_win: number;
  avg_loss: number;
}

export interface FeeStatistics {
  total_fees: number;
  open_fees: number;
  close_fees: number;
  avg_fee: number;
  fee_ratio: number;
  builder_fees: number;
}

export interface SideStatistics {
  total_trades: number;
  total_pnl: number;
  win_rate: number;
  avg_pnl: number;
}

export interface DirectionPreference {
  preferred_side: string;
  long_ratio: number;
  short_ratio: number;
  total_trades: number;
}

export interface DirectionStatistics {
  long_stats: SideStatistics;
  short_stats: SideStatistics;
  preference?: DirectionPreference;
}

export interface RiskMetrics {
  max_drawdown: number;
  max_drawdown_percent: number;
  drawdown_duration: number;
  recovery_time: number;
  volatility: number;
  sharpe_ratio: number;
  drawdown_history?: Array<{
    start_time: string;
    end_time: string;
    peak_value: number;
    trough_value: number;
    drawdown: number;
    duration: number;
  }>;
}

export interface StreakStatistics {
  longest_winning_streak: number;
  longest_losing_streak: number;
  current_streak: number;
  current_streak_type: string;
}

export interface SymbolStatistics {
  symbol: string;
  total_trades: number;
  completed_trades: number;
  total_pnl: number;
  win_rate: number;
  avg_pnl: number;
  max_win: number;
  max_loss: number;
  total_fees: number;
}

export interface DailyStatistics {
  date: string;
  total_trades: number;
  completed_trades: number;
  total_pnl: number;
  total_fees: number;
  win_rate: number;
}

export interface WeeklyStatistics {
  week: string;
  total_trades: number;
  completed_trades: number;
  total_pnl: number;
  total_fees: number;
  win_rate: number;
}

export interface MonthlyStatistics {
  month: string;
  total_trades: number;
  completed_trades: number;
  total_pnl: number;
  total_fees: number;
  win_rate: number;
}

export interface TimeSeriesStatistics {
  daily_stats: DailyStatistics[];
  weekly_stats: WeeklyStatistics[];
  monthly_stats: MonthlyStatistics[];
}

export interface TradeHistoryRecord {
  id: number;
  trader_id: string;
  symbol: string;
  side: 'long' | 'short' | '';
  action: string;
  quantity: number;
  signed_quantity?: number | null;
  execution_price: number;
  pnl?: number | null;
  fee: number;
  fee_token?: string | null;
  raw_dir: string;
  start_position?: number | null;
  builder_fee?: number | null;
  exchange_side?: string | null;
  timestamp: string;
  exchange_timestamp?: string | null;
  exchange_timestamp_ms?: number | null;
  created_at: string;
  updated_at: string;
}

export interface TradeHistoryListResponse {
  records: TradeHistoryRecord[];
  total: number;
  limit: number;
  offset: number;
  current_page: number;
  total_pages: number;
}

export interface TradeFrequencyStats {
  daily_average_trades: number;
  avg_trade_interval: number;
  most_active_hours: number[];
  most_active_days: string[];
}

export interface CumulativePnLPoint {
  timestamp: string;
  cumulative_pnl: number;
}

export interface PnLDistributionBin {
  range: string;
  count: number;
}

export interface TradeFrequencyPoint {
  timestamp: string;
  trade_count: number;
}

export interface TrendAnalysis {
  cumulative_pnl_series: CumulativePnLPoint[];
  pnl_distribution: PnLDistributionBin[];
  trade_frequency_trend: TradeFrequencyPoint[];
}

export interface ActionStats {
  total_trades: number;
  total_pnl: number;
  total_fees: number;
  avg_pnl: number;
}

export interface ActionStatistics {
  open_stats: ActionStats;
  close_stats: ActionStats;
}

export interface TradePair {
  open_record: TradeHistoryRecord;
  close_record: TradeHistoryRecord;
  matched_qty: number;
  holding_time: number;
  pnl: number;
  open_fee: number;
  close_fee: number;
}

export interface PairStatistics {
  total_pairs: number;
  pair_success_rate: number;
  avg_holding_time: number;
  min_holding_time: number;
  max_holding_time: number;
  unpaired_trades: number;
  pairs: TradePair[];
}

export interface TradeAnalytics {
  overview?: TradeOverview;
  pnl_stats?: PnLStatistics;
  win_rate_stats?: WinRateStatistics;
  fee_stats?: FeeStatistics;
  direction_stats?: DirectionStatistics;
  risk_metrics?: RiskMetrics;
  streak_stats?: StreakStatistics;
  symbol_stats?: Record<string, SymbolStatistics>;
  time_series_stats?: TimeSeriesStatistics;
  frequency_stats?: TradeFrequencyStats;
  trend_analysis?: TrendAnalysis;
  action_stats?: ActionStatistics;
  pair_stats?: PairStatistics;
}

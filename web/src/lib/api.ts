import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
  TraderInfo,
  AIModel,
  Exchange,
  CreateTraderRequest,
  UpdateModelConfigRequest,
  UpdateExchangeConfigRequest,
  CompetitionData,
  TradeHistoryListResponse,
  TradeAnalytics,
  TradeOverview,
  PnLStatistics,
  WinRateStatistics,
  FeeStatistics,
  RiskMetrics,
  SymbolStatistics,
  TimeSeriesStatistics,
  TradeFrequencyStats,
  ActionStatistics,
  TrendAnalysis,
  PairStatistics,
  TradePerformance,
} from '../types';

const API_BASE = '/api';

// Helper function to get auth headers
function getAuthHeaders(): Record<string, string> {
  const token = localStorage.getItem('auth_token');
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  return headers;
}

export const api = {
  // AI交易员管理接口
  async getTraders(): Promise<TraderInfo[]> {
    const res = await fetch(`${API_BASE}/my-traders`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取trader列表失败');
    return res.json();
  },

  // 获取公开的交易员列表（无需认证）
  async getPublicTraders(): Promise<any[]> {
    const res = await fetch(`${API_BASE}/traders`);
    if (!res.ok) throw new Error('获取公开trader列表失败');
    return res.json();
  },

  async createTrader(request: CreateTraderRequest): Promise<TraderInfo> {
    const res = await fetch(`${API_BASE}/traders`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify(request),
    });
    if (!res.ok) throw new Error('创建交易员失败');
    return res.json();
  },

  async deleteTrader(traderId: string): Promise<void> {
    const res = await fetch(`${API_BASE}/traders/${traderId}`, {
      method: 'DELETE',
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('删除交易员失败');
  },

  async startTrader(traderId: string): Promise<void> {
    const res = await fetch(`${API_BASE}/traders/${traderId}/start`, {
      method: 'POST',
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('启动交易员失败');
  },

  async stopTrader(traderId: string): Promise<void> {
    const res = await fetch(`${API_BASE}/traders/${traderId}/stop`, {
      method: 'POST',
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('停止交易员失败');
  },

  async updateTraderPrompt(traderId: string, customPrompt: string): Promise<void> {
    const res = await fetch(`${API_BASE}/traders/${traderId}/prompt`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify({ custom_prompt: customPrompt }),
    });
    if (!res.ok) throw new Error('更新自定义策略失败');
  },

  async getTraderConfig(traderId: string): Promise<any> {
    const res = await fetch(`${API_BASE}/traders/${traderId}/config`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取交易员配置失败');
    return res.json();
  },

  async updateTrader(traderId: string, request: CreateTraderRequest): Promise<TraderInfo> {
    const res = await fetch(`${API_BASE}/traders/${traderId}`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(request),
    });
    if (!res.ok) throw new Error('更新交易员失败');
    return res.json();
  },

  // AI模型配置接口
  async getModelConfigs(): Promise<AIModel[]> {
    const res = await fetch(`${API_BASE}/models`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取模型配置失败');
    return res.json();
  },

  // 获取系统支持的AI模型列表（无需认证）
  async getSupportedModels(): Promise<AIModel[]> {
    const res = await fetch(`${API_BASE}/supported-models`);
    if (!res.ok) throw new Error('获取支持的模型失败');
    return res.json();
  },

  async updateModelConfigs(request: UpdateModelConfigRequest): Promise<void> {
    const res = await fetch(`${API_BASE}/models`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(request),
    });
    if (!res.ok) throw new Error('更新模型配置失败');
  },

  // 交易所配置接口
  async getExchangeConfigs(): Promise<Exchange[]> {
    const res = await fetch(`${API_BASE}/exchanges`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取交易所配置失败');
    return res.json();
  },

  // 获取系统支持的交易所列表（无需认证）
  async getSupportedExchanges(): Promise<Exchange[]> {
    const res = await fetch(`${API_BASE}/supported-exchanges`);
    if (!res.ok) throw new Error('获取支持的交易所失败');
    return res.json();
  },

  async updateExchangeConfigs(request: UpdateExchangeConfigRequest): Promise<void> {
    const res = await fetch(`${API_BASE}/exchanges`, {
      method: 'PUT',
      headers: getAuthHeaders(),
      body: JSON.stringify(request),
    });
    if (!res.ok) throw new Error('更新交易所配置失败');
  },

  // 获取系统状态（支持trader_id）
  async getStatus(traderId?: string): Promise<SystemStatus> {
    const url = traderId
      ? `${API_BASE}/status?trader_id=${traderId}`
      : `${API_BASE}/status`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取系统状态失败');
    return res.json();
  },

  // 获取账户信息（支持trader_id）
  async getAccount(traderId?: string): Promise<AccountInfo> {
    const url = traderId
      ? `${API_BASE}/account?trader_id=${traderId}`
      : `${API_BASE}/account`;
    const res = await fetch(url, {
      cache: 'no-store',
      headers: {
        ...getAuthHeaders(),
        'Cache-Control': 'no-cache',
      },
    });
    if (!res.ok) throw new Error('获取账户信息失败');
    const data = await res.json();
    console.log('Account data fetched:', data);
    return data;
  },

  // 获取持仓列表（支持trader_id）
  async getPositions(traderId?: string): Promise<Position[]> {
    const url = traderId
      ? `${API_BASE}/positions?trader_id=${traderId}`
      : `${API_BASE}/positions`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取持仓列表失败');
    return res.json();
  },

  // 获取决策日志（支持trader_id）
  async getDecisions(traderId?: string): Promise<DecisionRecord[]> {
    const url = traderId
      ? `${API_BASE}/decisions?trader_id=${traderId}`
      : `${API_BASE}/decisions`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取决策日志失败');
    return res.json();
  },

  // 获取最新决策（支持trader_id）
  async getLatestDecisions(traderId?: string): Promise<DecisionRecord[]> {
    const url = traderId
      ? `${API_BASE}/decisions/latest?trader_id=${traderId}`
      : `${API_BASE}/decisions/latest`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取最新决策失败');
    return res.json();
  },

  async getTradeHistory(params: {
    traderId: string;
    limit?: number;
    offset?: number;
    symbol?: string;
    action?: string;
    side?: string;
    orderBy?: string;
    startTime?: string;
    endTime?: string;
  }): Promise<TradeHistoryListResponse> {
    const {
      traderId,
      limit,
      offset,
      symbol,
      action,
      side,
      orderBy,
      startTime,
      endTime,
    } = params;

    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', traderId);

    if (limit !== undefined) searchParams.set('limit', String(limit));
    if (offset !== undefined) searchParams.set('offset', String(offset));
    if (symbol) searchParams.set('symbol', symbol);
    if (action) searchParams.set('action', action);
    if (side) searchParams.set('side', side);
    if (orderBy) searchParams.set('order_by', orderBy);
    if (startTime) searchParams.set('start_time', startTime);
    if (endTime) searchParams.set('end_time', endTime);

    const res = await fetch(`${API_BASE}/trade-history?${searchParams.toString()}`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) {
      throw new Error('获取交易历史失败');
    }
    return res.json();
  },

  async getTradePerformanceStats(params: {
    trader_id: string;
    start_time?: string;
    end_time?: string;
    recent_limit?: number;
    sharpe_window?: number;
  }): Promise<TradePerformance> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', params.trader_id);
    if (params.start_time) searchParams.set('start_time', params.start_time);
    if (params.end_time) searchParams.set('end_time', params.end_time);
    if (params.recent_limit) searchParams.set('recent_limit', params.recent_limit.toString());
    if (params.sharpe_window) searchParams.set('sharpe_window', params.sharpe_window.toString());

    const res = await fetch(`${API_BASE}/trade-history/performance?${searchParams.toString()}`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) {
      throw new Error('获取交易表现失败');
    }
    return res.json();
  },

  // 获取统计信息（支持trader_id）
  async getStatistics(traderId?: string): Promise<Statistics> {
    const url = traderId
      ? `${API_BASE}/statistics?trader_id=${traderId}`
      : `${API_BASE}/statistics`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取统计信息失败');
    return res.json();
  },

  // 获取收益率历史数据（支持trader_id）
  async getEquityHistory(traderId?: string): Promise<any[]> {
    const url = traderId
      ? `${API_BASE}/equity-history?trader_id=${traderId}`
      : `${API_BASE}/equity-history`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取历史数据失败');
    return res.json();
  },

  // 批量获取多个交易员的历史数据（无需认证）
  async getEquityHistoryBatch(traderIds: string[]): Promise<any> {
    const res = await fetch(`${API_BASE}/equity-history-batch`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ trader_ids: traderIds }),
    });
    if (!res.ok) throw new Error('获取批量历史数据失败');
    return res.json();
  },

  // 获取前5名交易员数据（无需认证）
  async getTopTraders(): Promise<any[]> {
    const res = await fetch(`${API_BASE}/top-traders`);
    if (!res.ok) throw new Error('获取前5名交易员失败');
    return res.json();
  },

  // 获取公开交易员配置（无需认证）
  async getPublicTraderConfig(traderId: string): Promise<any> {
    const res = await fetch(`${API_BASE}/trader/${traderId}/config`);
    if (!res.ok) throw new Error('获取公开交易员配置失败');
    return res.json();
  },

  // 获取AI学习表现分析（支持trader_id）
  async getPerformance(traderId?: string): Promise<any> {
    const url = traderId
      ? `${API_BASE}/performance?trader_id=${traderId}`
      : `${API_BASE}/performance`;
    const res = await fetch(url, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取AI学习数据失败');
    return res.json();
  },

  // 获取竞赛数据（无需认证）
  async getCompetition(): Promise<CompetitionData> {
    const res = await fetch(`${API_BASE}/competition`);
    if (!res.ok) throw new Error('获取竞赛数据失败');
    return res.json();
  },

  // 用户信号源配置接口
  async getUserSignalSource(): Promise<{coin_pool_url: string, oi_top_url: string}> {
    const res = await fetch(`${API_BASE}/user/signal-sources`, {
      headers: getAuthHeaders(),
    });
    if (!res.ok) throw new Error('获取用户信号源配置失败');
    return res.json();
  },

  async saveUserSignalSource(coinPoolUrl: string, oiTopUrl: string): Promise<void> {
    const res = await fetch(`${API_BASE}/user/signal-sources`, {
      method: 'POST',
      headers: getAuthHeaders(),
      body: JSON.stringify({
        coin_pool_url: coinPoolUrl,
        oi_top_url: oiTopUrl,
      }),
    });
    if (!res.ok) throw new Error('保存用户信号源配置失败');
  },

  // Trade Analytics API
  async getTradeAnalytics(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    action?: string;
    start_time?: string;
    end_time?: string;
    group_by?: string;
    include_pairs?: boolean;
  }): Promise<TradeAnalytics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.action) searchParams.set('action', filter.action);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);
    if (filter.group_by) searchParams.set('group_by', filter.group_by);
    if (filter.include_pairs) searchParams.set('include_pairs', 'true');

    const res = await fetch(`${API_BASE}/trade-analytics?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取交易分析失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsOverview(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<TradeOverview> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/overview?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取概览统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsPnL(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<PnLStatistics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/pnl?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取盈亏统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsWinRate(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<WinRateStatistics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/win-rate?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取胜率统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsFees(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<FeeStatistics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/fees?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取费用统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsRisk(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<RiskMetrics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/risk?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取风险指标失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsSymbols(filter: {
    trader_id: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<Record<string, SymbolStatistics>> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/symbols?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取币种统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsTimeSeries(filter: {
    trader_id: string;
    group_by: string;
    start_time?: string;
    end_time?: string;
  }): Promise<TimeSeriesStatistics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    searchParams.set('group_by', filter.group_by);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/time-series?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取时间序列统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsFrequency(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<TradeFrequencyStats> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/frequency?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取交易频率统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsActions(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<ActionStatistics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/actions?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取交易类型统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsTrends(filter: {
    trader_id: string;
    symbol?: string;
    side?: string;
    start_time?: string;
    end_time?: string;
  }): Promise<TrendAnalysis> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.side) searchParams.set('side', filter.side);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);

    const res = await fetch(`${API_BASE}/trade-analytics/trends?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取趋势分析失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },

  async getTradeAnalyticsPairs(filter: {
    trader_id: string;
    symbol?: string;
    start_time?: string;
    end_time?: string;
    limit?: number;
  }): Promise<PairStatistics> {
    const searchParams = new URLSearchParams();
    searchParams.set('trader_id', filter.trader_id);
    if (filter.symbol) searchParams.set('symbol', filter.symbol);
    if (filter.start_time) searchParams.set('start_time', filter.start_time);
    if (filter.end_time) searchParams.set('end_time', filter.end_time);
    if (filter.limit) searchParams.set('limit', String(filter.limit));

    const res = await fetch(`${API_BASE}/trade-analytics/pairs?${searchParams.toString()}`, {
      headers: {
        'Content-Type': 'application/json',
      },
    });
    if (!res.ok) {
      const errorData = await res.json().catch(() => ({}));
      const errorMessage = errorData.error || `获取配对统计失败 (${res.status})`;
      throw new Error(errorMessage);
    }
    return res.json();
  },
};

import { useMemo } from 'react';
import useSWR from 'swr';
import { api } from '../lib/api';
import type { TradePerformance } from '../types';

export interface UseTradePerformanceOptions {
  start_time?: string;
  end_time?: string;
  recent_limit?: number;
  sharpe_window?: number;
  enabled?: boolean;
}

export function useTradePerformance(
  traderId?: string,
  options: UseTradePerformanceOptions = {},
) {
  const swrKey = useMemo(() => {
    if (!traderId || options.enabled === false) return null;
    return [
      'trade-performance',
      traderId,
      options.start_time,
      options.end_time,
      options.recent_limit,
      options.sharpe_window,
    ] as const;
  }, [
    traderId,
    options.start_time,
    options.end_time,
    options.recent_limit,
    options.sharpe_window,
    options.enabled,
  ]);

  const { data, error, isLoading, isValidating, mutate } = useSWR<TradePerformance>(
    swrKey,
    () =>
      api.getTradePerformanceStats({
        trader_id: traderId as string,
        start_time: options.start_time,
        end_time: options.end_time,
        recent_limit: options.recent_limit,
        sharpe_window: options.sharpe_window,
      }),
    {
      revalidateOnFocus: false,
      refreshInterval: 0,
    },
  );

  return {
    performance: data,
    error,
    isLoading,
    isValidating,
    refresh: mutate,
  };
}


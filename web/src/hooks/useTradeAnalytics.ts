import { useMemo } from 'react';
import useSWR from 'swr';
import { api } from '../lib/api';
import type { TradeAnalytics, TradeAnalyticsFilter } from '../types';

export interface UseTradeAnalyticsOptions {
  includePairs?: boolean;
  groupBy?: string;
  enabled?: boolean;
}

export function useTradeAnalytics(
  traderId?: string,
  filter?: Partial<TradeAnalyticsFilter>,
  options: UseTradeAnalyticsOptions = {},
) {
  const swrKey = useMemo(() => {
    if (!traderId || options.enabled === false) return null;
    return [
      'trade-analytics',
      traderId,
      filter?.symbol,
      filter?.side,
      filter?.action,
      filter?.start_time,
      filter?.end_time,
      options.includePairs,
      options.groupBy,
    ] as const;
  }, [traderId, filter, options.includePairs, options.groupBy, options.enabled]);

  const { data, error, isLoading, isValidating, mutate } = useSWR<TradeAnalytics>(
    swrKey,
    () =>
      api.getTradeAnalytics({
        trader_id: traderId as string,
        symbol: filter?.symbol,
        side: filter?.side,
        action: filter?.action,
        start_time: filter?.start_time,
        end_time: filter?.end_time,
        group_by: options.groupBy,
        include_pairs: options.includePairs,
      }),
    {
      revalidateOnFocus: false,
      refreshInterval: 0, // 不自动刷新，手动刷新
    },
  );

  return {
    data,
    error,
    isLoading,
    isValidating,
    refresh: mutate,
  };
}


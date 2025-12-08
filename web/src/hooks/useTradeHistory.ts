import { useCallback, useEffect, useMemo, useState } from 'react';
import useSWR from 'swr';

import { api } from '../lib/api';
import type { TradeHistoryListResponse } from '../types';

export interface TradeHistoryFilters {
  symbol?: string;
  action?: string;
  side?: string;
  orderBy?: string;
  startTime?: string;
  endTime?: string;
}

export interface UseTradeHistoryOptions {
  initialPageSize?: number;
  initialFilters?: TradeHistoryFilters;
}

export function useTradeHistory(
  traderId?: string,
  options: UseTradeHistoryOptions = {},
) {
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(options.initialPageSize ?? 50);
  const [filters, setFilters] = useState<TradeHistoryFilters>(
    options.initialFilters ?? {},
  );

  // Reset pagination whenever trader changes.
  useEffect(() => {
    setPage(1);
  }, [traderId]);

  const filterKey = useMemo(() => JSON.stringify(filters), [filters]);
  const offset = useMemo(
    () => Math.max(0, (page - 1) * pageSize),
    [page, pageSize],
  );

  const swrKey = useMemo(() => {
    if (!traderId) return null;
    return [
      'trade-history',
      traderId,
      page,
      pageSize,
      filterKey,
    ] as const;
  }, [filterKey, page, pageSize, traderId]);

  const { data, error, isLoading, isValidating, mutate } =
    useSWR<TradeHistoryListResponse>(
      swrKey,
      () =>
        api.getTradeHistory({
          traderId: traderId as string,
          limit: pageSize,
          offset,
          symbol: filters.symbol,
          action: filters.action,
          side: filters.side,
          orderBy: filters.orderBy,
          startTime: filters.startTime,
          endTime: filters.endTime,
        }),
      {
        keepPreviousData: true,
        revalidateOnFocus: false,
      },
    );

  const updateFilters = useCallback(
    (partial: TradeHistoryFilters) => {
      setFilters((prev) => {
        const next = { ...prev, ...partial };
        return next;
      });
      setPage(1);
    },
    [],
  );

  const setPageSafe = useCallback((nextPage: number) => {
    setPage((prev) => {
      const target = Number.isFinite(nextPage) ? Math.max(1, Math.floor(nextPage)) : prev;
      if (target === prev) return prev;
      return target;
    });
  }, []);

  const setPageSizeSafe = useCallback((nextSize: number) => {
    setPageSize((prev) => {
      const target = Number.isFinite(nextSize) ? Math.max(1, Math.floor(nextSize)) : prev;
      if (target === prev) return prev;
      return target;
    });
    setPage(1);
  }, []);

  const totalPages = data?.total_pages ?? 0;
  const resetFilters = useCallback(() => {
    setFilters({});
    setPage(1);
  }, []);

  return {
    records: data?.records ?? [],
    total: data?.total ?? 0,
    page,
    pageSize,
    offset,
    totalPages,
    filters,
    updateFilters,
    setPage: setPageSafe,
    setPageSize: setPageSizeSafe,
    resetFilters,
    isLoading,
    isValidating,
    error,
    refresh: mutate,
    hasMore:
      totalPages > 0
      ? page < totalPages
      : (data?.records?.length ?? 0) === pageSize,
  };
}


package trade_analytics

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// APIHandler API处理器
type APIHandler struct {
	service Service
}

// NewAPIHandler 创建API处理器
func NewAPIHandler(service Service) *APIHandler {
	return &APIHandler{service: service}
}

// RegisterRoutes 注册路由
func (h *APIHandler) RegisterRoutes(router *gin.RouterGroup) {
	analytics := router.Group("/trade-analytics")
	{
		analytics.GET("", h.handleGetAnalytics)
		analytics.GET("/overview", h.handleGetOverview)
		analytics.GET("/pnl", h.handleGetPnLStats)
		analytics.GET("/win-rate", h.handleGetWinRateStats)
		analytics.GET("/fees", h.handleGetFeeStats)
		analytics.GET("/risk", h.handleGetRiskMetrics)
		analytics.GET("/symbols", h.handleGetSymbolStats)
		analytics.GET("/time-series", h.handleGetTimeSeriesStats)
		analytics.GET("/pairs", h.handleGetPairStats)
		analytics.GET("/frequency", h.handleGetFrequencyStats)
		analytics.GET("/actions", h.handleGetActionStats)
		analytics.GET("/trends", h.handleGetTrendAnalysis)
	}
}

// parseAnalyticsFilter 解析查询参数为过滤器
func parseAnalyticsFilter(c *gin.Context) (*AnalyticsFilter, error) {
	filter := &AnalyticsFilter{}

	// 必需参数
	traderID := c.Query("trader_id")
	if traderID == "" {
		return nil, fmt.Errorf("trader_id is required")
	}
	filter.TraderID = traderID

	// 可选参数
	filter.Symbol = c.Query("symbol")
	filter.Side = c.Query("side")
	filter.Action = c.Query("action")

	// 时间参数
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			filter.StartTime = &startTime
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			filter.EndTime = &endTime
		}
	}

	// 分组方式
	filter.GroupBy = c.Query("group_by")

	// 是否包含配对分析
	if includePairsStr := c.Query("include_pairs"); includePairsStr != "" {
		if includePairs, err := strconv.ParseBool(includePairsStr); err == nil {
			filter.IncludePairs = includePairs
		}
	}

	return filter, nil
}

// handleGetAnalytics 获取完整分析
func (h *APIHandler) handleGetAnalytics(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analytics, err := h.service.GetAnalytics(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取分析失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

// handleGetOverview 获取概览统计
func (h *APIHandler) handleGetOverview(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	overview, err := h.service.GetOverview(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取概览统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, overview)
}

// handleGetPnLStats 获取盈亏统计
func (h *APIHandler) handleGetPnLStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetPnLStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取盈亏统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetWinRateStats 获取胜率统计
func (h *APIHandler) handleGetWinRateStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetWinRateStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取胜率统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetFeeStats 获取费用统计
func (h *APIHandler) handleGetFeeStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetFeeStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取费用统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetRiskMetrics 获取风险指标
func (h *APIHandler) handleGetRiskMetrics(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metrics, err := h.service.GetRiskMetrics(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取风险指标失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, metrics)
}

// handleGetSymbolStats 获取币种统计
func (h *APIHandler) handleGetSymbolStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetSymbolStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取币种统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetTimeSeriesStats 获取时间序列统计
func (h *APIHandler) handleGetTimeSeriesStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// group_by 参数是必需的
	if filter.GroupBy == "" {
		filter.GroupBy = c.Query("group_by")
		if filter.GroupBy == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "group_by parameter is required (day/week/month)"})
			return
		}
	}

	stats, err := h.service.GetTimeSeriesStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取时间序列统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetPairStats 获取配对统计
func (h *APIHandler) handleGetPairStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 限制返回数量
	limitStr := c.DefaultQuery("limit", "100")
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 100
	}

	stats, err := h.service.GetPairStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取配对统计失败: %v", err),
		})
		return
	}

	// 限制返回的配对数量
	if limit > 0 && len(stats.Pairs) > limit {
		stats.Pairs = stats.Pairs[:limit]
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetFrequencyStats 获取交易频率统计
func (h *APIHandler) handleGetFrequencyStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetFrequencyStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取交易频率统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetActionStats 获取交易类型统计
func (h *APIHandler) handleGetActionStats(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	stats, err := h.service.GetActionStats(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取交易类型统计失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleGetTrendAnalysis 获取趋势分析
func (h *APIHandler) handleGetTrendAnalysis(c *gin.Context) {
	filter, err := parseAnalyticsFilter(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	analysis, err := h.service.GetTrendAnalysis(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取趋势分析失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, analysis)
}


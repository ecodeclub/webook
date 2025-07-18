package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type MetricsBuilder struct {
	summaryVec *prometheus.SummaryVec
	counterVec *prometheus.CounterVec
}

func NewMetricsBuilder() *MetricsBuilder {
	summaryVec := promauto.NewSummaryVec(
		prometheus.SummaryOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request duration in seconds",
			Objectives: map[float64]float64{
				0.5:  0.05,
				0.9:  0.01,
				0.95: 0.005,
				0.99: 0.001,
			},
		},
		[]string{"method", "path", "status_code"},
	)

	counterVec := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status_code"},
	)

	return &MetricsBuilder{
		summaryVec: summaryVec,
		counterVec: counterVec,
	}
}

func (a *MetricsBuilder) Build() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		start := time.Now()

		// 处理请求
		ctx.Next()

		// 计算响应时间
		duration := time.Since(start).Seconds()

		// 获取请求信息
		method := ctx.Request.Method
		path := ctx.FullPath()
		if path == "" {
			path = ctx.Request.URL.Path
		}
		statusCode := strconv.Itoa(ctx.Writer.Status())

		// 记录响应时间指标
		a.summaryVec.WithLabelValues(method, path, statusCode).Observe(duration)

		// 记录访问次数指标
		a.counterVec.WithLabelValues(method, path, statusCode).Inc()
	}
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// PR Metrics
var (
	// PRCreatedTotal - количество созданных PR
	PRCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pr_created_total",
		Help: "Total number of pull requests created",
	})

	// PRMergedTotal - количество смерженных PR
	PRMergedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pr_merged_total",
		Help: "Total number of pull requests merged",
	})

	// PRReassignedTotal - количество переназначений ревьюверов
	PRReassignedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pr_reassigned_total",
		Help: "Total number of reviewer reassignments",
	})

	// PROpenCount - текущее количество открытых PR
	PROpenCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pr_open_count",
		Help: "Current number of open pull requests",
	})

	// PRCreationDuration - время создания PR
	PRCreationDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pr_creation_duration_seconds",
		Help:    "Duration of PR creation operation in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// PRMergeDuration - время merge PR
	PRMergeDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pr_merge_duration_seconds",
		Help:    "Duration of PR merge operation in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// PRReviewersAssigned - распределение количества назначенных ревьюверов
	PRReviewersAssigned = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "pr_reviewers_assigned",
		Help:    "Distribution of assigned reviewers count (0-2)",
		Buckets: []float64{0, 1, 2},
	})
)

// Team Metrics
var (
	// TeamCreatedTotal - количество созданных команд
	TeamCreatedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "team_created_total",
		Help: "Total number of teams created",
	})

	// TeamMembersCount - количество участников по командам
	TeamMembersCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "team_members_count",
		Help: "Number of team members by team",
	}, []string{"team_name"})

	// TeamActiveMembersCount - активные участники по командам
	TeamActiveMembersCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "team_active_members_count",
		Help: "Number of active team members by team",
	}, []string{"team_name"})
)

// User Metrics
var (
	// UserActiveStatusChanged - изменения статуса активности
	UserActiveStatusChanged = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "user_active_status_changed_total",
		Help: "Total number of user active status changes",
	}, []string{"status"})

	// UserReviewAssignmentsCount - количество назначенных review
	UserReviewAssignmentsCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "user_review_assignments_count",
		Help: "Number of review assignments per user",
	}, []string{"user_id"})

	// UserNoCandidatesErrors - ошибки "нет доступных кандидатов"
	UserNoCandidatesErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "user_no_candidates_errors_total",
		Help: "Total number of no candidates errors during reassignment",
	})
)

// HTTP Metrics
var (
	// HTTPRequestsTotal - общее количество HTTP запросов
	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
	}, []string{"method", "path", "status"})

	// HTTPRequestDuration - время обработки запроса
	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_duration_seconds",
		Help:    "Duration of HTTP request in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "path", "status"})

	// HTTPRequestSize - размер запроса
	HTTPRequestSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_request_size_bytes",
		Help:    "Size of HTTP request in bytes",
		Buckets: prometheus.ExponentialBuckets(100, 10, 8),
	}, []string{"method", "path"})

	// HTTPResponseSize - размер ответа
	HTTPResponseSize = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "http_response_size_bytes",
		Help:    "Size of HTTP response in bytes",
		Buckets: prometheus.ExponentialBuckets(100, 10, 8),
	}, []string{"method", "path"})
)

// Database Metrics
var (
	// DBTransactionDuration - время выполнения транзакций
	DBTransactionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "db_transaction_duration_seconds",
		Help:    "Duration of database transaction in seconds",
		Buckets: prometheus.DefBuckets,
	})

	// DBTransactionTotal - количество транзакций
	DBTransactionTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "db_transaction_total",
		Help: "Total number of database transactions",
	}, []string{"status"})

	// DBQueryDuration - время выполнения запросов
	DBQueryDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "db_query_duration_seconds",
		Help:    "Duration of database query in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	// DBConnectionPoolActive - активные соединения
	DBConnectionPoolActive = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connection_pool_active",
		Help: "Number of active database connections",
	})

	// DBConnectionPoolIdle - idle соединения
	DBConnectionPoolIdle = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "db_connection_pool_idle",
		Help: "Number of idle database connections",
	})
)

// Error Metrics
var (
	// ErrorsTotal - количество ошибок
	ErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "errors_total",
		Help: "Total number of errors",
	}, []string{"error_type", "layer"})

	// DomainErrorsTotal - доменные ошибки
	DomainErrorsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "domain_errors_total",
		Help: "Total number of domain errors",
	}, []string{"error_code"})
)

// Service Layer Metrics
var (
	// ServiceOperationDuration - время операций сервиса
	ServiceOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "service_operation_duration_seconds",
		Help:    "Duration of service operation in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	// RandomReviewerSelectionDuration - время выбора случайного ревьювера
	RandomReviewerSelectionDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "random_reviewer_selection_duration_seconds",
		Help:    "Duration of random reviewer selection in seconds",
		Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1},
	})
)

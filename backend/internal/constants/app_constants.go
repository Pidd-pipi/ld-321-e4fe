package constants

// 应用常量。
const (
	AppName                 = "agridispatch"
	APIVersion              = "v1"
	PageSizeDefault         = 10
	PageSizeMax             = 100
	OverviewCacheKey        = "agridispatch:overview"
	OverviewCacheTTLSeconds = 30
	WSPushIntervalSeconds   = 5
)

// 角色
const (
	RoleAdmin = "admin"
	RoleUser  = "user"
)

// 错误码
const (
	CodeOK              = 0
	CodeBadRequest      = 40000
	CodeUnauthorized    = 40100
	CodeForbidden       = 40300
	CodeNotFound        = 40400
	CodeConflict        = 40900
	CodeInternalError   = 50000
	CodeTooManyRequests = 42900
)

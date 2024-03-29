package service

const (
	ErrInvalidRequest      = 1001
	ErrNotFoundProductId   = 1002
	ErrUserCheckAuthFailed = 1003
	ErrUserServiceBusy     = 1004
	ErrActiveNotStart      = 1005//活动没有开始
	ErrActiveAlreadyEnd    = 1006//活动已经结束
	ErrActiveSaleOut       = 1007//商品已经售罄
	ErrProcessTimeout      = 1008
	ErrClientClosed        = 1009
)

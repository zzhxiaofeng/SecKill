package service

import (
	"sync"
	"time"

	"github.com/garyburd/redigo/redis"
)

const (
	ProductStatusNormal       = 0
	ProductStatusSaleOut      = 1
	ProductStatusForceSaleOut = 2
)

type RedisConf struct {
	RedisAddr        string
	RedisMaxIdle     int
	RedisMaxActive   int
	RedisIdleTimeout int
}

type EtcdConf struct {
	EtcdAddr          string
	Timeout           int
	EtcdSecKeyPrefix  string
	EtcdSecProductKey string
}

type AccessLimitConf struct {
	IPSecAccessLimit   int
	UserSecAccessLimit int
	IPMinAccessLimit   int
	UserMinAccessLimit int
}

type SecSkillConf struct {
	RedisBlackConf       RedisConf
	RedisProxy2LayerConf RedisConf
	RedisLayer2ProxyConf RedisConf

	EtcdConf          EtcdConf
	LogPath           string
	LogLevel          string
	SecProductInfoMap map[int]*SecProductInfoConf
	RWSecProductLock  sync.RWMutex
	CookieSecretKey   string

	ReferWhiteList []string

	ipBlackMap map[string]bool
	idBlackMap map[int]bool

	AccessLimitConf      AccessLimitConf
	blackRedisPool       *redis.Pool
	proxy2LayerRedisPool *redis.Pool
	layer2ProxyRedisPool *redis.Pool

	secLimitMgr *SecLimitMgr

	RWBlackLock                  sync.RWMutex
	WriteProxy2LayerGoroutineNum int
	ReadProxy2LayerGoroutineNum  int

	SecReqChan     chan *SecRequest
	SecReqChanSize int//管道大小

	UserConnMap     map[string]chan *SecResult
	UserConnMapLock sync.Mutex
}

type SecProductInfoConf struct {
	ProductId int
	StartTime int64
	EndTime   int64
	Status    int
	Total     int//商品总量
	Left      int//商品剩余量
}

type SecResult struct {
	ProductId int
	UserId    int
	Code      int
	Token     string
}

type SecRequest struct {
	ProductId     int
	Source        string
	AuthCode      string
	SecTime       string
	Nance         string
	UserId        int
	UserAuthSign  string
	AccessTime    time.Time
	ClientAddr    string
	ClientRefence string
	CloseNotify   <-chan bool `json:"-"`

	ResultChan chan *SecResult `json:"-"`
}

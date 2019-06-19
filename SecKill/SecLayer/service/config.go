package service

import (
	"sync"
	"time"

	etcd_client "go.etcd.io/etcd/clientv3"
	"github.com/garyburd/redigo/redis"
)

var (
	secLayerContext = &SecLayerContext{}
)

type SecProductInfoConf struct {
	ProductId         int
	StartTime         int64
	EndTime           int64
	Status            int
	Total             int
	Left              int
	OnePersonBuyLimit int//一个人只能购买一个
	BuyRate           float64//购买到的概率
	//每秒最多能卖多少个
	SoldMaxLimit int
	//限速控制
	secLimit *SecLimit
}

type RedisConf struct {
	RedisAddr        string
	RedisMaxIdle     int
	RedisMaxActive   int
	RedisIdleTimeout int
	RedisQueueName   string
}

type EtcdConf struct {
	EtcdAddr          string
	Timeout           int
	EtcdSecKeyPrefix  string
	EtcdSecProductKey string
}

type SecLayerConf struct {
	Proxy2LayerRedis RedisConf
	Layer2ProxyRedis RedisConf
	EtcdConfig       EtcdConf
	LogPath          string
	LogLevel         string

	WriteGoroutineNum      int
	ReadGoroutineNum       int
	HandleUserGoroutineNum int
	Read2handleChanSize    int
	Handle2WriteChanSize   int
	MaxRequestWaitTimeout  int

	SendToWriteChanTimeout  int
	SendToHandleChanTimeout int

	SecProductInfoMap map[int]*SecProductInfoConf
	TokenPasswd       string
}

type SecLayerContext struct {
	proxy2LayerRedisPool *redis.Pool
	layer2ProxyRedisPool *redis.Pool
	etcdClient           *etcd_client.Client
	RWSecProductLock     sync.RWMutex

	secLayerConf     *SecLayerConf
	waitGroup        sync.WaitGroup
	Read2HandleChan  chan *SecRequest
	Handle2WriteChan chan *SecResponse

	HistoryMap     map[int]*UserBuyHistory//key是商品id，value是用户购买过的商品数量，默认为0
	HistoryMapLock sync.Mutex

	//商品的计数
	productCountMgr *ProductCountMgr
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
	//CloseNotify   <-chan bool

	//ResultChan chan *SecResult
}

type SecResponse struct {
	ProductId int
	UserId    int
	Token     string//返回的消息，如果为空说明没抢到
	TokenTime int64
	Code      int//根据token是否为空设定相应的code值标示是否抢到
}

package service

import (
	"fmt"
	"time"

	"encoding/json"

	"github.com/astaxie/beego/logs"
	"github.com/garyburd/redigo/redis"

	"crypto/md5"
	"math/rand"
)

func initRedisPool(redisConf RedisConf) (pool *redis.Pool, err error) {
	pool = &redis.Pool{
		MaxIdle:     redisConf.RedisMaxIdle,
		MaxActive:   redisConf.RedisMaxActive,
		IdleTimeout: time.Duration(redisConf.RedisIdleTimeout) * time.Second,
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", redisConf.RedisAddr)
		},
	}

	conn := pool.Get()
	defer conn.Close()

	_, err = conn.Do("ping")
	if err != nil {
		logs.Error("ping redis failed, err:%v", err)
		return
	}
	return
}

func initRedis(conf *SecLayerConf) (err error) {

	secLayerContext.proxy2LayerRedisPool, err = initRedisPool(conf.Proxy2LayerRedis)
	if err != nil {
		logs.Error("init proxy2layer redis pool failed, err:%v", err)
		return
	}

	secLayerContext.layer2ProxyRedisPool, err = initRedisPool(conf.Layer2ProxyRedis)
	if err != nil {
		logs.Error("init layer2proxy redis pool failed, err:%v", err)
		return
	}

	return
}

func RunProcess() (err error) {

	for i := 0; i < secLayerContext.secLayerConf.ReadGoroutineNum; i++ {
		secLayerContext.waitGroup.Add(1)
		go HandleReader()
	}

	for i := 0; i < secLayerContext.secLayerConf.WriteGoroutineNum; i++ {
		secLayerContext.waitGroup.Add(1)
		go HandleWrite()
	}

	for i := 0; i < secLayerContext.secLayerConf.HandleUserGoroutineNum; i++ {//处理线程
		secLayerContext.waitGroup.Add(1)
		go HandleUser()
	}

	logs.Debug("all process goroutine started")
	secLayerContext.waitGroup.Wait()
	logs.Debug("wait all goroutine exited")
	return
}

func HandleReader() {

	logs.Debug("read goroutine running")
	for {
		conn := secLayerContext.proxy2LayerRedisPool.Get()//取链接
		for {//代码优化，取一个链接然后一直用这个链接循环取元素
			ret, err :=conn.Do("blpop", secLayerContext.secLayerConf.Proxy2LayerRedis.RedisQueueName, 0)//从队列出栈一个元素，即去一个元素
			if err != nil {
				logs.Error("pop from queue failed, err:%v", err)
				break
			}

			tmp, ok := ret.([]interface{})//如果可以转换成数组，说明是ok的
			if !ok || len(tmp) != 2{
				logs.Error("pop from queue failed, err:%v", err)
				continue
			}

			data, ok := tmp[1].([]byte)
			if !ok {
				logs.Error("pop from queue failed, err:%v", err)
				continue
			}
			
			logs.Debug("pop from queue, data:%s", string(data))

			var req SecRequest
			err = json.Unmarshal([]byte(data), &req)
			if err != nil {
				logs.Error("unmarshal to secrequest failed, err:%v", err)
				continue
			}

			now := time.Now().Unix()
			if now-req.AccessTime.Unix() >= int64(secLayerContext.secLayerConf.MaxRequestWaitTimeout) {
				logs.Warn("req[%v] is expire", req)
				continue
			}


			timer := time.NewTicker(time.Millisecond * time.Duration(secLayerContext.secLayerConf.SendToHandleChanTimeout))
			select {
			case secLayerContext.Read2HandleChan <- &req:
			case <-timer.C:
				logs.Warn("send to handle chan timeout, req:%v", req)
				break
			}
		}

		conn.Close()
	}
}

func HandleWrite() {
	logs.Debug("handle write running")

	for res := range secLayerContext.Handle2WriteChan {//从管道里取出来发到redis队列里
		err := sendToRedis(res)
		if err != nil {
			logs.Error("send to redis, err:%v, res:%v", err, res)
			continue
		}
	}
}

func sendToRedis(res *SecResponse) (err error) {

	data, err := json.Marshal(res)
	if err != nil {
		logs.Error("marshal failed, err:%v", err)
		return
	}

	conn := secLayerContext.layer2ProxyRedisPool.Get()//从链接池里取一个链接
	_, err = conn.Do("rpush", secLayerContext.secLayerConf.Layer2ProxyRedis.RedisQueueName, string(data))//往redis传递数据
	if err != nil {
		logs.Warn("rpush to redis failed, err:%v", err)
		return
	}

	return
}

func HandleUser() {

	logs.Debug("handle user running")
	for req := range secLayerContext.Read2HandleChan {//从管道取一个请求
		logs.Debug("begin process request:%v", req)
		res, err := HandleSecKill(req)
		if err != nil {
			logs.Warn("process request %v failed, err:%v", err)
			res = &SecResponse{
				Code: ErrServiceBusy,
			}
		}

		timer := time.NewTicker(time.Millisecond * time.Duration(secLayerContext.secLayerConf.SendToWriteChanTimeout))
		select {
		case secLayerContext.Handle2WriteChan <- res:
		case <-timer.C://超过设定的时间
			logs.Warn("send to response chan timeout, res:%v", res)
			break
		}

	}
	return
}

func HandleSecKill(req *SecRequest) (res *SecResponse, err error) {

	secLayerContext.RWSecProductLock.RLock()
	defer secLayerContext.RWSecProductLock.RUnlock()

	res = &SecResponse{}
	res.UserId=req.UserId
	res.ProductId=req.ProductId
	product, ok := secLayerContext.secLayerConf.SecProductInfoMap[req.ProductId]
	if !ok {//如果没有找到商品
		logs.Error("not found product:%v", req.ProductId)
		res.Code = ErrNotFoundProduct
		return
	}

	if product.Status == ProductStatusSoldout {//如果商品售罄
		res.Code = ErrSoldout
		return
	}

	now := time.Now().Unix()
	alreadySoldCount := product.secLimit.Check(now)//返回当前这一秒已经卖了多少商品
	if alreadySoldCount >= product.SoldMaxLimit {//如果这一秒卖的超过设定的最高售货量，就不用在卖了
		res.Code = ErrRetry//让用户重试
		return
	}

	secLayerContext.HistoryMapLock.Lock()
	userHistory, ok := secLayerContext.HistoryMap[req.UserId]
	if !ok {
		userHistory = &UserBuyHistory{//如果用户第一次来，创建用户的记录
			history: make(map[int]int, 16),
		}

		secLayerContext.HistoryMap[req.UserId] = userHistory
	}

	histryCount := userHistory.GetProductBuyCount(req.ProductId)
	secLayerContext.HistoryMapLock.Unlock()

	if histryCount >= product.OnePersonBuyLimit {//如果已经购买的数量大于设定的个人最高购买数量，则不允许继续购买
		res.Code = ErrAlreadyBuy
		return
	}

	curSoldCount := secLayerContext.productCountMgr.Count(req.ProductId)
	if curSoldCount >= product.Total {//如果卖出商品数量大于商品总数量
		res.Code = ErrSoldout
		product.Status = ProductStatusSoldout
		return
	}

	curRate := rand.Float64()//获取当前用户购买的概率
	if curRate > product.BuyRate {
		res.Code = ErrRetry
		return
	}

	userHistory.Add(req.ProductId, 1)//当前用户的商品购买数量加一
	secLayerContext.productCountMgr.Add(req.ProductId, 1)//商品卖出去的总数加一

	//用户id&商品id&当前时间&密钥
	res.Code = ErrSecKillSucc
	tokenData := fmt.Sprintf("userId=%d&productId=%d&timestamp=%d&security=%s",
		req.UserId, req.ProductId, now, secLayerContext.secLayerConf.TokenPasswd)

	res.Token = fmt.Sprintf("%x", md5.Sum([]byte(tokenData)))//用md5加密，不能被破解
	res.TokenTime = now

	return
}

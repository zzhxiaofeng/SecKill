package service

import (
	"time"
	"encoding/json"
	"fmt"

	"github.com/astaxie/beego/logs"
	"github.com/garyburd/redigo/redis"
)

func WriteHandle() {

	for {
		req := <-secKillConf.SecReqChan//取出一个请求然后发到redis队列中去
		conn := secKillConf.proxy2LayerRedisPool.Get()

		data, err := json.Marshal(req)
		if err != nil {
			logs.Error("json.Marshal failed, error:%v req:%v", err, req)
			conn.Close()
			continue
		}

		_, err = conn.Do("LPUSH", "sec_queue", data)
		if err != nil {
			logs.Error("lpush failed, err:%v, req:%v", err, req)
			conn.Close()
			continue
		}

		conn.Close()
	}

}

func ReadHandle() {
	for {

		conn := secKillConf.proxy2LayerRedisPool.Get()

		reply, err := conn.Do("RPOP", "recv_queue")
		data, err := redis.String(reply, err)
		if err==redis.ErrNil{//大坑，注意当redis取不出值的时候就要加上这个判断
			time.Sleep(time.Second)
			conn.Close()
			continue
		}
		if err != nil {
			logs.Error("rpop failed, err:%v", err)
			conn.Close()
			continue
		}

		var result SecResult
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			logs.Error("json.Unmarshal failed, err:%v", err)
			conn.Close()
			continue
		}

		userKey := fmt.Sprintf("%d_%d", result.UserId, result.ProductId)

		secKillConf.UserConnMapLock.Lock()
		resultChan, ok := secKillConf.UserConnMap[userKey]
		secKillConf.UserConnMapLock.Unlock()
		if !ok {
			conn.Close()
			logs.Warn("user not found:%v", userKey)
			continue
		}

		resultChan <- &result
		conn.Close()
	}
}

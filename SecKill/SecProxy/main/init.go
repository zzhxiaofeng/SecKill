package main

import (
	"encoding/json"
	"fmt"
	"go_dev/day14/SecKill/SecProxy/service"
	"time"

	"golang.org/x/net/context"

	"github.com/astaxie/beego/logs"
	etcd_client "go.etcd.io/etcd/clientv3"
	"go.etcd.io/etcd/mvcc/mvccpb"
	"github.com/garyburd/redigo/redis"
)

var (
	redisPool  *redis.Pool
	etcdClient *etcd_client.Client
)

func initEtcd() (err error) {
	cli, err := etcd_client.New(etcd_client.Config{
		Endpoints:   []string{secKillConf.EtcdConf.EtcdAddr},
		DialTimeout: time.Duration(secKillConf.EtcdConf.Timeout) * time.Second,
	})
	if err != nil {
		logs.Error("connect etcd failed, err:", err)
		return
	}

	etcdClient = cli
	return
}

func convertLogLevel(level string) int {

	switch level {
	case "debug":
		return logs.LevelDebug
	case "warn":
		return logs.LevelWarn
	case "info":
		return logs.LevelInfo
	case "trace":
		return logs.LevelTrace
	}

	return logs.LevelDebug
}

func initLogger() (err error) {
	config := make(map[string]interface{})
	config["filename"] = secKillConf.LogPath
	config["level"] = convertLogLevel(secKillConf.LogLevel)

	configStr, err := json.Marshal(config)
	if err != nil {
		fmt.Println("marshal failed, err:", err)
		return
	}

	logs.SetLogger(logs.AdapterFile, string(configStr))
	return
}

func loadSecConf() (err error) {

	resp, err := etcdClient.Get(context.Background(), secKillConf.EtcdConf.EtcdSecProductKey)//根据商品Key获取商品配置
	if err != nil {
		logs.Error("get [%s] from etcd failed, err:%v", secKillConf.EtcdConf.EtcdSecProductKey, err)
		return
	}

	var secProductInfo []service.SecProductInfoConf
	for k, v := range resp.Kvs {//resp是一个kv数组，是etcd配置
		logs.Debug("key[%v] valud[%v]", k, v)
		err = json.Unmarshal(v.Value, &secProductInfo)//将配置取出来放在secProductInfo
		if err != nil {
			logs.Error("Unmarshal sec product info failed, err:%v", err)
			return
		}

		logs.Debug("sec info conf is [%v]", secProductInfo)
	}

	updateSecProductInfo(secProductInfo)
	return
}

func initSec() (err error) {

	err = initLogger()
	if err != nil {
		logs.Error("init logger failed, err:%v", err)
		return
	}
	
	err = initEtcd()
	if err != nil {
		logs.Error("init etcd failed, err:%v", err)
		return
	}

	err = loadSecConf()
	if err != nil {
		logs.Error("load sec conf failed, err:%v", err)
		return
	}

	service.InitService(secKillConf)
	initSecProductWatcher()

	logs.Info("init sec succ")
	return
}

func initSecProductWatcher() {
	go watchSecProductKey(secKillConf.EtcdConf.EtcdSecProductKey)//监控哪个目录下的文件
}

func watchSecProductKey(key string) {

	cli, err := etcd_client.New(etcd_client.Config{
		Endpoints:   []string{"localhost:2379", "localhost:22379", "localhost:32379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		logs.Error("connect etcd failed, err:", err)
		return
	}

	logs.Debug("begin watch key:%s", key)
	for {
		rch := cli.Watch(context.Background(), key)
		var secProductInfo []service.SecProductInfoConf
		var getConfSucc = true

		for wresp := range rch {
			for _, ev := range wresp.Events {
				if ev.Type == mvccpb.DELETE {//如果删除则实现下列操作
					logs.Warn("key[%s] 's config deleted", key)
					continue
				}

				if ev.Type == mvccpb.PUT && string(ev.Kv.Key) == key {//如果更新则实现下列操作
					err = json.Unmarshal(ev.Kv.Value, &secProductInfo)
					if err != nil {
						logs.Error("key [%s], Unmarshal[%s], err:%v ", err)
						getConfSucc = false
						continue
					}
				}
				logs.Debug("get config from etcd, %s %q : %q\n", ev.Type, ev.Kv.Key, ev.Kv.Value)
			}

			if getConfSucc {//如果没问题就更新商品配置
				logs.Debug("get config from etcd succ, %v", secProductInfo)
				updateSecProductInfo(secProductInfo)
			}
		}

	}
}

func updateSecProductInfo(secProductInfo []service.SecProductInfoConf) {

	var tmp map[int]*service.SecProductInfoConf = make(map[int]*service.SecProductInfoConf, 1024)
	for _, v := range secProductInfo {
		produtInfo := v
		tmp[v.ProductId] = &produtInfo
	}

	secKillConf.RWSecProductLock.Lock()
	secKillConf.SecProductInfoMap = tmp
	secKillConf.RWSecProductLock.Unlock()
}

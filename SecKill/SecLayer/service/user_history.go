package service
//用户购买历史，判断用户是否已经购买过一次，如果购买过一次，则不允许再次购买

import (
	"sync"
)

type UserBuyHistory struct {
	history map[int]int//key是商品id，value是用户购买过的商品数量，默认为0
	lock    sync.RWMutex
}

func (p *UserBuyHistory) GetProductBuyCount(productId int) int {//通过商品Id获取购买数量
	p.lock.RLock()
	defer p.lock.RUnlock()

	count, _ := p.history[productId]
	return count
}

func (p *UserBuyHistory) Add(productId, count int) {//如果已经抢到一个商品，就将历史记录加一
	p.lock.Lock()
	defer p.lock.Unlock()

	cur, ok := p.history[productId]
	if !ok {//如果没买过商品，就等于count
		cur = count
	} else {//如果买过商品就加上count
		cur += count
	}

	p.history[productId] = cur
}

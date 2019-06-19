package service
//是否总数超限

import (
	"sync"
)

type ProductCountMgr struct {
	productCount map[int]int
	lock         sync.RWMutex
}

func NewProductCountMgr() (productMgr *ProductCountMgr) {//ProductCountMgr的构造函数
	productMgr = &ProductCountMgr{
		productCount: make(map[int]int, 128),
	}

	return
}

func (p *ProductCountMgr) Count(productId int) (count int) {
	p.lock.RLock()
	defer p.lock.RUnlock()

	count = p.productCount[productId]
	return

}

func (p *ProductCountMgr) Add(productId, count int) {

	p.lock.Lock()
	defer p.lock.Unlock()

	cur, ok := p.productCount[productId]
	if !ok {//如果商品没有计数
		cur = count
	} else {
		cur += count
	}

	p.productCount[productId] = cur
}

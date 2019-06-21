package product

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/logs"
	"fmt"
	"go_dev/day14/SecKill/SecAdmin/model"
)

type ProductController struct {
	beego.Controller
}

func (p *ProductController) ListProduct() {

	productModel := model.NewProductModel()
	productList, err := productModel.GetProductList()//获取所有商品的列表
	if err != nil {
		logs.Warn("get product list failed, err:%v", err)
		return
	}

	p.Data["product_list"] = productList
	p.TplName = "product/list.html"
	p.Layout = "layout/layout.html"
}



func (p *ProductController) CreateProduct() {
	p.TplName = "product/create.html"
	p.Layout = "layout/layout.html"
}

func (p *ProductController) SubmitProduct() {

	productName := p.GetString("product_name")//获取用户提交的product_name
	productTotal, err := p.GetInt("product_total")//获取用户提交的product_total
	
	p.TplName = "product/create.html"
	p.Layout = "layout/layout.html"
	errorMsg := "success"

	defer func(){
		if err != nil {
			p.Data["Error"] = errorMsg
			p.TplName = "product/error.html"//模板文件
			p.Layout = "layout/layout.html"
		}
	}()

	if len(productName) == 0 {
		logs.Warn("invalid product name, err:%v", err)
		errorMsg = fmt.Sprintf("invalid product name, err:%v", err)
		return
	}

	if err != nil {
		logs.Warn("invalid product total, err:%v", err)
		errorMsg = fmt.Sprintf("invalid product total, err:%v", err)
		return
	}

	productStatus, err := p.GetInt("product_status")
	if err != nil {
		logs.Warn("invalid product status, err:%v", err)
		errorMsg = fmt.Sprintf("invalid product status, err:%v", err)
		return 
	}

	productModel := model.NewProductModel()
	product := model.Product{
		ProductName: productName,
		Total: productTotal,
		Status: productStatus,
	}

	err = productModel.CreateProduct(&product)
	if err != nil {
		logs.Warn("create product failed, err:%v", err)
		errorMsg = fmt.Sprintf("create product failed, err:%v", err)
		return
	}
	logs.Debug("product name[%s], product total[%s], product status[%v]", productName, productTotal, productStatus)
}
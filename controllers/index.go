package controllers

import (
	"ShengXian/models"
	"fmt"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
)

type IndexController struct {
	beego.Controller
}

// 网站首页
func (c *IndexController) ShowIndex() {
	user := c.GetSession("user")
	if user == nil {
		c.Data["username"] = ""
	} else {
		c.Data["username"] = user.(models.User).Name
	}
	o := orm.NewOrm()
	//1.获取商品分类
	var goodsType []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsType)
	fmt.Println("goodsType = ", goodsType)
	c.Data["goodsType"] = goodsType
	//2.获取商品轮播图片
	var banners []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").RelatedSel("GoodsSKU").OrderBy("Index").All(&banners)
	c.Data["banners"] = banners
	//3.获取促销商品图片
	var promoteBanners []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&promoteBanners)
	c.Data["promoteBanners"] = promoteBanners
	//4.获取商品分类
	goods := make([]map[string]interface{}, len(goodsType)) //定义一个[]map切片，保存所有商品分类，每一个map代表一个分类
	for index, goodsType := range goodsType{
		temp := make(map[string]interface{})
		temp["type"] = goodsType //map["type"]代表一个商品分类
		goods[index] = temp
	}
	var goodsImage []models.IndexTypeGoodsBanner
	var goodsText []models.IndexTypeGoodsBanner
	for _,temp := range goods{
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU","GoodsType").Filter("GoodsType",temp["type"]).Filter("DisplayType", 1).OrderBy("Index").All(&goodsImage)
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU","GoodsType").Filter("GoodsType",temp["type"]).Filter("DisplayType", 0).OrderBy("Index").All(&goodsText)

		temp["goodsText"] = goodsText //map["goodsText"]代表该一个文字商品分类
		temp["goodsImage"] = goodsImage //map["goodsImage"]代表该一个图片商品分类
	}
	c.Data["goods"] = goods
	//获取购物车商品数量
	cartcount := GetCartCount(&c.Controller)
	//beego.Info("cartcount = ", cartcount)
	c.Data["cartcount"] = cartcount
	//指定视图页面
	c.TplName = "index.html"
}

// 获取当前登录用户
func GetUser(c *beego.Controller) {
	user := c.GetSession("user")
	if user != nil {
		c.Data["username"] = user.(models.User).Name
	} else {
		c.Data["username"] = ""
	}
}

package controllers

import (
	"ShengXian/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"math"
	"strconv"
)

type GoodsController struct {
	beego.Controller
}

// 展示商品详情
func (c *GoodsController) ShowDetail() {
	id := c.GetString("id")
	if id == "" {
		beego.Info("")
		c.Redirect("/", 302)
	}
	//1.获取商品类型
	o := orm.NewOrm()
	//代码优化后，获取商品类型都放入函数ShowGoodsLayout中
	//var goodsTypes []models.GoodsType
	//o.QueryTable("GoodsType").All(&goodsTypes)
	//c.Data["goodsTypes"] = goodsTypes
	//2.获取商品详情
	var goodsSKU models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("Id", id).One(&goodsSKU)
	c.Data["goodsSKU"] = goodsSKU
	//3. 获取新品推荐
	var newGoods []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", goodsSKU.GoodsType.Id).OrderBy("Time").Limit(2,0).All(&newGoods)
	c.Data["newGoods"] = newGoods
	//4.获取当前登录用户
	userObj := c.GetSession("user")
	if userObj != nil {
	//if userObj == nil {
		//c.Data["username"] = ""
	//} else {
		var user = userObj.(models.User)
		//代码优化后，在ShowGoodsLayout函数中获取当前登录用户
		//c.Data["username"] = user.Name
		//添加历史浏览记录
		//查询用户信息
		conn, _ := redis.Dial("tcp","192.168.31.20:6379")
		//插入历史纪录
		reply,err:=conn.Do("lpush", "history" + strconv.Itoa(user.Id), id)
		reply,_ = redis.Bool(reply, err) //把结果转换成Bool类型
		if reply == false{
			beego.Info("插入浏览数据错误")
		}
	}
	//代码优化后，在ShowGoodsLayout函数中设置布局
	//c.Layout = "goodsLayout.html"
	ShowGoodsLayout(&c.Controller)
	//获取购物车商品数量
	cartcount := GetCartCount(&c.Controller)
	//beego.Info("cartcount = ", cartcount)
	c.Data["cartcount"] = cartcount
	c.TplName = "detail.html"
}

//显示商品列表
func (c *GoodsController) ShowGoodsList() {
	//获取类型id
	typeId, err := c.GetInt("typeId")
	if err != nil {
		beego.Info("获取类型ID错误")
		c.Redirect("/",302)
		return
	}
	//1.获取商品类型
	o := orm.NewOrm()
	//代码优化后，获取商品类型都放入函数ShowGoodsLayout中
	//var goodsTypes []models.GoodsType
	//o.QueryTable("GoodsType").All(&goodsTypes)
	//c.Data["goodsTypes"] = goodsTypes
	//2.获取新品
	var newGoods []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).OrderBy("Time").Limit(2,0).All(&newGoods)
	c.Data["newGoods"] = newGoods
	//3.获取当前类型的商品列表
	var goodsSKUs []models.GoodsSKU
	//o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).All(&goodsSKUs)
	//分页处理
	pageSize := 2
	pageIndex,err := c.GetInt("pageIndex")
	if err != nil{
		pageIndex = 1
	}
	c.Data["pageIndex"] = pageIndex
	//计算开始查找位置，从0开始
	start := pageSize * (pageIndex - 1)
	//查找总记录数
	count, _ := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).Count()
	//获取总页码
	pageCount := math.Ceil(float64(count)/ float64(pageSize))
	//计算页码
	pageInfo := PageTool(int(pageCount), pageIndex)
	c.Data["pageInfo"] = pageInfo
	//分页查找商品
	//o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).Limit(pageSize, start).All(&goodsSKUs)
	//加入排序的分页
	sort := c.GetString("sort")
	c.Data["sort"] = sort
	if sort == "price" {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).OrderBy("Price").Limit(pageSize, start).All(&goodsSKUs)
	} else if sort == "sales" {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).OrderBy("Sales").Limit(pageSize, start).All(&goodsSKUs)
	} else {
		o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id",typeId).Limit(pageSize, start).All(&goodsSKUs)
	}
	c.Data["goods"] = goodsSKUs
	//4.获取当前的商品分类
	var goodsType models.GoodsType
	goodsType.Id = typeId
	o.Read(&goodsType)
	c.Data["goodsType"] = goodsType
 	//5.获取当前登录用户
 	//代码优化后，在ShowGoodsLayout函数中获取当前登录用户
	//userObj := c.GetSession("user")
	//	//if userObj == nil {
	//	//	c.Data["username"] = ""
	//	//} else {
	//	//	var user= userObj.(models.User)
	//	//	c.Data["username"] = user.Name
	//	//}
	//上一页页码
	pagePre := pageIndex - 1
	if pageIndex == 1{
		pagePre = 1
	}
	pageNext := pageIndex + 1
	if pageIndex == int(pageCount) {
		pageNext = pageIndex
	}
	c.Data["pagePre"] = pagePre
	c.Data["pageNext"] = pageNext
	//跳转页面
	//c.Layout = "goodsLayout.html"
	ShowGoodsLayout(&c.Controller)
	c.TplName = "list.html"
}

//分页助手函数
func PageTool(pageCount int, pageIndex int) []int {
	const page = 5 //显示页码按钮个数
	var pageIndexBuffer []int

	/*
		计算页码的思路:
			begin：开始显示页码；
			end：最后显示页码；
			1)	情况一：如果总页数小于等于5页；
				begin = 1
				end = 实际页数
			2)	情况二：如果查询结果大于5页，默认情况下是：
				begin = 当前页码 - 2
				end = 当前页码 + 2
				a)	如果begin小于等于0：
					begin = 1
					end = 5
				b)	如果end大于总页数：
					begin = 最后一页 - 4
					end = 最后一页
	*/
	if pageCount <= page {
		pageIndexBuffer = make([]int, pageCount)
		for index, _ := range pageIndexBuffer {
			pageIndexBuffer[index] = index + 1
		}
	} else {
		begin := pageIndex - 2
		end := pageIndex + 2
		if begin <= 0 {
			pageIndexBuffer = []int{1, 2, 3, 4, 5}
		} else if end > pageCount {
			pageIndexBuffer = []int{pageCount - 4, pageCount - 3, pageCount - 2, pageCount - 1, pageCount}
		} else {
			pageIndexBuffer = []int{pageIndex - 2, pageIndex - 1, pageIndex, pageIndex + 1, pageIndex + 2}
		}
	}
	return pageIndexBuffer
}

//显示布局内容
func ShowGoodsLayout(c *beego.Controller){
	//查询类型
	o := orm.NewOrm()
	var types []models.GoodsType
	o.QueryTable("GoodsType").All(&types)
	c.Data["types"] = types
	//获取用户信息
	user := c.GetSession("user")
	if user == nil {
		c.Data["username"] = ""
	} else {
		c.Data["username"] = user.(models.User).Name
	}
	//指定layout
	c.Layout = "goodsLayout.html"
}
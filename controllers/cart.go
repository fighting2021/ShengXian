package controllers

import (
	"ShengXian/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
)

type CartController struct {
	beego.Controller
}

// 添加购物车
func (c *CartController) HandleAddCart() {
	resp := make(map[string]interface{}) //封装响应数据
	defer c.ServeJSON() //返回json格式数据

	skuId, err1 := c.GetInt("skuId")
	goodsCount, err2 := c.GetInt("goodsCount")
	if err1 != nil || err2 != nil {
		resp["res"] = 1 //响应状态
		resp["errmsg"] = "获取数据信息错误" //响应信息
		c.Data["json"] = resp
		return
	}
	//beego.Info("skuId = ", skuId, ", goodsCount = ", goodsCount)
	// 检查商品是否存在
	o := orm.NewOrm()
	var goodsSKU models.GoodsSKU
	goodsSKU.Id = skuId
	err := o.Read(&goodsSKU)
	if err != nil{
		resp["res"] = 2
		resp["errmsg"] = "商品不存在"
		c.Data["json"] = resp
		return
	}
	// 检查商品是否超过了当前库存
	if (goodsCount > goodsSKU.Stock) {
		resp["res"] = 3
		resp["errmsg"] = "商品库存不足"
		c.Data["json"] = resp
		return
	}
	//获取当前登录用户
	user := c.GetSession("user")
	userId := user.(models.User).Id

	// 从redis服务器中获取购物车中商品数量（数据使用hash保存：cart_用户id, skuId, count）
	conn, _ := redis.Dial("tcp","192.168.31.20:6379")
	reply, err := conn.Do("hget","cart_" + strconv.Itoa(userId), skuId)
	preCount, _  := redis.Int(reply, err) //把返回结果转换成int类型
	// 更新用户购物车的商品数量
	conn.Do("hset","cart_" + strconv.Itoa(userId), skuId, goodsCount + preCount)
	//更新完成后返回购物车的商品数量
	resp["res"] = 5
	resp["cartcount"] = GetCartCount(&c.Controller) //购物车商品数量
	c.Data["json"] = resp
}

func GetCartCount(c *beego.Controller) int{
	user := c.GetSession("user")
	if user == nil{
		return 0
	}
	conn,_ := redis.Dial("tcp","192.168.31.20:6379")
	rep,err := conn.Do("hlen","cart_" + strconv.Itoa(user.(models.User).Id))
	cartCount, _ := redis.Int(rep, err)
	return cartCount
}

// 显示我的购物车
func (c *CartController) ShowCart() {
	// 获取当前登录用户，并传递给页面
	GetUser(&c.Controller)
	// 获取当前登录用户
	user := c.GetSession("user")
	// 从Redis中读取当前用户的购物车数据
	conn, _ := redis.Dial("tcp","192.168.31.20:6379")
	defer conn.Close()
	// 以map[string]int的形式获取购物车数据
	// key代表hash的field，value代表hash的value
	reply, _ := redis.IntMap(conn.Do("hgetall","cart_" + strconv.Itoa(user.(models.User).Id)))
	// 定义一个变量，封装购物车数据。该变量类型为[]map[string]interface{}
	var cartGoods = make([]map[string]interface{}, len(reply))

	//循环遍历，获取购物车商品数据
	totalCount := 0  //总件数
	totalPrice := 0 //总价格
	i := 0  //切片cartGoods的索引值，从0开始

	o := orm.NewOrm()
	for goodsSkuId,count := range reply {
		// 定义一个map，保存购物车商品和数量
		temp := make(map[string]interface{})
		// 根据ID获取GoodsSKU对象
		var goodsSku models.GoodsSKU
		id,_ := strconv.Atoi(goodsSkuId)
		goodsSku.Id = id
		o.Read(&goodsSku)
		// 把goodsSku和count保存到map对象中
		temp["goodsSku"] = goodsSku
		temp["count"] = count
		// 把map保存在cartGoods切片中
		cartGoods[i] = temp
		// 计算总件数
		totalCount += count
		// 计算总价格
		totalPrice += goodsSku.Price * count
		// 保存小计
		temp["addPrice"] = goodsSku.Price * count
		i += 1
	}
	// 传递数据给页面
	c.Data["totalCount"] = totalCount
	c.Data["totalPrice"] = totalPrice
	c.Data["goods"] = cartGoods
	//获取购物车商品数量
	cartcount := GetCartCount(&c.Controller)
	c.Data["cartcount"] = cartcount
	c.TplName = "cart.html"
}

// 修改购物车
func (c *CartController) HandleUpdateCart() {
	defer c.ServeJSON()
	resp := make(map[string]interface{})
	// 获取请求参数
	goodsSkuId, err1 := c.GetInt("goodsSkuId")
	count, err2 := c.GetInt("count")
	if err1 != nil || err2 != nil {
		resp["code"] = 1
		resp["errmsg"] = "参数不完整"
		c.Data["json"] = resp
		return
	}

	// 获取用户ID
	user := c.GetSession("user")
	if user == nil {
		resp["code"] = 2
		resp["errmsg"] = "用户未登录"
		c.Data["json"] = resp
		return
	}
	userId := user.(models.User).Id
	// 从redis中获取用户的购物车数据
	conn, err := redis.Dial("tcp","192.168.31.20:6379")
	if err != nil {
		resp["code"] = 3
		resp["errmsg"] = "redis服务器连接失败"
		c.Data["json"] = resp
		return
	}
	defer conn.Close()
	conn.Do("hset", "cart_" + strconv.Itoa(userId), goodsSkuId, count)

	// 修改成功后返回数据
	resp["code"] = 5
	resp["errmsg"] = "ok"
	c.Data["json"] = resp
}

// 删除购物车
func (c *CartController) HandleDelCart() {
	defer c.ServeJSON()
	resp := make(map[string]interface{})
	// 获取请求参数
	goodsSkuId, err := c.GetInt("goodsSkuId")
	if err != nil {
		resp["code"] = 1
		resp["errmsg"] = "参数不完整"
		c.Data["json"] = resp
		return
	}
	// 获取用户ID
	user := c.GetSession("user")
	if user == nil {
		resp["code"] = 2
		resp["errmsg"] = "用户未登录"
		c.Data["json"] = resp
		return
	}
	userId := user.(models.User).Id
	// 从redis中删除购物车商品
	conn, err := redis.Dial("tcp","192.168.31.20:6379")
	if err != nil {
		resp["code"] = 3
		resp["errmsg"] = "redis服务器连接失败"
		c.Data["json"] = resp
		return
	}
	defer conn.Close()
	conn.Do("hdel", "cart_" + strconv.Itoa(userId), goodsSkuId)
	// 删除成功后返回数据
	resp["code"] = 5
	resp["errmsg"] = "ok"
	c.Data["json"] = resp
}

// 获取购物车商品数量
func (c *CartController) HandleGetCart() {
	defer c.ServeJSON()
	resp := make(map[string]interface{})
	// 获取请求参数
	goodsSkuId, err := c.GetInt("goodsSkuId")
	if err != nil {
		resp["code"] = 1
		resp["errmsg"] = "参数不完整"
		c.Data["json"] = resp
		return
	}
	// 获取用户ID
	user := c.GetSession("user")
	if user == nil {
		resp["code"] = 2
		resp["errmsg"] = "用户未登录"
		c.Data["json"] = resp
		return
	}
	userId := user.(models.User).Id
	// 从redis中删除购物车商品
	conn, err := redis.Dial("tcp","192.168.31.20:6379")
	if err != nil {
		resp["code"] = 3
		resp["errmsg"] = "redis服务器连接失败"
		c.Data["json"] = resp
		return
	}
	defer conn.Close()
	count, err := redis.Int(conn.Do("hget", "cart_" + strconv.Itoa(userId), goodsSkuId))
	// 删除成功后返回数据
	resp["code"] = 5
	resp["errmsg"] = "ok"
	resp["count"] = count
	c.Data["json"] = resp
}
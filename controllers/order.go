package controllers

import (
	"ShengXian/models"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"time"
	"github.com/smartwalle/alipay"
)

type OrderController struct {
	beego.Controller
}

// 显示订单
func (c *OrderController) ShowOrder() {
	// 获取当前登录用户，并传递给页面
	GetUser(&c.Controller)
	// 获取用户id
	user := c.GetSession("user")
	userId := user.(models.User).Id
	// 获取参数
	skuids := c.GetStrings("skuid")
	if len(skuids) == 0 {
		beego.Info("参数不完整")
		c.Redirect("/user/cart", 302)
		return
	}
	// 获取购物城商品和数量
	goodsMap := make([]map[string]interface{}, len(skuids)) //保存购物车商品和数量
	conn, err := redis.Dial("tcp", "192.168.31.20:6379")
	if err != nil {
		beego.Info("Redis连接失败")
		c.Redirect("/user/cart", 302)
		return
	}
	o := orm.NewOrm()
	totalCount := 0 // 总件数
	totalPrice := 0 // 总金额
	transferPrice := 10 // 运费，默认为10
	for index, skuid := range skuids {
		temp := make(map[string]interface{})
		// 根据skuid获取goodsSku对象
		var goodsSku models.GoodsSKU
		id, _ := strconv.Atoi(skuid) // 把string转换成int类型
		goodsSku.Id = id
		o.Read(&goodsSku)
		temp["goods"] = goodsSku

		// 获取购物车中该商品的数量
		count, _ := redis.Int(conn.Do("hget", "cart_" + strconv.Itoa(userId), skuid))
		temp["count"] = count
		totalCount += count // 计算总件数
		// 计算小计
		total := goodsSku.Price * count
		temp["total"] = total
		// 计算总金额
		totalPrice += total
		// 保存数据到goodsMap中
		goodsMap[index] = temp
	}
	c.Data["goodsMap"] = goodsMap

	// 获取默认收获地址
	var addr models.Address
	o.QueryTable("Address").Filter("User__Id", userId).Filter("Isdefault", 1).One(&addr)
	c.Data["addr"] = addr

	// 总金额、运费、实付款
	c.Data["totalCount"] = totalCount
	c.Data["totalPrice"] = totalPrice
	c.Data["transferPrice"] = transferPrice
	c.Data["realPrice"] = totalPrice + transferPrice
	c.TplName = "place_order.html"
}

// 添加订单
func (c *OrderController) HandleAddOrder() {
	resp := make(map[string]interface{})
	defer c.ServeJSON()

	addrId, err1 := c.GetInt("addrId") // 邮寄地址
	payStyle, err2 := c.GetInt("pay_style") // 支付方式
	skuids := c.GetStrings("skuid") // 订单商品ID
	totalCount, err3 := c.GetInt("totalCount") // 总件数
	totalPrice, err4 := c.GetInt("totalPrice") // 总金额
	transferMoney, err5 := c.GetInt("transferMoney") // 运费
	// 判断请求参数是否完整
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil  || err5 != nil {
		resp["code"] = 1
		resp["errmsg"] = "参数不完整"
		c.Data["json"] = resp
		return
	}
	if len(skuids) == 0 {
		resp["code"] = 2
		resp["errmsg"] = "订单数据为空"
		c.Data["json"] = resp
		return
	}
	o := orm.NewOrm()
	o.Begin() // 开启事务
	// 获取当前用户
	user := c.GetSession("user").(models.User)
	// 添加订单OrderInfo
	var orderInfo models.OrderInfo
	orderInfo.User = &user // 订单所属用户
	orderInfo.OrderId = time.Now().Format("20060102150405") + strconv.Itoa(user.Id) // 订单号（日期时间 + 用户ID）
	var addr models.Address
	addr.Id = addrId
	o.Read(&addr)
	orderInfo.Address = &addr // 邮寄地址
	orderInfo.PayMethod = payStyle // 支付方式
	orderInfo.TotalCount = totalCount // 总件数
	orderInfo.TotalPrice = totalPrice // 总金额
	orderInfo.TransitPrice = transferMoney // 运费
	orderInfo.Orderstatus = 1 // 订单状态
	o.Insert(&orderInfo)

	// 连接Redis服务器
	conn, err := redis.Dial("tcp", "192.168.31.20:6379")
	if err != nil {
		resp["code"] = 3
		resp["errmsg"] = "Redis服务器连接失败"
		c.Data["json"] = resp
		o.Rollback() // 回滚事务
		return
	}

	// 添加订单商品OrderGoods
	for _, skuid := range skuids {
		// 获取订单商品GoodsSKU对象
		var goodsSku models.GoodsSKU
		id, _ := strconv.Atoi(skuid)
		goodsSku.Id = id

		i := 3 // 如果更新库存失败，需要重新读取商品库存。i代表循环次数
		for i > 0 {
			o.Read(&goodsSku)

			// 添加订单商品
			var orderGoods models.OrderGoods
			orderGoods.OrderInfo = &orderInfo // 所属订单
			orderGoods.GoodsSKU = &goodsSku   // 商品
			// 从redis服务器中获取购车商品数量
			count, _ := redis.Int(conn.Do("hget", "cart_"+strconv.Itoa(user.Id), skuid))

			// 判断库存是否足够
			if count > goodsSku.Stock {
				resp["code"] = 4
				resp["errmsg"] = goodsSku.Name + "库存不足"
				c.Data["json"] = resp
				o.Rollback() // 回滚事务
				return
			}

			preStock := goodsSku.Stock // 记录商品库存

			orderGoods.Count = count          // 购买数量
			orderGoods.Price = goodsSku.Price // 商品价格
			o.Insert(&orderGoods)

			// 修改商品库存和销量
			goodsSku.Stock -= count
			goodsSku.Sales += count
			//o.Update(&goodsSku)

			// 更新商品库存和销量时候处理并发
			updateCount, _ := o.QueryTable("GoodsSKU").Filter("Id", goodsSku.Id).Filter("Stock", preStock).Update(orm.Params{"Stock": goodsSku.Stock, "Sales": goodsSku.Sales})
			// 如果updateCount为0，代表库存已经发生改变。否则代表更新成功
			if updateCount == 0 {
				if i > 0 {
					i -= 1
					continue
				}
				resp["code"] = 5
				resp["errmsg"] = "商品库存改变,订单提交失败"
				c.Data["json"] = resp
				o.Rollback()
				return
			} else {
				// 清空用户购物车的商品数据
				conn.Do("hdel", "cart_" + strconv.Itoa(user.Id), skuid)
				break
			}
		}
	}
	o.Commit() // 提交事务
	// 如果添加成功，返回code=5
	resp["code"] = 5
	resp["errmsg"] = "ok"
	c.Data["json"] = resp
}

// 支付订单
func (c *OrderController) HandlePay() {
	var aliPublicKey = "MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA1LdUQ5LAUKvowjP51nPvkFBsqQ3R8ZTfUM/P25SFnc3rLxaWW9gqLatHtlA/kX3HxS9RAJkUpcjc5NM7hl6CIDtpcfBmwsr/Sh5ng9neNJqIuCclxe4GRuBwEjg4qCQClXrp17MxyXPDjqVggb2htJtinnicvvkQLmHGbFvf2AK4kEx5NQPAPdZv8Wi3t1yoFD+v6aE4RSz/Q7OglfLNgvOYKCPC9bFJRXDlSAOcUYmGkZSFwXnCISGgDEOdfjDHysI3xU+flHvoSGIcmPm2dqxDEgBXT79W1oKkDzp8KuFnA3Vd/5AdBwXOlnxem97UdIuWkNm0Vd14wE8G15K44wIDAQAB" // 可选，支付宝提供给我们用于签名验证的公钥，通过支付宝管理后台获取
	var privateKey = "MIIEpAIBAAKCAQEA1LdUQ5LAUKvowjP51nPvkFBsqQ3R8ZTfUM/P25SFnc3rLxaWW9gqLatHtlA/kX3HxS9RAJkUpcjc5NM7hl6CIDtpcfBmwsr/Sh5ng9neNJqIuCclxe4GRuBwEjg4qCQClXrp17MxyXPDjqVggb2htJtinnicvvkQLmHGbFvf2AK4kEx5NQPAPdZv8Wi3t1yoFD+v6aE4RSz/Q7OglfLNgvOYKCPC9bFJRXDlSAOcUYmGkZSFwXnCISGgDEOdfjDHysI3xU+flHvoSGIcmPm2dqxDEgBXT79W1oKkDzp8KuFnA3Vd/5AdBwXOlnxem97UdIuWkNm0Vd14wE8G15K44wIDAQABAoIBAQCHGckTAenTUtwKPCi54/iLmAjrdjOZVAxhrxs9Qx96EocE6TumKazgRKDPUjiNl22B94Ni9db/VIu3adGsjennvtRB4YXiwjtSP+1O+NkAYAXlsDd1dq/V5EZJzBtv7y8U8XQD43QDltrlhnO880v5AZepPsGPKXD1hHQZ7mBFOIyPaL04vNGBZgB4WUkm3W0jj8OCRlgiJ4oDnfpzGBo5Hiy7SGRpKBFPZrrqQ6D/rRqhihxTLxEANlOwDAbtJVyEXGc6UI/5WpjjG4ytzeoRnPtYUBk38LdVRnRYSjXJYWUU82H43qOFEIog/WNl7uzJu8aDVx+8rCrlzSMnKpuBAoGBAPIuMfqLFfD5cPcE4XE38FKMNVIY1jgDzUWmzmIaAZxEZ6QGGx/br9ZUQ8v8brxEy/4NjmqzgQWs6SdVf8Pb72pcHtPiariAQJJX6BPO+voy8CwNrfLJ/ycFmF/6rdpXGRMIrwq3hDQ92FqIYdqq9JyYtNCLY3Xc2lxGLMir9hBJAoGBAODatxPWGaKaNVdHl1r3Cvy3DD3rf5uuwLn8sUxaRaPg3OegZzS7KkWUrOmQFIkKJPFGo6ozcWdjM2E0hzCdzP25E2blQdT19N01+O24KvgY+9NhV6Xc7OqNLVGeX8V6F3R2nT951ehFY3uGQj5yZruRyTEwh0xfph8SnJpyk1fLAoGAdK4LCFHwfUxAv9KLJ3gzAiJpIiezNgOm55LpRlyPQBG4+U6zzNKUUigBDguN8okW20z+u6vGUsyu/HN1/hA9tjmx5JXmowOvmJJfzwNe6iKWkjL5BsuJceyBMGTfVB24h/IcG4a1XFIbpeqlcqwA9F2iaANwJP4F+oUY2In5IHkCgYAJcgaYxbij9PhipzY7cv86KHJSM49Tud6MsYm9HFmqjaqZ7EoJlActjxZIZW4HZ66vl/kOEEUmQ6uH3M4FA8v1qI2hF+ZRDNfxZvADBGaBr4L8mS02YXZuT+nkcNOzFvLfSJBId1S+AhQwvy4PM30PSgt6joBQfAAddAmyDzgTSwKBgQDLZM4/9Av0nqEGGtgu6er2gBUo4zLccf+D/OGrXt6OIrGPKFWzqJfJZmvS6hFcZHRUTCBNHe9tCLU4YiGx4HINK12MB8tsoGNAstxKOeyApuAxadvymaRZtWYcOvKUi9skoVPGHxNIJpRWhVvCiZ8uSgMEi8PYZl0heAZMDA0hbg==" // 必须，上一步中使用RSA签名验签工具 生成的私钥

	var appId = "2016092700611686"
	var client = alipay.New(appId, aliPublicKey, privateKey, false)

	//获取数据
	orderId := c.GetString("orderId")
	totalPrice := c.GetString("totalPrice")

	// 创建AliPayTradePagePay对象，该对象封装了提交给支付包的信息
	var p = alipay.AliPayTradePagePay{}
	p.NotifyURL = "" // 异步回调URL，只有当支付页面关闭后，支付宝才会请求该URL
	p.ReturnURL = "http://localhost:8080/user/payok" // 同步回调URL
	p.Subject = "天天生鲜购物平台"
	p.OutTradeNo = orderId
	p.TotalAmount = totalPrice
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"
	// 提交支付
	var url, err = client.TradePagePay(p)
	if err != nil {
		beego.Info(err)
	}
	// 获取回调URL
	var payURL = url.String()
	// 跳转到URL页面
	c.Redirect(payURL,302)
}

// 支付成功
func(c *OrderController) PayOk(){
	//获取数据
	orderId := c.GetString("out_trade_no")
	//校验数据
	if orderId == "" {
		beego.Info("支付返回数据错误")
		c.Redirect("/user/userorder", 302)
		return
	}
	// 修改订单状态
	o := orm.NewOrm()
	count,_:=o.QueryTable("OrderInfo").Filter("OrderId",orderId).Update(orm.Params{"Orderstatus":2})
	if count == 0 {
		beego.Info("更新数据失败")
	}
	//返回视图
	c.Redirect("/user/userorder",302)
}
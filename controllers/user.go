package controllers

import (
	"ShengXian/models"
	"encoding/base64"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/utils"
	"github.com/gomodule/redigo/redis"
	"math"
	"regexp"
	"strconv"
)

type UserController struct {
	beego.Controller
}

// 显示注册页面
func (c *UserController) ShowReg() {
	c.TplName = "register.html"
}

// 处理用户注册
func (c *UserController) HandleRegist() {
	// 获取请求参数
	name := c.GetString("user_name")
	pass := c.GetString("pwd")
	cpwd := c.GetString("cpwd")
	email := c.GetString("email")

	// 验证参数
	if name == "" || pass == "" || cpwd == "" || email == "" {
		c.Data["errMsg"] = "填写信息不完整，请重新注册"
		c.TplName = "register.html"
		return
	}
	if pass != cpwd {
		c.Data["errMsg"] = "两次输入密码不相同"
		c.TplName = "register.html"
		return
	}
	reg, _ := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
	res := reg.FindString(email)
	if res == "" {
		c.Data["errMsg"] = "邮箱格式不正确"
		c.TplName = "register.html"
		return
	}
	// 如果验证成功，执行保存操作
	var user models.User
	user.Name = name
	user.PassWord = pass
	user.Email = email
	o := orm.NewOrm()
	_, err := o.Insert(&user)
	if err != nil {
		c.Data["errMsg"] = "注册失败，请联系管理员"
		c.TplName = "register.html"
		return
	}
	// 发送邮件
	var config = `{"username":"896337156@qq.com","password":"bdxgebwcmympbajb","host":"smtp.qq.com","port":587}`
	e := utils.NewEMail(config) // 创建email对象
	e.From = "896337156@qq.com" // 发送来源
	e.To = []string{email}      // 接收者，可以有多个
	e.Subject = "天天生鲜用户注册"      // 邮件标题
	// 注意：IP地址为服务器的真实地址，本地为127.0.0.1
	e.Text = "127.0.0.1:8080/active?id=" + strconv.Itoa(user.Id) // 邮件正文
	e.Send()                                                     //发送邮件

	c.Ctx.WriteString("注册成功，请登录邮箱执行激活操作")
}

// 激活用户
func (c *UserController) ActiveUser() {
	// 获取请求参数ID
	id, _ := c.GetInt("id")
	// 根据ID查询用户
	o := orm.NewOrm()
	var user models.User
	user.Id = id
	err := o.Read(&user)
	// 判断用户是否存在。如果存在就把该用户的激活状态修改为true
	if err != nil {
		c.Data["errMsg"]= "用户不存在"
		c.TplName = "register.html"
		return
	}
	user.Active = true
	o.Update(&user)
	c.Ctx.WriteString(`<a href="/login">激活成功，点击跳转到登录页面</a>`)
}

// 显示登录页面
func (c *UserController) ShowLogin() {
	username := c.Ctx.GetCookie("username")
	data, _ := base64.StdEncoding.DecodeString(username)
	//fmt.Println("cookie's username is ", string(data))
	if string(data) == "" {
		c.Data["username"] = ""
		c.Data["checked"] = ""
	} else {
		c.Data["username"] = string(data)
		c.Data["checked"] = "checked"
	}
	c.TplName = "login.html"
}

// 处理用户登录
func (c *UserController) HandleLogin() {
	name := c.GetString("username")
	pass := c.GetString("pwd")
	if name == "" || pass == "" {
		c.Data["errMsg"] = "登录信息不完整"
		c.TplName = "login.html"
		return
	}
	var user models.User
	user.Name = name
	o := orm.NewOrm()
	err := o.Read(&user, "Name")
	if err != nil {
		c.Data["errMsg"] = "用户不存在"
		c.TplName = "login.html"
		return
	}
	if user.PassWord != pass {
		c.Data["errMsg"] = "密码不正确"
		c.TplName = "login.html"
		return
	}
	if !user.Active {
		c.Data["errMsg"] = "用户未激活"
		c.TplName = "login.html"
		return
	}
	// 记住用户名
	rem := c.GetString("rem")
	if rem == "on" {
		name = base64.StdEncoding.EncodeToString([]byte(name)) // 因为cookie不支持中文，所以使用Base64对内容进行加密
		c.Ctx.SetCookie("username", name, 7 * 24 * 60 * 60)
	} else {
		c.Ctx.SetCookie("username", "", 0)
	}
	// 把用户信息保存在Session中
	c.SetSession("user", user)
	//c.Ctx.WriteString("用户登录成功")
	c.Redirect("/", 302)
}

// 显示个人信息页面
func (c *UserController) ShowUserInfo() {
	// 从session获取当前登录用户
	user := c.GetSession("user")
	// 获取默认收获地址
	// 获取默认地址
	var addr models.Address
	o := orm.NewOrm()
	o.QueryTable("Address").Filter("Isdefault", true).One(&addr)
	// 把用户传递给页面
	c.Data["user"] = user
	c.Data["addr"] = addr
	c.Data["type"] = 1 // 1代表个人信息 2代表全部订单 3代表收获地址
	//获取用户浏览记录
	var goods []models.GoodsSKU
	conn,_ :=redis.Dial("tcp","192.168.31.20:6379")
	reply,err := conn.Do("lrange", "history"+strconv.Itoa(user.(models.User).Id), 0, 4)
	replyInts,_ := redis.Ints(reply, err) //把结果转换成[]Int类型
	for _,idVal := range replyInts{
		var temp models.GoodsSKU
		o.QueryTable("GoodsSKU").Filter("Id", idVal).One(&temp)
		goods = append(goods, temp)
	}
	c.Data["goods"] = goods

	c.Layout = "layout.html"
	c.TplName = "user_center_info.html"
}

// 注销
func (c *UserController) HandleLogout() {
	c.DelSession("user")
	c.Redirect("/", 302)
}

// 用户中心-收获地址
func (c *UserController) ShowUserSite() {
	// 从session获取当前登录用户
	user := c.GetSession("user")
	// 获取所有收获地址，按照isdefault降序排列
	o := orm.NewOrm()
	var addrs []models.Address
	o.QueryTable("Address").OrderBy("-isdefault").All(&addrs)
	c.Data["user"] = user
	c.Data["addrs"] = addrs
	c.Data["type"] = 3 // 1代表个人信息 2代表全部订单 3代表收获地址
	c.Layout = "layout.html"
	c.TplName = "user_center_site.html"
}

// 用户中心-添加收获地址
func (c *UserController) AddUserSite() {
	// 获取表单参数
	receiver := c.GetString("receiver")
	addr := c.GetString("addr")
	zipcode := c.GetString("zipcode")
	phone := c.GetString("phone")
	if receiver == "" || addr == "" || zipcode == "" || phone == "" {
		c.Data["errMsg"] = "输入数据不完整"
		c.ShowUserSite()
		return
	}
	// 获取所有收获地址，按照isdefault降序排列
	var address models.Address
	address.Addr = addr
	address.Receiver = receiver
	address.Zipcode = zipcode
	address.Phone = phone
	// 从session获取当前登录用户
	user := c.GetSession("user").(models.User)
	address.User = &user
	o := orm.NewOrm()
	_, err := o.Insert(&address)
	if err != nil {
		c.Data["errMsg"] = "添加失败"
		c.ShowUserSite()
	} else {
		c.Redirect("/user/usersite", 302)
	}
}

// 设置默认地址
func (c *UserController) UpdateUserSite() {
	// 获取默认地址
	var addr models.Address
	o := orm.NewOrm()
	o.QueryTable("Address").Filter("Isdefault", true).One(&addr)
	// 把默认地址修改为false
	addr.Isdefault = false
	o.Update(&addr)
	// 设置默认地址
	id, _ := c.GetInt("id")
	o.QueryTable("Address").Filter("Id", id).One(&addr)
	addr.Isdefault = true
	o.Update(&addr)
	// 修改完成后跳转会地址页面
	c.Redirect("/user/usersite", 302)
}

// 删除地址
func (c *UserController) DelUserSite() {
	var addr models.Address
	id, _ := c.GetInt("id")
	addr.Id = id
	o := orm.NewOrm()
	o.Delete(&addr)
	c.Redirect("/user/usersite", 302)
}

// 显示所有订单
func (c *UserController) ShowUserOrder() {
	user := c.GetSession("user")
	userId := user.(models.User).Id

	//分页处理
	pageSize := 2
	pageIndex,err := c.GetInt("pageIndex")
	if err != nil{
		pageIndex = 1
	}
	c.Data["pageIndex"] = pageIndex
	//计算开始查找位置，从0开始
	start := pageSize * (pageIndex - 1)
	//查找用户订单数量
	o := orm.NewOrm()
	count, _ := o.QueryTable("OrderInfo").Filter("User__Id", userId).Count()
	//获取总页码
	pageCount := math.Ceil(float64(count)/ float64(pageSize))
	//计算页码
	pageInfo := PageTool(int(pageCount), pageIndex)
	c.Data["pageInfo"] = pageInfo
	// 分页查询用户所有订单
	var orderInfos []models.OrderInfo
	o.QueryTable("OrderInfo").Filter("User__Id", userId).OrderBy("-Time").Limit(pageSize, start).All(&orderInfos)

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

	// 遍历所有订单，把订单和订单商品数据封装到goodsOrders切片中
	var goodsOrders = make([]map[string]interface{}, len(orderInfos))
	for index, orderInfo := range orderInfos {
		// 查询所有订单商品
		var orderGoods []models.OrderGoods
		o.QueryTable("OrderGoods").RelatedSel("OrderInfo","GoodsSKU").Filter("OrderInfo__Id",orderInfo.Id).All(&orderGoods)
		// 把订单和订单商品数据封装到map中
		temp := make(map[string]interface{})
		temp["orderInfo"] = orderInfo
		temp["orderGoods"] = orderGoods
		// 把map添加到goodsOrders切片中
		goodsOrders[index] = temp
	}
	c.Data["goodsOrders"] = goodsOrders
	c.Data["user"] = user
	c.Data["type"] = 2 // 1代表个人信息 2代表全部订单 3代表收获地址
	c.Layout = "layout.html"
	c.TplName = "user_center_order.html"
}

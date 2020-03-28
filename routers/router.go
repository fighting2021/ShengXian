package routers

import (
	"github.com/astaxie/beego/context"
	"github.com/astaxie/beego"
	"ShengXian/controllers"
)

func init() {
	//添加过滤器
	beego.InsertFilter("/user/*", beego.BeforeExec, loginFilter)
	beego.Router("/", &controllers.IndexController{}, "get:ShowIndex")
	beego.Router("/regist", &controllers.UserController{}, "get:ShowReg;post:HandleRegist")
	beego.Router("/active", &controllers.UserController{}, "get:ActiveUser")
	beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	beego.Router("/goodsDetail", &controllers.GoodsController{}, "get:ShowDetail")
	beego.Router("/goodsList", &controllers.GoodsController{}, "get:ShowGoodsList")
	beego.Router("/user/logout", &controllers.UserController{}, "get:HandleLogout")
	beego.Router("/user/userinfo", &controllers.UserController{}, "get:ShowUserInfo")
	beego.Router("/user/logout", &controllers.UserController{}, "get:HandleLogout")
	beego.Router("/user/usersite", &controllers.UserController{}, "get:ShowUserSite;post:AddUserSite")
	beego.Router("/user/updateUserSite", &controllers.UserController{}, "get:UpdateUserSite")
	beego.Router("/user/delUserSite", &controllers.UserController{}, "get:DelUserSite")
	beego.Router("/user/userorder", &controllers.UserController{}, "get:ShowUserOrder")
	beego.Router("/user/addCart", &controllers.CartController{}, "post:HandleAddCart")
	beego.Router("/user/cart", &controllers.CartController{}, "get:ShowCart")
	beego.Router("/user/updateCart", &controllers.CartController{}, "post:HandleUpdateCart")
	beego.Router("/user/delCart", &controllers.CartController{}, "post:HandleDelCart")
	beego.Router("/user/getCart", &controllers.CartController{}, "post:HandleGetCart")
	beego.Router("/user/showOrder", &controllers.OrderController{}, "post:ShowOrder")
	beego.Router("/user/addOrder", &controllers.OrderController{}, "post:HandleAddOrder")
	beego.Router("/user/pay", &controllers.OrderController{}, "get:HandlePay")
	beego.Router("/user/payok", &controllers.OrderController{}, "get:PayOk")
}

//登录过滤器
var loginFilter = func(ctx *context.Context) {
	user := ctx.Input.Session("user")
	if user == nil {
		ctx.Redirect(302, "/login")
	}
}

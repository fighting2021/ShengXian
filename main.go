package main

import (
	_ "ShengXian/routers"
	"github.com/astaxie/beego"
	_ "ShengXian/models"
)

func main() {
	beego.Run()
}


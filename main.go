package main

import (
	"github.com/ecodeclub/ginx/session"
	"github.com/ecodeclub/webook/ioc"
	"github.com/gotomicro/ego"
)

// export EGO_DEBUG=true
// 记得修改为你的配置文件
// go run main.go --config=config/config.yaml
func main() {
	// 先触发初始化
	egoApp := ego.New()
	app := ioc.InitApp()
	// 初始化 Session 机制
	session.SetDefaultProvider(app.Sp)
	err := egoApp.
		// Invoker 在 Ego 里面，应该叫做初始化函数
		Invoker().
		Serve(app.Web).
		Run()
	panic(err)
}

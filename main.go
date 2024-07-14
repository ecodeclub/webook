package main

import (
	"github.com/ecodeclub/webook/ioc"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/server/egin"
	"github.com/gotomicro/ego/server/egovernor"
)

// export EGO_DEBUG=true
// 记得修改为你的配置文件
// go run main.go --config=config/config.yaml
func main() {
	// 先触发初始化
	egoApp := ego.New()
	app, err := ioc.InitApp()
	if err != nil {
		panic(err)
	}
	err = egoApp.
		// Invoker 在 Ego 里面，应该叫做初始化函数
		Invoker().
		Serve(
			egovernor.Load("server.governor").Build(),
			app.Web,
			(*egin.Component)(app.Admin)).
		Job(app.Jobs...).
		Cron(app.Crons...).
		Run()
	panic(err)
}

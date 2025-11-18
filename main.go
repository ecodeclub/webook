package main

import (
	"context"

	"go.opentelemetry.io/otel/sdk/trace"

	"log/slog"

	"github.com/ecodeclub/webook/ioc"
	"github.com/gotomicro/ego"
	"github.com/gotomicro/ego/core/elog"
	"github.com/gotomicro/ego/server/egin"
	"github.com/gotomicro/ego/server/egovernor"
	slogzap "github.com/samber/slog-zap/v2"
)

// export EGO_DEBUG=true
// 记得修改为你的配置文件
// go run main.go --config=config/config.yaml
func main() {
	// 先触发初始化
	egoApp := ego.New()
	// 初始化
	tp := ioc.InitZipkinTracer()
	defer func(tp *trace.TracerProvider) {
		err := tp.Shutdown(context.Background())
		if err != nil {
			elog.Error("Shutdown zipkinTracer", elog.FieldErr(err))
		}
	}(tp)
	app, err := ioc.InitApp()
	if err != nil {
		panic(err)
	}

	initSlog()

	// 启动消费者
	for i := range app.Consumers {
		app.Consumers[i].Start(context.Background())
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
	if err != nil {
		elog.DefaultLogger.Error("App运行错误", elog.FieldErr(err))
	}
}

func initSlog() {
	zl := elog.DefaultLogger.ZapLogger()
  level := slog.LevelInfo
  if elog.DefaultLogger.IsDebugMode() {
    level = slog.LevelDebug
  }
	h := slogzap.Option{
		Level:    level , // slog 的阈值
		AddSource: true,           // 映射到 zap 的 caller
		Logger:    zl,             // 复用同一个 zap.Logger
	}

	logger := slog.New(h.NewZapHandler())
	slog.SetDefault(logger)
}

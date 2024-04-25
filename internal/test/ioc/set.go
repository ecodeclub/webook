package testioc

import "github.com/google/wire"

var BaseSet = wire.NewSet(InitDB, InitCache, InitMQ, InitES)

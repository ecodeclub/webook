package domain

import "time"

type User struct {
	Id         int64
	Email      string
	Password   string
	CreateTime time.Time
	UpdateTime time.Time
}

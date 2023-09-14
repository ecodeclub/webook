package encryption

import "time"

type Handle interface {
	Encryption(map[string]string, string, time.Duration) (encryptString string, err error)
	Decrypt(tokenStr string, secretCode string) (interface{}, error)
}

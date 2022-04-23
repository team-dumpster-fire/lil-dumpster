package state

type Backend interface {
	Set(key string, value interface{}) (err error)
	Get(key string, value interface{}) error
}

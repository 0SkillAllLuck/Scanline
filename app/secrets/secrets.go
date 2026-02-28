package secrets

import (
	"errors"

	"github.com/0skillallluck/scanline/internal/g"
)

var (
	ErrKeyNotFound = errors.New("key not found")
)

var getService = g.Lazy(func() Service {
	return newService()
})

type Service interface {
	Available() *ServiceError
	Delete(key string) error
	Get(key string) (Item, error)
	Has(key string) (bool, error)
	Set(key string, value Item) error
}

type Item struct {
	Label    string
	Password string
}

type ServiceError struct {
	Title string
	Body  string
	Fatal bool
}

func Healthcheck() *ServiceError {
	return getService().Available()
}

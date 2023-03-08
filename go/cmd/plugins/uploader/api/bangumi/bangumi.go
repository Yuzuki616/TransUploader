package bangumi

import (
	"errors"
	"github.com/go-resty/resty/v2"
)

var NotFoundError = errors.New("not found")

type Bangumi struct {
	client *resty.Client
}

func New() *Bangumi {
	return &Bangumi{
		client: resty.New(),
	}
}

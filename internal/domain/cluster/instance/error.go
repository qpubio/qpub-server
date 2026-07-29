package instance

import (
	"gorm.io/gorm"
)

var (
	ErrNotFound = gorm.ErrRecordNotFound
)

func IsNotFoundError(err error) bool {
	return err == ErrNotFound
}

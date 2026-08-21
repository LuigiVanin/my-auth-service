package interfaces

import "gorm.io/gorm"

type Page struct {
	Skip  int
	Limit int
}

type RepositoryOptions[T any] struct {
	Tx   *gorm.DB
	With []string

	Filter Page
}

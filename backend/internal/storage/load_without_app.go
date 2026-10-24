//go:build !testhelpers

package storage

import "gorm.io/gorm"

func loadWithoutApp() *gorm.DB {
	panic("storage.GetDb is called before app.New installed the database")
}

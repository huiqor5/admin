package seo

import (
	"os"
	"strings"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type TestQorSEOSetting struct {
	QorSEOSetting
}

func init() {
	if testdb, err := gorm.Open(postgres.Open(os.Getenv("DB_PARAMS")), &gorm.Config{}); err != nil {
		panic(err)
	} else {
		GlobalDB = testdb
	}
	GlobalDB.AutoMigrate(&TestQorSEOSetting{})
}

type Product struct {
	Name string
	SEO  Setting
}

func resetDB() {
	GlobalDB.Exec("truncate test_qor_seo_settings;")
}

func metaEqual(got, want string) bool {
	for _, s := range strings.Split(want, "\n") {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.Contains(got, s) {
			return false
		}
	}
	return true
}

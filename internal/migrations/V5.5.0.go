package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V5_5_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
	
	ALTER TABLE campaigns ADD IF NOT EXISTS attribs         JSONB NOT NULL DEFAULT '{}';

	`); err != nil {
		return err
	}

	return nil
}

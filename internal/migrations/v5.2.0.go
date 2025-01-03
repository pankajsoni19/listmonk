package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V5_2_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
	CREATE TYPE campaign_run_type AS ENUM ('list', 'event:sub');

	ALTER TABLE campaigns ADD IF NOT EXISTS run_type campaign_run_type NOT NULL DEFAULT 'list';

	`); err != nil {
		return err
	}

	return nil
}

package migrations

import (
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

func V5_4_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf, lo *log.Logger) error {
	if _, err := db.Exec(`
	
	CREATE TYPE campaign_traffic_type AS ENUM ('split', 'duplicate');

	ALTER TABLE campaigns ADD IF NOT EXISTS traffic_type campaign_traffic_type NOT NULL DEFAULT 'split';
	UPDATE campaigns SET messenger = concat('[{"weight":1,"name":"', messenger, '"}]');

	`); err != nil {
		return err
	}

	return nil
}

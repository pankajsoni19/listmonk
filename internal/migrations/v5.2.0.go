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

	ALTER TABLE subscriber_lists ADD CONSTRAINT idx_uniq UNIQUE ( subscriber_id, list_id);

	ALTER TABLE subscriber_lists DROP CONSTRAINT subscriber_lists_pkey;

	ALTER TABLE subscriber_lists ADD COLUMN id BIGSERIAL PRIMARY KEY;

	ALTER TABLE campaigns DROP max_subscriber_id;

	UPDATE settings SET value='false' where key = 'app.check_updates';
	
	`); err != nil {
		return err
	}

	return nil
}

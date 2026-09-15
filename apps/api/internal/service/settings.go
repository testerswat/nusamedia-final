package service

import (
	"context"
	"database/sql"
	"strings"
)

func (s Social) EnsurePlatformSettings(ctx context.Context) error {
	if _, err := s.DB.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS platform_settings(key text PRIMARY KEY,value text NOT NULL,updated_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO platform_settings(key,value) VALUES ('hero_image','/resources/hero.png'),('footer_image','/resources/footer.png') ON CONFLICT(key) DO NOTHING`)
	return err
}

type Settings struct {
	HeroImage   string `json:"hero"`
	FooterImage string `json:"footer"`
}

func (s Social) GetPublicSettings(ctx context.Context) (Settings, error) {
	var out Settings
	rows, err := s.DB.QueryContext(ctx, `SELECT key, value FROM platform_settings WHERE key IN ('hero_image','footer_image')`)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return out, err
		}
		switch key {
		case "hero_image":
			out.HeroImage = value
		case "footer_image":
			out.FooterImage = value
		}
	}
	return out, rows.Err()
}

func (s Social) UpdateMediaSetting(ctx context.Context, key, value string) error {
	key = strings.TrimSpace(key)
	value = strings.TrimSpace(value)
	if key != "hero_image" && key != "footer_image" {
		return sql.ErrNoRows
	}
	if value == "" {
		return sql.ErrNoRows
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO platform_settings(key,value) VALUES($1,$2) ON CONFLICT(key) DO UPDATE SET value=EXCLUDED.value, updated_at=now()`, key, value)
	return err
}

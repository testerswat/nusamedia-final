CREATE TABLE IF NOT EXISTS platform_settings(key text PRIMARY KEY,value text NOT NULL,updated_at timestamptz NOT NULL DEFAULT now());
INSERT INTO platform_settings(key,value) VALUES
('hero_image','/resources/hero.png'),
('footer_image','/resources/footer.png')
ON CONFLICT(key) DO NOTHING;

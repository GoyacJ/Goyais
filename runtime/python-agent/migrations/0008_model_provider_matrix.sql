PRAGMA foreign_keys = OFF;
BEGIN TRANSACTION;

CREATE TABLE model_configs_new (
  model_config_id TEXT PRIMARY KEY,
  provider TEXT NOT NULL CHECK(provider IN ('deepseek', 'minimax_cn', 'minimax_intl', 'zhipu', 'qwen', 'doubao', 'openai', 'anthropic', 'google', 'custom')),
  model TEXT NOT NULL,
  base_url TEXT,
  temperature REAL NOT NULL DEFAULT 0,
  max_tokens INTEGER,
  secret_ref TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

INSERT INTO model_configs_new(model_config_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at)
SELECT model_config_id, provider, model, base_url, temperature, max_tokens, secret_ref, created_at, updated_at
FROM model_configs;

DROP TABLE model_configs;
ALTER TABLE model_configs_new RENAME TO model_configs;

COMMIT;
PRAGMA foreign_keys = ON;

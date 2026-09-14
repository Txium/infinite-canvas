CREATE TABLE IF NOT EXISTS voice_profiles (
 user_id varchar(64) NOT NULL,
 character_id varchar(128) NOT NULL,
 character_name text,
 voice_provider text,
 voice_id text,
 voice_prompt text,
 default_speed numeric,
 default_emotion text,
 default_style text,
 updated_at timestamptz,
 PRIMARY KEY (user_id, character_id)
);
ALTER TABLE voice_profiles ADD COLUMN IF NOT EXISTS reference_images text;
CREATE TABLE IF NOT EXISTS media_ai_tasks (
 id varchar(64) PRIMARY KEY,
 user_id varchar(64), operation text, source_node_id text, adapter text,
 provider_task_id text, status text, outputs text, segments text, error_code text,
 created_at timestamptz, updated_at timestamptz
);
CREATE INDEX IF NOT EXISTS idx_media_ai_tasks_user_id ON media_ai_tasks(user_id);
CREATE INDEX IF NOT EXISTS idx_media_ai_tasks_status ON media_ai_tasks(status);

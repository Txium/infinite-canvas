ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS upstream_model_id text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS adapter text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_endpoint text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS upstream_request_sent boolean NOT NULL DEFAULT false;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS upstream_request_started_at text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS upstream_http_status bigint;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_task_status text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS error_code text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS frontend_selected_resolution text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS backend_resolved_resolution text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_requested_resolution text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_final_resolution text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_final_width bigint;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_final_height bigint;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS provider_original_result_url text;
ALTER TABLE canvas_image_tasks ADD COLUMN IF NOT EXISTS canvas_result_url text;

ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS adapter text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_endpoint text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS upstream_request_sent boolean NOT NULL DEFAULT false;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS upstream_request_started_at text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS upstream_http_status bigint;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_task_status text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS error_code text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS frontend_selected_resolution text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS backend_resolved_resolution text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_requested_resolution text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_final_resolution text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_final_width bigint;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_final_height bigint;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS provider_original_result_url text;
ALTER TABLE video_tasks ADD COLUMN IF NOT EXISTS canvas_result_url text;

ALTER TABLE model_routes ADD COLUMN IF NOT EXISTS adapter text;
ALTER TABLE model_routes ADD COLUMN IF NOT EXISTS endpoint text;
ALTER TABLE model_variants ADD COLUMN IF NOT EXISTS verification_status text;

CREATE INDEX IF NOT EXISTS idx_canvas_image_tasks_error_code ON canvas_image_tasks(error_code);
CREATE INDEX IF NOT EXISTS idx_video_tasks_error_code ON video_tasks(error_code);
CREATE INDEX IF NOT EXISTS idx_model_variants_verification_status ON model_variants(verification_status);

REVOKE ALL ON canvas_image_tasks, video_tasks, model_routes, model_variants FROM PUBLIC;
DO $$ BEGIN
  REVOKE ALL ON canvas_image_tasks, video_tasks, model_routes, model_variants FROM anon;
EXCEPTION WHEN undefined_object THEN NULL; END $$;
DO $$ BEGIN
  REVOKE ALL ON canvas_image_tasks, video_tasks, model_routes, model_variants FROM authenticated;
EXCEPTION WHEN undefined_object THEN NULL; END $$;

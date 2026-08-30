import { apiGet } from "@/services/api/request";

export type UserGenerationTask = {
    task_id: string; display_model_name: string; variant_name: string; task_type: "image" | "video" | "audio";
    status: string; sale_price_cents: number; refund_amount_cents: number; progress: number; created_at: string;
    completed_at: string; duration_seconds: number; input_summary: string; result_url: string; user_friendly_error: string;
};

export async function fetchUserGenerationTasks(token: string, params: { kind?: string; status?: string; days?: number; limit?: number } = {}) {
    return apiGet<UserGenerationTask[]>("/api/v1/generation-tasks", params, token);
}

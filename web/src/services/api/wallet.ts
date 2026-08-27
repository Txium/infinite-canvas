import { apiGet, apiPost, type ApiParams } from "@/services/api/request";

export type RechargeOrderStatus = "pending" | "approved" | "rejected";
export type RechargeOrder = { id: string; userId: string; username?: string; amountCents: number; credits: number; status: RechargeOrderStatus; paymentMethod: string; paymentNote: string; adminRemark: string; createdAt: string; reviewedAt: string };
export type RechargeOrderList = { items: RechargeOrder[]; total: number };
export type RechargePayment = { order: RechargeOrder; payUrl: string };

export function fetchRechargeOrders(token: string) { return apiGet<RechargeOrderList>("/api/v1/wallet/recharge-orders", undefined, token); }
export function createRechargeOrder(token: string, input: { amountCents: number; paymentMethod: string; paymentNote: string }) { return apiPost<RechargePayment>("/api/v1/wallet/recharge-orders", input, token); }
export function fetchAdminRechargeOrders(token: string, params?: ApiParams) { return apiGet<RechargeOrderList>("/api/admin/recharge-orders", params, token); }
export function reviewAdminRechargeOrder(token: string, id: string, status: "approved" | "rejected", remark = "") { return apiPost<RechargeOrder>(`/api/admin/recharge-orders/${encodeURIComponent(id)}/review`, { status, remark }, token); }

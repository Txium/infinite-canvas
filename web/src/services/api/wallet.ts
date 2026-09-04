import { apiGet, apiPost, type ApiParams } from "@/services/api/request";

export type RechargeOrderStatus = "pending" | "approved" | "rejected";
export type RechargeOrder = { id: string; userId: string; username?: string; amountCents: number; credits: number; refundableCents: number; status: RechargeOrderStatus; paymentMethod: string; paymentNote: string; providerTradeId: string; adminRemark: string; createdAt: string; reviewedAt: string };
export type RefundOrderStatus = "pending" | "processing" | "succeeded" | "rejected" | "failed";
export type RefundOrder = { id: string; rechargeOrderId: string; userId: string; username?: string; amountCents: number; reason: string; status: RefundOrderStatus; providerRefundAmountCents: number; adminRemark: string; failureMessage: string; createdAt: string; reviewedAt: string };
export type CreditLog = { id: string; type: string; amount: number; balance: number; frozenAmount: number; frozenBalance: number; relatedId: string; remark: string; createdAt: string };
export type CreditLogList = { items: CreditLog[]; total: number };
export type RechargeOrderList = { items: RechargeOrder[]; total: number };
export type RechargePayment = { order: RechargeOrder; payUrl: string };
export type RefundOrderList = { items: RefundOrder[]; total: number };

export function fetchRechargeOrders(token: string) { return apiGet<RechargeOrderList>("/api/v1/wallet/recharge-orders", undefined, token); }
export function createRechargeOrder(token: string, input: { amountCents: number; paymentMethod: string; paymentNote: string }) { return apiPost<RechargePayment>("/api/v1/wallet/recharge-orders", input, token); }
export function fetchWalletCreditLogs(token: string) { return apiGet<CreditLogList>("/api/v1/wallet/credit-logs", undefined, token); }
export function fetchRefundOrders(token: string) { return apiGet<RefundOrderList>("/api/v1/wallet/refund-orders", undefined, token); }
export function createRefundOrder(token: string, input: { rechargeOrderId: string; amountCents: number; reason: string }) { return apiPost<RefundOrder>("/api/v1/wallet/refund-orders", input, token); }
export function fetchAdminRechargeOrders(token: string, params?: ApiParams) { return apiGet<RechargeOrderList>("/api/admin/recharge-orders", params, token); }
export function reviewAdminRechargeOrder(token: string, id: string, status: "approved" | "rejected", remark = "") { return apiPost<RechargeOrder>(`/api/admin/recharge-orders/${encodeURIComponent(id)}/review`, { status, remark }, token); }
export function fetchAdminRefundOrders(token: string, params?: ApiParams) { return apiGet<RefundOrderList>("/api/admin/refund-orders", params, token); }
export function reviewAdminRefundOrder(token: string, id: string, action: "approve" | "reject" | "query", remark = "") { return apiPost<RefundOrder>(`/api/admin/refund-orders/${encodeURIComponent(id)}/review`, { action, remark }, token); }

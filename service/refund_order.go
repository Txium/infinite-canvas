package service

import (
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func RequestRefund(user model.AuthUser, rechargeOrderID string, amountCents int, reason string) (model.RefundOrder, error) {
	rechargeOrderID = strings.TrimSpace(rechargeOrderID)
	reason = strings.TrimSpace(reason)
	if rechargeOrderID == "" || amountCents < 100 {
		return model.RefundOrder{}, safeMessageError{message: "退款金额不能低于 1 元"}
	}
	if utf8.RuneCountInString(reason) < 2 || utf8.RuneCountInString(reason) > 200 {
		return model.RefundOrder{}, safeMessageError{message: "请填写 2 至 200 字的退款原因"}
	}
	order, err := repository.GetRechargeOrderByID(rechargeOrderID)
	if err != nil || order.UserID != user.ID {
		return model.RefundOrder{}, safeMessageError{message: "充值订单不存在"}
	}
	if order.Status != model.RechargeOrderApproved || order.PaymentMethod != "alipay" || order.ProviderTradeID == "" {
		return model.RefundOrder{}, safeMessageError{message: "该订单不支持支付宝原路退款"}
	}
	refund := model.RefundOrder{ID: newID("refund"), RechargeOrderID: order.ID, UserID: user.ID, AmountCents: amountCents, Reason: reason, Status: model.RefundOrderPending, ProviderTradeID: order.ProviderTradeID, CreatedAt: now(), UpdatedAt: now()}
	result, err := repository.CreateRefundOrder(refund, refund.CreatedAt)
	if err != nil {
		return model.RefundOrder{}, safeMessageError{message: "可退款余额不足，或该充值订单的可退款金额已用完"}
	}
	return result, nil
}

func ListUserRefundOrders(userID string) (model.RefundOrderList, error) {
	items, err := repository.ListUserRefundOrders(userID)
	return model.RefundOrderList{Items: items, Total: len(items)}, err
}

func ListRefundOrders(q model.Query) (model.RefundOrderList, error) {
	items, total, err := repository.ListRefundOrders(q)
	if err != nil {
		return model.RefundOrderList{}, err
	}
	for i := range items {
		if user, ok, _ := repository.GetUserByID(items[i].UserID); ok {
			items[i].Username = user.Username
		}
	}
	return model.RefundOrderList{Items: items, Total: int(total)}, nil
}

func ReviewRefundOrder(id, action, adminID, remark string) (model.RefundOrder, error) {
	action = strings.ToLower(strings.TrimSpace(action))
	remark = strings.TrimSpace(remark)
	refund, err := repository.GetRefundOrderByID(id)
	if err != nil {
		return model.RefundOrder{}, safeMessageError{message: "退款申请不存在"}
	}
	switch action {
	case "reject":
		if refund.Status != model.RefundOrderPending {
			return refund, safeMessageError{message: "只有待审核退款可以拒绝"}
		}
		return repository.ReleaseRefundOrder(id, model.RefundOrderRejected, adminID, remark, "", now())
	case "approve":
		if refund.Status == model.RefundOrderPending {
			refund, err = repository.StartRefundOrder(id, adminID, now())
			if err != nil {
				return refund, err
			}
		} else if refund.Status != model.RefundOrderProcessing {
			return refund, safeMessageError{message: "该退款申请不能再次提交"}
		}
		return submitOrQueryRefund(refund, adminID, false)
	case "query":
		if refund.Status != model.RefundOrderProcessing {
			return refund, safeMessageError{message: "只有处理中的退款需要查询"}
		}
		return submitOrQueryRefund(refund, adminID, true)
	default:
		return refund, safeMessageError{message: "退款审核操作无效"}
	}
}

func submitOrQueryRefund(refund model.RefundOrder, adminID string, query bool) (model.RefundOrder, error) {
	order, err := repository.GetRechargeOrderByID(refund.RechargeOrderID)
	if err != nil {
		return refund, err
	}
	var amount int
	if query {
		amount, err = QueryOfficialAlipayRefund(order, refund)
	} else {
		amount, err = CreateOfficialAlipayRefund(order, refund)
	}
	if err == nil {
		return repository.CompleteRefundOrder(refund.ID, amount, now())
	}
	var businessErr alipayBusinessError
	if errors.As(err, &businessErr) && businessErr.terminal && !query {
		return repository.ReleaseRefundOrder(refund.ID, model.RefundOrderFailed, adminID, "", businessErr.Error(), now())
	}
	var notFoundErr alipayRefundNotFoundError
	if errors.As(err, &notFoundErr) && query {
		return repository.SetRefundOrderMessage(refund.ID, "支付宝未查询到退款，可使用原退款请求号重新提交", now())
	}
	message := "支付宝结果暂不明确，请稍后点击查询退款结果"
	updated, saveErr := repository.SetRefundOrderMessage(refund.ID, message, now())
	if saveErr != nil {
		return refund, saveErr
	}
	return updated, safeMessageError{message: message}
}

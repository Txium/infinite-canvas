package service

import (
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func CreateRechargeOrder(user model.AuthUser, amountCents int, paymentMethod, paymentNote string) (model.RechargeOrder, error) {
	if amountCents < 100 { return model.RechargeOrder{}, safeMessageError{message: "充值金额不能低于 1 元"} }
	if amountCents > 10000000 { return model.RechargeOrder{}, safeMessageError{message: "充值金额超过限制"} }
	order := model.RechargeOrder{ID: newID("recharge"), UserID: user.ID, AmountCents: amountCents, Credits: amountCents, Status: model.RechargeOrderPending, PaymentMethod: strings.TrimSpace(paymentMethod), PaymentNote: strings.TrimSpace(paymentNote), CreatedAt: now(), UpdatedAt: now()}
	return repository.SaveRechargeOrder(order)
}

func ListUserRechargeOrders(userID string) (model.RechargeOrderList, error) {
	items, err := repository.ListUserRechargeOrders(userID)
	return model.RechargeOrderList{Items: items, Total: len(items)}, err
}

func ListRechargeOrders(q model.Query) (model.RechargeOrderList, error) {
	items, total, err := repository.ListRechargeOrders(q)
	if err != nil { return model.RechargeOrderList{}, err }
	for i := range items { if user, ok, _ := repository.GetUserByID(items[i].UserID); ok { items[i].Username = user.Username } }
	return model.RechargeOrderList{Items: items, Total: int(total)}, nil
}

func ReviewRechargeOrder(id string, status model.RechargeOrderStatus, adminID, remark string) (model.RechargeOrder, error) {
	if status != model.RechargeOrderApproved && status != model.RechargeOrderRejected { return model.RechargeOrder{}, safeMessageError{message: "审核状态无效"} }
	return repository.ReviewRechargeOrder(id, status, adminID, strings.TrimSpace(remark), now())
}

package repository

import (
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func CreateRefundOrder(refund model.RefundOrder, createdAt string) (model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return refund, err
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		var order model.RechargeOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", refund.RechargeOrderID, refund.UserID).First(&order).Error; err != nil {
			return err
		}
		if order.Status != model.RechargeOrderApproved || order.PaymentMethod != "alipay" || strings.TrimSpace(order.ProviderTradeID) == "" {
			return gorm.ErrInvalidValue
		}
		var reserved int64
		if err := tx.Model(&model.RefundOrder{}).
			Where("recharge_order_id = ? AND status IN ?", order.ID, []model.RefundOrderStatus{model.RefundOrderPending, model.RefundOrderProcessing, model.RefundOrderSucceeded}).
			Select("COALESCE(SUM(amount_cents), 0)").Scan(&reserved).Error; err != nil {
			return err
		}
		if refund.AmountCents <= 0 || reserved+int64(refund.AmountCents) > int64(order.Credits) {
			return gorm.ErrInvalidValue
		}
		result := tx.Model(&model.User{}).Where("id = ? AND credits >= ?", refund.UserID, refund.AmountCents).
			Updates(map[string]any{"credits": gorm.Expr("credits - ?", refund.AmountCents), "updated_at": createdAt})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return gorm.ErrInvalidValue
		}
		var user model.User
		if err := tx.Where("id = ?", refund.UserID).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&refund).Error; err != nil {
			return err
		}
		return tx.Create(&model.CreditLog{ID: "credit_refund_hold_" + refund.ID, UserID: refund.UserID, Type: model.CreditLogTypeRefundHold, Amount: -refund.AmountCents, Balance: user.Credits, RelatedID: refund.ID, Remark: "原路退款申请锁定", CreatedAt: createdAt}).Error
	})
	return refund, err
}

func GetRefundOrderByID(id string) (model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return model.RefundOrder{}, err
	}
	var item model.RefundOrder
	err = db.Where("id = ?", id).First(&item).Error
	return item, err
}

func ListUserRefundOrders(userID string) ([]model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var items []model.RefundOrder
	err = db.Where("user_id = ?", userID).Order("created_at desc").Limit(100).Find(&items).Error
	return items, err
}

func ListRefundOrders(q model.Query) ([]model.RefundOrder, int64, error) {
	db, err := DB()
	if err != nil {
		return nil, 0, err
	}
	q.Normalize()
	query := db.Model(&model.RefundOrder{})
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("id LIKE ? OR user_id LIKE ? OR recharge_order_id LIKE ? OR status LIKE ?", like, like, like, like)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []model.RefundOrder
	err = query.Order("created_at desc").Offset(q.Offset()).Limit(q.PageSize).Find(&items).Error
	return items, total, err
}

func RefundReservedAmounts(rechargeOrderIDs []string) (map[string]int, error) {
	result := map[string]int{}
	if len(rechargeOrderIDs) == 0 {
		return result, nil
	}
	db, err := DB()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		RechargeOrderID string
		Amount          int
	}
	err = db.Model(&model.RefundOrder{}).
		Select("recharge_order_id, COALESCE(SUM(amount_cents), 0) AS amount").
		Where("recharge_order_id IN ? AND status IN ?", rechargeOrderIDs, []model.RefundOrderStatus{model.RefundOrderPending, model.RefundOrderProcessing, model.RefundOrderSucceeded}).
		Group("recharge_order_id").Scan(&rows).Error
	for _, row := range rows {
		result[row.RechargeOrderID] = row.Amount
	}
	return result, err
}

func StartRefundOrder(id, adminID, reviewedAt string) (model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return model.RefundOrder{}, err
	}
	result := db.Model(&model.RefundOrder{}).Where("id = ? AND status = ?", id, model.RefundOrderPending).
		Updates(map[string]any{"status": model.RefundOrderProcessing, "reviewed_by": adminID, "reviewed_at": reviewedAt, "failure_message": "", "updated_at": reviewedAt})
	if result.Error != nil {
		return model.RefundOrder{}, result.Error
	}
	if result.RowsAffected != 1 {
		return model.RefundOrder{}, gorm.ErrInvalidValue
	}
	return GetRefundOrderByID(id)
}

func CompleteRefundOrder(id string, providerAmount int, reviewedAt string) (model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return model.RefundOrder{}, err
	}
	var result model.RefundOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		var item model.RefundOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return err
		}
		if item.Status == model.RefundOrderSucceeded {
			result = item
			return nil
		}
		if item.Status != model.RefundOrderProcessing || providerAmount != item.AmountCents {
			return gorm.ErrInvalidValue
		}
		if err := tx.Model(&item).Updates(map[string]any{"status": model.RefundOrderSucceeded, "provider_refund_amount_cents": providerAmount, "failure_message": "", "updated_at": reviewedAt}).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.Where("id = ?", item.UserID).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.CreditLog{ID: "credit_refund_paid_" + item.ID, UserID: item.UserID, Type: model.CreditLogTypePaymentRefund, Balance: user.Credits, RelatedID: item.ID, Remark: "支付宝原路退款成功", CreatedAt: reviewedAt}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).First(&result).Error
	})
	return result, err
}

func ReleaseRefundOrder(id string, status model.RefundOrderStatus, adminID, remark, failure, reviewedAt string) (model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return model.RefundOrder{}, err
	}
	var result model.RefundOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		var item model.RefundOrder
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).First(&item).Error; err != nil {
			return err
		}
		if item.Status == model.RefundOrderRejected || item.Status == model.RefundOrderFailed {
			result = item
			return nil
		}
		if status == model.RefundOrderRejected && item.Status != model.RefundOrderPending || status == model.RefundOrderFailed && item.Status != model.RefundOrderProcessing {
			return gorm.ErrInvalidValue
		}
		if err := tx.Model(&model.User{}).Where("id = ?", item.UserID).Updates(map[string]any{"credits": gorm.Expr("credits + ?", item.AmountCents), "updated_at": reviewedAt}).Error; err != nil {
			return err
		}
		var user model.User
		if err := tx.Where("id = ?", item.UserID).First(&user).Error; err != nil {
			return err
		}
		if err := tx.Model(&item).Updates(map[string]any{"status": status, "admin_remark": remark, "failure_message": failure, "reviewed_by": adminID, "reviewed_at": reviewedAt, "updated_at": reviewedAt}).Error; err != nil {
			return err
		}
		if err := tx.Create(&model.CreditLog{ID: "credit_refund_release_" + item.ID, UserID: item.UserID, Type: model.CreditLogTypeRefundRelease, Amount: item.AmountCents, Balance: user.Credits, RelatedID: item.ID, Remark: "原路退款锁定解除", OperatorID: adminID, CreatedAt: reviewedAt}).Error; err != nil {
			return err
		}
		return tx.Where("id = ?", id).First(&result).Error
	})
	return result, err
}

func SetRefundOrderMessage(id, message, updatedAt string) (model.RefundOrder, error) {
	db, err := DB()
	if err != nil {
		return model.RefundOrder{}, err
	}
	if err := db.Model(&model.RefundOrder{}).Where("id = ?", id).Updates(map[string]any{"failure_message": message, "updated_at": updatedAt}).Error; err != nil {
		return model.RefundOrder{}, err
	}
	return GetRefundOrderByID(id)
}

package repository

import (
	"strings"

	"github.com/tigerowo/infinite-canvas/model"
	"gorm.io/gorm"
)

func SaveRechargeOrder(order model.RechargeOrder) (model.RechargeOrder, error) {
	db, err := DB()
	if err != nil { return order, err }
	return order, db.Save(&order).Error
}

func ListUserRechargeOrders(userID string) ([]model.RechargeOrder, error) {
	db, err := DB()
	if err != nil { return nil, err }
	var items []model.RechargeOrder
	err = db.Where("user_id = ?", userID).Order("created_at desc").Limit(100).Find(&items).Error
	return items, err
}

func ListRechargeOrders(q model.Query) ([]model.RechargeOrder, int64, error) {
	db, err := DB()
	if err != nil { return nil, 0, err }
	q.Normalize()
	tx := db.Model(&model.RechargeOrder{})
	if keyword := strings.TrimSpace(q.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where("user_id LIKE ? OR payment_note LIKE ? OR provider_trade_id LIKE ? OR status LIKE ?", like, like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil { return nil, 0, err }
	var items []model.RechargeOrder
	err = tx.Order("created_at desc").Offset(q.Offset()).Limit(q.PageSize).Find(&items).Error
	return items, total, err
}

func ReviewRechargeOrder(id string, status model.RechargeOrderStatus, adminID, remark, reviewedAt string) (model.RechargeOrder, error) {
	db, err := DB()
	if err != nil { return model.RechargeOrder{}, err }
	var result model.RechargeOrder
	err = db.Transaction(func(tx *gorm.DB) error {
		var order model.RechargeOrder
		if err := tx.Where("id = ?", id).First(&order).Error; err != nil { return err }
		if order.Status != model.RechargeOrderPending { result = order; return nil }
		if err := tx.Model(&order).Updates(map[string]any{"status": status, "admin_remark": remark, "reviewed_by": adminID, "reviewed_at": reviewedAt, "updated_at": reviewedAt}).Error; err != nil { return err }
		if status == model.RechargeOrderApproved {
			if err := tx.Model(&model.User{}).Where("id = ?", order.UserID).Updates(map[string]any{"credits": gorm.Expr("credits + ?", order.Credits), "updated_at": reviewedAt}).Error; err != nil { return err }
			var user model.User
			if err := tx.Where("id = ?", order.UserID).First(&user).Error; err != nil { return err }
			if err := tx.Create(&model.CreditLog{ID: "credit_" + order.ID, UserID: order.UserID, Type: model.CreditLogTypeRecharge, Amount: order.Credits, Balance: user.Credits, RelatedID: order.ID, Remark: "充值到账", CreatedAt: reviewedAt}).Error; err != nil { return err }
		}
		return tx.Where("id = ?", id).First(&result).Error
	})
	return result, err
}

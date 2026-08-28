package service

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func CreateRechargeOrder(user model.AuthUser, amountCents int, paymentMethod, paymentNote string) (model.RechargePayment, error) {
	if amountCents < 100 { return model.RechargePayment{}, safeMessageError{message: "充值金额不能低于 1 元"} }
	if amountCents > 10000000 { return model.RechargePayment{}, safeMessageError{message: "充值金额超过限制"} }
	paymentMethod = strings.TrimSpace(paymentMethod)
	if paymentMethod != "alipay" { return model.RechargePayment{}, safeMessageError{message: "目前仅支持支付宝"} }
	if !config.DatabasePersistent() { return model.RechargePayment{}, safeMessageError{message: "平台数据库尚未启用持久化，为保护资金安全暂不能充值"} }
	if !config.PaymentConfigured() { return model.RechargePayment{}, safeMessageError{message: "在线支付尚未配置，请管理员先配置支付商户"} }
	order := model.RechargeOrder{ID: newID("recharge"), UserID: user.ID, AmountCents: amountCents, Credits: amountCents, Status: model.RechargeOrderPending, PaymentMethod: paymentMethod, PaymentNote: strings.TrimSpace(paymentNote), CreatedAt: now(), UpdatedAt: now()}
	saved, err := repository.SaveRechargeOrder(order)
	if err != nil { return model.RechargePayment{}, err }
	if config.AlipayConfigured() {
		payURL, err := createOfficialAlipayPayURL(saved)
		if err != nil { return model.RechargePayment{}, err }
		return model.RechargePayment{Order: saved, PayURL: payURL}, nil
	}
	params := map[string]string{"pid": config.Cfg.EpayMerchantID, "type": paymentMethod, "out_trade_no": saved.ID, "notify_url": strings.TrimRight(config.Cfg.PublicBaseURL, "/") + "/api/payments/epay/notify", "return_url": strings.TrimRight(config.Cfg.PublicBaseURL, "/") + "/wallet?payment=return", "name": "绘界方舟余额充值", "money": fmt.Sprintf("%.2f", float64(amountCents)/100), "sign_type": "MD5"}
	params["sign"] = signPayment(params, config.Cfg.EpayMerchantKey)
	values := url.Values{}
	for key, value := range params { values.Set(key, value) }
	return model.RechargePayment{Order: saved, PayURL: strings.TrimRight(config.Cfg.EpayAPIURL, "/") + "/submit.php?" + values.Encode()}, nil
}

func CompleteEpayRecharge(values url.Values) error {
	if strings.TrimSpace(config.Cfg.EpayMerchantKey) == "" || values.Get("pid") != config.Cfg.EpayMerchantID { return safeMessageError{message: "支付商户配置无效"} }
	provided := strings.ToLower(strings.TrimSpace(values.Get("sign")))
	params := map[string]string{}
	for key := range values { if key != "sign" && key != "sign_type" { params[key] = values.Get(key) } }
	expected := signPayment(params, config.Cfg.EpayMerchantKey)
	if len(provided) != len(expected) || subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) != 1 { return safeMessageError{message: "支付回调签名无效"} }
	if values.Get("trade_status") != "TRADE_SUCCESS" { return safeMessageError{message: "支付尚未成功"} }
	order, err := repository.GetRechargeOrderByID(values.Get("out_trade_no"))
	if err != nil { return err }
	paidCents, err := parseMoneyCents(values.Get("money"))
	if err != nil || paidCents != order.AmountCents { return safeMessageError{message: "支付金额不一致"} }
	_, err = repository.CompleteRechargeOrder(order.ID, values.Get("trade_no"), now())
	return err
}

func parseMoneyCents(value string) (int, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || parts[0] == "" { return 0, fmt.Errorf("invalid money") }
	yuan, err := strconv.Atoi(parts[0])
	if err != nil || yuan < 0 { return 0, fmt.Errorf("invalid money") }
	fraction := ""
	if len(parts) == 2 { fraction = parts[1] }
	if len(fraction) > 2 { return 0, fmt.Errorf("invalid money") }
	for len(fraction) < 2 { fraction += "0" }
	cents := 0
	if fraction != "" {
		cents, err = strconv.Atoi(fraction)
		if err != nil { return 0, fmt.Errorf("invalid money") }
	}
	return yuan*100 + cents, nil
}

func signPayment(params map[string]string, key string) string {
	keys := make([]string, 0, len(params))
	for name, value := range params { if name != "sign" && name != "sign_type" && value != "" { keys = append(keys, name) } }
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, name := range keys { parts = append(parts, name+"="+params[name]) }
	sum := md5.Sum([]byte(strings.Join(parts, "&") + key))
	return hex.EncodeToString(sum[:])
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

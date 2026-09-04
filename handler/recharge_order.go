package handler

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

type createRechargeOrderRequest struct {
	AmountCents   int    `json:"amountCents"`
	PaymentMethod string `json:"paymentMethod"`
	PaymentNote   string `json:"paymentNote"`
}
type reviewRechargeOrderRequest struct {
	Status model.RechargeOrderStatus `json:"status"`
	Remark string                    `json:"remark"`
}
type createRefundOrderRequest struct {
	RechargeOrderID string `json:"rechargeOrderId"`
	AmountCents     int    `json:"amountCents"`
	Reason          string `json:"reason"`
}
type reviewRefundOrderRequest struct {
	Action string `json:"action"`
	Remark string `json:"remark"`
}

func UserRechargeOrders(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	result, err := service.ListUserRechargeOrders(user.ID)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func EpayNotify(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	values := url.Values(r.Form)
	if err := service.CompleteEpayRecharge(values); err != nil {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("success"))
}

func AlipayNotify(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	if err := service.CompleteOfficialAlipayRecharge(r.Form); err != nil {
		http.Error(w, "fail", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("success"))
}

func CreateRechargeOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	var request createRechargeOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.CreateRechargeOrder(user, request.AmountCents, request.PaymentMethod, request.PaymentNote)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminRechargeOrders(w http.ResponseWriter, r *http.Request) {
	result, err := service.ListRechargeOrders(parseQuery(r))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminReviewRechargeOrder(w http.ResponseWriter, r *http.Request, id string) {
	admin, _ := service.UserFromContext(r.Context())
	var request reviewRechargeOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.ReviewRechargeOrder(id, request.Status, admin.ID, request.Remark)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func UserRefundOrders(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	result, err := service.ListUserRefundOrders(user.ID)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func CreateRefundOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	var request createRefundOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.RequestRefund(user, request.RechargeOrderID, request.AmountCents, request.Reason)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminRefundOrders(w http.ResponseWriter, r *http.Request) {
	result, err := service.ListRefundOrders(parseQuery(r))
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

func AdminReviewRefundOrder(w http.ResponseWriter, r *http.Request, id string) {
	admin, _ := service.UserFromContext(r.Context())
	var request reviewRefundOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.ReviewRefundOrder(id, request.Action, admin.ID, request.Remark)
	if err != nil {
		FailError(w, err)
		return
	}
	OK(w, result)
}

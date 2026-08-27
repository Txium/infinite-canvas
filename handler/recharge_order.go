package handler

import (
	"encoding/json"
	"net/http"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/service"
)

type createRechargeOrderRequest struct { AmountCents int `json:"amountCents"`; PaymentMethod string `json:"paymentMethod"`; PaymentNote string `json:"paymentNote"` }
type reviewRechargeOrderRequest struct { Status model.RechargeOrderStatus `json:"status"`; Remark string `json:"remark"` }

func UserRechargeOrders(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	result, err := service.ListUserRechargeOrders(user.ID)
	if err != nil { FailError(w, err); return }
	OK(w, result)
}

func CreateRechargeOrder(w http.ResponseWriter, r *http.Request) {
	user, _ := service.UserFromContext(r.Context())
	var request createRechargeOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.CreateRechargeOrder(user, request.AmountCents, request.PaymentMethod, request.PaymentNote)
	if err != nil { FailError(w, err); return }
	OK(w, result)
}

func AdminRechargeOrders(w http.ResponseWriter, r *http.Request) {
	result, err := service.ListRechargeOrders(parseQuery(r))
	if err != nil { FailError(w, err); return }
	OK(w, result)
}

func AdminReviewRechargeOrder(w http.ResponseWriter, r *http.Request, id string) {
	admin, _ := service.UserFromContext(r.Context())
	var request reviewRechargeOrderRequest
	_ = json.NewDecoder(r.Body).Decode(&request)
	result, err := service.ReviewRechargeOrder(id, request.Status, admin.ID, request.Remark)
	if err != nil { FailError(w, err); return }
	OK(w, result)
}

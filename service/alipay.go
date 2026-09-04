package service

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

var alipayHTTPClient = &http.Client{Timeout: 20 * time.Second}

func createOfficialAlipayPayURL(order model.RechargeOrder) (string, error) {
	biz, err := json.Marshal(map[string]string{
		"out_trade_no": order.ID,
		"total_amount": fmt.Sprintf("%.2f", float64(order.AmountCents)/100),
		"subject":      "绘界方舟余额充值",
		"product_code": "FAST_INSTANT_TRADE_PAY",
	})
	if err != nil {
		return "", err
	}
	params := map[string]string{
		"app_id":      config.Cfg.AlipayAppID,
		"method":      "alipay.trade.page.pay",
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"notify_url":  strings.TrimRight(config.Cfg.PublicBaseURL, "/") + "/api/payments/alipay/notify",
		"return_url":  strings.TrimRight(config.Cfg.PublicBaseURL, "/") + "/wallet?payment=return",
		"biz_content": string(biz),
	}
	signature, err := signAlipayRSA2(params, config.Cfg.AlipayAppPrivateKey)
	if err != nil {
		return "", safeMessageError{message: "支付宝应用私钥格式无效"}
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	return strings.TrimRight(config.Cfg.AlipayGatewayURL, "?") + "?" + values.Encode(), nil
}

func CompleteOfficialAlipayRecharge(values url.Values) error {
	if !config.AlipayConfigured() {
		return safeMessageError{message: "支付宝商户配置无效"}
	}
	if values.Get("app_id") != config.Cfg.AlipayAppID {
		return safeMessageError{message: "支付宝应用不匹配"}
	}
	if strings.TrimSpace(values.Get("seller_id")) == "" || values.Get("seller_id") != config.Cfg.AlipaySellerID {
		return safeMessageError{message: "支付宝收款商户不匹配"}
	}
	status := values.Get("trade_status")
	if status != "TRADE_SUCCESS" && status != "TRADE_FINISHED" {
		return safeMessageError{message: "支付尚未成功"}
	}
	if err := verifyAlipayRSA2(values, config.Cfg.AlipayPublicKey); err != nil {
		return safeMessageError{message: "支付宝回调签名无效"}
	}
	order, err := repository.GetRechargeOrderByID(values.Get("out_trade_no"))
	if err != nil {
		return err
	}
	paidCents, err := parseMoneyCents(values.Get("total_amount"))
	if err != nil || paidCents != order.AmountCents {
		return safeMessageError{message: "支付金额不一致"}
	}
	_, err = repository.CompleteRechargeOrder(order.ID, values.Get("trade_no"), "alipay", values.Get("seller_id"), now())
	return err
}

type alipayRefundResult struct {
	TradeNo      string `json:"trade_no"`
	OutTradeNo   string `json:"out_trade_no"`
	RefundFee    string `json:"refund_fee"`
	RefundAmount string `json:"refund_amount"`
	RefundStatus string `json:"refund_status"`
	OutRequestNo string `json:"out_request_no"`
	FundChange   string `json:"fund_change"`
	Code         string `json:"code"`
	Msg          string `json:"msg"`
	SubCode      string `json:"sub_code"`
	SubMsg       string `json:"sub_msg"`
}

type alipayBusinessError struct {
	message  string
	terminal bool
}

func (err alipayBusinessError) Error() string { return err.message }

type alipayRefundNotFoundError struct{}

func (alipayRefundNotFoundError) Error() string { return "支付宝尚未查询到该退款" }

func CreateOfficialAlipayRefund(order model.RechargeOrder, refund model.RefundOrder) (int, error) {
	result, err := callAlipayRefundAPI("alipay.trade.refund", map[string]string{
		"trade_no":       order.ProviderTradeID,
		"refund_amount":  fmt.Sprintf("%.2f", float64(refund.AmountCents)/100),
		"refund_reason":  refund.Reason,
		"out_request_no": refund.ID,
	}, "alipay_trade_refund_response")
	if err != nil {
		return 0, err
	}
	if result.FundChange != "Y" {
		return 0, fmt.Errorf("支付宝退款资金状态待查询")
	}
	// refund_fee is the cumulative refunded amount for the original trade, not
	// necessarily the amount of this partial refund. The signed request amount
	// plus fund_change=Y identifies this refund safely.
	return refund.AmountCents, nil
}

func QueryOfficialAlipayRefund(order model.RechargeOrder, refund model.RefundOrder) (int, error) {
	result, err := callAlipayRefundAPI("alipay.trade.fastpay.refund.query", map[string]string{
		"trade_no":       order.ProviderTradeID,
		"out_request_no": refund.ID,
	}, "alipay_trade_fastpay_refund_query_response")
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(result.RefundStatus) == "" && strings.TrimSpace(result.RefundAmount) == "" {
		return 0, alipayRefundNotFoundError{}
	}
	if result.RefundStatus != "REFUND_SUCCESS" {
		return 0, fmt.Errorf("支付宝退款状态待确认")
	}
	amount, err := parseMoneyCents(result.RefundAmount)
	if err != nil || amount != refund.AmountCents {
		return 0, fmt.Errorf("支付宝退款查询金额不一致")
	}
	return amount, nil
}

func callAlipayRefundAPI(method string, biz map[string]string, responseKey string) (alipayRefundResult, error) {
	if !config.AlipayConfigured() {
		return alipayRefundResult{}, fmt.Errorf("支付宝商户配置无效")
	}
	bizJSON, err := json.Marshal(biz)
	if err != nil {
		return alipayRefundResult{}, err
	}
	params := map[string]string{
		"app_id":      config.Cfg.AlipayAppID,
		"method":      method,
		"format":      "JSON",
		"charset":     "utf-8",
		"sign_type":   "RSA2",
		"timestamp":   time.Now().Format("2006-01-02 15:04:05"),
		"version":     "1.0",
		"biz_content": string(bizJSON),
	}
	signature, err := signAlipayRSA2(params, config.Cfg.AlipayAppPrivateKey)
	if err != nil {
		return alipayRefundResult{}, err
	}
	form := url.Values{}
	for key, value := range params {
		form.Set(key, value)
	}
	form.Set("sign", signature)
	request, err := http.NewRequest(http.MethodPost, config.Cfg.AlipayGatewayURL, bytes.NewBufferString(form.Encode()))
	if err != nil {
		return alipayRefundResult{}, err
	}
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=utf-8")
	response, err := alipayHTTPClient.Do(request)
	if err != nil {
		return alipayRefundResult{}, err
	}
	defer response.Body.Close()
	body, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return alipayRefundResult{}, err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return alipayRefundResult{}, fmt.Errorf("支付宝退款接口返回 HTTP %d", response.StatusCode)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil {
		return alipayRefundResult{}, err
	}
	raw, ok := envelope[responseKey]
	if !ok {
		return alipayRefundResult{}, fmt.Errorf("支付宝退款响应缺少结果")
	}
	var responseSign string
	if err := json.Unmarshal(envelope["sign"], &responseSign); err != nil || responseSign == "" {
		return alipayRefundResult{}, fmt.Errorf("支付宝退款响应缺少签名")
	}
	if err := verifyAlipayResponse(raw, responseSign, config.Cfg.AlipayPublicKey); err != nil {
		return alipayRefundResult{}, fmt.Errorf("支付宝退款响应验签失败")
	}
	var result alipayRefundResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return result, err
	}
	if result.Code != "10000" {
		message := strings.TrimSpace(result.SubMsg)
		if message == "" {
			message = strings.TrimSpace(result.Msg)
		}
		if message == "" {
			message = "支付宝拒绝退款请求"
		}
		terminal := result.Code == "40004" && result.SubCode != "ACQ.SYSTEM_ERROR" && result.SubCode != "ACQ.REFUND_CHARGE_ERROR"
		return result, alipayBusinessError{message: message, terminal: terminal}
	}
	return result, nil
}

func verifyAlipayResponse(raw json.RawMessage, signature, publicKeyText string) error {
	key, err := parseRSAPublicKey(publicKeyText)
	if err != nil {
		return err
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(signature))
	if err != nil {
		return err
	}
	digest := sha256.Sum256(raw)
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], decoded)
}

func signAlipayRSA2(params map[string]string, privateKeyText string) (string, error) {
	key, err := parseRSAPrivateKey(privateKeyText)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256([]byte(canonicalAlipayParams(params)))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func verifyAlipayRSA2(values url.Values, publicKeyText string) error {
	key, err := parseRSAPublicKey(publicKeyText)
	if err != nil {
		return err
	}
	params := map[string]string{}
	for name := range values {
		if name != "sign" && name != "sign_type" && values.Get(name) != "" {
			params[name] = values.Get(name)
		}
	}
	signature, err := base64.StdEncoding.DecodeString(strings.TrimSpace(values.Get("sign")))
	if err != nil {
		return err
	}
	digest := sha256.Sum256([]byte(canonicalAlipayParams(params)))
	return rsa.VerifyPKCS1v15(key, crypto.SHA256, digest[:], signature)
}

func canonicalAlipayParams(params map[string]string) string {
	keys := make([]string, 0, len(params))
	for key, value := range params {
		if key != "sign" && value != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		parts = append(parts, key+"="+params[key])
	}
	return strings.Join(parts, "&")
}

func decodeKeyDER(value string) ([]byte, error) {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\n`, "\n"))
	if block, _ := pem.Decode([]byte(value)); block != nil {
		return block.Bytes, nil
	}
	// The Alipay console normally displays application and Alipay public
	// keys as raw Base64 without PEM headers. Accept that format directly so
	// the official console value can be stored in a Render secret as-is.
	compact := strings.Join(strings.Fields(value), "")
	if compact == "" {
		return nil, fmt.Errorf("empty key")
	}
	der, err := base64.StdEncoding.DecodeString(compact)
	if err != nil {
		return nil, fmt.Errorf("invalid key encoding: %w", err)
	}
	return der, nil
}

func parseRSAPrivateKey(value string) (*rsa.PrivateKey, error) {
	der, err := decodeKeyDER(value)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(der)
}

func parseRSAPublicKey(value string) (*rsa.PublicKey, error) {
	der, err := decodeKeyDER(value)
	if err != nil {
		return nil, err
	}
	if key, err := x509.ParsePKIXPublicKey(der); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PublicKey(der)
}

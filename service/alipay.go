package service

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

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

func normalizedPEM(value string) []byte {
	value = strings.TrimSpace(strings.ReplaceAll(value, `\n`, "\n"))
	return []byte(value)
}

func parseRSAPrivateKey(value string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(normalizedPEM(value))
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PrivateKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PrivateKey(block.Bytes)
}

func parseRSAPublicKey(value string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode(normalizedPEM(value))
	if block == nil {
		return nil, fmt.Errorf("invalid public key PEM")
	}
	if key, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		if rsaKey, ok := key.(*rsa.PublicKey); ok {
			return rsaKey, nil
		}
	}
	return x509.ParsePKCS1PublicKey(block.Bytes)
}

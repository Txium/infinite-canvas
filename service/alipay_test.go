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
	"net/url"
	"testing"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
)

func alipayTestKeys(t *testing.T) (*rsa.PrivateKey, string, string, string) {
	t.Helper()
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	privateDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	publicDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	privatePEM := string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateDER}))
	publicPEM := string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicDER}))
	return privateKey, privatePEM, publicPEM, base64.StdEncoding.EncodeToString(publicDER)
}

func TestAlipayKeyParsersAcceptPEMAndRawBase64(t *testing.T) {
	privateKey, privatePEM, publicPEM, publicRaw := alipayTestKeys(t)
	privateDER, _ := x509.MarshalPKCS8PrivateKey(privateKey)
	privateRaw := base64.StdEncoding.EncodeToString(privateDER)

	for _, value := range []string{privatePEM, privateRaw} {
		parsed, err := parseRSAPrivateKey(value)
		if err != nil || parsed.N.Cmp(privateKey.N) != 0 {
			t.Fatalf("parse private key: %v", err)
		}
	}
	for _, value := range []string{publicPEM, publicRaw} {
		parsed, err := parseRSAPublicKey(value)
		if err != nil || parsed.N.Cmp(privateKey.N) != 0 {
			t.Fatalf("parse public key: %v", err)
		}
	}
}

func TestCreateOfficialAlipayPayURLUsesExpectedOrderAndSignature(t *testing.T) {
	privateKey, privatePEM, _, publicRaw := alipayTestKeys(t)
	previous := config.Cfg
	config.Cfg = config.Config{
		AlipayAppID:          "2021000000000000",
		AlipayAppPrivateKey:  privatePEM,
		AlipayPublicKey:      publicRaw,
		AlipaySellerID:       "2088000000000000",
		AlipayGatewayURL:     "https://openapi.alipay.com/gateway.do",
		AlipayPaymentEnabled: true,
		PublicBaseURL:        "https://canvas.example.com",
	}
	t.Cleanup(func() { config.Cfg = previous })

	payURL, err := createOfficialAlipayPayURL(model.RechargeOrder{ID: "recharge-test", AmountCents: 1234})
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(payURL)
	if err != nil {
		t.Fatal(err)
	}
	values := parsed.Query()
	if values.Get("method") != "alipay.trade.page.pay" || values.Get("notify_url") != "https://canvas.example.com/api/payments/alipay/notify" || values.Get("return_url") != "https://canvas.example.com/wallet?payment=return" {
		t.Fatalf("unexpected payment parameters: %v", values)
	}
	var biz map[string]string
	if err := json.Unmarshal([]byte(values.Get("biz_content")), &biz); err != nil {
		t.Fatal(err)
	}
	if biz["out_trade_no"] != "recharge-test" || biz["total_amount"] != "12.34" || biz["subject"] != "绘界方舟余额充值" || biz["product_code"] != "FAST_INSTANT_TRADE_PAY" {
		t.Fatalf("unexpected biz_content: %#v", biz)
	}
	signature, err := base64.StdEncoding.DecodeString(values.Get("sign"))
	if err != nil {
		t.Fatal(err)
	}
	params := map[string]string{}
	for key := range values {
		if key != "sign" {
			params[key] = values.Get(key)
		}
	}
	digest := sha256.Sum256([]byte(canonicalAlipayParams(params)))
	if err := rsa.VerifyPKCS1v15(&privateKey.PublicKey, crypto.SHA256, digest[:], signature); err != nil {
		t.Fatalf("verify generated payment signature: %v", err)
	}
}

func TestVerifyAlipayRSA2RejectsTamperedCallback(t *testing.T) {
	_, privatePEM, _, publicRaw := alipayTestKeys(t)
	params := map[string]string{"app_id": "app-1", "out_trade_no": "order-1", "total_amount": "10.00", "trade_status": "TRADE_SUCCESS"}
	signature, err := signAlipayRSA2(params, privatePEM)
	if err != nil {
		t.Fatal(err)
	}
	values := url.Values{}
	for key, value := range params {
		values.Set(key, value)
	}
	values.Set("sign", signature)
	values.Set("sign_type", "RSA2")
	if err := verifyAlipayRSA2(values, publicRaw); err != nil {
		t.Fatalf("valid callback rejected: %v", err)
	}
	values.Set("total_amount", "1000.00")
	if err := verifyAlipayRSA2(values, publicRaw); err == nil {
		t.Fatal("tampered callback signature unexpectedly accepted")
	}
}

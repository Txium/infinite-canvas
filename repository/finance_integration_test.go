package repository

import (
	"sync"
	"testing"

	"github.com/tigerowo/infinite-canvas/config"
	"github.com/tigerowo/infinite-canvas/model"
)

func useFinanceTestDB(t *testing.T) {
	t.Helper()
	previous := config.Cfg
	config.Cfg.StorageDriver = "sqlite"
	config.Cfg.DatabaseDSN = "file:" + t.Name() + "?mode=memory&cache=shared"
	db, dbErr = nil, nil
	dbOnce = sync.Once{}
	database, err := DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := database.DB()
	if err != nil {
		t.Fatal(err)
	}
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = sqlDB.Close()
		config.Cfg = previous
		db, dbErr = nil, nil
		dbOnce = sync.Once{}
	})
}

func seedFinanceUser(t *testing.T, credits, frozen int) model.User {
	t.Helper()
	user := model.User{ID: "user-1", Username: "finance-user", Role: model.UserRoleUser, Status: model.UserStatusActive, Credits: credits, FrozenCredits: frozen, CreatedAt: "2026-08-30T00:00:00Z", UpdatedAt: "2026-08-30T00:00:00Z"}
	database, _ := DB()
	if err := database.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	return user
}

// Test A: a replayed notify credits the pending order once.
func TestRechargeNotifyReplayCreditsOnce(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 0)
	database, _ := DB()
	order := model.RechargeOrder{ID: "order-a", UserID: "user-1", AmountCents: 1000, Credits: 1000, Status: model.RechargeOrderPending}
	if err := database.Create(&order).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRechargeOrder(order.ID, "trade-a", "alipay", "seller-1", "2026-08-30T01:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRechargeOrder(order.ID, "trade-a", "alipay", "seller-1", "2026-08-30T01:00:01Z"); err != nil {
		t.Fatal(err)
	}
	user, _, _ := GetUserByID("user-1")
	if user.Credits != 1000 {
		t.Fatalf("credits=%d, want 1000", user.Credits)
	}
	var count int64
	database.Model(&model.PaymentReceipt{}).Count(&count)
	if count != 1 {
		t.Fatalf("receipts=%d, want 1", count)
	}
}

// Test B: ¥1.69 sale and ¥1.50 upstream cost realizes ¥0.19 gross profit.
func TestSuccessfulTaskSettlementRecordsCostAndProfit(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 169)
	database, _ := DB()
	task := model.VideoTask{ID: "task-b", UserID: "user-1", Model: "seedance", ChannelID: "lec", Status: "completed", Credits: 169, SalePriceCents: 169, EstimatedProviderCostCents: 150, BillingID: "billing-b", BillingStatus: "frozen"}
	if err := database.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	if err := SettleVideoTaskFinancials(&task, "2026-08-30T02:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if task.ActualProviderCostCents != 150 || task.GrossProfitCents != 19 {
		t.Fatalf("cost=%d profit=%d", task.ActualProviderCostCents, task.GrossProfitCents)
	}
	user, _, _ := GetUserByID("user-1")
	if user.FrozenCredits != 0 {
		t.Fatalf("frozen=%d, want 0", user.FrozenCredits)
	}
}

// Test C: a failed task releases the full reservation and creates no revenue/cost.
func TestFailedTaskReleasesWithoutRevenue(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 169)
	log := model.CreditLog{ID: "credit_release_billing-c", UserID: "user-1", Type: model.CreditLogTypeAIRelease, Amount: 169, FrozenAmount: -169, RelatedID: "billing-c"}
	if _, _, err := ReleaseUserCredits("user-1", 169, log, "2026-08-30T03:00:00Z"); err != nil {
		t.Fatal(err)
	}
	user, _, _ := GetUserByID("user-1")
	if user.Credits != 169 || user.FrozenCredits != 0 {
		t.Fatalf("credits=%d frozen=%d", user.Credits, user.FrozenCredits)
	}
	database, _ := DB()
	var costs, settles int64
	database.Model(&model.ProviderLedger{}).Count(&costs)
	database.Model(&model.CreditLog{}).Where("type = ?", model.CreditLogTypeAISettle).Count(&settles)
	if costs != 0 || settles != 0 {
		t.Fatalf("costs=%d settles=%d", costs, settles)
	}
}

// Test D: concurrent duplicate completion settles and records cost once.
func TestConcurrentTaskSettlementIsIdempotent(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 169)
	database, _ := DB()
	task := model.VideoTask{ID: "task-d", UserID: "user-1", Model: "seedance", ChannelID: "lec", Status: "completed", Credits: 169, SalePriceCents: 169, EstimatedProviderCostCents: 150, BillingID: "billing-d", BillingStatus: "frozen"}
	if err := database.Create(&task).Error; err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			copy := task
			errs <- SettleVideoTaskFinancials(&copy, "2026-08-30T04:00:00Z")
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var costs, settles int64
	database.Model(&model.ProviderLedger{}).Where("task_id = ?", task.ID).Count(&costs)
	database.Model(&model.CreditLog{}).Where("id = ?", "credit_settle_billing-d").Count(&settles)
	if costs != 1 || settles != 1 {
		t.Fatalf("costs=%d settles=%d", costs, settles)
	}
}

// Test E: the same Alipay trade number cannot credit another order.
func TestTradeNumberUniqueAcrossOrders(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 0, 0)
	database, _ := DB()
	orders := []model.RechargeOrder{{ID: "order-e1", UserID: "user-1", AmountCents: 1000, Credits: 1000, Status: model.RechargeOrderPending}, {ID: "order-e2", UserID: "user-1", AmountCents: 1000, Credits: 1000, Status: model.RechargeOrderPending}}
	if err := database.Create(&orders).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRechargeOrder("order-e1", "trade-e", "alipay", "seller-1", "2026-08-30T05:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRechargeOrder("order-e2", "trade-e", "alipay", "seller-1", "2026-08-30T05:00:01Z"); err == nil {
		t.Fatal("duplicate trade number unexpectedly credited")
	}
	user, _, _ := GetUserByID("user-1")
	if user.Credits != 1000 {
		t.Fatalf("credits=%d, want 1000", user.Credits)
	}
}

// Test F: if the admin-adjustment ledger insert fails, the balance rolls back.
func TestAdminAdjustmentLedgerFailureRollsBack(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 1000, 0)
	database, _ := DB()
	if err := database.Exec(`CREATE TRIGGER fail_admin_adjust BEFORE INSERT ON credit_logs WHEN NEW.type = 'admin_adjust' BEGIN SELECT RAISE(ABORT, 'forced'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := AdjustUserCredits("user-1", 500, "admin-1", "test", "2026-08-30T06:00:00Z"); err == nil {
		t.Fatal("adjustment unexpectedly succeeded")
	}
	user, _, _ := GetUserByID("user-1")
	if user.Credits != 1000 {
		t.Fatalf("credits=%d, want 1000", user.Credits)
	}
}

// Test G: if the provider topup ledger insert fails, balance update rolls back.
func TestProviderTopupLedgerFailureRollsBack(t *testing.T) {
	useFinanceTestDB(t)
	database, _ := DB()
	zero := int64(0)
	provider := model.ModelProvider{ID: "lec", Code: "lec", Name: "LEC", BalanceCents: &zero}
	if err := database.Create(&provider).Error; err != nil {
		t.Fatal(err)
	}
	if err := database.Exec(`CREATE TRIGGER fail_provider_topup BEFORE INSERT ON provider_ledgers WHEN NEW.type = 'provider_topup' BEGIN SELECT RAISE(ABORT, 'forced'); END`).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := RecordProviderTopup("lec", 5000, "admin-1", "test", "", "2026-08-30T07:00:00Z"); err == nil {
		t.Fatal("topup unexpectedly succeeded")
	}
	var saved model.ModelProvider
	database.Where("id = ?", "lec").First(&saved)
	if saved.BalanceCents == nil || *saved.BalanceCents != 0 {
		t.Fatalf("balance changed after rollback")
	}
}

// User generation history repositories must never return another user's tasks.
func TestUserGenerationHistoryIsIsolatedByUser(t *testing.T) {
	useFinanceTestDB(t)
	database, _ := DB()
	tasks := []model.VideoTask{
		{ID: "task-user-a", UserID: "user-a", Model: "seedance", Status: "completed", CreatedAt: "2026-08-30T08:00:00Z"},
		{ID: "task-user-b", UserID: "user-b", Model: "hailuo", Status: "failed", CreatedAt: "2026-08-30T09:00:00Z"},
	}
	if err := database.Create(&tasks).Error; err != nil {
		t.Fatal(err)
	}
	items, err := ListUserVideoTasksForHistory("user-a", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].ID != "task-user-a" {
		t.Fatalf("user-a received unexpected tasks: %#v", items)
	}
	if _, found, err := GetUserVideoTask("user-a", "task-user-b"); err != nil || found {
		t.Fatalf("user-a could read user-b video task: found=%v err=%v", found, err)
	}
	images := []model.CanvasImageTask{
		{ID: "image-user-a", UserID: "user-a", Status: "completed"},
		{ID: "image-user-b", UserID: "user-b", Status: "completed"},
	}
	if err := database.Create(&images).Error; err != nil {
		t.Fatal(err)
	}
	imageItems, err := ListUserImageTasksForHistory("user-a", 20)
	if err != nil || len(imageItems) != 1 || imageItems[0].ID != "image-user-a" {
		t.Fatalf("user-a received unexpected image tasks: items=%#v err=%v", imageItems, err)
	}
	if _, found, err := GetUserCanvasImageTask("user-a", "image-user-b"); err != nil || found {
		t.Fatalf("user-a could read user-b image task: found=%v err=%v", found, err)
	}
}

// Time-scoped finance cards must not subtract all-time expenses from a selected period.
func TestFinanceSummaryScopesExpensesToSelectedPeriod(t *testing.T) {
	useFinanceTestDB(t)
	database, _ := DB()
	expenses := []model.OperatingExpense{
		{ID: "expense-old", Category: "server", AmountCNY: 1000, CreatedAt: "2026-08-01T00:00:00Z"},
		{ID: "expense-current", Category: "server", AmountCNY: 200, CreatedAt: "2026-08-30T00:00:00Z"},
	}
	if err := database.Create(&expenses).Error; err != nil {
		t.Fatal(err)
	}
	summary, err := AdminFinanceSummary("2026-08-30T00:00:00Z", "today", "2026-08-30T00:00:00Z", "")
	if err != nil {
		t.Fatal(err)
	}
	if summary.SelectedOperatingCostCents != 200 {
		t.Fatalf("selected operating cost=%d, want 200", summary.SelectedOperatingCostCents)
	}
	if summary.OperatingCostCents != 1200 {
		t.Fatalf("all-time operating cost=%d, want 1200", summary.OperatingCostCents)
	}
}

func TestCanonicalFinanceProviderAcceptsRouteIDs(t *testing.T) {
	cases := map[string]string{
		"provider_302":         "302",
		"provider_wavespeed":   "wavespeed",
		"provider_lec":         "lec",
		"provider_seedance_nz": "seedance_nz",
	}
	for input, want := range cases {
		if got := canonicalFinanceProvider(input); got != want {
			t.Fatalf("canonicalFinanceProvider(%q)=%q, want %q", input, got, want)
		}
	}
}

func TestRefundRequestReservesUnusedBalanceAndRejectRestoresOnce(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 1000, 0)
	database, _ := DB()
	recharge := model.RechargeOrder{ID: "recharge-refund-a", UserID: "user-1", AmountCents: 1000, Credits: 1000, Status: model.RechargeOrderApproved, PaymentMethod: "alipay", ProviderTradeID: "trade-refund-a"}
	if err := database.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}
	refund := model.RefundOrder{ID: "refund-a", RechargeOrderID: recharge.ID, UserID: "user-1", AmountCents: 600, Reason: "unused", Status: model.RefundOrderPending}
	if _, err := CreateRefundOrder(refund, "2026-08-30T10:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateRefundOrder(model.RefundOrder{ID: "refund-over", RechargeOrderID: recharge.ID, UserID: "user-1", AmountCents: 500, Reason: "over", Status: model.RefundOrderPending}, "2026-08-30T10:00:01Z"); err == nil {
		t.Fatal("refund above remaining original payment unexpectedly accepted")
	}
	user, _, _ := GetUserByID("user-1")
	if user.Credits != 400 {
		t.Fatalf("credits=%d, want 400 while refund is reserved", user.Credits)
	}
	if _, err := ReleaseRefundOrder(refund.ID, model.RefundOrderRejected, "admin-1", "not eligible", "", "2026-08-30T10:01:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := ReleaseRefundOrder(refund.ID, model.RefundOrderRejected, "admin-1", "replay", "", "2026-08-30T10:01:01Z"); err != nil {
		t.Fatal(err)
	}
	user, _, _ = GetUserByID("user-1")
	if user.Credits != 1000 {
		t.Fatalf("credits=%d, want 1000 after rejection replay", user.Credits)
	}
}

func TestSuccessfulPaymentRefundIsIdempotentAndKeepsBalanceDeducted(t *testing.T) {
	useFinanceTestDB(t)
	seedFinanceUser(t, 1000, 0)
	database, _ := DB()
	recharge := model.RechargeOrder{ID: "recharge-refund-b", UserID: "user-1", AmountCents: 1000, Credits: 1000, Status: model.RechargeOrderApproved, PaymentMethod: "alipay", ProviderTradeID: "trade-refund-b"}
	refund := model.RefundOrder{ID: "refund-b", RechargeOrderID: recharge.ID, UserID: "user-1", AmountCents: 1000, Reason: "unused", Status: model.RefundOrderPending}
	if err := database.Create(&recharge).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := CreateRefundOrder(refund, "2026-08-30T11:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := StartRefundOrder(refund.ID, "admin-1", "2026-08-30T11:01:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRefundOrder(refund.ID, 1000, "2026-08-30T11:02:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := CompleteRefundOrder(refund.ID, 1000, "2026-08-30T11:02:01Z"); err != nil {
		t.Fatal(err)
	}
	user, _, _ := GetUserByID("user-1")
	if user.Credits != 0 {
		t.Fatalf("credits=%d, want 0 after cash refund", user.Credits)
	}
	var paidLogs int64
	database.Model(&model.CreditLog{}).Where("type = ?", model.CreditLogTypePaymentRefund).Count(&paidLogs)
	if paidLogs != 1 {
		t.Fatalf("payment refund logs=%d, want 1", paidLogs)
	}
}

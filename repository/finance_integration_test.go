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

package service

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"gorm.io/gorm"
)

//go:embed catalog/model-market-v1.json
var modelCatalogJSON []byte

type catalogModel struct {
	model.MarketModel
	Variants []model.ModelVariant `json:"variants"`
}

type modelCatalog struct {
	Version   int                   `json:"version"`
	Providers []model.ModelProvider `json:"providers"`
	Models    []catalogModel        `json:"models"`
	Routes    []model.ModelRoute    `json:"routes"`
}

func defaultModelCatalog() ([]model.MarketModel, []model.ModelVariant, []model.ModelProvider, []model.ModelRoute, int, error) {
	var catalog modelCatalog
	if err := json.Unmarshal(modelCatalogJSON, &catalog); err != nil {
		return nil, nil, nil, nil, 0, err
	}
	now := time.Now().Format(time.RFC3339)
	models := make([]model.MarketModel, 0, len(catalog.Models))
	variants := make([]model.ModelVariant, 0)
	for _, entry := range catalog.Models {
		item := entry.MarketModel
		item.CreatedAt, item.UpdatedAt = now, now
		models = append(models, item)
		for _, variant := range entry.Variants {
			variant.CreatedAt, variant.UpdatedAt = now, now
			variants = append(variants, variant)
		}
	}
	for i := range catalog.Providers {
		catalog.Providers[i].CreatedAt, catalog.Providers[i].UpdatedAt = now, now
	}
	for i := range catalog.Routes {
		catalog.Routes[i].CreatedAt, catalog.Routes[i].UpdatedAt = now, now
	}
	return models, variants, catalog.Providers, catalog.Routes, catalog.Version, nil
}

func ensureDefaultModelCatalog() error {
	models, variants, providers, routes, version, err := defaultModelCatalog()
	if err != nil {
		return err
	}
	return repository.SyncDefaultModelCatalog(version, models, variants, providers, routes, time.Now().Format(time.RFC3339))
}

func ListMarketModels(category string, featured bool) ([]model.MarketModelCard, error) {
	if err := ensureDefaultModelCatalog(); err != nil {
		return nil, err
	}
	items, err := repository.ListMarketModels(strings.TrimSpace(category), featured)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	variants, err := repository.ListModelVariants(ids, true)
	if err != nil {
		return nil, err
	}
	routes, err := repository.ListEnabledModelRoutes(ids)
	if err != nil {
		return nil, err
	}
	providers, err := repository.ListModelProviders()
	if err != nil {
		return nil, err
	}
	providerByID := make(map[string]model.ModelProvider, len(providers))
	for _, provider := range providers {
		providerByID[provider.ID] = withProviderSecret(provider)
	}
	byModel := map[string][]model.ModelVariant{}
	for _, variant := range variants {
		byModel[variant.ModelID] = append(byModel[variant.ModelID], variant)
	}
	variantByID := map[string]model.ModelVariant{}
	for _, variant := range variants {
		variantByID[variant.ID] = variant
	}
	availableVariants := map[string]bool{}
	for _, route := range routes {
		variant := variantByID[route.VariantID]
		if provider, ok := providerByID[route.ProviderID]; ok && modelProviderReady(provider) && publicVariantPriced(variant) {
			availableVariants[route.VariantID] = true
		}
	}
	result := make([]model.MarketModelCard, 0, len(items))
	for _, item := range items {
		modelVariants := byModel[item.ID]
		if len(modelVariants) == 0 {
			continue
		}
		availableIDs := make([]string, 0)
		for _, variant := range modelVariants {
			if availableVariants[variant.ID] {
				availableIDs = append(availableIDs, variant.ID)
			}
		}
		// The public market must never advertise an un-routable or unpriced
		// variant as if it were usable. Admin pages still receive the complete
		// catalog, but users only see variants that passed provider readiness,
		// fixed-price, and enabled-route checks above.
		publicVariants := make([]model.PublicModelVariant, 0, len(availableIDs))
		for _, variant := range modelVariants {
			if !availableVariants[variant.ID] {
				continue
			}
			publicVariants = append(publicVariants, model.PublicModelVariant{ID: variant.ID, ModelID: variant.ModelID, Name: variant.Name, PriceCents: variant.PriceCents, PriceText: variant.PriceText, BillingUnit: variant.BillingUnit, PricingMode: variant.PricingMode, PriceFormula: variant.PriceFormula, PersonNote: variant.PersonNote, Remark: variant.Remark, Enabled: variant.Enabled, Sort: variant.Sort})
		}
		if len(publicVariants) == 0 {
			continue
		}
		result = append(result, model.MarketModelCard{MarketModel: item, Variants: publicVariants, AvailableVariantIDs: availableIDs, Available: len(availableIDs) > 0})
	}
	return result, nil
}

func AdminModelProviders() ([]model.ModelProvider, error) {
	if err := ensureDefaultModelCatalog(); err != nil {
		return nil, err
	}
	items, err := repository.ListModelProviders()
	if err != nil {
		return nil, err
	}
	routes, err := repository.ListModelRoutes()
	if err != nil {
		return nil, err
	}
	for i := range items {
		items[i].HasAPIKey = providerSecret(items[i].Code) != ""
		items[i].APIKey = ""
		applyProviderBalanceDefaults(&items[i])
		for _, route := range routes {
			if route.ProviderID != items[i].ID {
				continue
			}
			items[i].RouteCount++
			if route.Enabled {
				items[i].EnabledRouteCount++
			}
		}
		items[i].Ready = items[i].Enabled && strings.TrimSpace(items[i].BaseURL) != "" && items[i].HasAPIKey
		items[i].BalanceStatus, items[i].BalanceMessage = providerBalanceStatus(items[i])
	}
	return items, nil
}
func AdminModelRoutes() ([]model.ModelRoute, error) {
	if err := ensureDefaultModelCatalog(); err != nil {
		return nil, err
	}
	return repository.ListModelRoutes()
}
func AdminModelVariants() ([]model.ModelVariant, error) {
	if err := ensureDefaultModelCatalog(); err != nil {
		return nil, err
	}
	models, err := repository.ListAllMarketModels()
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(models))
	for _, item := range models {
		ids = append(ids, item.ID)
	}
	return repository.ListModelVariants(ids, false)
}
func AdminMarketModels() ([]model.MarketModel, error) {
	if err := ensureDefaultModelCatalog(); err != nil {
		return nil, err
	}
	return repository.ListAllMarketModels()
}

type ModelProviderConnectionTest struct {
	OK          bool     `json:"ok"`
	ProviderID  string   `json:"providerId"`
	Message     string   `json:"message"`
	BalanceText string   `json:"balanceText,omitempty"`
	Models      []string `json:"models,omitempty"`
}

// TestModelProviderConnection performs an authenticated, non-generation
// request.  It verifies that the server-side secret reaches the intended
// upstream without spending inference balance.
func TestModelProviderConnection(id string) (ModelProviderConnectionTest, error) {
	provider, err := repository.SavedModelProviderByID(strings.TrimSpace(id))
	if err != nil {
		return ModelProviderConnectionTest{}, errors.New("中转站不存在")
	}
	provider = withProviderSecret(provider)
	if !modelProviderReady(provider) {
		return ModelProviderConnectionTest{}, errors.New("请先启用中转站并配置 Base URL 与服务器 Secret")
	}
	channel := model.ModelChannel{ID: provider.ID, Protocol: "openai", Name: provider.Name, BaseURL: provider.BaseURL, APIKey: provider.APIKey, Timeout: 60, Enabled: true}
	result := ModelProviderConnectionTest{OK: true, ProviderID: provider.ID, Message: "服务器 Key 与 Base URL 连接正常"}
	var requestURL string
	switch provider.Code {
	case "wavespeed":
		requestURL = BuildModelChannelURL(channel, "/balance")
	case "seedance_nz":
		requestURL, err = providerOriginURL(provider.BaseURL, "/api/usage/wallet/")
	case "302":
		// 302 exposes a free, authenticated balance endpoint. The returned
		// amount is in PTC (not RMB), so it must be shown as-is and never be
		// written into the CNY balance field.
		requestURL, err = providerOriginURL(provider.BaseURL, "/dashboard/balance")
	default:
		requestURL = BuildModelChannelURL(channel, "/models")
	}
	if err != nil {
		return ModelProviderConnectionTest{}, err
	}
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return ModelProviderConnectionTest{}, err
	}
	SetModelChannelAuthHeader(request, channel)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "Mozilla/5.0 (compatible; InfiniteCanvas/0.5; +https://github.com/Txium/infinite-canvas)")
	response, err := HTTPClientForChannel(channel).Do(request)
	if err != nil {
		return ModelProviderConnectionTest{}, safeMessageError{message: "连接失败：上游接口无响应或网络不可达"}
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 512*1024))
	if response.StatusCode == http.StatusForbidden && provider.Code == "302" {
		return ModelProviderConnectionTest{}, safeMessageError{message: "302.AI 返回 403：当前 API Key 未开通余额查询权限，请在 302.AI API Key 高级设置中开启余额权限后再查询"}
	}
	if response.StatusCode >= http.StatusBadRequest {
		return ModelProviderConnectionTest{}, readAdminChannelError(body, response.StatusCode, "连接测试失败")
	}
	if provider.Code == "wavespeed" {
		var payload struct {
			Code int `json:"code"`
			Data struct {
				Balance float64 `json:"balance"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &payload) != nil || payload.Code != 200 {
			return ModelProviderConnectionTest{}, errors.New("WaveSpeed 余额响应无法解析")
		}
		result.BalanceText = "$" + strconv.FormatFloat(payload.Data.Balance, 'f', 2, 64) + " USD"
		result.Message = "连接正常；已读取美元余额（不会写入人民币余额）"
		return result, nil
	}
	if provider.Code == "302" {
		var payload struct {
			Data struct {
				Balance json.RawMessage `json:"balance"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &payload) != nil || len(payload.Data.Balance) == 0 {
			return ModelProviderConnectionTest{}, errors.New("302.AI 余额响应无法解析，请给 API Key 开通余额查询权限")
		}
		var balance string
		if json.Unmarshal(payload.Data.Balance, &balance) != nil {
			var number float64
			if json.Unmarshal(payload.Data.Balance, &number) != nil {
				return ModelProviderConnectionTest{}, errors.New("302.AI 余额响应无法解析")
			}
			balance = strconv.FormatFloat(number, 'f', 2, 64)
		}
		result.BalanceText = strings.TrimSpace(balance) + " PTC"
		result.Message = "连接正常；已读取 302.AI 余额"
		return result, nil
	}
	if provider.Code == "lec" {
		var payload map[string]any
		if json.Unmarshal(body, &payload) != nil {
			return ModelProviderConnectionTest{}, safeMessageError{message: "上游返回非 JSON 页面，可能拦截 Render 服务器访问；暂勿启用该线路"}
		}
		result.Models = append(result.Models, modelIDsFromPayload(payload["data"])...)
		sort.Strings(result.Models)
		if len(result.Models) == 0 {
			result.BalanceText = "上游未提供余额接口"
			result.Message = "连接正常；LEC 未提供可调用的余额接口"
		} else {
			result.BalanceText = "上游未提供余额接口"
			result.Message = fmt.Sprintf("连接正常；上游返回 %d 个模型，LEC 未提供可调用的余额接口", len(result.Models))
		}
		return result, nil
	}
	// seedance.nz wallet schemas may evolve. A successful authenticated HTTP
	// response is sufficient for connectivity. When the documented wallet
	// fields are present, preserve the provider's display currency verbatim;
	// never convert it into the platform's RMB cents field.
	if provider.Code == "seedance_nz" {
		var payload struct {
			Data struct {
				Amount      *float64 `json:"amount"`
				DisplayType string   `json:"display_type"`
			} `json:"data"`
		}
		if json.Unmarshal(body, &payload) == nil && payload.Data.Amount != nil {
			unit := strings.TrimSpace(payload.Data.DisplayType)
			if unit == "" {
				unit = "原币种"
			}
			result.BalanceText = strconv.FormatFloat(*payload.Data.Amount, 'f', 2, 64) + " " + unit
			result.Message = "连接正常；已读取 seedance.nz 钱包余额（原币种显示）"
			return result, nil
		}
	}
	result.Message = "连接正常；钱包接口鉴权通过（余额币种未自动换算）"
	return result, nil
}

func publicVariantPriced(variant model.ModelVariant) bool {
	if variant.PricingMode == "fixed" {
		return variant.PriceCents != nil && *variant.PriceCents > 0
	}
	if variant.PricingMode != "dynamic" {
		return false
	}
	// Dynamic LLM formulas declare both input and output RMB rates. A formula
	// such as "actual cost x 1.08" cannot be settled by the current wallet
	// implementation and must not become a one-cent public generation route.
	return len(regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`).FindAllString(variant.PriceFormula, -1)) >= 2
}

func modelIDsFromPayload(value any) []string {
	result := []string{}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			if id, ok := typed["id"].(string); ok {
				if id = strings.TrimSpace(id); id != "" {
					result = append(result, id)
				}
			}
			for key, item := range typed {
				if key == "id" {
					continue
				}
				walk(item)
			}
		}
	}
	walk(value)
	return result
}

func providerOriginURL(baseURL string, path string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", errors.New("Base URL 无效")
	}
	parsed.Path, parsed.RawPath, parsed.RawQuery, parsed.Fragment = path, "", "", ""
	return parsed.String(), nil
}

func SaveModelProvider(item model.ModelProvider) (model.ModelProvider, error) {
	allowed := map[string]bool{"302": true, "wavespeed": true, "lec": true, "seedance_nz": true}
	item.Code = strings.TrimSpace(item.Code)
	if !allowed[item.Code] {
		return model.ModelProvider{}, errors.New("第一版只允许配置 302.AI、WaveSpeed、LEC 和 seedance.nz")
	}
	now := time.Now().Format(time.RFC3339)
	if item.ID == "" {
		item.ID, item.CreatedAt = newID("provider"), now
		if item.BalanceCents != nil {
			item.BalanceCheckedAt = now
		}
	} else if saved, err := repository.SavedModelProviderByID(item.ID); err == nil {
		if item.CreatedAt == "" {
			item.CreatedAt = saved.CreatedAt
		}
		// Manual upstream balance changes must go through provider_topup or an
		// explicit balance-adjustment ledger; provider configuration cannot
		// silently overwrite financial state.
		item.BalanceCents = saved.BalanceCents
		item.BalanceCheckedAt = saved.BalanceCheckedAt
	}
	item.BaseURL, item.APIKey = strings.TrimSpace(item.BaseURL), ""
	applyProviderBalanceDefaults(&item)
	runtimeProvider := withProviderSecret(item)
	if item.Enabled && !modelProviderReady(runtimeProvider) {
		return model.ModelProvider{}, errors.New("启用供应商前必须在服务器 Secret 中配置 API Key，并填写 Base URL")
	}
	item.UpdatedAt = now
	if item.Timeout <= 0 {
		item.Timeout = 300
	}
	item.HasAPIKey = providerSecret(item.Code) != ""
	return item, repository.SaveModelProvider(item)
}
func SaveMarketModel(item model.MarketModel) (model.MarketModel, error) {
	now := time.Now().Format(time.RFC3339)
	if item.ID == "" {
		item.ID = newID("model")
		item.CreatedAt = now
	} else if saved, err := repository.SavedMarketModelByID(item.ID); err == nil {
		item.CreatedAt = saved.CreatedAt
	}
	if item.Status == "" {
		item.Status = "normal"
	}
	item.Name = strings.TrimSpace(item.Name)
	if item.Name == "" {
		return model.MarketModel{}, errors.New("模型名称不能为空")
	}
	if !map[string]bool{"llm": true, "image": true, "video": true, "person": true, "music": true, "voice": true, "3d": true, "tool": true}[item.Category] {
		return model.MarketModel{}, errors.New("模型分类无效")
	}
	if !map[string]bool{"normal": true, "busy": true, "maintenance": true}[item.Status] {
		return model.MarketModel{}, errors.New("模型运行状态无效")
	}
	item.UpdatedAt = now
	return item, repository.SaveMarketModel(item)
}
func SaveModelVariant(item model.ModelVariant) (model.ModelVariant, error) {
	now := time.Now().Format(time.RFC3339)
	if item.ID == "" {
		item.ID = newID("variant")
		item.CreatedAt = now
	} else if saved, err := repository.SavedModelVariantByID(item.ID); err == nil {
		item.ProviderCode = saved.ProviderCode
		item.UpstreamModelID = saved.UpstreamModelID
		item.CostCents = saved.CostCents
		item.CostText = saved.CostText
		item.PriceText = saved.PriceText
		item.MarginText = saved.MarginText
		item.PersonNote = saved.PersonNote
		item.RefundPolicy = saved.RefundPolicy
		item.SourceURL = saved.SourceURL
		item.Remark = saved.Remark
		item.Sort = saved.Sort
		item.CreatedAt = saved.CreatedAt
	}
	item.UpdatedAt = now
	return item, repository.SaveModelVariant(item)
}
func SaveModelRoute(item model.ModelRoute) (model.ModelRoute, error) {
	variant, err := repository.SavedModelVariantByID(strings.TrimSpace(item.VariantID))
	if err != nil {
		return model.ModelRoute{}, errors.New("模型档位不存在")
	}
	provider, err := repository.SavedModelProviderByID(strings.TrimSpace(item.ProviderID))
	if err != nil {
		return model.ModelRoute{}, errors.New("供应商不存在")
	}
	item.ModelID = variant.ModelID
	item.UpstreamModelID, item.Protocol = strings.TrimSpace(item.UpstreamModelID), strings.TrimSpace(item.Protocol)
	if item.UpstreamModelID == "" {
		return model.ModelRoute{}, errors.New("实际上游模型 ID 不能为空")
	}
	if item.Protocol == "" {
		return model.ModelRoute{}, errors.New("线路协议不能为空")
	}
	if item.Priority <= 0 {
		item.Priority = 1
	}
	if item.Enabled {
		marketModel, modelErr := repository.SavedMarketModelByID(variant.ModelID)
		if modelErr != nil || !marketModel.Enabled || marketModel.Status == "maintenance" {
			return model.ModelRoute{}, errors.New("模型未上架或正在维护，不能启用线路")
		}
		fixedPriced := variant.PricingMode == "fixed" && variant.PriceCents != nil && *variant.PriceCents > 0
		dynamicPriced := variant.PricingMode == "dynamic" && strings.TrimSpace(variant.PriceFormula) != ""
		if !variant.Enabled || (!fixedPriced && !dynamicPriced) {
			return model.ModelRoute{}, errors.New("启用线路前必须为档位设置售价或有效的动态价格公式并上架")
		}
		if !modelProviderReady(withProviderSecret(provider)) {
			return model.ModelRoute{}, errors.New("启用线路前必须启用供应商并配置 Base URL 与服务器 Secret")
		}
	}
	now := time.Now().Format(time.RFC3339)
	if item.ID == "" {
		item.ID = newID("route")
		item.CreatedAt = now
	} else if saved, savedErr := repository.SavedModelRouteByID(item.ID); savedErr == nil {
		item.CreatedAt = saved.CreatedAt
	}
	item.UpdatedAt = now
	return item, repository.SaveModelRoute(item)
}

func AdminModelReadiness() (model.ModelReadiness, error) {
	if err := ensureDefaultModelCatalog(); err != nil {
		return model.ModelReadiness{}, err
	}
	providers, err := repository.ListModelProviders()
	if err != nil {
		return model.ModelReadiness{}, err
	}
	marketModels, err := repository.ListAllMarketModels()
	if err != nil {
		return model.ModelReadiness{}, err
	}
	modelIDs := make([]string, 0, len(marketModels))
	enabledModels := map[string]bool{}
	for _, item := range marketModels {
		modelIDs = append(modelIDs, item.ID)
		enabledModels[item.ID] = item.Enabled && item.Status != "maintenance"
	}
	variants, err := repository.ListModelVariants(modelIDs, false)
	if err != nil {
		return model.ModelReadiness{}, err
	}
	routes, err := repository.ListModelRoutes()
	if err != nil {
		return model.ModelReadiness{}, err
	}
	result := model.ModelReadiness{ProviderCount: len(providers), Issues: []model.ModelReadinessIssue{}}
	providerReady := map[string]bool{}
	for _, item := range providers {
		ready := modelProviderReady(withProviderSecret(item))
		providerReady[item.ID] = ready
		if ready {
			result.ReadyProviderCount++
			continue
		}
		missing := []string{}
		if !item.Enabled {
			missing = append(missing, "未启用")
		}
		if strings.TrimSpace(item.BaseURL) == "" {
			missing = append(missing, "缺 Base URL")
		}
		if providerSecret(item.Code) == "" {
			missing = append(missing, "缺服务器 Secret")
		}
		result.Issues = append(result.Issues, model.ModelReadinessIssue{Level: "warning", Scope: "provider", ID: item.ID, Message: item.Name + "：" + strings.Join(missing, "、")})
	}
	enabledRoutesByVariant := map[string]int{}
	for _, route := range routes {
		if !route.Enabled {
			continue
		}
		result.EnabledRouteCount++
		if providerReady[route.ProviderID] && strings.TrimSpace(route.UpstreamModelID) != "" {
			enabledRoutesByVariant[route.VariantID]++
		}
	}
	missingRouteCount, dynamicCount := 0, 0
	for _, variant := range variants {
		if !variant.Enabled || !enabledModels[variant.ModelID] {
			continue
		}
		result.EnabledVariantCount++
		if variant.PricingMode == "dynamic" {
			if strings.TrimSpace(variant.PriceFormula) == "" {
				dynamicCount++
				continue
			}
			if enabledRoutesByVariant[variant.ID] > 0 {
				result.AvailableVariantCount++
			} else {
				missingRouteCount++
			}
			continue
		}
		if variant.PricingMode != "fixed" || variant.PriceCents == nil || *variant.PriceCents <= 0 {
			dynamicCount++
			continue
		}
		if enabledRoutesByVariant[variant.ID] > 0 {
			result.AvailableVariantCount++
		} else {
			missingRouteCount++
		}
	}
	if missingRouteCount > 0 {
		result.Issues = append(result.Issues, model.ModelReadinessIssue{Level: "error", Scope: "route", Message: fmt.Sprintf("%d 个已上架固定价档位没有可用线路", missingRouteCount)})
	}
	if dynamicCount > 0 {
		result.Issues = append(result.Issues, model.ModelReadinessIssue{Level: "warning", Scope: "pricing", Message: fmt.Sprintf("%d 个动态价或未定价档位暂不允许生成", dynamicCount)})
	}
	result.Ready = result.ReadyProviderCount > 0 && result.AvailableVariantCount > 0
	return result, nil
}

type MarketRouteCandidate struct {
	RouteID       string
	Channel       model.ModelChannel
	UpstreamModel string
}

func ResolveMarketRoutes(variantID string) ([]MarketRouteCandidate, bool, error) {
	// Render's free filesystem is ephemeral. A generation request may be the
	// first request after a deployment, before the model-market page has seeded
	// the bundled catalog. Route resolution must therefore initialize it too.
	if err := ensureDefaultModelCatalog(); err != nil {
		return nil, false, err
	}
	variant, variantErr := repository.MarketVariantByID(strings.TrimSpace(variantID))
	if errors.Is(variantErr, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if variantErr != nil {
		return nil, false, variantErr
	}
	if !variant.Enabled {
		return nil, true, errors.New("当前模型档位未上架")
	}
	routes, err := repository.EnabledRoutesForVariant(strings.TrimSpace(variantID))
	if err != nil {
		return nil, true, err
	}
	result := make([]MarketRouteCandidate, 0, len(routes))
	for _, route := range routes {
		provider, providerErr := repository.ModelProviderByID(route.ProviderID)
		provider = withProviderSecret(provider)
		if providerErr != nil || !modelProviderReady(provider) {
			continue
		}
		baseURL := provider.BaseURL
		protocol := route.Protocol
		if marketModel, marketErr := repository.SavedMarketModelByID(variant.ModelID); marketErr == nil && marketModel.Category == "llm" && provider.Code == "wavespeed" {
			// WaveSpeed serves LLMs from its OpenAI-compatible host, separate
			// from the media prediction API, while keeping the same account key.
			baseURL = "https://llm.wavespeed.ai/v1"
			protocol = "openai"
		}
		result = append(result, MarketRouteCandidate{RouteID: route.ID, Channel: model.ModelChannel{ID: provider.ID, Protocol: protocol, Name: provider.Name, BaseURL: baseURL, APIKey: provider.APIKey, Models: []string{route.UpstreamModelID}, Weight: 1, Timeout: provider.Timeout, Enabled: true}, UpstreamModel: route.UpstreamModelID})
	}
	if len(result) == 0 {
		return nil, true, errors.New("指定模型渠道不可用")
	}
	return result, true, nil
}

func ResolveMarketRouteForRoute(variantID string, routeID string) (model.ModelChannel, string, bool, error) {
	candidates, routed, err := ResolveMarketRoutes(variantID)
	if err != nil || !routed {
		return model.ModelChannel{}, "", routed, err
	}
	if routeID != "" {
		for _, candidate := range candidates {
			if candidate.RouteID == routeID {
				return candidate.Channel, candidate.UpstreamModel, true, nil
			}
		}
	}
	return candidates[0].Channel, candidates[0].UpstreamModel, true, nil
}

func ResolveMarketRoute(variantID string) (model.ModelChannel, string, bool, error) {
	return ResolveMarketRouteForRoute(variantID, "")
}

func modelProviderReady(item model.ModelProvider) bool {
	return item.Enabled && strings.TrimSpace(item.BaseURL) != "" && strings.TrimSpace(item.APIKey) != ""
}

func withProviderSecret(item model.ModelProvider) model.ModelProvider {
	item.APIKey = providerSecret(item.Code)
	item.HasAPIKey = item.APIKey != ""
	return item
}

func providerSecret(code string) string {
	environmentNames := map[string][]string{
		"302":         {"MODEL_PROVIDER_302_API_KEY", "API_PROVIDER_302_KEY", "302_API_KEY"},
		"wavespeed":   {"MODEL_PROVIDER_WAVESPEED_API_KEY", "API_PROVIDER_WAVESPEED_KEY", "WAVESPEED_API_KEY"},
		"lec":         {"MODEL_PROVIDER_LEC_API_KEY", "API_PROVIDER_LEC_KEY", "LEC_API_KEY"},
		"seedance_nz": {"MODEL_PROVIDER_SEEDANCE_NZ_API_KEY", "API_PROVIDER_SEEDANCE_NZ_KEY", "SEEDANCE_NZ_API_KEY"},
	}
	for _, name := range environmentNames[strings.TrimSpace(code)] {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func applyProviderBalanceDefaults(item *model.ModelProvider) {
	if item.WarningBalanceCents <= 0 {
		item.WarningBalanceCents = 10000
	}
	if item.CriticalBalanceCents <= 0 {
		item.CriticalBalanceCents = 3000
	}
	if item.LowBalanceCents <= 0 {
		item.LowBalanceCents = 1000
	}
	if item.CriticalBalanceCents > item.WarningBalanceCents {
		item.CriticalBalanceCents = item.WarningBalanceCents
	}
	if item.LowBalanceCents > item.CriticalBalanceCents {
		item.LowBalanceCents = item.CriticalBalanceCents
	}
}

func providerBalanceStatus(item model.ModelProvider) (string, string) {
	if !item.Enabled {
		return "disabled", "供应商当前停用"
	}
	if strings.TrimSpace(item.BaseURL) == "" || !item.HasAPIKey {
		return "not_ready", "Base URL 或服务器 Secret 未配置"
	}
	if item.BalanceCents == nil {
		return "unknown", "需要登录上游查看并手工记录"
	}
	if *item.BalanceCents < item.LowBalanceCents {
		return "very_low", "余额极低，请尽快充值"
	}
	if *item.BalanceCents < item.CriticalBalanceCents {
		return "critical", "余额不足"
	}
	if *item.BalanceCents < item.WarningBalanceCents {
		return "warning", "余额偏低"
	}
	return "normal", "余额正常（手工记录）"
}

func sameOptionalInt64(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func MarketModelCost(variantID string) (int, bool, error) {
	cost, _, routed, err := MarketModelPricing(variantID)
	return cost, routed, err
}

func MarketModelPricing(variantID string) (int, string, bool, error) {
	return MarketModelPricingForRequest(variantID, nil)
}

// MarketModelPricingForRequest returns a conservative RMB reservation for a
// market request. Dynamic LLM variants are reserved from the declared
// input/output per-million-token formula; a later usage-aware settlement can
// refund any unused reservation without exposing provider pricing to clients.
func MarketModelPricingForRequest(variantID string, body []byte) (int, string, bool, error) {
	variant, err := repository.MarketVariantByID(strings.TrimSpace(variantID))
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, err
	}
	if !variant.Enabled || variant.PricingMode == "disabled" {
		return 0, "", false, errors.New("当前模型档位未上架")
	}
	if variant.PricingMode == "dynamic" {
		if strings.TrimSpace(variant.PriceFormula) == "" {
			return 0, "", false, errors.New("动态价格尚未配置")
		}
		return dynamicVariantReservation(variant.PriceFormula, body), strings.TrimSpace(variant.BillingUnit), true, nil
	}
	if variant.PriceCents == nil {
		return 0, "", false, errors.New("模型售价尚未配置")
	}
	return int(*variant.PriceCents), strings.TrimSpace(variant.BillingUnit), true, nil
}

// MarketModelEstimatedProviderCost returns the configured upstream cost in
// cents for the same billing units used to reserve the customer sale price.
func MarketModelEstimatedProviderCost(variantID string, units int) (int64, error) {
	variant, err := repository.MarketVariantByID(strings.TrimSpace(variantID))
	if errors.Is(err, gorm.ErrRecordNotFound) || variant.CostCents == nil {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if units < 1 {
		units = 1
	}
	return *variant.CostCents * int64(units), nil
}

func dynamicVariantReservation(formula string, body []byte) int {
	prices := regexp.MustCompile(`([0-9]+(?:\.[0-9]+)?)`).FindAllString(formula, -1)
	if len(prices) < 2 {
		return 1
	}
	inputRate, _ := strconv.ParseFloat(prices[0], 64)
	outputRate, _ := strconv.ParseFloat(prices[1], 64)
	inputTokens := 1
	if len(body) > 0 {
		inputTokens = len([]rune(string(body))) / 4
		if inputTokens < 1 {
			inputTokens = 1
		}
	}
	outputTokens := 512
	var request struct {
		MaxTokens int `json:"max_tokens"`
	}
	if json.Unmarshal(body, &request) == nil && request.MaxTokens > 0 {
		outputTokens = request.MaxTokens
	}
	reservation := (float64(inputTokens)*inputRate + float64(outputTokens)*outputRate) / 10000
	credits := int(reservation)
	if float64(credits) < reservation {
		credits++
	}
	if credits < 1 {
		credits = 1
	}
	return credits
}

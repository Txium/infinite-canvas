package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

const providerCatalogSyncInterval = 6 * time.Hour

var providerCatalogSyncOnce sync.Once

type ProviderCatalogSyncResult struct {
	ProviderID     string   `json:"providerId"`
	ProviderCode   string   `json:"providerCode"`
	UpstreamModels int      `json:"upstreamModels"`
	CheckedRoutes  int      `json:"checkedRoutes"`
	DisabledRoutes []string `json:"disabledRoutes"`
	CheckedAt      string   `json:"checkedAt"`
}

// StartProviderCatalogSync removes stale selectable routes without deleting
// historical model/task records. Only providers with an authoritative catalog
// endpoint are eligible; a failed or empty catalog response changes nothing.
func StartProviderCatalogSync() {
	providerCatalogSyncOnce.Do(func() {
		go func() {
			timer := time.NewTimer(10 * time.Second)
			defer timer.Stop()
			<-timer.C
			for {
				syncReadyProviderCatalogs()
				time.Sleep(providerCatalogSyncInterval)
			}
		}()
	})
}

func syncReadyProviderCatalogs() {
	providers, err := repository.ListModelProviders()
	if err != nil {
		log.Printf("list providers for catalog sync failed: %v", err)
		return
	}
	for _, provider := range providers {
		if provider.Code != "lec" && provider.Code != "wavespeed" {
			continue
		}
		if _, err := SyncModelProviderCatalog(provider.ID); err != nil {
			log.Printf("provider catalog sync skipped provider=%s err=%v", provider.Code, err)
		}
	}
}

func SyncModelProviderCatalog(id string) (ProviderCatalogSyncResult, error) {
	provider, err := repository.SavedModelProviderByID(strings.TrimSpace(id))
	if err != nil {
		return ProviderCatalogSyncResult{}, errors.New("中转站不存在")
	}
	provider = withProviderSecret(provider)
	if !modelProviderReady(provider) {
		return ProviderCatalogSyncResult{}, errors.New("中转站未启用或服务器 Secret 未配置")
	}
	if provider.Code != "lec" && provider.Code != "wavespeed" {
		return ProviderCatalogSyncResult{}, errors.New("该中转站没有可安全使用的权威模型目录接口")
	}

	channel := model.ModelChannel{ID: provider.ID, Protocol: "openai", Name: provider.Name, BaseURL: provider.BaseURL, APIKey: provider.APIKey, Timeout: 60, Enabled: true}
	request, err := http.NewRequest(http.MethodGet, BuildModelChannelURL(channel, "/models"), nil)
	if err != nil {
		return ProviderCatalogSyncResult{}, err
	}
	SetModelChannelAuthHeader(request, channel)
	request.Header.Set("Accept", "application/json")
	response, err := HTTPClientForChannel(channel).Do(request)
	if err != nil {
		return ProviderCatalogSyncResult{}, errors.New("上游模型目录暂时无法访问，本次未修改任何路由")
	}
	defer response.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(response.Body, 4*1024*1024))
	if response.StatusCode >= http.StatusBadRequest {
		return ProviderCatalogSyncResult{}, readAdminChannelError(body, response.StatusCode, "上游模型目录读取失败，本次未修改任何路由")
	}
	var payload any
	if json.Unmarshal(body, &payload) != nil {
		return ProviderCatalogSyncResult{}, errors.New("上游模型目录格式无效，本次未修改任何路由")
	}
	ids := catalogModelIDs(payload)
	if len(ids) == 0 {
		return ProviderCatalogSyncResult{}, errors.New("上游返回空模型目录，本次未修改任何路由")
	}
	available := make(map[string]bool, len(ids))
	for _, item := range ids {
		available[item] = true
	}

	models, err := repository.ListAllMarketModels()
	if err != nil {
		return ProviderCatalogSyncResult{}, err
	}
	categories := map[string]string{}
	for _, item := range models {
		categories[item.ID] = item.Category
	}
	checked := 0
	eligible := func(route model.ModelRoute) bool {
		upstream := strings.TrimSpace(route.UpstreamModelID)
		if upstream == "" || strings.ContainsAny(upstream, "* ") || strings.HasPrefix(upstream, "/") {
			return false
		}
		if provider.Code == "wavespeed" && categories[route.ModelID] == "llm" {
			return false
		}
		checked++
		return true
	}
	checkedAt := time.Now().UTC().Format(time.RFC3339)
	disabled, err := repository.ApplyProviderCatalog(provider.ID, available, checkedAt, eligible)
	if err != nil {
		return ProviderCatalogSyncResult{}, err
	}
	disabledIDs := make([]string, 0, len(disabled))
	for _, route := range disabled {
		disabledIDs = append(disabledIDs, fmt.Sprintf("%s -> %s", route.VariantID, route.UpstreamModelID))
	}
	sort.Strings(disabledIDs)
	return ProviderCatalogSyncResult{ProviderID: provider.ID, ProviderCode: provider.Code, UpstreamModels: len(ids), CheckedRoutes: checked, DisabledRoutes: disabledIDs, CheckedAt: checkedAt}, nil
}

func catalogModelIDs(value any) []string {
	seen := map[string]bool{}
	var walk func(any)
	walk = func(current any) {
		switch typed := current.(type) {
		case []any:
			for _, item := range typed {
				walk(item)
			}
		case map[string]any:
			for _, key := range []string{"id", "model_id"} {
				if id, ok := typed[key].(string); ok {
					id = strings.TrimSpace(id)
					if id != "" {
						seen[id] = true
					}
				}
			}
			for _, item := range typed {
				walk(item)
			}
		}
	}
	walk(value)
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	sort.Strings(result)
	return result
}

package service

import (
	"errors"
	"strings"
	"time"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
	"gorm.io/gorm"
)

func ListMarketModels(category string, featured bool) ([]model.MarketModelCard, error) {
	if err := repository.SeedMarketModels(defaultMarketModels()); err != nil { return nil, err }
	items, err := repository.ListMarketModels(strings.TrimSpace(category), featured)
	if err != nil { return nil, err }
	ids := make([]string, 0, len(items)); for _, item := range items { ids = append(ids, item.ID) }
	prices, err := repository.ListModelPrices(ids); if err != nil { return nil, err }
	routes, err := repository.ListEnabledModelRoutes(ids); if err != nil { return nil, err }
	byModel := map[string][]model.ModelPrice{}; for _, price := range prices { byModel[price.ModelID] = append(byModel[price.ModelID], price) }
	available := map[string]bool{}; for _, route := range routes { available[route.ModelID] = true }
	result := make([]model.MarketModelCard, 0, len(items)); for _, item := range items { result = append(result, model.MarketModelCard{MarketModel:item, Prices:byModel[item.ID], Available:available[item.ID]}) }
	return result, nil
}

func AdminModelProviders() ([]model.ModelProvider, error) { items, err := repository.ListModelProviders(); for i := range items { if items[i].APIKey != "" { items[i].APIKey = "" } }; return items, err }
func AdminModelRoutes() ([]model.ModelRoute, error) { return repository.ListModelRoutes() }
func SaveModelProvider(item model.ModelProvider) (model.ModelProvider, error) { now := time.Now().Format(time.RFC3339); if item.ID == "" { item.ID = newID("provider"); item.CreatedAt = now } else if saved, err := repository.SavedModelProviderByID(item.ID); err == nil { if item.APIKey == "" { item.APIKey = saved.APIKey }; if item.CreatedAt == "" { item.CreatedAt = saved.CreatedAt } }; item.UpdatedAt = now; if item.Timeout <= 0 { item.Timeout = 300 }; return item, repository.SaveModelProvider(item) }
func SaveMarketModel(item model.MarketModel) (model.MarketModel, error) { now := time.Now().Format(time.RFC3339); if item.ID == "" { item.ID = newID("model"); item.CreatedAt = now }; if item.Status == "" { item.Status = "normal" }; item.UpdatedAt = now; return item, repository.SaveMarketModel(item) }
func SaveModelRoute(item model.ModelRoute) (model.ModelRoute, error) { now := time.Now().Format(time.RFC3339); if item.ID == "" { item.ID = newID("route"); item.CreatedAt = now }; item.UpdatedAt = now; return item, repository.SaveModelRoute(item) }
func SaveModelPrice(item model.ModelPrice) (model.ModelPrice, error) { now := time.Now().Format(time.RFC3339); if item.ID == "" { item.ID = newID("price"); item.CreatedAt = now }; if item.Currency == "" { item.Currency = "CNY" }; item.UpdatedAt = now; return item, repository.SaveModelPrice(item) }

func ResolveMarketRoute(modelID string) (model.ModelChannel, string, bool, error) {
	routes, err := repository.EnabledRoutesForModel(strings.TrimSpace(modelID))
	if err != nil { return model.ModelChannel{}, "", false, err }
	for _, route := range routes {
		provider, providerErr := repository.ModelProviderByID(route.ProviderID)
		if providerErr != nil { continue }
		return model.ModelChannel{ID:provider.ID, Protocol:route.Protocol, Name:provider.Name, BaseURL:provider.BaseURL, APIKey:provider.APIKey, Models:[]string{route.UpstreamModelID}, Weight:1, Timeout:provider.Timeout, Enabled:true}, route.UpstreamModelID, true, nil
	}
	return model.ModelChannel{}, "", false, nil
}

func MarketModelCost(modelID string) (int, bool, error) {
	price, err := repository.MarketModelPrice(strings.TrimSpace(modelID))
	if errors.Is(err, gorm.ErrRecordNotFound) { return 0, false, nil }
	return price.PriceCredits, err == nil, err
}

func defaultMarketModels() []model.MarketModel {
	now := time.Now().Format(time.RFC3339)
	makeModel := func(id,name,category,description string,modes,resolutions,durations []string, featured bool, sort int) model.MarketModel { return model.MarketModel{ID:id,Name:name,Category:category,Description:description,Modes:modes,Resolutions:resolutions,Durations:durations,Ratios:[]string{"16:9","9:16","1:1"},Speed:"normal",Featured:featured,Status:"normal",Enabled:true,Sort:sort,CreatedAt:now,UpdatedAt:now} }
	items := []model.MarketModel{
		makeModel("sd20_fast","Seedance 2.0 极速","video","快速多模态视频生成",[]string{"text-to-video","image-to-video"},[]string{"720p"},[]string{"5","10","15"},true,10),
		makeModel("sd20_standard","Seedance 2.0 标准","video","高质量多模态视频生成",[]string{"text-to-video","image-to-video","multimodal"},[]string{"720p","1080p"},[]string{"5","10","15"},true,20),
		makeModel("sd25","Seedance 2.5","video","新一代 Seedance 视频模型",[]string{"text-to-video","image-to-video"},[]string{"720p","1080p"},[]string{"5","10"},true,30),
		makeModel("sd20_face_720","Seedance 2.0 人物 720P","person","人物参考与角色一致性",[]string{"image-to-video","multimodal"},[]string{"720p"},[]string{"10","15"},true,40),
		makeModel("sd20_face_1080","Seedance 2.0 人物 1080P","person","高清人物参考专线",[]string{"image-to-video","multimodal"},[]string{"1080p"},[]string{"10","15"},false,50),
		makeModel("veo31_fast","Veo 3.1 Fast","video","快速电影感视频生成",[]string{"text-to-video","image-to-video"},[]string{"720p","1080p"},[]string{"5","8"},true,60),
		makeModel("kling30","Kling 3.0","video","高质量运动与镜头控制",[]string{"text-to-video","image-to-video"},[]string{"720p","1080p"},[]string{"5","10"},true,70),
		makeModel("wan27","Wan 2.7","video","通用文生与图生视频",[]string{"text-to-video","image-to-video"},[]string{"720p"},[]string{"5","10"},false,80),
		makeModel("wan_ultra_fast","Wan Ultra Fast","video","低成本快速预览",[]string{"text-to-video","image-to-video"},[]string{"480p","720p"},[]string{"5"},false,90),
		makeModel("infinitetalk","InfiniteTalk","video","人物口型与说话视频",[]string{"image-to-video","audio-to-video"},[]string{"720p"},[]string{"5","10"},false,100),
		makeModel("nano2","Nano Banana 2","image","高性价比图片生成与编辑",[]string{"text-to-image","image-to-image"},[]string{"1K","2K"},nil,true,110),
		makeModel("nano_pro","Nano Banana Pro","image","高质量图片生成与编辑",[]string{"text-to-image","image-to-image"},[]string{"1K","2K","4K"},nil,true,120),
		makeModel("seedream45","Seedream 4.5","image","商业视觉与中文排版",[]string{"text-to-image","image-to-image"},[]string{"1K","2K","4K"},nil,false,130),
		makeModel("seedream5","Seedream 5","image","新一代高质量图片模型",[]string{"text-to-image","image-to-image"},[]string{"1K","2K","4K"},nil,true,140),
		makeModel("flux2_klein","Flux 2 Klein","image","快速写实图片生成",[]string{"text-to-image","image-to-image"},[]string{"1K","2K"},nil,false,150),
		makeModel("qwen_image","Qwen Image","image","中文语义与文字生成",[]string{"text-to-image","image-to-image"},[]string{"1K","2K"},nil,false,160),
		makeModel("suno","Suno","audio","歌曲与纯音乐生成",[]string{"text-to-audio"},nil,nil,false,170),
		makeModel("tts","多语言 TTS","audio","多语言语音合成",[]string{"text-to-speech"},nil,nil,false,180),
		makeModel("video_upscale","视频超分","tool","视频清晰度增强",[]string{"video-to-video"},[]string{"1080p","2K","4K"},nil,false,190),
		makeModel("image_tools","图片超分与抠图","tool","图片放大、修复与背景移除",[]string{"image-to-image"},[]string{"2K","4K"},nil,false,200),
	}
	for i := range items {
		switch items[i].Category {
		case "image":
			items[i].MaxReferenceImages = 4
		case "video":
			items[i].MaxReferenceImages = 2
			items[i].SupportsFirstLastFrame = true
		case "person":
			items[i].MaxReferenceImages = 4
			items[i].SupportsPerson = true
			items[i].SupportsFirstLastFrame = true
		}
	}
	items[9].SupportsAudioReference = true
	return items
}

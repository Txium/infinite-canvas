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
	now := time.Now().Format(time.RFC3339)
	if err := repository.SyncDefaultModelCatalog(2, defaultMarketModels(), defaultModelPrices(), []string{"wan27","seedream45","flux2_klein","image_tools","gpt_image2_default","gpt_image2_2k","gpt_image2_4k","gpt_image2_pro_4k"}, now); err != nil { return nil, err }
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
		makeModel("deepseek_v4_flash","DeepSeek V4 Flash","llm","快速低价语言模型",[]string{"text"},nil,nil,true,10), makeModel("deepseek_v4_pro","DeepSeek V4 Pro","llm","高质量推理语言模型",[]string{"text"},nil,nil,false,20), makeModel("gpt54_mini","GPT-5.4 Mini","llm","高性价比通用语言模型",[]string{"text"},nil,nil,false,30), makeModel("gpt55","GPT-5.5","llm","旗舰通用语言模型",[]string{"text"},nil,nil,false,40), makeModel("claude_sonnet46","Claude Sonnet 4.6","llm","写作与代码主力模型",[]string{"text"},nil,nil,true,50), makeModel("claude_opus48","Claude Opus 4.8","llm","复杂推理旗舰模型",[]string{"text"},nil,nil,false,60), makeModel("gemini31_pro","Gemini 3.1 Pro","llm","长上下文多模态模型",[]string{"text"},nil,nil,false,70), makeModel("gemini_flash","Gemini Flash","llm","快速多模态语言模型",[]string{"text"},nil,nil,false,80), makeModel("qwen37_max","Qwen 3.7 Max","llm","中文与复杂任务模型",[]string{"text"},nil,nil,false,90), makeModel("kimi","Kimi","llm","按上游成本加 10%",[]string{"text"},nil,nil,false,100), makeModel("glm","GLM","llm","按上游成本加 10%",[]string{"text"},nil,nil,false,110),
		makeModel("gpt_image2","GPT Image 2","image","日常高清生图与编辑",[]string{"text-to-image","image-to-image"},[]string{"标准"},nil,true,200), makeModel("gpt_image2_hd","GPT Image 2 高清","image","高清图片生成与编辑",[]string{"text-to-image","image-to-image"},[]string{"高清"},nil,false,210), makeModel("mj_standard","Midjourney MJ","image","高审美图片生成",[]string{"text-to-image","image-to-image"},nil,nil,true,220), makeModel("mj_turbo","Midjourney Turbo","image","快速高审美图片生成",[]string{"text-to-image","image-to-image"},nil,nil,false,230), makeModel("nano2_lite","Nano Banana 2 Lite","image","低价智能生图与改图",[]string{"text-to-image","image-to-image"},nil,nil,false,240), makeModel("nano2","Nano Banana 2","image","智能生图与改图",[]string{"text-to-image","image-to-image"},nil,nil,true,250), makeModel("nano_pro","Nano Banana Pro","image","高质量智能改图",[]string{"text-to-image","image-to-image"},nil,nil,false,260), makeModel("seedream5_lite","Seedream 5 Lite","image","快速商业视觉生成",[]string{"text-to-image","image-to-image"},nil,nil,false,270), makeModel("seedream5","Seedream 5 Pro","image","高质量商业视觉生成",[]string{"text-to-image","image-to-image"},nil,nil,false,280), makeModel("flux2","FLUX 2","image","快速写实图片生成",[]string{"text-to-image","image-to-image"},nil,nil,false,290), makeModel("qwen_image","Qwen Image","image","中文语义与文字生成",[]string{"text-to-image","image-to-image"},nil,nil,false,300),
		makeModel("wan_ultra_fast","Wan Ultra Fast 5秒","video","超低价快速视频",[]string{"text-to-video","image-to-video"},[]string{"720p"},[]string{"5"},false,400), makeModel("wan_ultra_fast_10","Wan Ultra Fast 10秒","video","低价十秒视频",[]string{"text-to-video","image-to-video"},[]string{"720p"},[]string{"10"},false,410), makeModel("grok_imagine_video","Grok Imagine Video","video","低价创意视频生成",[]string{"text-to-video","image-to-video"},nil,nil,false,420), makeModel("sd20_fast","Seedance 2.0 人物特惠 720P","video","人物参考特惠线路",[]string{"image-to-video","multimodal"},[]string{"720p"},nil,true,430), makeModel("sd20_face_720","Seedance 2.0 人物 720P","person","人物参考与角色一致性",[]string{"image-to-video","multimodal"},[]string{"720p"},nil,true,440), makeModel("sd20_face_1080","Seedance 2.0 人物 1080P","person","高清人物参考",[]string{"image-to-video","multimodal"},[]string{"1080p"},nil,false,450), makeModel("sd20_standard","Seedance 2.0 Fast","video","快速多模态视频生成",[]string{"text-to-video","image-to-video","multimodal"},nil,nil,false,460), makeModel("sd25","Seedance 2.5","video","新一代多模态视频模型",[]string{"text-to-video","image-to-video"},nil,nil,false,470), makeModel("kling30","Kling 3.0","video","高质量运动与镜头控制",[]string{"text-to-video","image-to-video"},nil,nil,true,480), makeModel("veo31_fast","Veo 3.1 Fast","video","快速电影感视频",[]string{"text-to-video","image-to-video"},nil,nil,true,490), makeModel("veo31_quality","Veo 3.1 Quality","video","高质量电影感视频",[]string{"text-to-video","image-to-video"},nil,nil,false,500), makeModel("hailuo_minimax","Hailuo / MiniMax Video","video","通用创意视频生成",[]string{"text-to-video","image-to-video"},nil,nil,false,510), makeModel("vidu","Vidu","video","快速低价视频生成",[]string{"text-to-video","image-to-video"},nil,nil,false,520),
		makeModel("infinitetalk","InfiniteTalk","person","人物说话与口型驱动",[]string{"image-to-video","audio-to-video"},nil,nil,false,600), makeModel("wan_animate","Wan Animate","person","人物动作生成",[]string{"image-to-video"},nil,nil,false,610), makeModel("lip_sync","Lip Sync","person","音视频口型同步",[]string{"audio-to-video"},nil,nil,false,620), makeModel("talking_photo","图片说话","person","静态人物图片口播",[]string{"image-to-video","audio-to-video"},nil,nil,false,630), makeModel("person_replace","人物替换","person","视频人物替换",[]string{"video-to-video"},nil,nil,false,640), makeModel("motion_transfer","动作迁移","person","人物动作迁移",[]string{"video-to-video"},nil,nil,false,650), makeModel("digital_human","数字人口播","person","数字人视频口播",[]string{"text-to-video","audio-to-video"},nil,nil,false,660),
		makeModel("suno","Suno","music","歌曲与音乐生成",[]string{"text-to-audio"},nil,nil,false,700), makeModel("ai_bgm","AI BGM","music","背景音乐生成",[]string{"text-to-audio"},nil,nil,false,710), makeModel("ai_song","AI 歌曲","music","完整歌曲生成",[]string{"text-to-audio"},nil,nil,false,720), makeModel("music_extend","音乐续写","music","延长已有音乐",[]string{"audio-to-audio"},nil,nil,false,730), makeModel("song_cover","歌曲翻唱","music","声音与风格翻唱",[]string{"audio-to-audio"},nil,nil,false,740), makeModel("music_stems","音乐分轨","music","人声与乐器分离",[]string{"audio-to-audio"},nil,nil,false,750),
		makeModel("tts","AI 配音 TTS","voice","文本转语音",[]string{"text-to-speech"},nil,nil,false,800), makeModel("premium_tts","高品质配音","voice","高自然度配音",[]string{"text-to-speech"},nil,nil,false,810), makeModel("voice_clone","声音克隆","voice","参考音色克隆",[]string{"audio-to-audio"},nil,nil,false,820), makeModel("whisper","Whisper 语音转文字","voice","语音识别与转写",[]string{"audio-to-text"},nil,nil,false,830), makeModel("realtime_voice","实时语音","voice","实时语音交互",[]string{"audio"},nil,nil,false,840), makeModel("multi_voice","多人配音","voice","多角色语音合成",[]string{"text-to-speech"},nil,nil,false,850),
		makeModel("hunyuan3d_rapid","Hunyuan 3D Rapid","3d","快速图片或文字生成 3D",[]string{"text-to-3d","image-to-3d"},nil,nil,false,900), makeModel("tripo3d","Tripo 3D","3d","通用 3D 模型生成",[]string{"text-to-3d","image-to-3d"},nil,nil,false,910), makeModel("seed3d","Seed3D","3d","高质量 3D 生成",[]string{"text-to-3d","image-to-3d"},nil,nil,false,920), makeModel("meshy6","Meshy 6","3d","专业 3D 资产生成",[]string{"text-to-3d","image-to-3d"},nil,nil,false,930),
		makeModel("remove_bg","AI 抠图 / 去背景","tool","智能主体抠图",[]string{"image-to-image"},nil,nil,false,1000), makeModel("replace_bg","换背景","tool","智能替换图片背景",[]string{"image-to-image"},nil,nil,false,1010), makeModel("image_upscale","图片高清 / 超分","tool","图片清晰度增强",[]string{"image-to-image"},nil,nil,false,1020), makeModel("video_upscale","视频高清 / 超分","tool","视频清晰度增强",[]string{"video-to-video"},nil,nil,false,1030), makeModel("outpaint","智能扩图","tool","画面智能扩展",[]string{"image-to-image"},nil,nil,false,1040), makeModel("image_restore","图片修复","tool","老照片与瑕疵修复",[]string{"image-to-image"},nil,nil,false,1050), makeModel("video_extend","视频延长","tool","视频内容续写",[]string{"video-to-video"},nil,nil,false,1060), makeModel("video_edit","视频编辑","tool","自然语言编辑视频",[]string{"video-to-video"},nil,nil,false,1070), makeModel("subtitles","字幕生成","tool","语音识别并生成字幕",[]string{"audio-to-text"},nil,nil,false,1080),
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
	return items
}

func defaultModelPrices() []model.ModelPrice {
	now := time.Now().Format(time.RFC3339)
	price := func(id, modelID, variant, unit string, cents int) model.ModelPrice { return model.ModelPrice{ID:id,ModelID:modelID,Variant:variant,BillingMode:"request",Unit:unit,PriceCredits:cents,Currency:"CNY",Enabled:true,CreatedAt:now,UpdatedAt:now} }
	return []model.ModelPrice{
		price("price_deepseek_v4_flash_in","deepseek_v4_flash","输入","百万 Token",120), price("price_deepseek_v4_flash_out","deepseek_v4_flash","输出","百万 Token",240), price("price_deepseek_v4_pro_in","deepseek_v4_pro","输入","百万 Token",350), price("price_deepseek_v4_pro_out","deepseek_v4_pro","输出","百万 Token",700), price("price_gpt54_mini_in","gpt54_mini","输入","百万 Token",590), price("price_gpt54_mini_out","gpt54_mini","输出","百万 Token",3500), price("price_gpt55_in","gpt55","输入","百万 Token",3900), price("price_gpt55_out","gpt55","输出","百万 Token",23500), price("price_claude_sonnet46_in","claude_sonnet46","输入","百万 Token",2300), price("price_claude_sonnet46_out","claude_sonnet46","输出","百万 Token",11500), price("price_claude_opus48_in","claude_opus48","输入","百万 Token",3900), price("price_claude_opus48_out","claude_opus48","输出","百万 Token",19500), price("price_gemini31_pro_in","gemini31_pro","输入","百万 Token",1500), price("price_gemini31_pro_out","gemini31_pro","输出","百万 Token",9200), price("price_gemini_flash_in","gemini_flash","输入","百万 Token",300), price("price_gemini_flash_out","gemini_flash","输出","百万 Token",1800), price("price_qwen37_max_in","qwen37_max","输入","百万 Token",1900), price("price_qwen37_max_out","qwen37_max","输出","百万 Token",5800),
		price("price_gpt_image2","gpt_image2","", "张",15), price("price_gpt_image2_hd","gpt_image2_hd","", "张",20), price("price_mj_standard","mj_standard","", "次",49), price("price_mj_turbo","mj_turbo","", "次",89), price("price_nano2_lite","nano2_lite","", "张",15), price("price_nano2","nano2","", "张",35), price("price_nano_pro","nano_pro","", "张",69), price("price_seedream5_lite","seedream5_lite","", "张",25), price("price_seedream5","seedream5","", "张",39), price("price_flux2","flux2","", "张起",15), price("price_qwen_image","qwen_image","", "张起",20),
		price("price_wan_ultra_fast_5","wan_ultra_fast","", "次",49), price("price_wan_ultra_fast_10","wan_ultra_fast_10","", "次",89), price("price_grok_imagine_video","grok_imagine_video","", "次起",59), price("price_sd20_fast","sd20_fast","", "次",199), price("price_sd20_face_720","sd20_face_720","", "次",299), price("price_sd20_face_1080","sd20_face_1080","", "次",319), price("price_sd20_standard","sd20_standard","", "秒起",60), price("price_sd25","sd25","", "秒起",80), price("price_kling30","kling30","", "秒起",65), price("price_veo31_fast","veo31_fast","", "秒起",75), price("price_veo31_quality","veo31_quality","", "秒起",150), price("price_hailuo_minimax","hailuo_minimax","", "秒起",69), price("price_vidu","vidu","", "秒起",39),
		price("price_infinitetalk","infinitetalk","", "秒",25), price("price_wan_animate","wan_animate","", "秒",35), price("price_lip_sync","lip_sync","", "秒起",19), price("price_talking_photo","talking_photo","", "次起",29), price("price_person_replace","person_replace","", "秒起",39), price("price_motion_transfer","motion_transfer","", "秒起",39), price("price_digital_human","digital_human","", "秒起",29),
		price("price_suno","suno","", "次",89), price("price_ai_bgm","ai_bgm","", "次起",39), price("price_ai_song","ai_song","", "次起",69), price("price_music_extend","music_extend","", "次起",39), price("price_song_cover","song_cover","", "次起",69), price("price_music_stems","music_stems","", "次起",29),
		price("price_tts","tts","", "次起",5), price("price_premium_tts","premium_tts","", "次起",10), price("price_voice_clone","voice_clone","", "次起",19), price("price_whisper","whisper","", "次起",5), price("price_realtime_voice","realtime_voice","", "次起",10), price("price_multi_voice","multi_voice","", "次起",19),
		price("price_hunyuan3d_rapid","hunyuan3d_rapid","", "次起",29), price("price_tripo3d","tripo3d","", "次起",49), price("price_seed3d","seed3d","", "次起",69), price("price_meshy6","meshy6","", "次起",599),
		price("price_remove_bg","remove_bg","", "张",5), price("price_replace_bg","replace_bg","", "张",10), price("price_image_upscale","image_upscale","", "张起",10), price("price_video_upscale","video_upscale","", "秒起",19), price("price_outpaint","outpaint","", "张",10), price("price_image_restore","image_restore","", "张",10), price("price_video_extend","video_extend","", "秒起",39), price("price_video_edit","video_edit","", "次起",39), price("price_subtitles","subtitles","", "次起",5),
	}
}

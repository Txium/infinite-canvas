# 灵感画布全链路排查与扩展验收

## 边界与验收规则

- 仅 staging；Production 不修改。保留 Image2 已工作路径、钱包与金融记录。
- 配置就绪不等于真实可用。未经当前 Provider/Adapter/档位完整实测的能力统一 UNVERIFIED；失败测试不能记为已验证成功。
- 每次付费测试记录预算、规格、local/upstream task、请求画质、原始/保存媒体元数据、结算或解冻证据。未知状态不重复下单。
- 本次尚未新增付费生成，等待测试总预算；也未创建收费 Worker 或修改线上数据库。

## 已取得的基线证据

| 层级 | 现有实现/证据 | 缺口与风险 |
| --- | --- | --- |
| Web/API | Dockerfile 将 Next.js 3000 与 Go 8080 放在同一容器，render.yaml 为 staging Free | 无独立媒体/音频/GPU Worker；镜像未安装 ffmpeg/ffprobe |
| 数据库 | 配置声明 PostgreSQL；VideoTask/ModelProvider/MarketModel/ModelVariant/ModelRoute 已存在 | 实际数据库主机、staging/production 隔离与备份尚未现场核实；不能假定 Neon |
| Provider | staging 管理页显示 302、WaveSpeed、LEC、seedance.nz 均 api_key_present=true | 有 Key 不证明每个 endpoint 授权有效；302 余额权限 403 不等同生成鉴权失败 |
| Provider URL | 302=https://api.302.ai；WaveSpeed=https://api.wavespeed.ai/api/v3；LEC=https://api.paipu.net；seedance.nz=https://api.seedance.nz | 本轮仅只读检查，未同步目录或改线路 |
| Registry | service/model_market.go 按 variant→route→provider→upstream model 解析；route.Protocol 存在 | availability 主要依赖启用、Key、URL与价格，缺少验收证据、完整 Adapter/endpoint 校验 |
| MJ | handler/midjourney_302.go 有独立 JSON Adapter、MJ endpoint 与双鉴权，不是 Image2 字符串替换 | 只支持无参考图 JSON；提交后同步轮询；取消/10分钟超时直接退款，未持久化独立 MJ 上游任务供恢复 |
| 视频状态 | service/video_task.go 已有 reconciling 和后台低频查询 | timeout 仍归 failed；finished 未明确成功映射；完成但无 URL 仍可能结算；只扫描48小时窗口 |
| 永久审计 | model/video_task.go 有 provider关联、upstream model/task、请求响应、账务字段 | 完成任务30天自动删除，不能满足永久审计；缺阶段事件/原始与保存URL/媒体元数据 |
| 媒体保存 | handler/generated_media.go 原字节下载并上传，未在此函数重编码 | 不代表结果解析和前端所有路径不降质；没有原始/保存元数据或校验和对比证据 |
| 结果解析 | handler/video_task.go 优先 video_url/url，前端 collectMediaUrls 遍历任意 URL | original_url 未优先；存在误选非原始输出的风险，需 Provider 样本复现 |
| 分辨率 | web/src/lib/market-video-resolution.ts 对部分档位写死720p/1080p | 缺前端选项→payload→实际媒体的持久化追踪；不可据 UI 证明真1080p |
| 全景 | Panorama节点、2:1球形提示词、canvas-panorama-viewer 已有 | 不应重建；需要明确水平360与完整球形360×180的入口和验收，720是俗称 |

## 验证清单

- Image2 Low 1K：上一轮有真实成功证据；本轮保护性回归待做。其他 Image2 档位 UNVERIFIED。
- MJ Standard/Turbo：此前有鉴权失败记录；本轮成功验收 UNVERIFIED。
- LEC 900：上一轮有15.103991秒、1280×720完整播放、¥1.69单笔结算证据；不能据此证明1080p。
- 用户提及 Seedance满血、933、海螺：记录为已尝试，需核对精确variant/provider/task；未获得当前成功证据前均 UNVERIFIED。
- 模型广场其他模型均 UNVERIFIED，不用卡片或余额查询作为成功证据。

## 分阶段交付与真实验收门槛

1. 基线与可观测性：补数据库/部署只读核对、staging环境隔离、统一Registry验证证据；记录 REQUEST_RECEIVED → MODEL_RESOLVED → PROVIDER_RESOLVED → PROVIDER_CONFIG_OK → ADAPTER_SELECTED → PARAM_VALIDATION_PASSED → PAYLOAD_BUILT → UPSTREAM_REQUEST_START → UPSTREAM_RESPONSE_RECEIVED → UPSTREAM_TASK_ID_SAVED。只记录Key存在与否；敏感请求/签名URL脱敏。
2. MJ与视频正确性：持久化接单信息，显式Adapter，错误分类；timeout未知不退款；补偿查询；完成缺URL不丢失任务；保留原始URL并实测原文件和Canvas文件尺寸/码率/帧率/编码/时长/大小。
3. 原生Media Worker：在视频节点提供逐帧/时间码、原片首尾当前帧、裁剪、抽音频、移除独立字幕轨；使用ffprobe真实fps，输出新增节点且保留来源；尾帧继续视频。VFR视频需单独处理时间戳，不能把平均fps当每帧恒定时距。
4. 音频Worker：Whisper ASR、pyannote说话人时间段、Demucs人声/BGM；重叠语音分离单独能力，不能伪装为diarization。逐个核对代码及权重许可、硬件、模型下载条款后接入。
5. Voice Profile：角色绑定Provider/Voice ID/描述/语速/情绪/风格，TTS、Voice Design、Clone分开；试听生成需费用确认。评估CosyVoice与ElevenLabs，不承诺未经验证能力。
6. 图片创作：复用全景节点；增加九宫格分镜、细节特写、人物三视图工作流，保留参考图；等待LibTV具体链接核对交互，不复制未知许可代码。
7. 烧录字幕：仅作为后续独立AI字幕修复能力；不能将删除subtitle stream描述成去除画面字幕。

## 未决部署选择

- Media Worker运行位置、持久化队列、对象存储授权方式，需基于真实部署检查决定；不能沿用浏览器Key或把临时磁盘当持久存储。
- Audio/GPU Worker的可用机器与费用上限尚未授权。不自动购买Render实例、GPU或其他服务。
- 每个修复的状态必须区分：代码已改／回归通过／staging已部署／真实生成通过；只有最后一项可宣布对应模型链路验收完成。

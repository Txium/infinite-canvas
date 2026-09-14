# 云端媒体能力交付边界

本批已发布staging：`bd6eddc`，Render部署`dep-dajp02fqj5pc73eh17v0`确认Live。新音色面板通过能力与档案GET加载十项待接入能力，无读取错误；未执行档案写入验收。不购买实例/GPU/存储，不调用付费模型。Production不变。

## P0/P1 基础处理

- 视频节点「视频处理」：播放器、时间轴、±1/5秒、逐帧、HH:MM:SS:FF与帧编号。ffprobe返回fps/duration/width/height/codec/bitrate；VFR使用真实帧时间戳，时间码是标称fps而非广播drop-frame。
- 按需生成8格160×90缩略图拼图，仅用作预览；不会预先保存全部高清帧。当前实现为手动点击加载，非自动缓存服务。
- 首帧、尾帧、当前帧从原视频解码为PNG；尾帧定位结束前0.08秒，未实现黑帧识别。
- 截帧新节点保存source_video_node_id/timestamp/operation/output_url，同时保留已有metadata字段。
- 当前帧可连接新图片或视频节点；尾帧可连接下一镜的firstFrameNodeId。仅创建草稿，不自动生成或扣费。
- 裁剪生成新视频：默认流复制；关键帧边界不保证逐帧精确；显式精确模式才H.264/AAC重新编码。
- 音频提取：AAC/MP3复制原音轨，其余转AAC；音频节点保存来源、时长、编码。无声视频复制视频轨。
- 删除独立字幕轨使用重新封装；不等于擦除烧录字幕。MP4不兼容编码会失败，不能声称支持所有格式。
- 独立Media Worker接收API转发的原文件，不调用Provider，不进入钱包。单任务并发、100MB/10分钟上限、子进程120秒超时；Web/API不执行FFmpeg。
- Worker工作目录区分于对象存储的原始/最终文件。正常任务立即清理；异常退出残留job目录1小时TTL，每分钟清理一次并跳过当前活跃任务。浏览器缩略图URL关闭面板时释放；没有永久缩略图云缓存。

**部署依赖仍未满足：MEDIA_WORKER_URL/TOKEN及独立FFmpeg服务未配置，所以线上媒体运算目前不可用。** 不自动升级当前免费实例。尚未建立持久Media作业队列、租约及进程重启后继续处理；短任务仍是独立Worker内同步响应。

## P2/P3 接口预留

登录后GET `/api/v1/media-ai/capabilities` 返回十种COMING_SOON能力：asr、tts、voice_design、voice_clone、bgm_separation、speaker_diarization、speech_separation、remove_burned_subtitle、video_repair、video_inpainting。

POST `/api/v1/media-ai/tasks` 解析MediaAIRequest，通过明确operation选择DisabledMediaAIAdapter；已知未配置能力返回503，未知operation返回400。不会伪造queued任务、调用Provider或扣费。

- Adapter契约：Submit(context, request) → upstream ID；Poll(context, ID) → typed task。
- 任务结构预留：用户/源节点/operation/adapter/upstream ID/status/outputs/segments/error/timestamps。
- 输出包含音频/视频URL、label、Voice ID、duration、codec；语音段含speaker_id/start_time/end_time/text。
- 状态预留queued/processing/completed/failed/timed_out_unknown/cancelled；本批**没有实现这些状态的执行器或资金结算**，不能直接将Adapter启用。
- ASR/分离模型未下载；Self-hosted、官方API、中转站均未实现具体Adapter。GPU不是所有能力的硬性前提，后续按模型资源需求选择，当前不采购。
- Voice Design试听/选择、Voice Clone参考声音上传及授权确认、TTS提交/结果建节点，均未实现真实执行流程。面板显示待接入，不提供可点击的假生成。

## 角色音色

图片/音频/视频节点工具栏「音色与AI」提供Voice Profile表单和已有档案选择。默认以当前节点ID作为character_id，也允许填写统一角色ID；未自动改造现有角色资产库。

GET/POST `/api/v1/voice-profiles` 查询/保存当前用户的档案。复合主键(user_id, character_id)隔离用户；不接受客户端指定用户ID。保存不验证Voice ID、不生成试听，也不允许填写Provider Key。

## 图片工具

保留九宫格分镜、细节特写、人物三视图、球形全景。新增人像质感/打光参考图提示词工作流。集中提供已有多角度、元素编辑、宫格切分、放大/超分入口。

真实图层分离、片段重拍、视频超分具体Provider还未实现；不能用这些预设声称复刻LibTV全部能力。

## 验收

只运行类型检查、单元/接口/数据库结构回归，不调用Image2、MJ、Seedance、933、海螺。本机无FFmpeg，实际原片输出尚未验证。数据库结构回归使用SQLite；不代表云端PostgreSQL迁移已完成。上线和真实效果仍需分开验收。

# 灵感画布 Media Worker

独立于 Canvas Web/API、Provider 与钱包的 FFmpeg 服务，不调用付费模型。

## 启动

1. 安装 Docker；从当前目录构建：`docker build -t canvas-media-worker .`
2. 为 Worker 与 Canvas Go API 配置相同的随机 `MEDIA_WORKER_TOKEN`（至少32字符），不要提交到仓库或填进浏览器。两端同时设置 `MEDIA_WORKER_MAX_MB`；测试环境当前为25，升级资源后可直接改为100、500或更高，无需改代码。
3. 启动独立容器并注入该环境变量；容器监听8090。建议使用私有网络、1GB以上内存、独立临时盘并限制CPU/内存。
4. 仅在 staging Go API 设置 `MEDIA_WORKER_URL=http://你的私网Worker:8090` 与 `MEDIA_WORKER_TOKEN`。Production 不配置。
5. 页面打开视频节点 → 视频处理。未配置时明确提示，不会转向任何付费Provider。

不提供默认口令，不自动创建云端资源。跨公网必须通过HTTPS；只开放给API网络。

## 当前实现

- 时间轴按需输出8格160×90低清拼图；PNG原帧只在截帧时提取。AAC/MP3原音轨复制，其他音轨转AAC；可单独输出无声视频。
- 专属临时目录canvas-media-worker/job-*：任务结束立即清理，异常残留1小时TTL，每分钟扫描，跳过活跃任务；绝不清理对象存储original/final。该目录不用于用户长期存储。

- 原视频上传、ffprobe元数据和真实帧时间戳；最多100MB、10分钟、单Worker同时一个任务，每个子进程最多120秒。
- 原分辨率PNG首帧、尾部约0.08秒和当前时间点截帧；不会截图浏览器播放器。
- MP4片段裁剪：默认流复制（边界受关键帧限制）；明确选择精确模式才用H.264 CRF18/AAC编码，原文件不覆盖。
- 提取第一条音频轨：AAC直接copy到M4A，其他编码转AAC。
- 移除独立字幕轨：只映射视频与音频并stream copy。非MP4兼容编码可能失败，不会静默转码。
- 输出按现有画布文件存储方式保存，带sourceVideoId和mediaOperation；可由尾帧创建下一镜节点。

## 明确限制

- 首尾帧目前为时间定位，不承诺自动检测黑帧；无输出会报错。VFR逐帧使用时间戳，显示时间码为标称平均fps非广播drop-frame时间码。
- 当前短任务同步返回，尚无持久化任务队列或断点恢复；断网重做媒体处理不会产生模型扣费。长视频/大量用户需先升级为持久队列。
- ASR、说话人识别、BGM分离、重叠语音分离、音色设计、烧录字幕修复不属于此Worker，尚未实现。
- 此版本未自动创建/部署Worker，线上缺少配置时处理不可用；不能把前端入口称为部署完成。

## 验证

`python -m unittest test_server.py` 检查参数、原分辨率命令、stream copy及范围校验。

真实FFmpeg验收需安装ffmpeg/ffprobe，先使用自有短样片验证：PNG尺寸等于原片、AAC提取未重编码、裁剪时长和字幕stream移除。不要仅以单元测试替代媒体验收。

## 许可证检查入口

- FFmpeg：<https://ffmpeg.org/legal.html>；实际发行包的GPL/LGPL与编码库构建选项需一并保留告知。
- Whisper代码与权重：<https://github.com/openai/whisper>（MIT）。
- pyannote Community-1：<https://huggingface.co/pyannote/speaker-diarization-community-1>（模型卡CC-BY-4.0；下载需接受条款）。
- Demucs：<https://github.com/facebookresearch/demucs>；代码MIT，部署前仍需核对所选权重与依赖。
- CosyVoice/SpeechBrain及具体权重许可尚未完成逐项核查，不随本Worker打包。

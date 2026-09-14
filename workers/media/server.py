"""Isolated, bounded FFmpeg worker. Accepts uploaded bytes, never remote URLs."""
import hmac
import json
import math
import os
import subprocess
import tempfile
import threading
import shutil
import time
from email.parser import BytesParser
from email.policy import default
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

MAX_BYTES = min(100, max(1, int(os.environ.get("MEDIA_MAX_MB", "25")))) << 20
SLOT = threading.BoundedSemaphore(1)
FORMATS = "mov,matroska,webm,avi"
TEMP_ROOT = Path(tempfile.gettempdir()) / "canvas-media-worker"
TEMP_TTL = 3600
ACTIVE_DIRS = set()
TEMP_LOCK = threading.Lock()


class MediaError(ValueError):
    def __init__(self, code, message):
        super().__init__(message)
        self.code = code


def cleanup_temporary(now=None):
    """Only expired worker-owned job directories; never originals or final storage."""
    now = time.time() if now is None else now
    if not TEMP_ROOT.exists() or TEMP_ROOT.is_symlink():
        return
    root = TEMP_ROOT.resolve()
    with TEMP_LOCK:
        for item in root.iterdir():
            if item.name in ACTIVE_DIRS or not item.name.startswith("job-") or item.is_symlink() or not item.is_dir():
                continue
            if item.resolve().parent == root and now - item.stat().st_mtime > TEMP_TTL:
                shutil.rmtree(item)


def cleanup_loop():
    while True:
        try:
            cleanup_temporary()
        except OSError:
            pass
        time.sleep(60)


def run(args):
    result = subprocess.run(args, capture_output=True, timeout=120, check=False)
    if result.returncode:
        raise ValueError("无法处理该视频格式或文件已损坏，原视频未修改")
    return result.stdout


def probe(path):
    raw = run(["ffprobe", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", FORMATS,
               "-show_streams", "-show_format", "-of", "json", str(path)])
    data = json.loads(raw)
    video = next((s for s in data["streams"] if s["codec_type"] == "video"), None)
    if not video:
        raise ValueError("文件不含视频轨")
    duration = float(data["format"].get("duration", video.get("duration", 0)))
    if not math.isfinite(duration) or not 0 < duration <= 600:
        raise ValueError("首阶段支持10分钟以内的视频")
    numerator, denominator = map(float, video.get("avg_frame_rate", "0/1").split("/"))
    fps = numerator / denominator if denominator else 0
    if not 0 < fps <= 240:
        raise ValueError("无法读取有效视频帧率")
    frame_data = json.loads(run(["ffprobe", "-v", "error", "-protocol_whitelist", "file,pipe", "-format_whitelist", FORMATS,
                                "-select_streams", "v:0", "-show_entries", "frame=best_effort_timestamp_time", "-of", "json", str(path)]))
    timestamps = [float(f["best_effort_timestamp_time"]) for f in frame_data.get("frames", []) if "best_effort_timestamp_time" in f]
    origin = timestamps[0] if timestamps else 0
    timestamps = [round(t-origin, 6) for t in timestamps]
    return {"duration": duration, "fps": fps, "frameTimes": timestamps, "width": video["width"], "height": video["height"],
            "codec": video["codec_name"], "bitrate": data["format"].get("bit_rate"), "bytes": path.stat().st_size,
            "audioCodec": next((s["codec_name"] for s in data["streams"] if s["codec_type"] == "audio"), None),
            "subtitleTracks": sum(s["codec_type"] == "subtitle" for s in data["streams"])}


def build_command(source, output, action, meta, start, end, precise=False):
    if not math.isfinite(start) or not math.isfinite(end):
        raise ValueError("时间参数无效")
    args = ["ffmpeg", "-nostdin", "-v", "error", "-threads", "1", "-filter_threads", "1", "-protocol_whitelist", "file,pipe", "-format_whitelist", FORMATS, "-i", str(source)]
    if action in ("first", "last", "frame"):
        position = 0 if action == "first" else max(0, meta["duration"] - .08) if action == "last" else start
        if not 0 <= position < meta["duration"]:
            raise ValueError("截帧时间超出视频范围")
        args += ["-ss", str(position), "-map", "0:v:0", "-frames:v", "1", "-c:v", "png"]
    elif action == "thumbnails":
        interval = max(meta["duration"] / 8, .001)
        args += ["-map", "0:v:0", "-vf", f"fps=1/{interval},scale=160:90:force_original_aspect_ratio=decrease,pad=160:90:(ow-iw)/2:(oh-ih)/2,tile=8x1", "-frames:v", "1", "-c:v", "png"]
    elif action == "clip":
        if not 0 <= start < end <= meta["duration"]:
            raise ValueError("请设置有效起止时间")
        args += ["-ss", str(start), "-t", str(end-start), "-map", "0:v:0", "-map", "0:a:0?"]
        args += ["-c:v", "libx264", "-crf", "18", "-preset", "fast", "-c:a", "aac"] if precise else ["-c", "copy"]
        args += ["-avoid_negative_ts", "make_zero", "-movflags", "+faststart"]
    elif action == "audio":
        if not meta["audioCodec"]:
            raise MediaError("VIDEO_HAS_NO_AUDIO", "原视频没有音频轨，无法提取音频；没有创建音频结果")
        args += ["-map", "0:a:0", "-vn", "-c:a", "copy" if meta["audioCodec"] in ("aac", "mp3") else "aac"]
    elif action == "mute":
        args += ["-map", "0:v:0", "-c:v", "copy", "-an", "-sn", "-movflags", "+faststart"]
    elif action == "subtitles":
        if not meta["subtitleTracks"]:
            raise ValueError("没有独立字幕轨；烧录在画面中的字幕不能用此功能移除")
        args += ["-map", "0:v", "-map", "0:a?", "-c", "copy", "-sn"]
    else:
        raise ValueError("不支持的处理操作")
    return args + ["-y", str(output)]


class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path != "/health":
            return self.reply(404, {"msg": "接口不存在"})
        ready = bool(shutil.which("ffmpeg") and shutil.which("ffprobe"))
        return self.reply(200 if ready else 503, {"ready": ready, "ffmpeg_available": bool(shutil.which("ffmpeg")), "ffprobe_available": bool(shutil.which("ffprobe"))})

    def log_message(self, *_):
        pass  # Never log authorization or media payloads.

    def reply(self, status, data, content_type="application/json"):
        if isinstance(data, dict):
            data = json.dumps(data, ensure_ascii=False).encode()
        self.send_response(status)
        self.send_header("Content-Type", content_type)
        self.send_header("Content-Length", str(len(data)))
        self.send_header("Cache-Control", "no-store")
        self.end_headers()
        self.wfile.write(data)

    def do_POST(self):
        token = os.environ.get("MEDIA_WORKER_TOKEN", "")
        if not token or not hmac.compare_digest(self.headers.get("Authorization", ""), "Bearer " + token):
            return self.reply(401, {"msg": "Worker认证失败"})
        if self.path != "/process":
            return self.reply(404, {"msg": "接口不存在"})
        if not SLOT.acquire(blocking=False):
            return self.reply(429, {"msg": "媒体Worker正在处理其他任务，请稍后重试"})
        try:
            size = int(self.headers.get("Content-Length", "0"))
            if not 0 < size <= MAX_BYTES:
                raise MediaError("MEDIA_FILE_TOO_LARGE", f"本Worker支持{MAX_BYTES >> 20}MB以内的上传，请选择更短的片段")
            self.connection.settimeout(30)
            content_type = self.headers.get("Content-Type", "")
            if not content_type.startswith("multipart/form-data;") or "\r" in content_type or "\n" in content_type:
                raise ValueError("必须上传原始视频文件")
            body = self.rfile.read(size)
            if len(body) != size:
                raise ValueError("上传未完成")
            form = BytesParser(policy=default).parsebytes(("Content-Type: " + content_type + "\r\nMIME-Version: 1.0\r\n\r\n").encode() + body)
            values = {part.get_param("name", header="content-disposition"): part.get_payload(decode=True) for part in form.iter_parts()}
            TEMP_ROOT.mkdir(mode=0o700, parents=True, exist_ok=True)
            if TEMP_ROOT.is_symlink():
                raise ValueError("临时目录无效")
            cleanup_temporary()
            with tempfile.TemporaryDirectory(prefix="job-", dir=TEMP_ROOT) as directory:
                with TEMP_LOCK:
                    ACTIVE_DIRS.add(Path(directory).name)
                source = Path(directory) / "input"
                source.write_bytes(values.get("file", b""))
                meta = probe(source)
                action = values.get("action", b"probe").decode()
                if action == "probe":
                    return self.reply(200, meta)
                extension = "png" if action in ("first", "last", "frame", "thumbnails") else "m4a" if action == "audio" else "mp4"
                if action == "audio" and meta["audioCodec"] == "mp3":
                    extension = "mp3"
                output = Path(directory) / ("output." + extension)
                run(build_command(source, output, action, meta, float(values.get("start", b"0")), float(values.get("end", b"0")), values.get("precise") == b"true"))
                if not output.exists() or not 0 < output.stat().st_size <= MAX_BYTES:
                    raise ValueError("未得到有效输出；请调整时间范围或缩短片段")
                return self.reply(200, output.read_bytes(), {"png": "image/png", "m4a": "audio/mp4", "mp3": "audio/mpeg", "mp4": "video/mp4"}[extension])
        except MediaError as error:
            self.reply(422, {"error_code": error.code, "msg": str(error)})
        except subprocess.TimeoutExpired:
            self.reply(504, {"error_code": "MEDIA_PROCESSING_TIMEOUT", "msg": "媒体处理超时，原文件保留，可缩短片段后重试"})
        except (ValueError, KeyError, TypeError, AttributeError, OSError):
            self.reply(400, {"msg": "处理失败：请检查视频格式、音轨和时间范围；最长10分钟/100MB，原文件保留"})
        finally:
            if "directory" in locals():
                with TEMP_LOCK:
                    ACTIVE_DIRS.discard(Path(directory).name)
            SLOT.release()


if __name__ == "__main__":
    if len(os.environ.get("MEDIA_WORKER_TOKEN", "")) < 32:
        raise SystemExit("MEDIA_WORKER_TOKEN must contain at least 32 characters")
    threading.Thread(target=cleanup_loop, daemon=True).start()
    ThreadingHTTPServer((os.environ.get("BIND_HOST", "127.0.0.1"), int(os.environ.get("PORT", "8090"))), Handler).serve_forever()

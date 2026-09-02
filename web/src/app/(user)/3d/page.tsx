"use client";

import { App, Button, Input, Upload } from "antd";
import { Box, Download, LoaderCircle, UploadCloud } from "lucide-react";
import { useEffect, useState } from "react";

import { requestVideoGeneration } from "@/services/api/video";
import { useEffectiveConfig } from "@/stores/use-config-store";
import type { ReferenceImage } from "@/types/image";

export default function ThreeDGenerationPage() {
    const { message } = App.useApp();
    const effectiveConfig = useEffectiveConfig();
    const [image, setImage] = useState<ReferenceImage | null>(null);
    const [prompt, setPrompt] = useState("");
    const [resultUrl, setResultUrl] = useState("");
    const [loading, setLoading] = useState(false);
    const [marketVariant, setMarketVariant] = useState("");
    const model = marketVariant || effectiveConfig.videoModel || "seed3d_20__01";

    useEffect(() => setMarketVariant(new URLSearchParams(window.location.search).get("marketVariant") || ""), []);

    const generate = async () => {
        if (!image) { message.info("请先上传一张物体参考图"); return; }
        setLoading(true);
        setResultUrl("");
        try {
            const result = await requestVideoGeneration({ ...effectiveConfig, model, videoModel: model }, prompt, [image]);
            setResultUrl(result.url);
            message.success("3D 模型已生成，可下载 GLB 文件");
        } catch (error) {
            message.error(error instanceof Error ? error.message : "3D 模型生成失败");
        } finally {
            setLoading(false);
        }
    };

    return <main className="h-full overflow-y-auto px-4 py-6 md:px-8">
        <section className="mx-auto max-w-3xl rounded-2xl border border-stone-200 p-6 dark:border-stone-800">
            <div className="mb-6 flex items-center gap-3"><Box className="size-6" /><div><h1 className="text-xl font-semibold">Seed3D 2.0 图片转 3D</h1><p className="text-sm text-stone-500">上传单个物体图片，生成可下载的 GLB 三维模型。</p></div></div>
            <Upload.Dragger accept="image/*" maxCount={1} showUploadList={Boolean(image)} beforeUpload={(file) => {
                const reader = new FileReader();
                reader.onload = () => setImage({ id: `${Date.now()}`, name: file.name, type: file.type, dataUrl: String(reader.result || "") });
                reader.readAsDataURL(file);
                return false;
            }} onRemove={() => { setImage(null); setResultUrl(""); }}>
                <UploadCloud className="mx-auto mb-2 size-8 text-stone-400" /><p>点击或拖入物体参考图</p>
            </Upload.Dragger>
            <Input.TextArea className="mt-4" value={prompt} onChange={(event) => setPrompt(event.target.value)} rows={4} placeholder="可选：描述希望保留的材质、形状或细节" />
            <Button className="mt-4" type="primary" block disabled={!image || loading} icon={loading ? <LoaderCircle className="size-4 animate-spin" /> : <Box className="size-4" />} onClick={() => void generate()}>{loading ? "生成中" : "生成 3D 模型"}</Button>
            {resultUrl ? <a className="mt-4 flex items-center justify-center gap-2 rounded-xl border border-stone-200 p-3 dark:border-stone-800" href={resultUrl} target="_blank" rel="noreferrer" download><Download className="size-4" />下载 GLB 模型</a> : null}
        </section>
    </main>;
}

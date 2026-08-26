"use client";

import { App, Button, Empty, Input, Segmented, Tag } from "antd";
import { Check, ImageIcon, RefreshCw, Search, Settings2, Video } from "lucide-react";
import { useMemo, useState } from "react";

import { filterModelsByCapability, normalizeLocalChannels, useConfigStore, useEffectiveConfig, type ModelCapability } from "@/stores/use-config-store";

type MarketModel = {
    key: string;
    model: string;
    channelId: string;
    channelName: string;
    capability: "image" | "video";
};

export default function ModelMarketPage() {
    const { message } = App.useApp();
    const [capability, setCapability] = useState<"image" | "video">("image");
    const [query, setQuery] = useState("");
    const config = useConfigStore((state) => state.config);
    const updateConfig = useConfigStore((state) => state.updateConfig);
    const openConfigDialog = useConfigStore((state) => state.openConfigDialog);
    const effectiveConfig = useEffectiveConfig();

    const allModels = useMemo<MarketModel[]>(() => {
        const channels = effectiveConfig.channelMode === "remote"
            ? effectiveConfig.publicChannels.map((channel) => ({ id: channel.id || "remote", name: channel.name || "云端渠道", protocol: channel.protocol || "openai", purpose: "general", models: channel.models || [] }))
            : normalizeLocalChannels(config);
        return channels.flatMap((channel) => (["image", "video"] as const).flatMap((kind) => {
            if (channel.purpose !== "general" && channel.purpose !== kind) return [];
            return filterModelsByCapability(channel.models || [], kind, channel.protocol || "openai").map((model) => ({
                key: `${channel.id}::${kind}::${model}`,
                model,
                channelId: channel.id || "",
                channelName: channel.name || "模型渠道",
                capability: kind,
            }));
        }));
    }, [config, effectiveConfig]);

    const visibleModels = useMemo(() => {
        const keyword = query.trim().toLowerCase();
        return allModels.filter((item) => item.capability === capability && (!keyword || `${item.model} ${item.channelName}`.toLowerCase().includes(keyword)));
    }, [allModels, capability, query]);

    const activeModel = capability === "image" ? config.imageModel : config.videoModel;
    const activeChannelId = capability === "image" ? config.imageChannelId : config.videoChannelId;
    const chooseModel = (item: MarketModel) => {
        if (item.capability === "image") {
            updateConfig("imageModel", item.model);
            updateConfig("imageChannelId", item.channelId);
        } else {
            updateConfig("videoModel", item.model);
            updateConfig("videoChannelId", item.channelId);
        }
        updateConfig("model", item.model);
        updateConfig("activeChannelId", item.channelId);
        message.success(`已切换到 ${item.model}`);
    };

    const imageCount = allModels.filter((item) => item.capability === "image").length;
    const videoCount = allModels.filter((item) => item.capability === "video").length;

    return (
        <main className="min-h-[calc(100vh-64px)] bg-stone-50 px-4 py-8 dark:bg-stone-950 md:px-8">
            <div className="mx-auto max-w-7xl">
                <section className="mb-7 flex flex-col gap-5 rounded-3xl border border-stone-200 bg-white p-6 shadow-sm dark:border-stone-800 dark:bg-stone-900 md:flex-row md:items-end md:justify-between">
                    <div>
                        <div className="mb-2 flex items-center gap-2 text-sm text-stone-500"><RefreshCw className="size-4" /> 模型来自你已经接入的通用 Key</div>
                        <h1 className="text-3xl font-semibold tracking-tight text-stone-950 dark:text-white">模型广场</h1>
                        <p className="mt-2 max-w-2xl text-sm leading-6 text-stone-500">图片和视频模型分开展示。选择模型不会购买新的 Key，只会切换当前画布使用的模型。</p>
                    </div>
                    <Button icon={<Settings2 className="size-4" />} onClick={() => openConfigDialog(false)}>管理通用 Key / 拉取模型</Button>
                </section>

                <div className="mb-6 flex flex-col gap-3 md:flex-row md:items-center md:justify-between">
                    <Segmented
                        size="large"
                        value={capability}
                        onChange={(value) => setCapability(value as "image" | "video")}
                        options={[
                            { label: `图片模型 ${imageCount}`, value: "image", icon: <ImageIcon className="size-4" /> },
                            { label: `视频模型 ${videoCount}`, value: "video", icon: <Video className="size-4" /> },
                        ]}
                    />
                    <Input allowClear value={query} onChange={(event) => setQuery(event.target.value)} prefix={<Search className="size-4 text-stone-400" />} placeholder="搜索模型或渠道" className="md:max-w-sm" />
                </div>

                {visibleModels.length ? (
                    <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                        {visibleModels.map((item) => {
                            const selected = item.model === activeModel && item.channelId === activeChannelId;
                            return (
                                <article key={item.key} className={`rounded-2xl border bg-white p-5 shadow-sm transition dark:bg-stone-900 ${selected ? "border-blue-500 ring-2 ring-blue-500/15" : "border-stone-200 hover:-translate-y-0.5 hover:shadow-md dark:border-stone-800"}`}>
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex min-w-0 items-center gap-3">
                                            <div className="grid size-11 shrink-0 place-items-center rounded-xl bg-stone-100 dark:bg-stone-800">{item.capability === "image" ? <ImageIcon className="size-5" /> : <Video className="size-5" />}</div>
                                            <div className="min-w-0"><h2 className="truncate font-medium text-stone-950 dark:text-white" title={item.model}>{item.model}</h2><p className="mt-1 truncate text-xs text-stone-500">{item.channelName}</p></div>
                                        </div>
                                        {selected ? <Tag color="blue" icon={<Check className="size-3" />}>当前</Tag> : null}
                                    </div>
                                    <div className="mt-5 flex items-center justify-between gap-3">
                                        <div className="flex flex-wrap gap-2"><Tag>{item.capability === "image" ? "生图" : "视频"}</Tag><Tag>按渠道计费</Tag></div>
                                        <Button type={selected ? "default" : "primary"} disabled={selected} onClick={() => chooseModel(item)}>{selected ? "正在使用" : "切换使用"}</Button>
                                    </div>
                                </article>
                            );
                        })}
                    </div>
                ) : (
                    <div className="rounded-3xl border border-dashed border-stone-300 bg-white py-20 dark:border-stone-700 dark:bg-stone-900">
                        <Empty description={query ? "没有匹配的模型" : `还没有${capability === "image" ? "图片" : "视频"}模型`}>
                            <Button type="primary" onClick={() => openConfigDialog(false)}>配置通用 Key 并拉取模型</Button>
                        </Empty>
                    </div>
                )}
            </div>
        </main>
    );
}

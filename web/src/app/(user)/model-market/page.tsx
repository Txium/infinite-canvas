"use client";

import { App, Button, Empty, Input, Tag } from "antd";
import { Check, ImageIcon, MessageSquareText, RefreshCw, Search, Settings2, Video } from "lucide-react";
import { useMemo, useState } from "react";

import { filterModelsByCapability, normalizeLocalChannels, useConfigStore, useEffectiveConfig } from "@/stores/use-config-store";

type MarketModel = {
    key: string;
    model: string;
    channelId: string;
    channelName: string;
    capability: "image" | "video" | "text";
    available: boolean;
};

type MarketCapability = MarketModel["capability"];

const capabilityMeta: Record<MarketCapability, { noun: string; apiLabel: string; color: string }> = {
    image: { noun: "图片", apiLabel: "生图 API", color: "blue" },
    video: { noun: "视频", apiLabel: "视频 API", color: "purple" },
    text: { noun: "语言", apiLabel: "文本 API", color: "green" },
};

export default function ModelMarketPage() {
    const { message } = App.useApp();
    const [capability, setCapability] = useState<MarketCapability>("image");
    const [channelFilter, setChannelFilter] = useState("all");
    const [query, setQuery] = useState("");
    const config = useConfigStore((state) => state.config);
    const updateConfig = useConfigStore((state) => state.updateConfig);
    const openConfigDialog = useConfigStore((state) => state.openConfigDialog);
    const effectiveConfig = useEffectiveConfig();

    const allModels = useMemo<MarketModel[]>(() => {
        const channels = effectiveConfig.channelMode === "remote"
            ? effectiveConfig.publicChannels.map((channel) => ({ id: channel.id || "remote", name: channel.name || "云端渠道", protocol: channel.protocol || "openai", purpose: "general", models: channel.models || [] }))
            : normalizeLocalChannels(config);
        const connected = channels.flatMap((channel) => (["image", "video", "text"] as const).flatMap((kind) => {
            if (channel.purpose !== "general" && channel.purpose !== kind) return [];
            return filterModelsByCapability(channel.models || [], kind, channel.protocol || "openai").map((model) => ({
                key: `${channel.id}::${kind}::${model}`,
                model,
                channelId: channel.id || "",
                channelName: channel.name || "模型渠道",
                capability: kind,
                available: true,
            }));
        }));
        return connected;
    }, [config, effectiveConfig]);

    const visibleModels = useMemo(() => {
        const keyword = query.trim().toLowerCase();
        return allModels.filter((item) => item.capability === capability && (channelFilter === "all" || item.channelId === channelFilter) && (!keyword || `${item.model} ${item.channelName}`.toLowerCase().includes(keyword)));
    }, [allModels, capability, channelFilter, query]);

    const activeModel = capability === "image" ? config.imageModel : capability === "video" ? config.videoModel : config.textModel;
    const activeChannelId = capability === "image" ? config.imageChannelId : capability === "video" ? config.videoChannelId : config.textChannelId;
    const chooseModel = (item: MarketModel) => {
        if (!item.available) {
            message.info(`${item.model} 已加入模型库，请先在通用 Key 渠道中拉取到该模型`);
            openConfigDialog(false);
            return;
        }
        if (item.capability === "image") {
            updateConfig("imageModel", item.model);
            updateConfig("imageChannelId", item.channelId);
        } else if (item.capability === "video") {
            updateConfig("videoModel", item.model);
            updateConfig("videoChannelId", item.channelId);
        } else {
            updateConfig("textModel", item.model);
            updateConfig("textChannelId", item.channelId);
        }
        updateConfig("model", item.model);
        updateConfig("activeChannelId", item.channelId);
        message.success(`已切换到 ${item.model}`);
    };

    const scopedModels = allModels;
    const imageCount = scopedModels.filter((item) => item.capability === "image").length;
    const videoCount = scopedModels.filter((item) => item.capability === "video").length;
    const textCount = scopedModels.filter((item) => item.capability === "text").length;
    const meta = capabilityMeta[capability];
    const channelOptions = useMemo(() => {
        const map = new Map<string, string>();
        scopedModels.filter((item) => item.capability === capability).forEach((item) => map.set(item.channelId, item.channelName));
        return Array.from(map, ([id, name]) => ({ id, name, count: scopedModels.filter((item) => item.capability === capability && item.channelId === id).length }));
    }, [capability, scopedModels]);

    return (
        <main className="h-full min-h-0 overflow-y-auto bg-stone-50 px-4 py-6 dark:bg-stone-950 md:px-8">
            <div className="mx-auto max-w-[1500px]">
                <section className="mb-7 flex flex-col gap-5 rounded-3xl border border-stone-200 bg-white p-6 shadow-sm dark:border-stone-800 dark:bg-stone-900 md:flex-row md:items-end md:justify-between">
                    <div>
                        <div className="mb-2 flex items-center gap-2 text-sm text-stone-500"><RefreshCw className="size-4" /> 模型来自你已经接入的通用 Key</div>
                        <h1 className="text-3xl font-semibold tracking-tight text-stone-950 dark:text-white">模型广场</h1>
                        <p className="mt-2 max-w-2xl text-sm leading-6 text-stone-500">只展示新上游实际返回、当前能够调用的模型；不再混入参考模型或未接入模型。</p>
                    </div>
                    <Button icon={<Settings2 className="size-4" />} onClick={() => openConfigDialog(false)}>管理通用 Key / 拉取模型</Button>
                </section>

                <div className="mb-5 flex flex-col gap-3 rounded-2xl border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900 md:flex-row md:items-center md:justify-between">
                    <Tag color="green">已接入可用 {allModels.length}</Tag>
                    <Input allowClear value={query} onChange={(event) => setQuery(event.target.value)} prefix={<Search className="size-4 text-stone-400" />} placeholder="搜索模型或渠道" className="md:max-w-sm" />
                </div>
                <div className="grid gap-5 lg:grid-cols-[260px_minmax(0,1fr)]">
                    <aside className="h-fit rounded-2xl border border-stone-200 bg-white p-4 dark:border-stone-800 dark:bg-stone-900 lg:sticky lg:top-4">
                        <div className="mb-3 text-sm font-semibold">模型类型</div>
                        <div className="grid gap-2">
                            {([{ value: "image", label: "图片生成", count: imageCount, icon: <ImageIcon className="size-4" /> }, { value: "video", label: "视频生成", count: videoCount, icon: <Video className="size-4" /> }, { value: "text", label: "语言模型", count: textCount, icon: <MessageSquareText className="size-4" /> }] as const).map((item) => <button key={item.value} className={`flex items-center justify-between rounded-xl px-3 py-2 text-left text-sm transition ${capability === item.value ? "bg-stone-900 text-white dark:bg-white dark:text-stone-950" : "hover:bg-stone-100 dark:hover:bg-stone-800"}`} onClick={() => { setCapability(item.value); setChannelFilter("all"); }}><span className="flex items-center gap-2">{item.icon}{item.label}</span><span>{item.count}</span></button>)}
                        </div>
                        <div className="mb-3 mt-6 text-sm font-semibold">供应渠道</div>
                        <div className="grid max-h-80 gap-1 overflow-y-auto pr-1">
                            <button className={`flex justify-between rounded-lg px-3 py-2 text-left text-sm ${channelFilter === "all" ? "bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-300" : "hover:bg-stone-100 dark:hover:bg-stone-800"}`} onClick={() => setChannelFilter("all")}><span>全部渠道</span><span>{scopedModels.filter((item) => item.capability === capability).length}</span></button>
                            {channelOptions.map((item) => <button key={item.id} className={`flex justify-between gap-3 rounded-lg px-3 py-2 text-left text-sm ${channelFilter === item.id ? "bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-300" : "hover:bg-stone-100 dark:hover:bg-stone-800"}`} onClick={() => setChannelFilter(item.id)}><span className="truncate">{item.name}</span><span>{item.count}</span></button>)}
                        </div>
                    </aside>
                    <section>
                        <div className="mb-4 flex items-center justify-between"><div><h2 className="text-xl font-semibold">{meta.noun}模型</h2><p className="mt-1 text-sm text-stone-500">找到 {visibleModels.length} 个模型 · 可直接切换使用</p></div><Tag color={meta.color}>{meta.apiLabel}</Tag></div>
                        {visibleModels.length ? <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
                        {visibleModels.map((item) => {
                            const selected = item.model === activeModel && item.channelId === activeChannelId;
                            return (
                                <article key={item.key} className={`rounded-2xl border bg-white p-5 shadow-sm transition dark:bg-stone-900 ${selected ? "border-blue-500 ring-2 ring-blue-500/15" : "border-stone-200 hover:-translate-y-0.5 hover:shadow-md dark:border-stone-800"}`}>
                                    <div className="flex items-start justify-between gap-3">
                                        <div className="flex min-w-0 items-center gap-3">
                                            <div className="grid size-11 shrink-0 place-items-center rounded-xl bg-stone-100 dark:bg-stone-800">{item.capability === "image" ? <ImageIcon className="size-5" /> : item.capability === "video" ? <Video className="size-5" /> : <MessageSquareText className="size-5" />}</div>
                                            <div className="min-w-0"><h2 className="truncate font-medium text-stone-950 dark:text-white" title={item.model}>{item.model}</h2><p className="mt-1 truncate text-xs text-stone-500">{item.channelName}</p></div>
                                        </div>
                                        {selected ? <Tag color="blue" icon={<Check className="size-3" />}>当前</Tag> : !item.available ? <Tag>待接入</Tag> : null}
                                    </div>
                                    <div className="mt-5 flex items-center justify-between gap-3">
                                        <div className="flex flex-wrap gap-2"><Tag>{item.capability === "image" ? "生图" : item.capability === "video" ? "视频" : "语言"}</Tag><Tag>{item.available ? "按渠道计费" : "需要渠道支持"}</Tag></div>
                                        <Button type={selected ? "default" : item.available ? "primary" : "default"} disabled={selected} onClick={() => chooseModel(item)}>{selected ? "正在使用" : item.available ? "切换使用" : "接入模型"}</Button>
                                    </div>
                                </article>
                            );
                        })}
                        </div> : (
                    <div className="rounded-3xl border border-dashed border-stone-300 bg-white py-20 dark:border-stone-700 dark:bg-stone-900">
                        <Empty description={query ? "没有匹配的模型" : `还没有可用的${meta.noun}模型`}>
                            <Button type="primary" onClick={() => openConfigDialog(false)}>配置新上游并拉取模型</Button>
                        </Empty>
                    </div>
                )}
                    </section>
                </div>
            </div>
        </main>
    );
}

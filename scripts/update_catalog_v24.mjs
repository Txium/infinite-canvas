import fs from "node:fs";

const path = new URL("../service/catalog/model-market-v1.json", import.meta.url);
const catalog = JSON.parse(fs.readFileSync(path, "utf8"));
catalog.version = 24;

const models = new Map(catalog.models.map((item) => [item.id, item]));
const variants = new Map(catalog.models.flatMap((item) => item.variants.map((variant) => [variant.id, variant])));

for (const variant of variants.values()) {
    variant.verificationStatus = "UNVERIFIED";
}

for (const id of ["gpt_image_2__01", "seedance_2__01", "lec_seed_2_0_900"]) {
    if (variants.has(id)) variants.get(id).verificationStatus = "VERIFIED";
}
for (const id of [
    "midjourney__01", "midjourney__02",
    "lec_ac_seedance_2_0_933_pro_720p", "lec_seedance_2_0_933_stable",
    "hailuo_h3__01", "hailuo_h3__02", "hailuo_h3__03", "hailuo_h3__04",
]) {
    if (variants.has(id)) variants.get(id).verificationStatus = "TESTING";
}

const capabilities = {
    gpt_image_2: { modes: ["text-to-image", "image-to-image"], resolutions: ["1K", "2K", "4K"], ratios: ["1:1", "16:9", "9:16", "3:2", "2:3"] },
    midjourney: { modes: ["text-to-image"], ratios: ["1:1", "16:9", "9:16", "3:2", "2:3"] },
    seedance_2: { modes: ["text-to-video", "image-to-video"], resolutions: ["480P", "720P"], durations: ["5", "10", "15"], ratios: ["16:9", "9:16"] },
    hailuo_h3: { modes: ["text-to-video", "image-to-video", "reference-to-video"], resolutions: ["480P", "768P", "2K"], durations: ["6", "10"], ratios: ["16:9", "9:16"] },
};
for (const [id, fields] of Object.entries(capabilities)) {
    if (models.has(id)) Object.assign(models.get(id), fields);
}

for (const route of catalog.routes) {
    const category = models.get(route.modelId)?.category || "";
    if (route.providerId === "provider_302" && route.variantId.startsWith("midjourney__")) {
        route.adapter = "302_midjourney";
        route.endpoint = route.upstreamModelId;
    } else if (route.providerId === "provider_lec") {
        route.adapter = "lec_video";
        route.endpoint = "/v1/videos";
    } else if (route.providerId === "provider_seedance_nz") {
        route.adapter = category === "video" ? "openai_video" : "openai_compatible";
        route.endpoint = category === "video" ? "/videos" : "/chat/completions";
    } else if (route.providerId === "provider_wavespeed") {
        route.adapter = `wavespeed_${category || "generic"}`;
        route.endpoint = category === "llm" ? "/chat/completions" : `/predictions/${route.upstreamModelId}`;
    } else {
        route.adapter = "unverified";
        route.endpoint = "unverified";
    }
}

fs.writeFileSync(path, `${JSON.stringify(catalog, null, 2)}\n`);

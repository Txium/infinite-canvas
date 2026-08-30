export function fixedMarketVideoResolution(model: string) {
    const value = model.trim().toLowerCase();
    if (!value) return "";
    if (value.includes("_480p") || value.includes("-480p")) return "480p";
    if (value.includes("_720p") || value.includes("-720p")) return "720p";
    if (value === "seedance_2__01" || value === "lec_seedance_2_0") return "720p";
    if (/^kling_3__(?:0[1-8])$/.test(value)) return "720p";
    if (/^kling_3__(?:09|1[0-6])$/.test(value)) return "1080p";
    return "";
}

export function resolutionConfigValue(resolution: string) {
    return resolution.replace(/p$/i, "");
}

import type { AssetLibraryItem } from "@/services/api/assets";
import type { Prompt } from "@/services/api/prompts";

const createdAt = "2026-01-01T00:00:00.000Z";

const starterCoverUrls: Record<string, string> = {
    "product-hero": "https://images.unsplash.com/photo-1523275335684-37898b6baf30?auto=format&fit=crop&w=1200&q=82",
    "ecommerce-detail": "https://images.unsplash.com/photo-1542291026-7eec264c27ff?auto=format&fit=crop&w=1200&q=82",
    "fashion-editorial": "https://images.unsplash.com/photo-1515886657613-9f3515b0c78f?auto=format&fit=crop&w=1200&q=82",
    poster: "https://images.unsplash.com/photo-1541701494587-cb58502866ab?auto=format&fit=crop&w=1200&q=82",
    food: "https://images.unsplash.com/photo-1504674900247-0877df9cc836?auto=format&fit=crop&w=1200&q=82",
    interior: "https://images.unsplash.com/photo-1600607687939-ce8a6c25118c?auto=format&fit=crop&w=1200&q=82",
    "ip-character": "https://images.unsplash.com/photo-1577083552431-6e5fd01aa342?auto=format&fit=crop&w=1200&q=82",
    storyboard: "https://images.unsplash.com/photo-1485846234645-a62644f84728?auto=format&fit=crop&w=1200&q=82",
    "cinematic-video": "https://images.unsplash.com/photo-1489599849927-2ee91cede3ba?auto=format&fit=crop&w=1200&q=82",
    "social-cover": "https://images.unsplash.com/photo-1550745165-9bc0b252726f?auto=format&fit=crop&w=1200&q=82",
    "product-angles": "https://images.unsplash.com/photo-1556228578-8c89e6adf883?auto=format&fit=crop&w=1200&q=82",
    restore: "https://images.unsplash.com/photo-1516035069371-29a1b244cc32?auto=format&fit=crop&w=1200&q=82",
};

export const starterPrompts: Prompt[] = [
    prompt("product-hero", "高端产品主视觉", "商业摄影", ["产品", "广告", "质感"], "为【产品名称】制作高端商业主视觉：产品居中悬浮，材质细节清晰，背景使用【品牌色】渐变与柔和体积光，加入克制的倒影和空气感，构图简洁，留出标题与卖点排版区域，真实摄影质感，4K。"),
    prompt("ecommerce-detail", "电商详情页卖点图", "电商设计", ["电商", "详情页", "转化"], "为【产品名称】设计电商详情页卖点图，核心卖点是【卖点】，用近景特写展示功能与材质，画面包含清晰的信息层级、图标占位和短标题留白，色彩符合【品牌风格】，真实、可信、具有购买欲。"),
    prompt("fashion-editorial", "时尚杂志人像", "人物摄影", ["人像", "时尚", "杂志"], "一位【人物描述】身穿【服装】，置身【场景】，采用时尚杂志大片构图，柔和侧逆光，肤质自然，服装纹理清晰，低饱和电影色调，85mm 镜头，浅景深，高级而克制。"),
    prompt("poster", "品牌活动海报", "平面设计", ["海报", "品牌", "排版"], "设计一张【活动名称】品牌海报，主视觉为【主体】，采用大胆但整洁的网格排版，主标题区域清晰，包含日期、地点与行动按钮占位，色彩使用【配色】，兼顾社交媒体缩略图识别度。"),
    prompt("food", "餐饮菜单主图", "商业摄影", ["美食", "餐饮", "摄影"], "【菜品名称】商业美食摄影，45 度俯拍，食材新鲜、热气自然，餐具与桌面风格为【风格】，光线温暖但不过曝，突出质感与份量，背景干净，适合菜单与外卖平台。"),
    prompt("interior", "室内空间效果图", "空间设计", ["室内", "建筑", "效果图"], "【空间类型】室内设计效果图，风格为【设计风格】，自然采光，材质真实，动线合理，家具比例准确，广角但无明显畸变，画面整洁、具有生活气息，建筑可视化品质。"),
    prompt("ip-character", "品牌 IP 角色设定", "角色设计", ["IP", "角色", "设定"], "为【品牌/项目】设计一个【性格关键词】的 IP 角色，核心特征是【识别元素】，输出正面全身角色立绘，造型简洁、轮廓易识别、配色不超过四种，适合表情包、周边和短视频延展。"),
    prompt("storyboard", "15 秒短视频分镜", "视频分镜", ["视频", "分镜", "Seedance"], "为【主题】制作 15 秒竖屏短视频：0-3 秒用强视觉钩子展示【冲突/结果】；3-8 秒用两个镜头说明过程；8-12 秒突出【核心卖点】；12-15 秒定格品牌与行动提示。写明每镜景别、运镜、动作、光线、音效和转场，人物与产品保持一致。"),
    prompt("cinematic-video", "电影感运镜视频", "视频分镜", ["视频", "电影感", "运镜"], "【主体】位于【场景】，镜头从【起始景别】缓慢推进并轻微环绕，主体完成【动作】，环境中的风、光影与粒子产生自然变化；电影级灯光，运动连贯，物理真实，无闪烁、无变形、无突然切镜。"),
    prompt("social-cover", "社交媒体封面", "内容设计", ["封面", "小红书", "短视频"], "制作【平台】封面，主题【标题】，主体占画面 60%，第一眼能看懂内容；使用高对比配色、简洁背景和明确标题留白，避免过多小字，适配手机缩略图，真实精致而非廉价模板感。"),
    prompt("product-angles", "产品多角度一致性", "商业摄影", ["产品", "多视角", "一致性"], "以参考图中的产品为唯一主体，保持品牌、结构、材质、颜色和比例完全一致，生成【正面/侧面/背面/细节】视角，使用相同摄影棚、灯光和背景，禁止添加不存在的零件或文字。"),
    prompt("restore", "老照片修复", "图像编辑", ["修复", "清晰", "照片"], "修复这张照片：去除划痕、噪点、污渍与压缩痕迹，恢复合理的面部和织物细节，校正曝光与白平衡，保持人物身份、年代感和原始构图，不改变五官，不做过度磨皮。"),
];

export const starterAssets: AssetLibraryItem[] = [
    asset("negative-quality", "通用负面约束", ["负面词", "通用"], "避免低清晰度、过曝、欠曝、噪点、文字乱码、水印、重复主体、肢体畸形、手指错误、脸部变形、透视错误、塑料质感、过度锐化与不自然景深。"),
    asset("lighting", "商业摄影布光词典", ["灯光", "摄影"], "柔光箱主光；侧后方轮廓光；顶部柔和补光；黑旗控制反射；背景渐变光；轻微体积光；高光不过曝；阴影保留材质细节。"),
    asset("camera", "常用镜头与景别", ["镜头", "运镜"], "24mm 环境广角；35mm 纪实中景；50mm 自然视角；85mm 人像特写；100mm 微距细节。景别：大全景、全景、中景、近景、特写、大特写。"),
    asset("motion", "视频运镜词典", ["视频", "运镜"], "缓慢推进、平稳拉远、横向跟拍、低机位仰拍、轻微环绕、手持纪实、俯拍下降、焦点转移、固定镜头内动作。一次只选 1-2 种运镜，减少变形。"),
    asset("consistency", "角色一致性约束", ["角色", "一致性"], "保持人物面部特征、发型、年龄、服装、配饰、身材比例与肤色一致；保持左右方向和标志位置正确；镜头变化时不新增或删除关键元素。"),
    asset("product-consistency", "产品一致性约束", ["产品", "一致性"], "严格保留参考产品的轮廓、结构、材质、颜色、品牌位置和尺寸比例；不得重设计包装，不得生成错误文字，不得增加按钮、接口或装饰。"),
    asset("palette-luxury", "高级黑金配色", ["配色", "品牌"], "主色 #111111，辅助色 #2B2B2B，强调色 #C9A86A，背景色 #F4F0E8。适合珠宝、美妆、酒店和高端消费品。"),
    asset("palette-fresh", "清新科技配色", ["配色", "科技"], "主色 #2563EB，辅助色 #7DD3FC，强调色 #22C55E，背景色 #F8FAFC，正文色 #172033。适合 SaaS、智能硬件和年轻品牌。"),
    asset("copy-hook", "短视频开场钩子", ["文案", "短视频"], "结果前置：我用【方法】把【问题】变成了【结果】。\n反常识：别再【常见做法】了，真正有效的是【新方法】。\n挑战式：给我【时间】，看看能不能完成【目标】。"),
    asset("copy-cta", "通用行动文案", ["文案", "转化"], "立即体验；领取同款方案；保存这份清单；评论区告诉我你的选择；点击了解完整流程；现在开始制作你的第一版。"),
    asset("seedance-shot", "Seedance 分镜格式", ["Seedance", "分镜"], "镜头 1（0-3s）：景别 + 主体动作 + 环境。\n镜头 2（3-8s）：运镜 + 关键过程 + 光线变化。\n镜头 3（8-12s）：产品/人物特写 + 卖点。\n镜头 4（12-15s）：结果定格 + 品牌留白。"),
    asset("delivery-check", "生成前检查清单", ["工作流", "检查"], "主体是否明确？画面比例是否正确？参考图用途是否写清？镜头是否过多？文字是否应后期添加？人物/产品一致性是否约束？负面词是否包含变形、闪烁和乱码？"),
];

function prompt(id: string, title: string, category: string, tags: string[], text: string): Prompt {
    return { id: `starter-${id}`, title, category, tags, prompt: text, coverUrl: starterCoverUrls[id] || "", preview: text, githubUrl: "", createdAt, updatedAt: createdAt };
}

function asset(id: string, title: string, tags: string[], content: string): AssetLibraryItem {
    return { id: `starter-${id}`, title, type: "text", tags, content, category: "创作工具箱", description: "可直接复制或加入我的素材后编辑。", coverUrl: "", url: "", createdAt, updatedAt: createdAt };
}

const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const ts = require('typescript');

test('video task listing refreshes wallet only when task state changes', async () => {
    const filename = path.resolve(__dirname, '../src/services/api/video.ts');
    const source = ts.createSourceFile(filename, fs.readFileSync(filename, 'utf8'), ts.ScriptTarget.Latest, true);
    const fn = source.statements.find(node => ts.isFunctionDeclaration(node) && node.name?.text === 'listVideoGenerationTasks');
    let tasks = [{id:'retry-1',status:'processing',billingStatus:'frozen'}];
    let refreshes = 0;
    const context = vm.createContext({ usesAccountProxy:()=>true, aiHeaders:()=>({}), aiApiUrl:()=>'/video-tasks', normalizeVideoResponse:x=>x,
        useUserStore:{getState:()=>({token:'test'})}, axios:{get:async()=>({data:{code:0,data:tasks}})}, refreshRemoteUser:()=>{refreshes++;}, lastVideoWalletSignature:'' });
    vm.runInContext(ts.transpileModule(fn.getText(source).replace(/^export /,''), {compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText,context);
    await context.listVideoGenerationTasks({});
    await context.listVideoGenerationTasks({});
    assert.equal(refreshes,1);
    tasks = [{id:'retry-1',status:'completed',billingStatus:'settled'}];
    await context.listVideoGenerationTasks({});
    assert.equal(refreshes,2);
});

test('image-to-video retry keeps its direct image instead of ancestral image config', () => {
    const filename = path.resolve(__dirname, '../src/app/(user)/canvas/[id]/canvas-client-page.tsx');
    const source = ts.createSourceFile(filename, fs.readFileSync(filename, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    const fn = source.statements.find(node => ts.isFunctionDeclaration(node) && node.name?.text === 'findRetrySourceNode');
    assert.ok(fn);
    const code = ts.transpileModule(fn.getText(source), {compilerOptions:{target:ts.ScriptTarget.ES2022}}).outputText;
    const context = vm.createContext({CanvasNodeType:{Video:'video',Config:'config'},isCanvasImageNodeType:type=>type==='image'});
    vm.runInContext(code, context);
    const nodes = [{id:'config',type:'config'}, {id:'image',type:'image',metadata:{content:'https://example.com/image.png'}}, {id:'video',type:'video'}];
    const edges = [{fromNodeId:'config',toNodeId:'image'}, {fromNodeId:'image',toNodeId:'video'}];
    assert.equal(context.findRetrySourceNode('video',nodes,edges).id,'video');
    assert.equal(context.findRetrySourceNode('image',nodes,edges).id,'config');
});

test('canvas retries create a fresh task and send the previous video task as retry origin', () => {
    const canvas = fs.readFileSync(path.resolve(__dirname, '../src/app/(user)/canvas/[id]/canvas-client-page.tsx'), 'utf8');
    const api = fs.readFileSync(path.resolve(__dirname, '../src/services/api/video.ts'), 'utf8');
    assert.ok(canvas.includes('const previousVideoTaskId = node.type === CanvasNodeType.Video ? node.metadata?.videoTaskId || "" : ""'));
    assert.ok(canvas.includes('`client_video_retry_${nanoid()}`'));
    assert.ok(canvas.includes('`client_image_task_${node.id}_${nanoid()}`'));
    assert.ok(canvas.includes('retryOfTaskId: previousVideoTaskId || undefined'));
    assert.ok(!canvas.includes('retryOfTaskId: retryVideoTaskId'));
    assert.ok(api.includes('accountProxy && createOptions.retryOfTaskId'));
    assert.ok(api.includes('"X-Retry-Video-Task-ID": createOptions.retryOfTaskId'));
});

test('toolbar node creation reads the live selection when wiring references', () => {
    const canvas = fs.readFileSync(path.resolve(__dirname, '../src/app/(user)/canvas/[id]/canvas-client-page.tsx'), 'utf8');
    const toolbar = fs.readFileSync(path.resolve(__dirname, '../src/app/(user)/canvas/components/canvas-toolbar.tsx'), 'utf8');
    assert.match(canvas, /\.filter\(\(node\) => selectedNodeIdsRef\.current\.has\(node\.id\)\)/);
    assert.doesNotMatch(canvas, /\.filter\(\(node\) => selectedNodeIds\.has\(node\.id\)\)/);
    assert.match(toolbar, /onPointerDown=\{\(event\) => event\.stopPropagation\(\)\}/);
});

test('canvas media nodes retain cloud market model selections', () => {
    const source = fs.readFileSync(path.resolve(__dirname, '../src/app/(user)/canvas/components/canvas-node-prompt-panel.tsx'), 'utf8');
    assert.match(source, /globalConfig\.marketModels\.find\(\(item\) => item\.id === savedModel && item\.capability === mode\)/);
    assert.match(source, /savedMarketModel \|\| availableModels\.includes\(savedModel\)/);
    assert.match(source, /<ModelPicker[^>]*value=\{config\.model\}[^>]*channelId=\{config\.activeChannelId\}[^>]*capability="video"/);
    assert.doesNotMatch(source, /<ModelPicker[^>]*value=\{config\.videoModel\}/);
});

test('model picker applies pointer selections as well as radix value changes', () => {
    const source = fs.readFileSync(path.resolve(__dirname, '../src/components/model-picker.tsx'), 'utf8');
    assert.match(source, /onClick=\{\(\) => \{\s*onChange\(option\.model, option\.channelId, option\.capability\);\s*setOpen\(false\);/s);
});

// Exercise the real TypeScript modules with mocked HTTP, without paid calls.
function loadModule(relative, mocks) {
    const filename = path.resolve(__dirname, '..', relative);
    const code = ts.transpileModule(fs.readFileSync(filename, 'utf8'), {
        compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2022 },
    }).outputText;
    const exports = {};
    vm.runInNewContext(code, { exports, require: (id) => id in mocks ? mocks[id] : require(id), console }, { filename });
    return exports;
}

function deferred() {
    let resolve, reject;
    const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
    return { promise, resolve, reject };
}

const { buildGenerationConfig } = loadModule('src/app/(user)/canvas/utils/generation-config.ts', {
    '@/stores/use-config-store': { defaultConfig: {}, normalizeLocalChannels: (config) => config.channels || [], modelMatchesCapability: () => true },
    '@/lib/market-video-resolution': { fixedMarketVideoResolution: () => '', resolutionConfigValue: (value) => value },
    './canvas-panorama': { PANORAMA_IMAGE_SIZE: '2048x1024', isPanoramaNodeType: () => false },
});

for (const mode of ['image', 'video', 'audio', 'text']) {
    test(`${mode}: saved node selection overrides global default for display, price and request`, () => {
        const config = { channelMode: 'remote', publicChannels: [], marketModels: [{ id: 'selected', capability: mode }],
            [`${mode}Model`]: 'global-default', [`${mode}ChannelId`]: 'global-channel' };
        const selected = buildGenerationConfig(config, { metadata: { model: 'selected', channelId: 'selected-channel' } }, mode);
        assert.equal(selected.model, 'selected');
        assert.equal(selected.activeChannelId, 'selected-channel');
        assert.equal(selected[`${mode}ChannelId`], 'selected-channel');
        const reselected = buildGenerationConfig(config, { metadata: { model: 'global-default', channelId: 'global-channel' } }, mode);
        assert.equal(reselected.model, 'global-default');
        assert.equal(reselected.activeChannelId, 'global-channel');
    });
}

test('unavailable or wrong-capability saved model resolves identically for rendering and submitting', () => {
    const config = { channelMode: 'remote', publicChannels: [], marketModels: [{ id: 'old-video', capability: 'video' }],
        imageModel: 'gpt-image', imageChannelId: 'platform' };
    for (const model of ['removed', 'old-video']) {
        const result = buildGenerationConfig(config, { metadata: { model, channelId: 'old-channel' } }, 'image');
        assert.equal(result.model, 'gpt-image');
        assert.equal(result.activeChannelId, 'platform');
    }
});

test('configuration picker, price and submission share the resolved node model', () => {
    const panel = fs.readFileSync(path.resolve(__dirname, '../src/app/(user)/canvas/components/canvas-config-node-panel.tsx'), 'utf8');
    const page = fs.readFileSync(path.resolve(__dirname, '../src/app/(user)/canvas/[id]/canvas-client-page.tsx'), 'utf8');
    assert.match(panel, /const config = buildGenerationConfig\(globalConfig, node, mode\)/);
    assert.match(panel, /<ModelPicker[^>]*value=\{config\.model\}/);
    assert.match(panel, /item\.id === config\.model/);
    assert.match(page, /const generationConfig = buildGenerationConfig\(effectiveConfig, sourceNode, mode\)/);
    assert.match(page, /import \{ buildGenerationConfig \} from "\.\.\/utils\/generation-config"/);
});

function userStore(fetchCurrentUser) {
    class ApiError extends Error { constructor(status) { super('request failed'); this.status = status; } }
    const { useUserStore } = loadModule('src/stores/use-user-store.ts', {
        '@/services/api/auth': { AUTH_TOKEN_KEY: 'audit', fetchCurrentUser },
        '@/services/api/request': { ApiError },
        // Persistence is not under test; the real Zustand state transitions are.
        'zustand/middleware': { persist: (initializer) => initializer },
    });
    return { store: useUserStore, ApiError };
}

test('old account response cannot overwrite a newly logged-in account', async () => {
    const pending = deferred();
    const { store } = userStore(() => pending.promise);
    store.getState().setSession('A', { id: 'A', credits: 100 });
    const request = store.getState().hydrateUser();
    store.getState().setSession('B', { id: 'B', credits: 200 });
    pending.resolve({ id: 'A', credits: 999 });
    await request;
    assert.equal(store.getState().user.id, 'B');
    assert.equal(store.getState().user.credits, 200);
});

test('old unauthorized response cannot log out the new account', async () => {
    const pending = deferred();
    const { store, ApiError } = userStore(() => pending.promise);
    store.getState().setSession('A', { id: 'A' });
    const request = store.getState().hydrateUser();
    store.getState().setSession('B', { id: 'B' });
    pending.reject(new ApiError(401));
    await request;
    assert.equal(store.getState().token, 'B');
});

test('latest wallet refresh wins over a slower older refresh', async () => {
    const one = deferred(), two = deferred();
    let calls = 0;
    const { store } = userStore(() => (++calls === 1 ? one : two).promise);
    store.getState().setSession('A', { id: 'A', credits: 100 });
    const first = store.getState().hydrateUser();
    const second = store.getState().hydrateUser();
    two.resolve({ id: 'A', credits: 90 }); await second;
    one.resolve({ id: 'A', credits: 100 }); await first;
    assert.equal(store.getState().user.credits, 90);
});

test('temporary network failure preserves session; explicit 401 clears it', async () => {
    let failure = new Error('network');
    const { store, ApiError } = userStore(async () => { throw failure; });
    store.getState().setSession('A', { id: 'A', credits: 100 });
    await store.getState().hydrateUser();
    assert.equal(store.getState().token, 'A');
    failure = new ApiError(401);
    await store.getState().hydrateUser();
    assert.equal(store.getState().token, '');
});

test('model market handles null arrays and excludes disabled variants', async () => {
    let response = null;
    const { fetchModelMarket } = loadModule('src/services/api/model-market.ts', {
        '@/services/api/request': { apiGet: async () => response },
    });
    assert.equal((await fetchModelMarket()).length, 0);
    response = [{ id: 'image', modes: null, ratios: null, resolutions: null, durations: null,
        availableVariantIds: ['enabled', 'disabled'],
        variants: [{ id: 'enabled', enabled: true }, { id: 'disabled', enabled: false }],
    }];
    const items = await fetchModelMarket();
    assert.equal(items[0].variants.length, 1);
    assert.equal(items[0].variants[0].id, 'enabled');
    assert.equal(items[0].modes.length, 0);
});

test('canvas measurement waits for project mount, tracks resize and preserves saved viewport', () => {
    // Execute the actual observer effect, including its dependency wiring. No
    // browser or paid API is involved; deployed DOM behavior is checked separately.
    const filename = path.resolve(__dirname, '../src/app/(user)/canvas/[id]/canvas-client-page.tsx');
    const source = ts.createSourceFile(filename, fs.readFileSync(filename, 'utf8'), ts.ScriptTarget.Latest, true, ts.ScriptKind.TSX);
    let effect;
    function visit(node) {
        if (ts.isCallExpression(node) && node.expression.getText(source) === 'useEffect'
            && node.arguments[0]?.getText(source).includes('new ResizeObserver(')) effect = node;
        ts.forEachChild(node, visit);
    }
    visit(source);
    assert.ok(effect, 'canvas observer effect exists');
    const state = { projectLoaded: false, containerRef: { current: null } };
    const measurements = [], observers = [];
    let dependency, cleanup;
    const context = vm.createContext({
        ...state,
        setSize: (size) => measurements.push({ ...size }),
        setViewport: () => assert.fail('measurement must not overwrite the saved viewport'),
        ResizeObserver: class {
            constructor(callback) { this.callback = callback; observers.push(this); }
            observe(element) { this.element = element; }
            disconnect() { this.disconnected = true; }
        },
        useEffect: (callback, deps) => {
            if (dependency && deps.length === dependency.length && deps.every((value, i) => Object.is(value, dependency[i]))) return;
            cleanup?.();
            dependency = [...deps];
            cleanup = callback();
        },
    });
    const runEffect = () => vm.runInContext(effect.getText(source), context);
    runEffect();
    assert.equal(observers.length, 0);
    let rect = { width: 622, height: 760 };
    const element = { getBoundingClientRect: () => rect };
    state.containerRef.current = element;
    context.projectLoaded = true;
    runEffect();
    assert.equal(observers.length, 1, 'observer attaches after async restoration');
    assert.equal(observers[0].element, element);
    assert.deepEqual(measurements.at(-1), rect);
    rect = { width: 390, height: 844 };
    observers[0].callback();
    assert.deepEqual(measurements.at(-1), rect);
    context.projectLoaded = false;
    state.containerRef.current = null;
    runEffect();
    assert.equal(observers[0].disconnected, true);
    state.containerRef.current = element;
    context.projectLoaded = true;
    runEffect();
    assert.equal(observers.length, 2, 'newly mounted project is measured again');
    assert.deepEqual(measurements.at(-1), rect);
    cleanup();
    assert.equal(observers[1].disconnected, true);
});

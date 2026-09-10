const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const vm = require('node:vm');
const ts = require('typescript');

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

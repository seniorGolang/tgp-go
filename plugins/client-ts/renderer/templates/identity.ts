import {VersionASTg} from "./version";

const storageKey = "tgp-client-instance-id";

type NodeProcess = {versions?: {node?: string}};
type NodeRequire = (id: string) => {hostname: () => string};

let cachedClientName: string | undefined;

function isNodeRuntime(): boolean {
    const proc = (globalThis as {process?: NodeProcess}).process;
    return typeof proc?.versions?.node === "string";
}

function readNodeHostname(): string {
    const req = (globalThis as {require?: NodeRequire}).require;
    return req!("node:os").hostname();
}

function resolveBrowserToken(): string {
    const ua = navigator.userAgent.toLowerCase();
    if (ua.includes("edg/")) {
        return "edge";
    }
    if (ua.includes("firefox/")) {
        return "firefox";
    }
    if (ua.includes("opr/") || ua.includes("opera")) {
        return "opera";
    }
    if (ua.includes("chrome/")) {
        return "chrome";
    }
    if (ua.includes("safari/")) {
        return "safari";
    }
    return "unknown";
}

function resolveInstanceId(): string {
    const stored = localStorage.getItem(storageKey);
    if (stored) {
        return stored;
    }
    const created = crypto.randomUUID();
    localStorage.setItem(storageKey, created);
    return created;
}

export function resolveDefaultClientName(): string {
    if (cachedClientName !== undefined) {
        return cachedClientName;
    }
    if (isNodeRuntime()) {
        const hostname = readNodeHostname();
        cachedClientName = hostname !== ""
            ? `${hostname}_astg_ts_${VersionASTg}`
            : `astg_ts_${VersionASTg}`;
        return cachedClientName;
    }
    const browserToken = resolveBrowserToken();
    const instanceId = resolveInstanceId();
    cachedClientName = `${browserToken}_${instanceId}_astg_ts_${VersionASTg}`;
    return cachedClientName;
}

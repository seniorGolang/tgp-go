import {VersionASTg} from "./version";

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

function fnv1a32(input: string): string {
    let hash = 0x811c9dc5;
    for (let i = 0; i < input.length; i++) {
        hash ^= input.charCodeAt(i);
        hash = Math.imul(hash, 0x01000193);
    }
    return (hash >>> 0).toString(16).padStart(8, "0");
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

function resolveBrowserId(): string {
    return fnv1a32(navigator.userAgent);
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
    const browserId = resolveBrowserId();
    cachedClientName = `${browserToken}_${browserId}_astg_ts_${VersionASTg}`;
    return cachedClientName;
}

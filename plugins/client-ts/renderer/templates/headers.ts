import {type ClientOptions} from "./options";

export const headerClientId = "X-Client-Id";

export async function buildClientHeaders(options: ClientOptions): Promise<Record<string, string>> {
    let userHeaders: Record<string, string> = {};
    if (options.headers && typeof (options.headers) === "function") {
        userHeaders = await options.headers();
    } else if (options.headers) {
        userHeaders = options.headers;
    }
    const result: Record<string, string> = {...userHeaders};
    if (!(headerClientId in result)) {
        result[headerClientId] = options.clientName;
    }
    return result;
}

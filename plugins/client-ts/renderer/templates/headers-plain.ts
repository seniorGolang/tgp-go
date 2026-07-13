import {type ClientOptions} from "./options";

export async function buildClientHeaders(options: ClientOptions): Promise<Record<string, string>> {
    if (options.headers && typeof (options.headers) === "function") {
        return options.headers();
    }
    if (options.headers) {
        return options.headers;
    }
    return {};
}

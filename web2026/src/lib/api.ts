import { setToken } from "./auth.svelte";

// Wrapper fetch untuk endpoint authenticated. Selalu bawa header Authorization.
// Kalau backend balas 401 (token invalid/expired), langsung auto-logout:
// token dihapus dari localStorage & state, Root.svelte otomatis pindah ke halaman Login.
export async function apiFetch(
    path_api: string,
    token: string | null,
    init: RequestInit = {}
): Promise<Response> {
    const res = await fetch(path_api, {
        ...init,
        headers: {
            ...(init.headers ?? {}),
            "Content-Type": "application/json",
            Authorization: "Bearer " + (token ?? ""),
        },
    });

    if (res.status === 401) {
        setToken(null);
    }

    return res;
}

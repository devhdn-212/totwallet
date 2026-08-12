// src/lib/useSlowQuery.ts
import { writable } from 'svelte/store';
import { apiFetch } from './api';

export interface SlowQueryRecord {
    id: number;
    query: string;
    duration_ms: number;
    created_datetime: string;
}

export interface SlowQueryResponse {
    status: number;
    record: SlowQueryRecord[];
    total?: number;
    message?: string;
}

// 1 halaman = 100 baris, sesuai default limit di backend (internal/service/slowquery.go).
export const SLOWQUERY_PAGE_SIZE = 100;

// Batas warna badge durasi: >=2s merah (parah), >=1s kuning (waspada), sisanya abu-abu.
function durationClass(ms: number): string {
    if (ms >= 2000) return 'bg-red-100 text-red-700';
    if (ms >= 1000) return 'bg-yellow-100 text-yellow-700';
    return 'bg-muted text-muted-foreground';
}

export function useSlowQuery(path_api: string, token: string | null) {
    const listHome = writable<any[]>([]);
    const total = writable(0);
    const isLoading = writable(false);

    async function load(page: number = 1) {
        isLoading.set(true);
        try {
            const offset = (page - 1) * SLOWQUERY_PAGE_SIZE;
            const res = await apiFetch(path_api + 'api/slowquery', token, {
                method: 'POST',
                body: JSON.stringify({ limit: SLOWQUERY_PAGE_SIZE, offset }),
            });

            const json: SlowQueryResponse = await res.json();

            if (json.status === 200) {
                const record = json.record ?? [];
                total.set(json.total ?? record.length);
                listHome.set(
                    record.map((item, index) => ({
                        home_no: offset + index + 1,
                        home_query: item.query,
                        home_duration_ms: item.duration_ms,
                        home_duration_css: durationClass(item.duration_ms),
                        home_create: item.created_datetime,
                    }))
                );
            }
        } catch (err) {
            console.error(err);
        } finally {
            isLoading.set(false);
        }
    }

    return { listHome, total, isLoading, load };
}

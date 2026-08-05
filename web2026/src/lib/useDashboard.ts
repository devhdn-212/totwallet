// src/lib/useDashboard.ts
import { writable } from 'svelte/store';

export interface DashboardRecord {
    total_deposit_today: string;
    total_withdraw_today: string;
    total_member: number;
    total_transaksi: number;
}

export interface DashboardResponse {
    status: number;
    record: DashboardRecord;
    message?: string;
}

export function useDashboard(path_api: string, token: string) {
    const summary = writable<DashboardRecord | null>(null);
    const isLoading = writable(false);

    async function load() {
        isLoading.set(true);
        try {
            const res = await fetch(path_api + 'api/dashboard', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer ' + token,
                },
                body: JSON.stringify({}),
            });

            const json: DashboardResponse = await res.json();

            if (json.status === 200) {
                summary.set(json.record);
            }
        } catch (err) {
            console.error(err);
        } finally {
            isLoading.set(false);
        }
    }

    return { summary, isLoading, load };
}

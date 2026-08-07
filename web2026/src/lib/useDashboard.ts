// src/lib/useDashboard.ts
import { writable } from 'svelte/store';
import { apiFetch } from './api';

export interface DashboardMonthly {
    bulan: string;
    debit: string;
    credit: string;
}

export interface DashboardRecord {
    total_deposit_today: string;
    total_withdraw_today: string;
    total_debit_today: string;
    total_member: number;
    total_transaksi: number;
    chart: DashboardMonthly[];
}

export interface DashboardResponse {
    status: number;
    record: DashboardRecord;
    message?: string;
}

export function useDashboard(path_api: string, token: string | null) {
    const summary = writable<DashboardRecord | null>(null);
    const isLoading = writable(false);

    async function load() {
        isLoading.set(true);
        try {
            const res = await apiFetch(path_api + 'api/dashboard', token, {
                method: 'POST',
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

// src/lib/useTransaksi.ts
import { writable } from 'svelte/store';

export interface TrxRecord {
    idtrx: string;
    notrx: string;
    username: string;
    tipe: string;
    source: string;
    amount: string;
    saldo_before: string;
    saldo_after: string;
    refno: string;
    status: string;
    keterangan: string;
    created_datetime: string;
}

export interface TrxResponse {
    status: number;
    record: TrxRecord[];
    total?: number;
    message?: string;
}

const TIPE_CLASS: Record<string, string> = {
    CREDIT: 'bg-blue-100 text-blue-700',
    DEBIT: 'bg-orange-100 text-orange-700',
};

// 1 halaman = 500 baris, sesuai batas maksimal limit di backend (internal/service/wallet_transaction.go).
export const TRX_PAGE_SIZE = 500;

export function useTransaksi(path_api: string, token: string) {
    const listHome = writable<any[]>([]);
    const total = writable(0);
    const isLoading = writable(false);

    async function load(page: number = 1) {
        isLoading.set(true);
        try {
            const offset = (page - 1) * TRX_PAGE_SIZE;
            const res = await fetch(path_api + 'api/transaksi', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer ' + token,
                },
                body: JSON.stringify({ limit: TRX_PAGE_SIZE, offset }),
            });

            const json: TrxResponse = await res.json();

            if (json.status === 200) {
                const record = json.record ?? [];
                total.set(json.total ?? record.length);
                listHome.set(
                    record.map((item, index) => ({
                        home_no: offset + index + 1,
                        home_notrx: item.notrx,
                        home_username: item.username,
                        home_tipe: item.tipe,
                        home_tipe_css: TIPE_CLASS[item.tipe] ?? '',
                        home_source: item.source,
                        home_amount: item.amount,
                        home_saldo_before: item.saldo_before,
                        home_saldo_after: item.saldo_after,
                        home_refno: item.refno,
                        home_keterangan: item.keterangan,
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

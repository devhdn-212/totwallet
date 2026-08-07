// src/lib/useMember.ts
import { writable } from 'svelte/store';
import { STATUS_CLASS } from './status';
import { apiFetch } from './api';

export type MemberStatus = 'Y' | 'N';

export interface MemberRecord {
    username: string;
    token: string;
    nama: string;
    saldo: string;
    status: MemberStatus;
    created_datetime: string;
    updated_datetime: string;
}

export interface MemberResponse {
    status: number;
    record: MemberRecord[];
    message?: string;
}

export function useMember(path_api: string, token: string | null) {
    const listHome = writable<any[]>([]);
    const totalrecord = writable(0);
    const isLoading = writable(false);

    async function load() {
        isLoading.set(true);
        try {
            const res = await apiFetch(path_api + 'api/member', token, {
                method: 'POST',
                body: JSON.stringify({}),
            });

            const json: MemberResponse = await res.json();

            if (json.status === 200) {
                const record = json.record ?? [];

                totalrecord.set(record.length);

                listHome.set(
                    record.map((item, index) => ({
                        home_no: index + 1,
                        home_username: item.username,
                        home_token: item.token,
                        home_nama: item.nama,
                        home_saldo: item.saldo,
                        home_status: item.status,
                        home_status_css: STATUS_CLASS[item.status],
                        home_create: item.created_datetime,
                        home_update: item.updated_datetime,
                    }))
                );
            }
        } catch (err) {
            console.error(err);
        } finally {
            isLoading.set(false);
        }
    }

    return { listHome, totalrecord, isLoading, load };
}

// src/lib/useAdmin.ts
import { writable } from 'svelte/store';
import { STATUS_CLASS } from './status';

export type AdminStatus = 'Y' | 'N';

export interface AdminRecord {
    admin_username: string;
    admin_role: string;
    admin_name: string;
    admin_ipaddress: string;
    admin_lastlogin: string;
    admin_joindate: string;
    admin_status: AdminStatus;
    admin_created: string;
    admin_updated: string;
}

export interface AdminResponse {
    status: number;
    record: AdminRecord[];
    message?: string;
}

export function useAdmin(path_api: string, token: string) {
    const listHome = writable<any[]>([]);
    const totalrecord = writable(0);
    const isLoading = writable(false);

    async function load() {
        isLoading.set(true);
        try {
            const res = await fetch(path_api + 'api/admin', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: 'Bearer ' + token,
                },
                body: JSON.stringify({}),
            });

            const json: AdminResponse = await res.json();

            if (json.status === 200) {
                const record = json.record ?? [];

                totalrecord.set(record.length);

                listHome.set(
                    record.map((item, index) => ({
                        home_no: index + 1,
                        home_id: item.admin_username,
                        home_name: item.admin_name,
                        home_rule: item.admin_role,
                        home_ipaddress: item.admin_ipaddress,
                        home_lastlogin: item.admin_lastlogin,
                        home_status: item.admin_status,
                        home_status_css: STATUS_CLASS[item.admin_status],
                        home_create: item.admin_created,
                        home_update: item.admin_updated,
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

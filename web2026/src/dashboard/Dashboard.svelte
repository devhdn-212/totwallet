<script lang="ts">
    import { useDashboard } from "../lib/useDashboard";
    import { decimal } from "../lib/helpers";
    import { getAuth } from "../lib/auth.svelte";
    import { RefreshCw } from "lucide-svelte"

    let { HandleLogout, path_api } = $props();

    const auth = getAuth();
    const dashboard = useDashboard(path_api, auth.token);

    const summary = $derived(dashboard.summary);
    const isLoading = $derived(dashboard.isLoading);

    dashboard.load();
    $effect(() => {
        if (!auth.token) {
            HandleLogout();
        }
    });

    const stats = $derived([
        { label: "Total Deposit Hari Ini", value: $summary ? decimal($summary.total_deposit_today) : "-" },
        { label: "Total Withdraw Hari Ini", value: $summary ? decimal($summary.total_withdraw_today) : "-" },
        { label: "Total Member", value: $summary ? decimal(String($summary.total_member)) : "-" },
        { label: "Total Transaksi", value: $summary ? decimal(String($summary.total_transaksi)) : "-" },
    ]);
</script>

<div class="flex flex-col gap-6">
  <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
    {#each stats as stat}
      <div class="bg-background rounded-xl border p-4 flex flex-col gap-1">
        <span class="text-xs text-muted-foreground">{stat.label}</span>
        {#if $isLoading}
            <RefreshCw size={16} class="animate-spin text-muted-foreground" />
        {:else}
            <span class="text-2xl font-bold">{stat.value}</span>
        {/if}
      </div>
    {/each}
  </div>

  <div class="bg-background rounded-xl border p-6">
    <h2 class="font-semibold text-lg mb-4">Selamat Datang</h2>
    <p class="text-muted-foreground text-sm">Pilih menu di sidebar untuk mulai mengelola data.</p>
  </div>
</div>

<script lang="ts">
    import { BarChart } from "layerchart";
    import * as Chart from "$lib/components/ui/chart/index.js";
    import { useDashboard } from "../lib/useDashboard";
    import { decimal } from "../lib/helpers";
    import { getAuth } from "../lib/auth.svelte";
    import { RefreshCw } from "lucide-svelte"

    let { path_api } = $props();

    const auth = getAuth();
    const dashboard = useDashboard(path_api, auth.token);

    const summary = $derived(dashboard.summary);
    const isLoading = $derived(dashboard.isLoading);

    dashboard.load();

    const stats = $derived([
        { label: "Total Deposit Hari Ini", value: $summary ? decimal($summary.total_deposit_today) : "-" },
        { label: "Total Withdraw Hari Ini", value: $summary ? decimal($summary.total_withdraw_today) : "-" },
        { label: "Total Debit Hari Ini", value: $summary ? decimal($summary.total_debit_today) : "-" },
        { label: "Total Member", value: $summary ? decimal(String($summary.total_member)) : "-" },
        { label: "Total Transaksi", value: $summary ? decimal(String($summary.total_transaksi)) : "-" },
    ]);

    const chartData = $derived(
        ($summary?.chart ?? []).map((m) => ({
            bulan: m.bulan,
            debit: Number(m.debit),
            credit: Number(m.credit),
        }))
    );

    const chartConfig = {
        debit: { label: "Debit", color: "#f43f5e" },
        credit: { label: "Credit", color: "#10b981" },
    } satisfies Chart.ChartConfig;

    const bulanNama = ["Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"];
    function monthLabel(bulan: string): string {
        const [_, m] = bulan.split("-").map(Number);
        return bulanNama[m - 1] ?? bulan;
    }
</script>

<div class="flex flex-col gap-6">
  <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-5 gap-4">
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
    <h2 class="font-semibold text-lg mb-1">Transaksi Per Bulan</h2>
    <p class="text-muted-foreground text-sm mb-4">Total debit vs credit per bulan (1 tahun terakhir)</p>

    {#if $isLoading}
      <div class="flex items-center justify-center h-72">
        <RefreshCw size={20} class="animate-spin text-muted-foreground" />
      </div>
    {:else if chartData.length}
      <Chart.Container config={chartConfig} class="h-72 w-full">
        <BarChart
          data={chartData}
          x="bulan"
          axis="x"
          seriesLayout="group"
          bandPadding={0.25}
          legend
          series={[
            { key: "debit", label: "Debit", color: chartConfig.debit.color },
            { key: "credit", label: "Credit", color: chartConfig.credit.color },
          ]}
          props={{
            xAxis: { format: (d) => monthLabel(String(d)) },
          }}
        >
          {#snippet tooltip()}
            <Chart.Tooltip />
          {/snippet}
        </BarChart>
      </Chart.Container>
    {:else}
      <p class="text-muted-foreground text-sm h-72 flex items-center justify-center">Belum ada data transaksi.</p>
    {/if}
  </div>

  <div class="bg-background rounded-xl border p-6">
    <h2 class="font-semibold text-lg mb-4">Selamat Datang</h2>
    <p class="text-muted-foreground text-sm">Pilih menu di sidebar untuk mulai mengelola data.</p>
  </div>
</div>

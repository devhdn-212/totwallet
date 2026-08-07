<script lang="ts">
    import { decimal } from "../lib/helpers";
    import { TRX_PAGE_SIZE } from "../lib/useTransaksi";
    import { Toaster } from "$lib/components/ui/sonner"
    import { Badge } from "$lib/components/ui/badge"
    import { Button } from "$lib/components/ui/button"
    import * as Select from "$lib/components/ui/select"
    import DepositWithdrawModal from "../components/DepositWithdrawModal.svelte"
    import { RefreshCw, ArrowDownCircle, ArrowUpCircle } from "lucide-svelte"

    let {
        RefreshPage,
        record = [],
        total = 0,
        currentPage = 1,
        GoToPage,
        token = "",
        path_api = "",
        title_page = "",
        isLoading = false } = $props();

    let modalOpen = $state(false);
    let modalMode = $state<'deposit' | 'withdraw'>('deposit');

    function openModal(mode: 'deposit' | 'withdraw') {
        modalMode = mode;
        modalOpen = true;
    }

    const totalPages = $derived(Math.max(1, Math.ceil(total / TRX_PAGE_SIZE)));
    const pageOptions = $derived(Array.from({ length: totalPages }, (_, i) => i + 1));
    const rangeStart = $derived(total === 0 ? 0 : (currentPage - 1) * TRX_PAGE_SIZE + 1);
    const rangeEnd = $derived(Math.min(currentPage * TRX_PAGE_SIZE, total));
</script>

<div class="flex flex-col gap-4">
    <Toaster richColors position="top-right" theme="light" />
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <h1 class="text-xl font-semibold">{title_page}</h1>
            <p class="text-sm text-muted-foreground">Deposit / withdraw saldo member & riwayat transaksi</p>
        </div>

        <div class="flex items-center gap-2">
            <Button size="sm" class="cursor-pointer bg-blue-600 hover:bg-blue-700" onclick={() => openModal('deposit')}>
                <ArrowDownCircle size={14} />
                Deposit
            </Button>
            <Button size="sm" class="cursor-pointer bg-orange-600 hover:bg-orange-700" onclick={() => openModal('withdraw')}>
                <ArrowUpCircle size={14} />
                Withdraw
            </Button>
            <Button variant="outline" size="sm" class="cursor-pointer" onclick={RefreshPage} disabled={isLoading}>
                <RefreshCw size={14} class={isLoading ? "animate-spin" : ""} />
                Refresh
            </Button>
        </div>
    </div>

    <!-- Table -->
    <div class="rounded-xl border bg-background overflow-hidden">
      <div class="overflow-x-auto">
        <div class="max-h-[800px] overflow-y-auto">
          <table class="w-full text-sm min-w-[900px]">
              <thead class="bg-muted/50 border-b sticky top-0 z-10">
                  <tr>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[1%]">NO</th>
                      <th class="text-left px-4 py-3 font-medium text-muted-foreground">NOTRX</th>
                      <th class="text-left px-4 py-3 font-medium text-muted-foreground">USERNAME</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground">TIPE</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground">SOURCE</th>
                      <th class="text-right px-4 py-3 font-medium text-muted-foreground">AMOUNT</th>
                      <th class="text-right px-4 py-3 font-medium text-muted-foreground">SALDO AKHIR</th>
                      <th class="text-left px-4 py-3 font-medium text-muted-foreground">KETERANGAN</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground">WAKTU</th>
                  </tr>
              </thead>
              <tbody>
                  {#if isLoading}
                      <tr>
                          <td colspan="9" class="text-center py-10 text-muted-foreground">
                              <div class="flex items-center justify-center gap-2">
                                  <RefreshCw size={16} class="animate-spin" />
                                  Memuat data...
                              </div>
                          </td>
                      </tr>
                  {:else if record.length === 0}
                      <tr>
                          <td colspan="9" class="text-center py-10 text-muted-foreground">
                              Belum ada transaksi
                          </td>
                      </tr>
                  {:else}
                      {#each record as rec}
                      <tr class="border-b last:border-0 hover:bg-muted/30 transition-colors">
                          <td class="px-4 py-3 text-muted-foreground text-center">{rec.home_no}</td>
                          <td class="px-4 py-3 whitespace-nowrap">{rec.home_notrx}</td>
                          <td class="px-4 py-3">{rec.home_username}</td>
                          <td class="px-4 py-3 text-center">
                              <Badge variant="secondary" class="{rec.home_tipe_css}">{rec.home_tipe}</Badge>
                          </td>
                          <td class="px-4 py-3 text-center text-muted-foreground">{rec.home_source}</td>
                          <td class="px-4 py-3 text-right font-medium">{decimal(rec.home_amount)}</td>
                          <td class="px-4 py-3 text-right">{decimal(rec.home_saldo_after)}</td>
                          <td class="px-4 py-3 text-muted-foreground text-xs">{rec.home_keterangan}</td>
                          <td class="px-4 py-3 whitespace-nowrap text-center">{rec.home_create}</td>
                      </tr>
                      {/each}
                  {/if}
              </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Paging -->
    <div class="flex items-center justify-between">
        <p class="text-xs text-muted-foreground">
            Menampilkan {rangeStart}-{rangeEnd} dari {total} transaksi
        </p>
        {#if totalPages > 1}
        <div class="flex items-center gap-2">
            <span class="text-xs text-muted-foreground">Halaman</span>
            <Select.Root type="single" value={String(currentPage)} onValueChange={(v) => v && GoToPage(Number(v))}>
                <Select.Trigger class="w-24">
                    {currentPage} / {totalPages}
                </Select.Trigger>
                <Select.Content>
                    {#each pageOptions as p}
                        <Select.Item value={String(p)}>Halaman {p}</Select.Item>
                    {/each}
                </Select.Content>
            </Select.Root>
        </div>
        {/if}
    </div>
</div>

<DepositWithdrawModal
    bind:open={modalOpen}
    mode={modalMode}
    token={token}
    path_api={path_api}
    onSuccess={RefreshPage} />

<script lang="ts">
    import { SLOWQUERY_PAGE_SIZE } from "../lib/useSlowQuery";
    import { Badge } from "$lib/components/ui/badge"
    import { Button } from "$lib/components/ui/button"
    import * as Select from "$lib/components/ui/select"
    import { RefreshCw } from "lucide-svelte"

    let {
        RefreshPage,
        record = [],
        total = 0,
        currentPage = 1,
        GoToPage,
        title_page = "",
        isLoading = false } = $props();

    const totalPages = $derived(Math.max(1, Math.ceil(total / SLOWQUERY_PAGE_SIZE)));
    const pageOptions = $derived(Array.from({ length: totalPages }, (_, i) => i + 1));
    const rangeStart = $derived(total === 0 ? 0 : (currentPage - 1) * SLOWQUERY_PAGE_SIZE + 1);
    const rangeEnd = $derived(Math.min(currentPage * SLOWQUERY_PAGE_SIZE, total));

    function formatDuration(ms: number): string {
        if (ms >= 1000) return (ms / 1000).toFixed(2) + "s";
        return ms + "ms";
    }
</script>

<div class="flex flex-col gap-4">
    <!-- Header -->
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <div>
            <h1 class="text-xl font-semibold">{title_page}</h1>
            <p class="text-sm text-muted-foreground">Query database yang lebih lambat dari 500ms, kecatat otomatis</p>
        </div>

        <div class="flex flex-wrap items-center gap-2">
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
          <table class="w-full text-sm min-w-[700px]">
              <thead class="bg-muted/50 border-b sticky top-0 z-10">
                  <tr>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[1%]">NO</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[1%]">DURASI</th>
                      <th class="text-left px-4 py-3 font-medium text-muted-foreground">QUERY</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground">WAKTU</th>
                  </tr>
              </thead>
              <tbody>
                  {#if isLoading}
                      <tr>
                          <td colspan="4" class="text-center py-10 text-muted-foreground">
                              <div class="flex items-center justify-center gap-2">
                                  <RefreshCw size={16} class="animate-spin" />
                                  Memuat data...
                              </div>
                          </td>
                      </tr>
                  {:else if record.length === 0}
                      <tr>
                          <td colspan="4" class="text-center py-10 text-muted-foreground">
                              Belum ada query lambat tercatat
                          </td>
                      </tr>
                  {:else}
                      {#each record as rec}
                      <tr class="border-b last:border-0 hover:bg-muted/30 transition-colors">
                          <td class="px-4 py-3 text-muted-foreground text-center">{rec.home_no}</td>
                          <td class="px-4 py-3 text-center">
                              <Badge variant="secondary" class="{rec.home_duration_css}">{formatDuration(rec.home_duration_ms)}</Badge>
                          </td>
                          <td class="px-4 py-3 font-mono text-xs whitespace-pre-wrap break-all">{rec.home_query}</td>
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
    <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
        <p class="text-xs text-muted-foreground">
            Menampilkan {rangeStart}-{rangeEnd} dari {total} query lambat
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

<div class="flex flex-col gap-6">
  <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-4 gap-4">
    {#each [
      { label: "Total Deposit Hari Ini", value: "-", trend: "" },
      { label: "Total Withdraw Hari Ini", value: "-", trend: "" },
      { label: "Total Member", value: "-", trend: "" },
      { label: "Total Transaksi", value: "-", trend: "" },
    ] as stat}
      <div class="bg-background rounded-xl border p-4 flex flex-col gap-1">
        <span class="text-xs text-muted-foreground">{stat.label}</span>
        <span class="text-2xl font-bold">{stat.value}</span>
        <span class="text-xs text-green-500">{stat.trend}</span>
      </div>
    {/each}
  </div>

  <div class="bg-background rounded-xl border p-6">
    <h2 class="font-semibold text-lg mb-4">Selamat Datang</h2>
    <p class="text-muted-foreground text-sm">Pilih menu di sidebar untuk mulai mengelola data.</p>
  </div>
</div>
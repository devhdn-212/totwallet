<script lang="ts">
    import { getInputValue, sanitizeidlower, sanitizestringnormal, sanitizefloat } from "../lib/helpers";
    import * as z from "zod";
    import { toast } from "svelte-sonner"
    import { Toaster } from "$lib/components/ui/sonner"
    import { Badge } from "$lib/components/ui/badge"
    import { Button } from "$lib/components/ui/button"
    import { Input } from "$lib/components/ui/input"
    import { Label } from "$lib/components/ui/label"
    import * as Dialog from "$lib/components/ui/dialog"
    import { RefreshCw, ArrowDownCircle, ArrowUpCircle } from "lucide-svelte"

    let {
        RefreshPage,
        record = [],
        token = "",
        path_api = "",
        title_page = "",
        isLoading = false } = $props();

    let loadingSave = $state(false)
    let modalOpen = $state(false)
    type sMode = 'deposit' | 'withdraw';
    type sForm = {
        mode: sMode;
        username: string;
        amount: string;
        refno: string;
        keterangan: string;
    }
    const initialForm: sForm = { mode: 'deposit', username: '', amount: '', refno: '', keterangan: '' };
    let form = $state<sForm>({ ...initialForm });

    function openModal(mode: sMode) {
        form = { ...initialForm, mode };
        modalOpen = true;
    }
    function handleSanitizeInput(e: Event, field: keyof typeof form, type: string): void {
        const target = getInputValue(e);
        let value = ""
        switch (type){
            case "idlower":
                value = sanitizeidlower(target);
                break;
            case "string_normal":
                value = sanitizestringnormal(target);
                break;
            case "float":
                value = sanitizefloat(target);
                break;
        }
        form[field] = value;
    }
    async function HandleSave() {
        const trxSchema = z.object({
            username: z.string().min(4, "Username minimal 4 karakter").trim(),
            amount: z.string().refine((v) => Number(v) > 0, "Nominal harus lebih dari 0"),
        });
        const parsedData = trxSchema.safeParse({ username: form.username, amount: form.amount });
        if (!parsedData.success) {
            toast.error('Error', {description: parsedData.error.issues[0].message});
            return;
        }
        loadingSave = true;
        try {
            const endpoint = form.mode === 'deposit' ? 'api/transaksi/deposit' : 'api/transaksi/withdraw';
            const res = await fetch(path_api + endpoint, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: "Bearer " + token,
                },
                body: JSON.stringify({
                    username: form.username,
                    amount: form.amount,
                    refno: form.refno,
                    keterangan: form.keterangan,
                }),
            });
            const json = await res.json();
            if (json.status == 200) {
                toast.success('Berhasil', {description: `${form.mode === 'deposit' ? 'Deposit' : 'Withdraw'} berhasil diproses`});
                RefreshPage();
                modalOpen = false;
            } else {
                toast.error('Error', {description: json.message});
            }
        } catch (e) {
            toast.error('Koneksi Gagal', { description: 'Tidak dapat terhubung ke server' })
        } finally {
            loadingSave = false;
        }
    }
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
                      {#each record as rec, i}
                      <tr class="border-b last:border-0 hover:bg-muted/30 transition-colors">
                          <td class="px-4 py-3 text-muted-foreground text-center">{i + 1}</td>
                          <td class="px-4 py-3 whitespace-nowrap">{rec.home_notrx}</td>
                          <td class="px-4 py-3">{rec.home_username}</td>
                          <td class="px-4 py-3 text-center">
                              <Badge variant="secondary" class="{rec.home_tipe_css}">{rec.home_tipe}</Badge>
                          </td>
                          <td class="px-4 py-3 text-center text-muted-foreground">{rec.home_source}</td>
                          <td class="px-4 py-3 text-right font-medium">{rec.home_amount}</td>
                          <td class="px-4 py-3 text-right">{rec.home_saldo_after}</td>
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
</div>

<!-- Modal Deposit/Withdraw -->
<Dialog.Root bind:open={modalOpen}>
  <Dialog.Content class="max-w-md">
    <Dialog.Header>
      <Dialog.Title>{form.mode === 'deposit' ? 'Deposit' : 'Withdraw'} Saldo Member</Dialog.Title>
    </Dialog.Header>

    <div class="flex flex-col gap-4 py-2">
      <div class="flex flex-col gap-2">
        <Label for="trx-username">Username Member <span class="text-destructive">*</span></Label>
        <Input
          id="trx-username"
          maxlength="30"
          placeholder="Contoh: budi"
          oninput={(e) => handleSanitizeInput(e, "username","idlower")}
          bind:value={form.username}/>
      </div>
      <div class="flex flex-col gap-2">
        <Label for="trx-amount">Nominal <span class="text-destructive">*</span></Label>
        <Input
          id="trx-amount"
          placeholder="Contoh: 100000"
          oninput={(e) => handleSanitizeInput(e, "amount","float")}
          bind:value={form.amount}/>
      </div>
      <div class="flex flex-col gap-2">
        <Label for="trx-refno">No. Referensi</Label>
        <Input
          id="trx-refno"
          maxlength="50"
          placeholder="Opsional, mis. no. rekening/bukti transfer"
          bind:value={form.refno}/>
      </div>
      <div class="flex flex-col gap-2">
        <Label for="trx-keterangan">Keterangan</Label>
        <Input
          id="trx-keterangan"
          maxlength="250"
          placeholder="Opsional"
          oninput={(e) => handleSanitizeInput(e, "keterangan","string_normal")}
          bind:value={form.keterangan}/>
      </div>
    </div>
    <Dialog.Footer>
      <Button class="cursor-pointer" variant="outline" onclick={() => modalOpen = false}>Batal</Button>
      {#if loadingSave}
            <Button class="cursor-pointer" disabled>
                <RefreshCw size={14} class="animate-spin" />
                    Memproses...
            </Button>
        {:else}
            <Button class="cursor-pointer" onclick={HandleSave} disabled={!form.username || !form.amount}>
                {form.mode === 'deposit' ? 'Deposit' : 'Withdraw'}
            </Button>
        {/if}
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

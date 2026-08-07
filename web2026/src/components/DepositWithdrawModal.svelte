<script lang="ts">
    import { getInputValue, sanitizeidlower, sanitizestringnormal, sanitizefloat } from "../lib/helpers";
    import { apiFetch } from "../lib/api";
    import * as z from "zod";
    import { toast } from "svelte-sonner"
    import { Button } from "$lib/components/ui/button"
    import { Input } from "$lib/components/ui/input"
    import { Label } from "$lib/components/ui/label"
    import * as Dialog from "$lib/components/ui/dialog"
    import { RefreshCw } from "lucide-svelte"

    type sMode = 'deposit' | 'withdraw';

    let {
        open = $bindable(false),
        mode = 'deposit',
        username = '',
        usernameLocked = false,
        token = '',
        path_api = '',
        onSuccess = () => {},
    }: {
        open?: boolean;
        mode?: sMode;
        username?: string;
        usernameLocked?: boolean;
        token?: string;
        path_api?: string;
        onSuccess?: () => void;
    } = $props();

    let loadingSave = $state(false);
    type sForm = { username: string; amount: string; refno: string; keterangan: string };
    let form = $state<sForm>({ username: '', amount: '', refno: '', keterangan: '' });

    // Reset form tiap kali modal dibuka — username diisi dari prop (dikunci kalau usernameLocked,
    // dipakai pas modal dibuka dari halaman Member buat username tertentu).
    $effect(() => {
        if (open) {
            form = { username, amount: '', refno: '', keterangan: '' };
        }
    });

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
            const endpoint = mode === 'deposit' ? 'api/transaksi/deposit' : 'api/transaksi/withdraw';
            const res = await apiFetch(path_api + endpoint, token, {
                method: "POST",
                body: JSON.stringify({
                    username: form.username,
                    amount: form.amount,
                    refno: form.refno,
                    keterangan: form.keterangan,
                }),
            });
            const json = await res.json();
            if (json.status == 200) {
                toast.success('Berhasil', {description: `${mode === 'deposit' ? 'Deposit' : 'Withdraw'} berhasil diproses`});
                onSuccess();
                open = false;
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

<Dialog.Root bind:open={open}>
  <Dialog.Content class="max-w-md">
    <Dialog.Header>
      <Dialog.Title>{mode === 'deposit' ? 'Deposit' : 'Withdraw'} Saldo Member</Dialog.Title>
    </Dialog.Header>

    <div class="flex flex-col gap-4 py-2">
      <div class="flex flex-col gap-2">
        <Label for="trx-username">Username Member <span class="text-destructive">*</span></Label>
        <Input
          id="trx-username"
          maxlength="30"
          placeholder="Contoh: budi"
          disabled={usernameLocked}
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
      <Button class="cursor-pointer" variant="outline" onclick={() => open = false}>Batal</Button>
      {#if loadingSave}
            <Button class="cursor-pointer" disabled>
                <RefreshCw size={14} class="animate-spin" />
                    Memproses...
            </Button>
        {:else}
            <Button class="cursor-pointer" onclick={HandleSave} disabled={!form.username || !form.amount}>
                {mode === 'deposit' ? 'Deposit' : 'Withdraw'}
            </Button>
        {/if}
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

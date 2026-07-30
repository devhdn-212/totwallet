<script lang="ts">
    import { getStatusLabel } from '../lib/status'
    import { getInputValue, sanitizeidlower,sanitizestringnormal } from "../lib/helpers";
    import * as z from "zod";
    import { toast } from "svelte-sonner"
    import { Toaster } from "$lib/components/ui/sonner"
    import { Badge } from "$lib/components/ui/badge"
    import { Button } from "$lib/components/ui/button"
    import { Input } from "$lib/components/ui/input"
    import { Label } from "$lib/components/ui/label"
    import { Info } from "lucide-svelte"
    import * as Alert from "$lib/components/ui/alert"
    import * as Dialog from "$lib/components/ui/dialog"
    import * as Select from "$lib/components/ui/select"
    import { RefreshCw, Plus, Search, Pencil, Eye, EyeOff, Copy } from "lucide-svelte"


    let {
        RefreshPage,
        record = [],
        token = "",
        path_api = "",
        title_page = "",
        totalrecord=0,
        isLoading=false} = $props();


    let showPassword = $state(false)
    let loadingSave = $state(false)
    let searchHome: string = $state('');
    let filterHome: any[] = $state([]);
    let modalOpen = $state(false)
    type Status = 'Y' | 'N';
    type sForm = {
        sData: string;
        field_id: string;
        field_id_flag: boolean;
        field_name: string;
        field_password: string;
        field_status: Status;
        field_token: string;
        field_created: string | null;
        field_updated: string | null;
    }
    const initialForm: sForm = {
        sData: 'New',
        field_id: '',
        field_id_flag: false,
        field_password: '',
        field_name: '',
        field_status: 'Y',
        field_token: '',
        field_created: '',
        field_updated: '',
    };
    let form = $state<sForm>({ ...initialForm });
    $effect(() => {
        if (searchHome) {
            filterHome = record.filter(
                (item) =>
                    item.home_username
                        .toLowerCase()
                        .includes(searchHome.toLowerCase())  ||
                    item.home_nama
                        .toLowerCase()
                        .includes(searchHome.toLowerCase())
            );

        }else{
            filterHome = [...record];
        }
    });

    const NewData = (e: string,id: string, name: string, status: Status, token_: string,
                        created: string, updated: string) => {
        if(e=="New"){
            clearForm()
        }else{
            form.sData = e
            form.field_id = id
            form.field_id_flag = true
            form.field_name = name
            form.field_status = status
            form.field_token = token_
            form.field_created = created
            form.field_updated = updated
        }

        modalOpen = true
    };
    function clearForm() {
        form = {...initialForm};
    }
    function copyToken() {
        navigator.clipboard.writeText(form.field_token);
        toast.success('Tersalin', {description: 'Token member disalin ke clipboard'});
    }
    async function HandleSave() {
        const memberSchema = z.object({
            id: z.string().min(4,"Username is too short. Minimum 4 characters required").trim(),
            password: z.string().optional(),
            name: z.string().min(4,"Name is too short. Minimum 4 characters required").regex(/^[a-zA-Z0-9 ]+$/, {message: "Name Only letters, numbers, and spaces are allowed"}),
            status: z.string(),
            sData: z.enum(["New", "Edit"]),
        }).superRefine((data, ctx) => {
            if (data.sData === "New") {
                if (!data.password || data.password.trim().length < 6) {
                    ctx.addIssue({
                        path: ["password"],
                        message: "Password is required (minimum 6 characters)",
                        code: z.ZodIssueCode.custom,
                    });
                }
            }
        });
        const parsedData = memberSchema.safeParse({
            sData: form.sData,
            id: form.field_id,
            password: form.field_password,
            name: form.field_name,
            status: form.field_status
        });
        if(!parsedData.success){
            toast.error('Error', {description:parsedData.error.issues[0].message});
            return;
        }
        loadingSave = true
        try{
            const res = await fetch(path_api+"api/member/save", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                    Authorization: "Bearer " + token,
                },
                body: JSON.stringify({
                    type: form.sData,
                    username: form.field_id,
                    nama: form.field_name,
                    password: form.field_password,
                    status: form.field_status,
                }),
            });
            const json = await res.json();
            if (json.status == 200) {
                toast.info('Info', {description:json.message});
                RefreshPage()
                modalOpen = false;
                if(form.sData == "New"){
                    clearForm()
                }
            } else {
                if (json.status === 409 || json.message === "Duplicate Entry") {
                    toast.error("Duplicate Entry", {
                    description: `Member dengan username "${form.field_id}" sudah ada`,
                    })
                    return
                }else{
                    toast.error('Error', {description:json.message});
                }
            }
        }catch(e){
            toast.error('Koneksi Gagal', { description: 'Tidak dapat terhubung ke server' })
        }finally {
            loadingSave = false
        }
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
      }
      form[field] = value;
  }
  const handleKeyboard_checkenter = (e: any) => {
      let keyCode = e.which || e.keyCode;
      if (keyCode === 13) {
          e.preventDefault();
      }
  };
</script>


<div class="flex flex-col gap-4">
    <Toaster richColors position="top-right" theme="light" />
    <!-- Header -->
    <div class="flex items-center justify-between">
        <div>
            <h1 class="text-xl font-semibold">{title_page}</h1>
            <p class="text-sm text-muted-foreground">Kelola akun member wallet</p>
        </div>

        <!-- Buttons -->
        <div class="flex items-center gap-2">
            <Button size="sm" class="cursor-pointer" onclick={() => NewData("New","","","Y","","","")}>
                <Plus size={14} />
                New
            </Button>
            <Button variant="outline" size="sm" class="cursor-pointer" onclick={RefreshPage} disabled={isLoading}>
                <RefreshCw size={14} class={isLoading ? "animate-spin" : ""} />
                Refresh
            </Button>
        </div>
    </div>

    <!-- Search -->
    <div class="relative w-full max-w-sm">
      <Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground" />
      <Input
        placeholder="Cari username, nama..."
        class="pl-8"
        bind:value={searchHome}
        onkeypress={handleKeyboard_checkenter}/>
    </div>

    <!-- Table -->
    <div class="rounded-xl border bg-background overflow-hidden">
      <div class="overflow-x-auto">
        <div class="max-h-[800px] overflow-y-auto">
          <table class="w-full text-sm min-w-[800px] ">
              <thead class="bg-muted/50 border-b sticky top-0 z-10">
                  <tr>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[1%]">#</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[1%]">STATUS</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[1%]">NO</th>
                      <th class="text-left px-4 py-3 font-medium text-muted-foreground w-[5%]">USERNAME</th>
                      <th class="text-left px-4 py-3 font-medium text-muted-foreground w-[100%]">NAMA</th>
                      <th class="text-right px-4 py-3 font-medium text-muted-foreground w-[15%]">SALDO</th>
                      <th class="text-center px-4 py-3 font-medium text-muted-foreground w-[15%]">JOIN</th>
                  </tr>
              </thead>
              <tbody>
                  {#if isLoading}
                      <tr>
                          <td colspan="7" class="text-center py-10 text-muted-foreground">
                              <div class="flex items-center justify-center gap-2">
                                  <RefreshCw size={16} class="animate-spin" />
                                  Memuat data...
                              </div>
                          </td>
                      </tr>
                  {:else if filterHome.length === 0}
                      <tr>
                          <td colspan="7" class="text-center py-10 text-muted-foreground">
                              Data tidak ditemukan
                          </td>
                      </tr>
                  {:else}
                      {#each filterHome as rec,i}
                      <tr class="border-b last:border-0 hover:bg-muted/30 transition-colors">
                          <td class="px-4 py-3">
                              <div class="flex items-center justify-end gap-1">
                                  <Button variant="ghost" size="icon"
                                        onclick={() => NewData("Edit",
                                            rec.home_username,
                                            rec.home_nama,
                                            rec.home_status,
                                            rec.home_token,
                                            rec.home_create,
                                            rec.home_update)}>
                                      <Pencil size={14} />
                                  </Button>
                              </div>
                          </td>
                          <td class="px-4 py-3 font-medium text-center">
                              <Badge variant="secondary" class="{rec.home_status_css}"> {getStatusLabel(rec.home_status)}</Badge>
                          </td>
                          <td class="px-4 py-3 text-muted-foreground text-center">{i + 1}</td>
                          <td class="px-4 py-3 ">{rec.home_username}</td>
                          <td class="px-4 py-3">{rec.home_nama}</td>
                          <td class="px-4 py-3 text-right font-medium">{rec.home_saldo}</td>
                          <td class="px-4 py-3 whitespace-nowrap text-center">{rec.home_create}</td>
                      </tr>
                      {/each}
                  {/if}
              </tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Footer info -->
    <p class="text-xs text-muted-foreground">
      Menampilkan {filterHome.length} dari {totalrecord} data
    </p>
</div>

<!-- Modal New -->
<Dialog.Root bind:open={modalOpen}>
  <Dialog.Content class="max-w-md">
    <Dialog.Header>
      <Dialog.Title>{title_page} / {form.sData}</Dialog.Title>
    </Dialog.Header>

    <div class="flex flex-col gap-4 py-2">
      <div class="flex flex-col gap-2">
        <Label for="member-id">Username <span class="text-destructive">*</span></Label>
        <Input
          maxlength="30"
          disabled={form.field_id_flag}
          placeholder="Contoh: budi"
          oninput={(e) => handleSanitizeInput(e, "field_id","idlower")}
          bind:value={form.field_id}/>
      </div>
      <div class="flex flex-col gap-2">
        <Label for="member-password">Password {#if form.sData=="New"}<span class="text-destructive">*</span>{/if}</Label>
        <div class="relative">
            <Input
                maxlength="20"
                type={showPassword ? "text" : "password"}
                placeholder={form.sData=="New" ? "••••••••" : "Kosongkan jika tidak diubah"}
                bind:value={form.field_password}/>
                <button
                    type="button"
                    class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                    onclick={() => showPassword = !showPassword}>
                    {#if showPassword}
                    <EyeOff size={16} />
                    {:else}
                    <Eye size={16} />
                    {/if}
            </button>
        </div>
      </div>
      <div class="flex flex-col gap-2">
        <Label for="member-name">Nama <span class="text-destructive">*</span></Label>
        <Input
          id="member-name"
          maxlength="50"
          placeholder="Contoh: Budi Santoso"
          oninput={(e) => handleSanitizeInput(e, "field_name","string_normal")}
          bind:value={form.field_name}/>
      </div>
      <div class="flex flex-col gap-2">
        <Label>Status <span class="text-destructive">*</span></Label>
        <Select.Root type="single" bind:value={form.field_status}>
          <Select.Trigger class="w-full">
            {form.field_status === "Y" ? "ACTIVE" : "DEACTIVE"}
          </Select.Trigger>
          <Select.Content>
            <Select.Item value="Y">ACTIVE</Select.Item>
            <Select.Item value="N">DEACTIVE</Select.Item>
          </Select.Content>
        </Select.Root>
      </div>
      {#if form.sData != "New"}
      <div class="flex flex-col gap-2">
        <Label for="member-token">Token (buat integrasi API game)</Label>
        <div class="flex gap-2">
            <Input id="member-token" readonly value={form.field_token} class="font-mono text-xs"/>
            <Button type="button" variant="outline" size="icon" onclick={copyToken}>
                <Copy size={14} />
            </Button>
        </div>
      </div>
      {/if}
    </div>
    {#if form.sData != "New"}
    <div class="flex flex-col gap-2">
      <Alert.Root class="bg-blue-50 border-blue-200">
        <Info size={16} class="text-blue-600" />
        <Alert.Description class="text-xs text-blue-700 flex flex-col gap-1">
          <span><span class="font-medium">Created :</span> {form.field_created}</span>
          <span><span class="font-medium">Updated :</span> {form.field_updated}</span>
        </Alert.Description>
      </Alert.Root>
    </div>
    {/if}
    <Dialog.Footer>
      <Button class="cursor-pointer" variant="outline" onclick={() => modalOpen = false}>Batal</Button>
      {#if loadingSave}
            <Button class="cursor-pointer" disabled>
                <RefreshCw size={14} class="animate-spin" />
                    Menyimpan...
            </Button>
        {:else}
            <Button class="cursor-pointer" onclick={HandleSave} disabled={!form.field_id || !form.field_name || !form.field_status}>
                Save
            </Button>
        {/if}
    </Dialog.Footer>
  </Dialog.Content>
</Dialog.Root>

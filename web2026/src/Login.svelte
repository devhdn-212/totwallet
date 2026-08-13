<script lang="ts">
    import * as z from "zod";
    import { onMount } from "svelte";
    import { replace } from "@mateothegreat/svelte5-router";
    import { getInputValue, sanitizeUsername } from "./lib/helpers";
    import { setToken } from "./lib/auth.svelte";
    import { toast } from "svelte-sonner"
    import { Toaster } from "$lib/components/ui/sonner"
    import { Button } from "$lib/components/ui/button"
    import { Input } from "$lib/components/ui/input"
    import { Label } from "$lib/components/ui/label"
    import * as Card from "$lib/components/ui/card"
    import { Eye, EyeOff } from "lucide-svelte"

    let {path_api} = $props();
    let user_field = $state({
        username: "",
        password: "",
        loading: false,
        showPassword: false,
        ipaddress: "0.0.0.0",
        timezone: "Asia/Jakarta"
    })

    // Saat halaman login pertama dibuka, minta real IP dari /api/health dulu (biar IP
    // asli pengguna yang tersimpan di tbl_admin.ipaddress, bukan 0.0.0.0). Kalau gagal,
    // biarkan default "0.0.0.0" (tetap lolos validasi required).
    onMount(async () => {
        try {
            const res = await fetch(path_api + "api/health");
            const json = await res.json();
            const realIp = json?.record?.real_ip;
            if (realIp && realIp !== "0.0.0.0") {
                user_field.ipaddress = realIp;
            }
        } catch {
            // fallback: tetap "0.0.0.0"
        }
    });

    async function handleLogin() {
         const userSchema = z.object({
            username: z.string().min(4,"Username must be at least 4 characters"),
            pass: z.string().min(4,"Password must be at least 4 characters"),
        });
        const parsedData = userSchema.safeParse({
            username: user_field.username,
            pass: user_field.password
        });
        if(!parsedData.success){
            toast.error('Error', {description:parsedData.error.issues[0].message});
        }else{
            // flag_button = false;
            const res = await fetch(path_api+"api/auth", {
                method: "POST",
                headers: {
                    "Content-Type": "application/json",
                },
                body: JSON.stringify({
                    username: user_field.username,
                    password: user_field.password,
                    ipaddress: user_field.ipaddress,
                    timezone: user_field.timezone,
                }),
            });
            const json = await res.json();
            if (json.status == 200) {
                setToken(json.token)
                replace("/");
            } else if (json.status == 401) {
                // flag_button = true;
                toast.error('Error', {description:json.message});
                user_field.username = ""
                user_field.password = ""
            } else {
                // flag_button = true;
                toast.error('Error', {description: json.message || "Server trouble, please contact admin"});
            }
        }
    }
    function handleUsernameInput(e:Event):void {
        const target = getInputValue(e);
        let value = sanitizeUsername(target);
        
        user_field.username = value;
    }
</script>

<main class="min-h-screen flex items-center justify-center bg-muted/40 px-4">
    <Toaster richColors position="top-right" theme="light" />
    <Card.Root class="w-full max-w-sm">
        <Card.Header>
            <Card.Title class="text-2xl">WALLET ADMIN</Card.Title>
            <Card.Description>Masukkan username dan password kamu</Card.Description>
        </Card.Header>

        <Card.Content class="flex flex-col gap-4">
            <div class="flex flex-col gap-2">
                <Label for="username">Username</Label>
                <Input
                    id="username"
                    oninput={handleUsernameInput}
                    type="text"
                    maxlength="10"
                    placeholder="Masukkan username...."
                    bind:value={user_field.username}/>
            </div>

            <div class="flex flex-col gap-2">
                <Label for="password">Password</Label>
                <div class="relative">
                    <Input
                            id="password"
                            type={user_field.showPassword ? "text" : "password"}
                            placeholder="••••••••"
                            maxlength="20"
                            class="pr-10"
                            bind:value={user_field.password}/>
                    <button
                            type="button"
                            class="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground hover:text-foreground transition-colors"
                            onclick={() => user_field.showPassword = !user_field.showPassword}>
                            {#if user_field.showPassword}
                            <EyeOff size={16} />
                            {:else}
                            <Eye size={16} />
                            {/if}
                    </button>
                </div>
            </div>
        </Card.Content>
        <Card.Footer>
            <Button
                class="w-full cursor-pointer"
                onclick={handleLogin}
                disabled={user_field.loading || !user_field.username || !user_field.password}>
                {user_field.loading ? "Loading..." : "Login"}
            </Button>
        </Card.Footer>
  </Card.Root>
</main>
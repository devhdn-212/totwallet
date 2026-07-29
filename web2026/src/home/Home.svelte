<script lang="ts">
  import { Button } from "$lib/components/ui/button"
  import { Separator } from "$lib/components/ui/separator"
  import {
    LayoutDashboard, Users, ShoppingCart,
    Menu, ShieldCheck, LogOut
  } from "lucide-svelte"

  import Dashboard from "../dashboard/Dashboard.svelte"
  import Admin from "../admin/Admin.svelte"
  import Member from "../member/Member.svelte"
  import Transaksi from "../transaksi/Transaksi.svelte"

  let { HandleLogout, path_api } = $props()

  let sidebarOpen = $state(true)
  let activePage = $state("dashboard")
  let activeLabel = $state("Dashboard")

  function navigate(key: string, label: string) {
    activePage = key
    activeLabel = label
  }

  const menus = [
    { key: "dashboard", label: "Dashboard", icon: LayoutDashboard },
    { key: "transaksi", label: "Transaksi", icon: ShoppingCart },
    { key: "member", label: "Member", icon: Users },
    { key: "admin", label: "Admin", icon: ShieldCheck },
  ]
</script>

<div class="flex h-screen bg-muted/40 overflow-hidden">

    <!-- Sidebar -->
    <aside class={`flex flex-col bg-background border-r transition-all duration-300 ${sidebarOpen ? "w-60" : "w-16"}`}>

        <!-- Logo -->
        <div class="flex items-center gap-3 px-4 py-4 h-16 border-b">
            {#if sidebarOpen}
                <span class="font-bold text-lg tracking-tight">Wallet Panel</span>
            {/if}
        </div>

        <!-- Menu -->
        <nav class="flex-1 overflow-y-auto py-3 px-2 flex flex-col gap-1">
            {#each menus as menu}
                <button
                    class={`flex items-center gap-3 w-full px-3 py-2 rounded-md text-sm font-medium transition-colors
                    ${activePage === menu.key
                        ? "bg-primary text-primary-foreground"
                        : "hover:bg-muted"}`}
                    onclick={() => navigate(menu.key, menu.label)}>
                    <menu.icon size={18} class="shrink-0" />
                    {#if sidebarOpen}
                    <span>{menu.label}</span>
                    {/if}
                </button>
            {/each}
        </nav>
        <Separator />

        <!-- Bottom -->
        <div class="px-2 py-3 flex flex-col gap-1">
            <button
                class="flex items-center gap-3 w-full px-3 py-2 rounded-md text-sm font-medium text-destructive hover:bg-destructive/10 transition-colors"
                onclick={HandleLogout}>
                <LogOut size={18} class="shrink-0" />
                {#if sidebarOpen}
                <span>Logout</span>
                {/if}
            </button>
        </div>
    </aside>

  <!-- Main -->
    <div class="flex flex-col flex-1 overflow-hidden">
        <!-- Topbar -->
        <header class="h-16 border-b bg-background flex items-center px-4 gap-4">
            <Button variant="ghost" size="icon" onclick={() => sidebarOpen = !sidebarOpen}>
                <Menu size={20} />
            </Button>
            <span class="font-semibold text-sm">{activeLabel}</span>
        </header>

        <!-- Content Area -->
        <main class="flex-1 overflow-y-auto p-6">
            {#if activePage === "dashboard"}
                <Dashboard />
            {:else if activePage === "transaksi"}
                <Transaksi HandleLogout={HandleLogout} {path_api} />
            {:else if activePage === "member"}
                <Member HandleLogout={HandleLogout} {path_api} />
            {:else if activePage === "admin"}
                <Admin HandleLogout={HandleLogout} {path_api} />
            {:else}
                <div class="bg-background rounded-xl border p-6">
                    <p class="text-muted-foreground text-sm">Halaman <strong>{activeLabel}</strong> belum tersedia.</p>
                </div>
            {/if}
        </main>
    </div>
</div>

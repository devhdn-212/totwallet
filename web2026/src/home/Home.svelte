<script lang="ts">
  import { Button } from "$lib/components/ui/button"
  import { Separator } from "$lib/components/ui/separator"
  import {
    LayoutDashboard, Users, ShoppingCart,
    Menu, ShieldCheck, LogOut, Gauge, X
  } from "lucide-svelte"

  import Dashboard from "../dashboard/Dashboard.svelte"
  import Admin from "../admin/Admin.svelte"
  import Member from "../member/Member.svelte"
  import Transaksi from "../transaksi/Transaksi.svelte"
  import SlowQuery from "../slowquery/SlowQuery.svelte"

  let { HandleLogout, path_api } = $props()

  let sidebarOpen = $state(true)
  let mobileOpen = $state(false)
  let activePage = $state("dashboard")
  let activeLabel = $state("Dashboard")

  function navigate(key: string, label: string) {
    activePage = key
    activeLabel = label
    mobileOpen = false
  }

  function toggleSidebar() {
    sidebarOpen = !sidebarOpen
    mobileOpen = !mobileOpen
  }

  const menus = [
    { key: "dashboard", label: "Dashboard", icon: LayoutDashboard },
    { key: "transaksi", label: "Transaksi", icon: ShoppingCart },
    { key: "member", label: "Member", icon: Users },
    { key: "admin", label: "Admin", icon: ShieldCheck },
    { key: "slowquery", label: "Slow Query", icon: Gauge },
  ]
</script>

<div class="flex h-screen bg-muted/40 overflow-hidden">

    <!-- Backdrop (hanya di mobile, saat drawer terbuka) -->
    {#if mobileOpen}
        <button
            type="button"
            class="fixed inset-0 z-40 bg-black/40 cursor-pointer lg:hidden"
            onclick={() => mobileOpen = false}
            aria-label="Tutup menu"></button>
    {/if}

    <!-- Sidebar — di mobile jadi drawer (sliding dari kiri), di desktop tetap kolom yang bisa collapse -->
    <aside class={`flex flex-col bg-background border-r transition-all duration-300
        fixed inset-y-0 left-0 z-50 w-60
        ${mobileOpen ? "translate-x-0" : "-translate-x-full"}
        lg:static lg:translate-x-0
        ${sidebarOpen ? "lg:w-60" : "lg:w-16"}`}>

        <!-- Logo -->
        <div class="flex items-center justify-between gap-3 px-4 py-4 h-16 border-b">
            <div class="flex items-center gap-3 min-w-0">
                {#if sidebarOpen || mobileOpen}
                    <span class="font-bold text-lg tracking-tight truncate">Wallet Panel</span>
                {/if}
            </div>
            <!-- Tombol tutup drawer (mobile) -->
            <button
                class="lg:hidden text-muted-foreground hover:text-foreground transition-colors shrink-0"
                onclick={() => mobileOpen = false}
                aria-label="Tutup menu">
                <X size={20} />
            </button>
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
                    {#if sidebarOpen || mobileOpen}
                    <span class="truncate">{menu.label}</span>
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
                {#if sidebarOpen || mobileOpen}
                <span>Logout</span>
                {/if}
            </button>
        </div>
    </aside>

  <!-- Main -->
    <div class="flex flex-col flex-1 min-w-0 overflow-hidden">
        <!-- Topbar -->
        <header class="h-16 border-b bg-background flex items-center px-4 gap-4 shrink-0">
            <Button variant="ghost" size="icon" onclick={toggleSidebar} aria-label="Toggle menu">
                <Menu size={20} />
            </Button>
            <span class="font-semibold text-sm truncate">{activeLabel}</span>
        </header>

        <!-- Content Area -->
        <main class="flex-1 overflow-y-auto p-3 sm:p-6">
            {#if activePage === "dashboard"}
                <Dashboard {path_api} />
            {:else if activePage === "transaksi"}
                <Transaksi {path_api} />
            {:else if activePage === "member"}
                <Member {path_api} />
            {:else if activePage === "admin"}
                <Admin {path_api} />
            {:else if activePage === "slowquery"}
                <SlowQuery {path_api} />
            {:else}
                <div class="bg-background rounded-xl border p-6">
                    <p class="text-muted-foreground text-sm">Halaman <strong>{activeLabel}</strong> belum tersedia.</p>
                </div>
            {/if}
        </main>
    </div>
</div>

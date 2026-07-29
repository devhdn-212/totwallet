<script lang="ts">
    import Home from "./Home.svelte";
    import { useTransaksi } from "../lib/useTransaksi";
    import { getAuth } from "../lib/auth.svelte";
    let {HandleLogout,path_api} = $props();

    const auth = getAuth();
    const trx = useTransaksi(path_api, auth.token);

    const listHome = $derived(trx.listHome);
    const isLoading = $derived(trx.isLoading);

    trx.load();
    $effect(() => {
        if (!auth.token) {
            HandleLogout();
        }
    });
</script>
<Home RefreshPage={trx.load}
      record={$listHome}
      token={auth.token}
      path_api={path_api}
      title_page="Transaksi"
      isLoading={$isLoading} />

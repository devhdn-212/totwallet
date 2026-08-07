<script lang="ts">
    import Home from "./Home.svelte";
    import { useTransaksi } from "../lib/useTransaksi";
    import { getAuth } from "../lib/auth.svelte";
    let {HandleLogout,path_api} = $props();

    const auth = getAuth();
    const trx = useTransaksi(path_api, auth.token);

    const listHome = $derived(trx.listHome);
    const total = $derived(trx.total);
    const isLoading = $derived(trx.isLoading);

    let currentPage = $state(1);

    function goToPage(page: number) {
        currentPage = page;
        trx.load(page);
    }
    function refresh() {
        trx.load(currentPage);
    }

    trx.load(currentPage);
    $effect(() => {
        if (!auth.token) {
            HandleLogout();
        }
    });
</script>
<Home RefreshPage={refresh}
      record={$listHome}
      total={$total}
      currentPage={currentPage}
      GoToPage={goToPage}
      token={auth.token}
      path_api={path_api}
      title_page="Transaksi"
      isLoading={$isLoading} />

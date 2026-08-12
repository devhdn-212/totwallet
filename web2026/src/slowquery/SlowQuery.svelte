<script lang="ts">
    import Home from "./Home.svelte";
    import { useSlowQuery } from "../lib/useSlowQuery";
    import { getAuth } from "../lib/auth.svelte";
    let {path_api} = $props();

    const auth = getAuth();
    const sq = useSlowQuery(path_api, auth.token);

    const listHome = $derived(sq.listHome);
    const total = $derived(sq.total);
    const isLoading = $derived(sq.isLoading);

    let currentPage = $state(1);

    function goToPage(page: number) {
        currentPage = page;
        sq.load(page);
    }
    function refresh() {
        sq.load(currentPage);
    }

    sq.load(currentPage);
</script>
<Home RefreshPage={refresh}
      record={$listHome}
      total={$total}
      currentPage={currentPage}
      GoToPage={goToPage}
      path_api={path_api}
      title_page="Slow Query"
      isLoading={$isLoading} />

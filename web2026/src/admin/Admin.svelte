<script lang="ts">
    import Home from "./Home.svelte";
    import { useAdmin } from "../lib/useAdmin";
    import { getAuth } from "../lib/auth.svelte";
    let {path_api} = $props();

    const auth = getAuth();
    const admin = useAdmin(path_api, auth.token);

    const listHome = $derived(admin.listHome);
    const totalrecord = $derived(admin.totalrecord);
    const isLoading = $derived(admin.isLoading);

    admin.load();
</script>
<Home RefreshPage={admin.load}
      record={$listHome}
      token={auth.token ?? ""}
      path_api={path_api}
      title_page="Admin"
      totalrecord={$totalrecord}
      isLoading={$isLoading} />

<script lang="ts">
    import Home from "./Home.svelte";
    import { useMember } from "../lib/useMember";
    import { getAuth } from "../lib/auth.svelte";
    let {HandleLogout,path_api} = $props();

    const auth = getAuth();
    const member = useMember(path_api, auth.token);

    const listHome = $derived(member.listHome);
    const totalrecord = $derived(member.totalrecord);
    const isLoading = $derived(member.isLoading);

    member.load();
    $effect(() => {
        if (!auth.token) {
            HandleLogout();
        }
    });
</script>
<Home RefreshPage={member.load}
      record={$listHome}
      token={auth.token}
      path_api={path_api}
      title_page="Member"
      totalrecord={$totalrecord}
      isLoading={$isLoading} />

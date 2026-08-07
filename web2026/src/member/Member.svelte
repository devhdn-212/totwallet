<script lang="ts">
    import Home from "./Home.svelte";
    import { useMember } from "../lib/useMember";
    import { getAuth } from "../lib/auth.svelte";
    let {path_api} = $props();

    const auth = getAuth();
    const member = useMember(path_api, auth.token);

    const listHome = $derived(member.listHome);
    const totalrecord = $derived(member.totalrecord);
    const isLoading = $derived(member.isLoading);

    member.load();
</script>
<Home RefreshPage={member.load}
      record={$listHome}
      token={auth.token ?? ""}
      path_api={path_api}
      title_page="Member"
      totalrecord={$totalrecord}
      isLoading={$isLoading} />

<script lang="ts">
    import { replace } from "@mateothegreat/svelte5-router";
    import Login from "./Login.svelte";
    import Home from "./home/Home.svelte";
    import { getAuth, setToken } from "./lib/auth.svelte";

    let {path_api} = $props();

    const auth = getAuth();
    console.log("AUTH : ",auth.token)

    async function HandleLogout() {
        try {
        await fetch(path_api + "api/auth/logout", {
            method: "POST",
            headers: {
            "Content-Type": "application/json",
            Authorization: "Bearer " + auth.token,
            },
            body: JSON.stringify({}),
        });
        } catch (e) {
        // ignore error, tetap logout
        }
        setToken(null);
        replace("/");
    };
</script>

{#if auth.token}
    <Home {path_api} HandleLogout={() => {
        HandleLogout()
    }} />
{:else}
    <Login {path_api} />
{/if}
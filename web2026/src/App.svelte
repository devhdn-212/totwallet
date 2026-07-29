<script lang="ts">
    import { Router, type RouteConfig } from "@mateothegreat/svelte5-router";
    import Root from "./Root.svelte";

  // const path_api: string = "http://localhost:6061/";
  const path_api: string = "/";


 
  const routes: RouteConfig[] = [
    { path: "/", component: Root, props: { path_api } },
  ];
  
</script>


<Router {routes} mode="hash" />
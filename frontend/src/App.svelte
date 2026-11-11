<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import { DashboardStatus as statusEvent } from "../bindings/events/index.js";
  import { Dashboard, type DashboardStatus } from "../bindings/github.com/traveling-developer/shellight/cmd/shellight/index.js";
  import StatusShape from "./StatusShape.svelte";

  // null until the first answer from Go.
  let status: DashboardStatus | null = $state(null);

  // Follow the poll rounds first, then ask for the current status, so no
  // update in between is lost; a pushed status is newer than the answer.
  let pushed = false;
  Events.On(statusEvent, (ev) => {
    pushed = true;
    status = ev.data;
  });
  Dashboard.Status().then((s) => {
    if (!pushed) status = s;
  });
</script>

<div class="layout">
  <aside class="sidebar">
    <!-- Room for the traffic lights; the window is dragged by it. -->
    <div class="titlebar"></div>
    <nav>
      {#if status && !status.problem}
        {#each status.workspaces ?? [] as ws (ws.name)}
          <h2>{ws.name}</h2>
          <ul>
            {#each ws.sandboxes ?? [] as sb (sb.name)}
              <li>
                <StatusShape state={sb.state} />
                <span class="name">{sb.name}</span>
                <span class="phase" class:error={sb.state === "error"}>{sb.phase}</span>
              </li>
            {/each}
          </ul>
        {:else}
          <p class="empty">No sandboxes</p>
        {/each}
      {/if}
    </nav>
  </aside>

  <main class="content">
    {#if status?.problem}
      <h1>{status.problem}</h1>
      {#if status.detail}
        <p class="detail">{status.detail}</p>
      {/if}
    {:else if status}
      <p>{status.gateway}</p>
    {/if}
  </main>
</div>

<style>
  .layout {
    display: flex;
    height: 100%;
  }

  .sidebar {
    flex: 0 0 220px;
    display: flex;
    flex-direction: column;
    background: var(--sidebar-bg);
    border-right: 1px solid var(--separator);
  }

  .titlebar {
    flex: none;
    height: 40px;
    --wails-draggable: drag;
  }

  nav {
    flex: 1;
    overflow-y: auto;
    padding: 0 10px 14px;
  }

  h2 {
    margin: 0;
    padding: 14px 8px 4px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  h2:first-child {
    padding-top: 6px;
  }

  ul {
    display: flex;
    flex-direction: column;
    gap: 2px;
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 7px 8px;
    border-radius: 6px;
  }

  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .phase,
  .empty {
    color: var(--text-secondary);
  }

  .phase.error {
    color: var(--error);
  }

  .empty {
    margin: 6px 8px;
  }

  .content {
    flex: 1;
    overflow-y: auto;
    padding: 22px 28px 28px;
  }

  h1 {
    margin: 0 0 8px;
    font-size: 15px;
    font-weight: 600;
  }

  .detail {
    color: var(--text-secondary);
    user-select: text;
    -webkit-user-select: text;
    overflow-wrap: anywhere;
  }
</style>

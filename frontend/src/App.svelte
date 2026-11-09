<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import { DashboardStatus as statusEvent } from "../bindings/events/index.js";
  import { Dashboard, type DashboardStatus } from "../bindings/github.com/traveling-developer/shellight/cmd/shellight/index.js";

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
    {#if status && !status.problem}
      {#each status.workspaces ?? [] as ws (ws.name)}
        <h2>{ws.name}</h2>
        <ul>
          {#each ws.sandboxes ?? [] as sb (sb.name)}
            <li>
              <span class="dot" data-state={sb.state}></span>
              <span class="name">{sb.name}</span>
              <span class="phase">{sb.phase}</span>
            </li>
          {/each}
        </ul>
      {:else}
        <p class="empty">No sandboxes</p>
      {/each}
    {/if}
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
    flex: 0 0 240px;
    overflow-y: auto;
    padding: 8px 10px;
    box-sizing: border-box;
    background: var(--sidebar-bg);
    border-right: 1px solid var(--separator);
  }

  h2 {
    margin: 12px 8px 4px;
    font-size: 11px;
    font-weight: 600;
    color: var(--text-secondary);
  }

  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }

  li {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    border-radius: 6px;
  }

  .dot {
    flex: none;
    width: 8px;
    height: 8px;
    border-radius: 50%;
  }

  .dot[data-state="ok"] {
    background: var(--ok);
  }

  .dot[data-state="busy"] {
    background: var(--busy);
  }

  .dot[data-state="error"] {
    background: var(--error);
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

  .empty {
    margin: 12px 8px;
  }

  .content {
    flex: 1;
    overflow-y: auto;
    padding: 20px 24px;
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

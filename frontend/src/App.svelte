<script lang="ts">
  import { Events } from "@wailsio/runtime";
  import {
    DashboardSelect as selectEvent,
    DashboardStatus as statusEvent,
  } from "../bindings/events/index.js";
  import {
    Dashboard,
    type DashboardSandbox,
    type DashboardStatus,
    type SandboxKey,
  } from "../bindings/github.com/traveling-developer/shellight/cmd/shellight/index.js";
  import SandboxDetails from "./SandboxDetails.svelte";
  import StatusShape from "./StatusShape.svelte";

  // null until the first answer from Go.
  let status = $state<DashboardStatus | null>(null);
  let selected = $state<SandboxKey | null>(null);
  // The first sandbox is selected once, when the first sandboxes arrive,
  // unless the menu asked for a sandbox.
  let autoSelected = false;

  const sandboxes = $derived((status?.workspaces ?? []).flatMap((ws) => ws.sandboxes ?? []));
  const selectedSandbox = $derived(sandboxes.find((sb) => isSelected(sb)) ?? null);

  function isSelected(sb: DashboardSandbox): boolean {
    return sb.workspace === selected?.workspace && sb.name === selected?.name;
  }

  function select(sb: DashboardSandbox) {
    selected = { workspace: sb.workspace, name: sb.name };
  }

  // apply shows s. The selection is kept while its sandbox is missing, so a
  // sandbox that is gone shows no details, and one that comes back (e.g.
  // after a gateway outage) is selected again.
  function apply(s: DashboardStatus) {
    status = s;
    if (!autoSelected && sandboxes.length > 0) {
      autoSelected = true;
      select(sandboxes[0]);
    }
  }

  // Follow the poll rounds first, then ask for the current status, so no
  // update in between is lost; a pushed status is newer than the answer.
  let pushed = false;
  Events.On(statusEvent, (ev) => {
    pushed = true;
    apply(ev.data);
  });
  Dashboard.Status().then((s) => {
    if (!pushed) apply(s);
  });

  // A sandbox clicked in the menu: the same order as for the status.
  function selectKey(key: SandboxKey) {
    autoSelected = true;
    selected = { workspace: key.workspace, name: key.name };
  }
  let selectPushed = false;
  Events.On(selectEvent, (ev) => {
    selectPushed = true;
    selectKey(ev.data);
  });
  Dashboard.Selection().then((key) => {
    if (key && !selectPushed) selectKey(key);
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
                <button class:selected={isSelected(sb)} onclick={() => select(sb)}>
                  <StatusShape state={sb.state} />
                  <span class="name">{sb.name}</span>
                  <span class="phase" class:error={sb.state === "error"}>{sb.phase}</span>
                </button>
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
      <div>
        <h1>{status.problem}</h1>
        {#if status.detail}
          <p class="detail">{status.detail}</p>
        {/if}
      </div>
    {:else if selectedSandbox}
      <SandboxDetails sandbox={selectedSandbox} forwardErr={status?.forwardErr ?? ""} />
    {:else if sandboxes.length > 0}
      <p class="hint">No sandbox selected</p>
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

  li button {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 7px 8px;
    border: 0;
    border-radius: 6px;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
  }

  li button.selected {
    background: var(--selected-bg);
  }

  .selected .name {
    font-weight: 600;
  }

  .name {
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .phase,
  .empty,
  .hint {
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
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 22px;
    overflow-y: auto;
    padding: 22px 28px 28px;
  }

  .hint {
    margin: 0;
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

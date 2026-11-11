<script lang="ts">
  import { Browser } from "@wailsio/runtime";
  import type { DashboardSandbox } from "../bindings/github.com/traveling-developer/shellight/cmd/shellight/index.js";
  import StatusShape from "./StatusShape.svelte";

  // forwardErr is set if listing the port forwards failed.
  let { sandbox, forwardErr }: { sandbox: DashboardSandbox; forwardErr: string } = $props();

  const forwards = $derived(sandbox.forwards ?? []);

  function open(url: string) {
    Browser.OpenURL(url).catch((err) => console.error(`open ${url}:`, err));
  }
</script>

<header>
  <h1>{sandbox.name}</h1>
  <p class="subtitle">
    <StatusShape state={sandbox.state} />
    {sandbox.phase} · workspace {sandbox.workspace}
  </p>
</header>

{#if sandbox.reason}
  <p class="error-box">{sandbox.reason}</p>
{/if}

<dl class="tiles">
  <div>
    <dt>Created</dt>
    <dd>{sandbox.created}</dd>
  </div>
  <div>
    <dt>Policy version</dt>
    <dd>v{sandbox.policyVersion}</dd>
  </div>
</dl>

<section>
  <h2>
    Port forwards
    {#if !forwardErr && forwards.length === 0}
      <span class="none">none</span>
    {/if}
  </h2>
  {#if forwardErr}
    <p class="error-box">{forwardErr}</p>
  {:else if forwards.length > 0}
    <ul class="framed">
      {#each forwards as f (f.url)}
        <li>
          <button class="link" onclick={() => open(f.url)}>{f.hostPort} ↗</button>
          <span class="target">→ sandbox :{f.port}</span>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  header {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  h1 {
    margin: 0;
    font-size: 22px;
    font-weight: 700;
    overflow-wrap: anywhere;
  }

  .subtitle {
    display: flex;
    align-items: center;
    gap: 6px;
    margin: 0;
    color: var(--text-subtitle);
  }

  .tiles {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
    gap: 1px;
    margin: 0;
    background: var(--separator);
    border: 1px solid var(--separator);
    border-radius: 10px;
    overflow: hidden;
  }

  .tiles div {
    padding: 10px 14px;
    background: var(--tile-bg);
  }

  dt {
    font-size: 11px;
    color: var(--text-secondary);
  }

  dd {
    margin: 0;
    font-weight: 500;
  }

  section {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  h2 {
    display: flex;
    align-items: baseline;
    gap: 8px;
    margin: 0;
    font-size: 15px;
    font-weight: 700;
  }

  .none {
    font-size: 13px;
    font-weight: 400;
    color: var(--text-secondary);
  }

  .framed {
    margin: 0;
    padding: 0;
    list-style: none;
    border: 1px solid var(--separator);
    border-radius: 10px;
    overflow: hidden;
  }

  .framed li {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    padding: 10px 14px;
  }

  .framed li + li {
    border-top: 1px solid var(--separator);
  }

  .link {
    flex: 1 1 200px;
    padding: 0;
    border: 0;
    background: none;
    text-align: left;
    font: 600 13px "SF Mono", ui-monospace, monospace;
    color: var(--link);
    text-decoration: underline;
    cursor: pointer;
  }

  .link:hover {
    color: var(--link-hover);
  }

  .target {
    color: var(--text-secondary);
  }
</style>

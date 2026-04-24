<script lang="ts">
  import { connState, lastSeq, records, role } from '$lib/stores';
  import { deleteRecord, setRole, writeRecord } from '$lib/tauri';
  import type { Role } from '$lib/types';

  let newId = '';
  let newData = '';
  let pending = false;
  let errorMsg = '';

  async function onAdd() {
    if (!newId || pending) return;
    pending = true;
    errorMsg = '';
    try {
      await writeRecord(newId, newData);
      newId = '';
      newData = '';
    } catch (err) {
      errorMsg = String(err);
    } finally {
      pending = false;
    }
  }

  async function onDelete(id: string) {
    if (pending) return;
    pending = true;
    errorMsg = '';
    try {
      await deleteRecord(id);
    } catch (err) {
      errorMsg = String(err);
    } finally {
      pending = false;
    }
  }

  async function onRoleChange(value: Role) {
    try {
      await setRole(value);
      role.set(value);
    } catch (err) {
      errorMsg = String(err);
    }
  }
</script>

<main>
  <h1>appunvs desktop</h1>

  <section class="status">
    <div>
      conn: <strong>{$connState}</strong>
    </div>
    <div>
      last_seq: <strong>{$lastSeq}</strong>
    </div>
    <div>
      role:
      <select
        value={$role}
        on:change={(e) => onRoleChange((e.currentTarget as HTMLSelectElement).value as Role)}
      >
        <option value="provider">provider</option>
        <option value="connector">connector</option>
        <option value="both">both</option>
      </select>
    </div>
  </section>

  <section class="add">
    <h2>add / upsert</h2>
    <form on:submit|preventDefault={onAdd}>
      <input placeholder="id" bind:value={newId} required />
      <input placeholder="data" bind:value={newData} />
      <button type="submit" disabled={pending || !newId}>write</button>
    </form>
    {#if errorMsg}
      <p class="err">{errorMsg}</p>
    {/if}
  </section>

  <section class="list">
    <h2>records ({$records.length})</h2>
    {#if $records.length === 0}
      <p class="empty">no records yet</p>
    {:else}
      <ul>
        {#each $records as r (r.id)}
          <li>
            <span class="id">{r.id}</span>
            <span class="data">{r.data}</span>
            <span class="seq">seq {r.seq}</span>
            <button on:click={() => onDelete(r.id)} disabled={pending}>del</button>
          </li>
        {/each}
      </ul>
    {/if}
  </section>
</main>

<style>
  main {
    max-width: 720px;
    margin: 1.5rem auto;
    padding: 0 1rem;
    font-family: system-ui, sans-serif;
  }
  h1 {
    margin-bottom: 0.25rem;
  }
  .status {
    display: flex;
    gap: 1.5rem;
    padding: 0.5rem 0.75rem;
    background: #f5f5f5;
    border-radius: 6px;
    font-size: 0.9rem;
  }
  .add form {
    display: flex;
    gap: 0.5rem;
  }
  .add input {
    flex: 1;
    padding: 0.35rem 0.5rem;
  }
  ul {
    list-style: none;
    padding: 0;
  }
  li {
    display: flex;
    gap: 0.75rem;
    align-items: center;
    padding: 0.35rem 0;
    border-bottom: 1px solid #eee;
  }
  .id {
    font-family: ui-monospace, monospace;
    font-weight: 600;
  }
  .data {
    flex: 1;
    color: #444;
  }
  .seq {
    color: #888;
    font-size: 0.8rem;
  }
  .empty {
    color: #888;
  }
  .err {
    color: #b00020;
  }
</style>

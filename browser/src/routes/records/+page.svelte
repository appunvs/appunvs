<script lang="ts">
  import { v4 as uuidv4 } from 'uuid';
  import { Role } from '$lib/pb/wire';
  import {
    connState,
    deviceId,
    lastSeq,
    records,
    role,
    userId
  } from '$lib/stores';
  import { getEngine } from '../engine-context';

  let newData = '';

  function onRoleChange(ev: Event) {
    const v = (ev.target as HTMLSelectElement).value as Role;
    getEngine().setRole(v);
  }

  async function onAdd() {
    const data = newData.trim();
    if (!data) return;
    await getEngine().addRecord(uuidv4(), data);
    newData = '';
  }

  async function onDelete(id: string) {
    await getEngine().deleteRecord(id);
  }
</script>

<section>
  <h1>appunvs browser</h1>

  <dl>
    <dt>device_id</dt><dd><code>{$deviceId}</code></dd>
    <dt>user_id</dt><dd><code>{$userId}</code></dd>
    <dt>conn</dt><dd>{$connState}</dd>
    <dt>last_seq</dt><dd>{$lastSeq}</dd>
  </dl>

  <fieldset>
    <legend>Role</legend>
    <select value={$role} on:change={onRoleChange}>
      <option value={Role.PROVIDER}>provider</option>
      <option value={Role.CONNECTOR}>connector</option>
      <option value={Role.BOTH}>both</option>
    </select>
  </fieldset>

  <fieldset>
    <legend>Add record</legend>
    <form on:submit|preventDefault={onAdd}>
      <input type="text" bind:value={newData} placeholder="data" />
      <button type="submit">add</button>
    </form>
  </fieldset>

  <h2>Records ({$records.length})</h2>
  <table border="1" cellpadding="4">
    <thead>
      <tr><th>id</th><th>data</th><th>seq</th><th>updated_at</th><th></th></tr>
    </thead>
    <tbody>
      {#each $records as r (r.id)}
        <tr>
          <td><code>{r.id}</code></td>
          <td>{r.data}</td>
          <td>{r.seq}</td>
          <td>{new Date(r.updated_at).toISOString()}</td>
          <td><button on:click={() => onDelete(r.id)}>delete</button></td>
        </tr>
      {/each}
    </tbody>
  </table>
</section>

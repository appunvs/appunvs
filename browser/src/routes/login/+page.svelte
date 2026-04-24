<script lang="ts">
  import { goto } from '$app/navigation';
  import { v4 as uuidv4 } from 'uuid';
  import { login, signup, deviceId, setDeviceId, registerDevice } from '$lib/auth';

  let mode: 'login' | 'signup' = 'login';
  let email = '';
  let password = '';
  let busy = false;
  let errorMsg = '';

  async function onSubmit() {
    errorMsg = '';
    busy = true;
    try {
      if (!deviceId()) setDeviceId(uuidv4());

      if (mode === 'signup') {
        await signup(email, password);
      } else {
        await login(email, password);
      }
      await registerDevice();
      await goto('/');
    } catch (e) {
      errorMsg = e instanceof Error ? e.message : String(e);
    } finally {
      busy = false;
    }
  }
</script>

<section>
  <h1>appunvs — {mode === 'login' ? 'sign in' : 'create account'}</h1>

  <form on:submit|preventDefault={onSubmit}>
    <label>
      email
      <input type="email" bind:value={email} required autocomplete="email" />
    </label>
    <label>
      password
      <input
        type="password"
        bind:value={password}
        required
        minlength={mode === 'signup' ? 8 : 1}
        autocomplete={mode === 'login' ? 'current-password' : 'new-password'}
      />
    </label>
    <button type="submit" disabled={busy}>
      {busy ? '…' : mode === 'signup' ? 'create account' : 'sign in'}
    </button>
  </form>

  {#if errorMsg}
    <p role="alert" style="color: red">{errorMsg}</p>
  {/if}

  <p>
    {#if mode === 'login'}
      no account? <button type="button" on:click={() => (mode = 'signup')}>create one</button>
    {:else}
      already have an account? <button type="button" on:click={() => (mode = 'login')}>sign in</button>
    {/if}
  </p>
</section>

<style>
  section { max-width: 28rem; margin: 3rem auto; font-family: system-ui, sans-serif; }
  form label { display: block; margin: 0.75rem 0; }
  input { display: block; width: 100%; padding: 0.5rem; font-size: 1rem; }
  button[type='submit'] { padding: 0.6rem 1.25rem; font-size: 1rem; }
</style>

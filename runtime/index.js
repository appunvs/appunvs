// Runtime SDK entry point.  When the host shell embeds RuntimeSDK and
// mounts a RuntimeView (via the SubRuntime native module — TODO), this
// file is what the bundled JS bootstraps into.
//
// In dev (`npm run ios` / `npm run android` from this directory), the
// CLI-spawned app loads this entry directly, so what you see is the
// "runtime by itself" harness — the placeholder UI from src/index.tsx.
import { AppRegistry } from 'react-native';
import App from './src';
import { name as appName } from './app.json';

AppRegistry.registerComponent(appName, () => App);

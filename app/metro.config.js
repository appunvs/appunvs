// Default Metro config — kept here so we have a single place to add custom
// resolver / transformer rules later (e.g. SVG transformer, monorepo
// workspace lookups).
const { getDefaultConfig } = require('expo/metro-config');

const config = getDefaultConfig(__dirname);

module.exports = config;

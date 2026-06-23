#!/usr/bin/env node

'use strict';

const { spawnSync } = require('child_process');
const path = require('path');
const fs   = require('fs');
const os   = require('os');

const BINARY_DIR = path.join(__dirname, '..', 'bin', 'binaries');

const version = require('../package.json').version;

if (process.argv.includes('-v') || process.argv.includes('--version')) {
  console.log(`octonote v${version}`);
  process.exit(0);
}

if (process.argv.includes('--update')) {
  console.log('Checking for updates...');
  const https = require('https');
  const req = https.get('https://registry.npmjs.org/octonote/latest', (res) => {
    let data = '';
    res.on('data', (chunk) => data += chunk);
    res.on('end', () => {
      try {
        const latest = JSON.parse(data).version;
        if (latest && isNewerVersion(latest, version)) {
          console.log(`\nA new version of octonote is available: v${version} -> v${latest}`);
          console.log('To update, run:\n  npm install -g octonote@latest\n');
        } else {
          console.log(`octonote is already up-to-date (v${version}).`);
        }
      } catch (err) {
        console.error('Failed to check for updates:', err.message);
      }
      process.exit(0);
    });
  });
  req.on('error', (err) => {
    console.error('Failed to check for updates:', err.message);
    process.exit(1);
  });
  req.setTimeout(10000, () => {
    req.destroy();
    console.error('Update check timed out.');
    process.exit(1);
  });
  return;
}

function isNewerVersion(latest, current) {
  const l = latest.split('.').map(Number);
  const c = current.split('.').map(Number);
  for (let i = 0; i < Math.max(l.length, c.length); i++) {
    const lVal = l[i] || 0;
    const cVal = c[i] || 0;
    if (lVal > cVal) return true;
    if (lVal < cVal) return false;
  }
  return false;
}



function getPlatformBinary() {
  const platform = process.platform;
  const arch     = process.arch;

  const platformMap = {
    'darwin-arm64': 'octonote-darwin-arm64',
    'darwin-x64':   'octonote-darwin-amd64',
    'linux-arm64':  'octonote-linux-arm64',
    'linux-x64':    'octonote-linux-amd64',
    'win32-x64':    'octonote-windows-amd64.exe',
    'win32-arm64':  'octonote-windows-arm64.exe',
  };

  const key    = `${platform}-${arch}`;
  const binary = platformMap[key];

  if (!binary) {
    console.error(`[octonote] Unsupported platform: ${key}`);
    console.error('  Supported: darwin-arm64, darwin-x64, linux-arm64, linux-x64, win32-x64');
    process.exit(1);
  }

  return path.join(BINARY_DIR, binary);
}

const binaryPath = getPlatformBinary();

if (!fs.existsSync(binaryPath)) {
  console.error(`[octonote] Binary not found: ${binaryPath}`);
  console.error('  Try reinstalling: npm install -g octonote');
  process.exit(1);
}

if (process.platform !== 'win32') {
  try {
    fs.accessSync(binaryPath, fs.constants.X_OK);
  } catch (_) {
    fs.chmodSync(binaryPath, 0o755);
  }
}

const result = spawnSync(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env,
});

if (result.error) {
  console.error(`[octonote] Failed to launch: ${result.error.message}`);
  process.exit(1);
}

process.exit(result.status ?? 0);

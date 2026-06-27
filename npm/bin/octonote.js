#!/usr/bin/env node

const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');
const crypto = require('crypto');
const platform = os.platform();
const arch = os.arch();

const VERSION = require('../package.json').version;
const fs = require('fs');

const packageMap = {
  'darwin-arm64': '@divyo-argha/octonote-darwin-arm64',
  'darwin-x64': '@divyo-argha/octonote-darwin-x64',
  'linux-arm64': '@divyo-argha/octonote-linux-arm64',
  'linux-x64': '@divyo-argha/octonote-linux-x64',
  'win32-x64': '@divyo-argha/octonote-windows-x64',
  'win32-arm64': '@divyo-argha/octonote-windows-arm64'
};

const binName = platform === 'win32' ? 'octonote.exe' : 'octonote';
const packageName = packageMap[`${platform}-${arch}`];

// Run CLI
runCli();

function runCli() {
  if (!packageName) {
    console.error(`Unsupported platform/architecture: ${platform}-${arch}`);
    process.exit(1);
  }

  try {
    const packagePath = require.resolve(`${packageName}/package.json`);
    const binPath = path.join(path.dirname(packagePath), binName);
    
    execFileSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
  } catch (err) {
    if (err.code === 'MODULE_NOT_FOUND') {
      const devBinPath = path.resolve(__dirname, '..', '..', binName);
      if (fs.existsSync(devBinPath)) {
        try {
          execFileSync(devBinPath, process.argv.slice(2), { stdio: 'inherit' });
          return;
        } catch (execErr) {
          if (execErr.status !== undefined) {
            process.exit(execErr.status);
          }
          console.error(execErr);
          process.exit(1);
        }
      }
      console.error(`The native binary for your platform (${packageName}) was not installed.`);
      console.error(`Ensure you are using a supported package manager and did not ignore optional dependencies.`);
    } else {
      console.error(err);
    }
    process.exit(1);
  }
}

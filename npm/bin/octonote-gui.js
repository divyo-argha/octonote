#!/usr/bin/env node

const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');

const platform = os.platform();
const arch = os.arch();

const packageMap = {
  'darwin-arm64': '@divyo-argha/octonote-gui-darwin-arm64',
  'darwin-x64': '@divyo-argha/octonote-gui-darwin-x64',
  'linux-arm64': '@divyo-argha/octonote-gui-linux-arm64',
  'linux-x64': '@divyo-argha/octonote-gui-linux-x64',
  'win32-x64': '@divyo-argha/octonote-gui-windows-x64'
};

const binName = platform === 'win32' ? 'octonote-gui.exe' : 'octonote-gui';
const packageName = packageMap[`${platform}-${arch}`];

if (!packageName) {
  console.error(`Unsupported platform/architecture for GUI: ${platform}-${arch}`);
  process.exit(1);
}

try {
  const packagePath = require.resolve(`${packageName}/package.json`);
  const binPath = path.join(path.dirname(packagePath), binName);
  
  execFileSync(binPath, process.argv.slice(2), { stdio: 'inherit' });
} catch (err) {
  if (err.code === 'MODULE_NOT_FOUND') {
    console.error(`The native GUI binary for your platform (${packageName}) was not installed.`);
    console.error(`Ensure you are using a supported package manager and did not ignore optional dependencies.`);
  } else {
    console.error(err);
  }
  process.exit(1);
}

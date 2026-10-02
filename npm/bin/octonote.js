#!/usr/bin/env node

const os = require('os');
const path = require('path');
const { execFileSync, execSync } = require('child_process');
const fs = require('fs');

const platform = os.platform();
const arch = os.arch();

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
  const args = process.argv.slice(2);

  // If asking for GUI and desktop app exists, launch directly
  if (args[0] === 'gui' || args[0] === '--gui' || args[0] === '-g') {
    if (platform === 'darwin') {
      const appPaths = [
        path.resolve(__dirname, '..', '..', 'gui', 'build', 'bin', 'octoNote.app'),
        path.resolve(os.homedir(), 'Applications', 'octoNote.app'),
        '/Applications/octoNote.app'
      ];
      for (const p of appPaths) {
        if (fs.existsSync(p)) {
          execSync(`open "${p}"`);
          console.log('✓ octoNote desktop app launched.');
          process.exit(0);
        }
      }
    } else if (platform === 'win32') {
      const winPaths = [
        path.resolve(__dirname, '..', '..', 'octonote-gui.exe'),
        path.resolve(__dirname, '..', '..', 'gui', 'build', 'bin', 'octonote.exe'),
        path.resolve(os.homedir(), 'octonote-gui.exe')
      ];
      for (const p of winPaths) {
        if (fs.existsSync(p)) {
          execSync(`start "" "${p}"`);
          console.log('✓ octoNote desktop app launched.');
          process.exit(0);
        }
      }
    } else {
      const linuxPaths = [
        path.resolve(__dirname, '..', '..', 'octonote-gui'),
        path.resolve(__dirname, '..', '..', 'gui', 'build', 'bin', 'octonote'),
        '/usr/local/bin/octonote-gui'
      ];
      for (const p of linuxPaths) {
        if (fs.existsSync(p)) {
          execSync(`"${p}" &`);
          console.log('✓ octoNote desktop app launched.');
          process.exit(0);
        }
      }
    }
  }

  // 1. Try local dev repository binary first (for npm link or local checkout)
  const devBinPath = path.resolve(__dirname, '..', '..', binName);
  if (fs.existsSync(devBinPath)) {
    try {
      execFileSync(devBinPath, args, { stdio: 'inherit' });
      return;
    } catch (execErr) {
      if (execErr.status !== undefined) {
        process.exit(execErr.status);
      }
      console.error(execErr);
      process.exit(1);
    }
  }

  // 2. Try pre-bundled binaries directory
  const bundledBinPath = path.resolve(__dirname, 'binaries', `${binName}-${platform}-${arch}`);
  if (fs.existsSync(bundledBinPath)) {
    try {
      execFileSync(bundledBinPath, args, { stdio: 'inherit' });
      return;
    } catch (execErr) {
      if (execErr.status !== undefined) {
        process.exit(execErr.status);
      }
      console.error(execErr);
      process.exit(1);
    }
  }

  // 3. Try platform optional dependency package
  if (packageName) {
    try {
      const packagePath = require.resolve(`${packageName}/package.json`);
      const binPath = path.join(path.dirname(packagePath), binName);
      if (fs.existsSync(binPath)) {
        execFileSync(binPath, args, { stdio: 'inherit' });
        return;
      }
    } catch (err) {
      // module not found, proceed to fallback error
    }
  }

  console.error(`The native binary for your platform (${platform}-${arch}) was not found.`);
  console.error(`Please install octonote globally: npm install -g octonote`);
  console.error(`Or build from source: make all`);
  process.exit(1);
}

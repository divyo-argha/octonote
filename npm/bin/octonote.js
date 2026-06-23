#!/usr/bin/env node

const os = require('os');
const path = require('path');
const { execFileSync } = require('child_process');

const platform = os.platform();
const arch = os.arch();

const VERSION = require('../package.json').version;
const https = require('https');
const http = require('http');
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

// Intercept "install-gui" command
if (process.argv[2] === 'install-gui') {
  installGui();
} else {
  runCli();
}

function followRedirects(url) {
  return new Promise((resolve, reject) => {
    const attempt = (currentUrl, remaining) => {
      if (remaining <= 0) return reject(new Error('Too many redirects'));
      const lib = currentUrl.startsWith('https') ? https : http;
      lib.get(currentUrl, { headers: { 'User-Agent': 'octonote-installer/1.0' } }, res => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          attempt(res.headers.location, remaining - 1);
        } else {
          resolve(res);
        }
      }).on('error', reject);
    };
    attempt(url, 5);
  });
}

async function download(url, destPath) {
  const res = await followRedirects(url);
  if (res.statusCode !== 200) {
    res.resume();
    throw new Error(`HTTP ${res.statusCode} for ${url}`);
  }
  return new Promise((resolve, reject) => {
    const writeStream = fs.createWriteStream(destPath);
    res.pipe(writeStream);
    res.on('end', resolve);
    writeStream.on('error', reject);
  });
}

async function installGui() {
  console.log(`Installing octonote-gui v${VERSION}...`);
  
  let assetName = '';
  if (platform === 'darwin') {
    assetName = 'octonote-gui-darwin';
  } else if (platform === 'linux') {
    assetName = 'octonote-gui-linux';
  } else if (platform === 'win32') {
    assetName = 'octonote-gui-windows.exe';
  } else {
    console.error(`Unsupported platform for GUI: ${platform}`);
    process.exit(1);
  }

  const downloadUrl = `https://github.com/divyo-argha/octonote/releases/download/v${VERSION}/${assetName}`;
  
  // Find global NPM bin folder
  let binDir = '';
  try {
    const { execSync } = require('child_process');
    binDir = execSync('npm bin -g').toString().trim();
  } catch (err) {
    // Fallback if npm bin -g fails
    if (platform === 'win32') {
      binDir = path.join(process.env.APPDATA, 'npm');
    } else {
      binDir = '/usr/local/bin';
    }
  }

  const targetBinName = platform === 'win32' ? 'octonote-gui.exe' : 'octonote-gui';
  const targetPath = path.join(binDir, targetBinName);
  const tmpPath = path.join(os.tmpdir(), targetBinName);

  try {
    console.log(`Downloading GUI binary from ${downloadUrl}...`);
    await download(downloadUrl, tmpPath);
    
    console.log(`Installing GUI binary to ${targetPath}...`);
    try {
      fs.copyFileSync(tmpPath, targetPath);
    } catch (err) {
      if (err.code === 'EACCES' || err.code === 'EPERM') {
        console.error(`Permission denied writing to ${targetPath}.`);
        if (platform !== 'win32') {
          console.error(`Please run the command with sudo:`);
          console.error(`  sudo octonote install-gui`);
        } else {
          console.error(`Please run your terminal as Administrator and try again.`);
        }
        process.exit(1);
      }
      throw err;
    }

    if (platform !== 'win32') {
      fs.chmodSync(targetPath, 0o755);
    }
    
    // Clean up temp file
    try { fs.unlinkSync(tmpPath); } catch (e) {}

    console.log(`\n✓ octonote-gui v${VERSION} successfully installed!`);
    console.log(`Run 'octonote-gui' to open the desktop application.`);
  } catch (err) {
    console.error(`Installation failed: ${err.message}`);
    process.exit(1);
  }
}

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
      console.error(`The native binary for your platform (${packageName}) was not installed.`);
      console.error(`Ensure you are using a supported package manager and did not ignore optional dependencies.`);
    } else {
      console.error(err);
    }
    process.exit(1);
  }
}

#!/usr/bin/env node
const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');
const https = require('https');
const http = require('http');

const VERSION = require('../package.json').version;
const REPO_OWNER = 'divyo-argha';
const REPO_NAME = 'octonote';
const GITHUB_BASE = `https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/v${VERSION}`;

const PACKAGES = {
  // CLI packages only
  '@divyo-argha/octonote-darwin-arm64': { os: 'darwin', cpu: 'arm64', releaseAsset: 'octonote-darwin-arm64' },
  '@divyo-argha/octonote-darwin-x64':   { os: 'darwin', cpu: 'x64',   releaseAsset: 'octonote-darwin-amd64' },
  '@divyo-argha/octonote-linux-arm64':  { os: 'linux',  cpu: 'arm64', releaseAsset: 'octonote-linux-arm64' },
  '@divyo-argha/octonote-linux-x64':    { os: 'linux',  cpu: 'x64',   releaseAsset: 'octonote-linux-amd64' },
  '@divyo-argha/octonote-windows-x64':  { os: 'win32',  cpu: 'x64',   releaseAsset: 'octonote-windows-amd64.exe' }
};

function followRedirects(url) {
  return new Promise((resolve, reject) => {
    const attempt = (currentUrl, remaining) => {
      if (remaining <= 0) return reject(new Error('Too many redirects'));
      const lib = currentUrl.startsWith('https') ? https : http;
      lib.get(currentUrl, { headers: { 'User-Agent': 'octonote-publisher/1.0' } }, res => {
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

async function main() {
  const buildDir = path.join(__dirname, '..', 'build');
  if (!fs.existsSync(buildDir)) fs.mkdirSync(buildDir);

  const localDistDir = path.join(__dirname, '..', '..', 'dist');

  for (const [pkgName, info] of Object.entries(PACKAGES)) {
    const isWindows = info.os === 'win32';
    const baseName = 'octonote';
    const pkgDir = path.join(buildDir, pkgName.replace('@', '').replace('/', '-'));
    
    console.log(`Processing ${pkgName}...`);
    
    if (!fs.existsSync(pkgDir)) fs.mkdirSync(pkgDir, { recursive: true });
    
    const binName = isWindows ? `${baseName}.exe` : baseName;
    const destPath = path.join(pkgDir, binName);
    
    try {
      const localFilePath = path.join(localDistDir, info.releaseAsset);
      if (fs.existsSync(localFilePath)) {
        console.log(`  Using local binary from ${localFilePath}`);
        fs.copyFileSync(localFilePath, destPath);
      } else {
        const downloadUrl = `${GITHUB_BASE}/${info.releaseAsset}`;
        console.log(`  Downloading ${downloadUrl}...`);
        await download(downloadUrl, destPath);
      }
      
      if (!isWindows) fs.chmodSync(destPath, 0o755);
      
      const pkgJson = {
        name: pkgName,
        version: VERSION,
        os: [info.os],
        cpu: [info.cpu],
        publishConfig: { access: 'public' }
      };
      
      fs.writeFileSync(path.join(pkgDir, 'package.json'), JSON.stringify(pkgJson, null, 2));
      
      console.log(`  Publishing ${pkgName}...`);
      execSync('npm publish --access public', { cwd: pkgDir, stdio: 'inherit' });
    } catch (err) {
      console.error(`  Failed for ${pkgName}: ${err.message}`);
    }
  }
  
  console.log('Publishing main wrapper package...');
  try {
    execSync('npm publish --access public', { cwd: path.join(__dirname, '..'), stdio: 'inherit' });
  } catch (err) {
    console.error(`Failed to publish main package:`, err.message);
  }
}

main().catch(console.error);

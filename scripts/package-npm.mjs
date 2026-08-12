#!/usr/bin/env node
import { chmod, copyFile, mkdir, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { fileURLToPath } from 'node:url';

const platformTargets = [
  ['linux-amd64', 'linux-x64'],
  ['linux-arm64', 'linux-arm64'],
  ['darwin-amd64', 'darwin-x64'],
  ['darwin-arm64', 'darwin-arm64'],
  ['windows-amd64.exe', 'win32-x64.exe'],
  ['windows-arm64.exe', 'win32-arm64.exe'],
];

export async function preparePackage({ artifacts, out, packageName, commandName, binaryPrefix, version, description, keywords = [] }) {
  await mkdir(join(out, 'bin'), { recursive: true });
  for (const [artifactSuffix, npmSuffix] of platformTargets) {
    const src = join(artifacts, `${binaryPrefix}-${artifactSuffix}`);
    const dst = join(out, 'bin', `${commandName}-${npmSuffix}`);
    await copyFile(src, dst);
    await chmod(dst, 0o755);
  }

  const pkg = {
    name: packageName,
    version,
    description,
    license: 'GPL-3.0-or-later',
    repository: { type: 'git', url: 'https://github.com/NetVar1337/unleash' },
    homepage: 'https://github.com/NetVar1337/unleash',
    author: 'NetVar1337',
    keywords,
    bin: { [commandName]: 'bin/run.js' },
    os: ['darwin', 'linux', 'win32'],
    cpu: ['x64', 'arm64'],
    files: ['bin/'],
  };
  await writeFile(join(out, 'package.json'), `${JSON.stringify(pkg, null, 2)}\n`);
  await writeFile(join(out, 'bin', 'run.js'), launcher(commandName));
  await chmod(join(out, 'bin', 'run.js'), 0o755);
}

function launcher(commandName) {
  return `#!/usr/bin/env node
const { execFileSync } = require("child_process");
const path = require("path");
const os = require("os");

const platform = os.platform();
const arch = os.arch();
let bin;
if (platform === "win32") {
  bin = arch === "arm64" ? "${commandName}-win32-arm64.exe" : "${commandName}-win32-x64.exe";
} else if (platform === "darwin") {
  bin = arch === "arm64" ? "${commandName}-darwin-arm64" : "${commandName}-darwin-x64";
} else {
  bin = arch === "arm64" ? "${commandName}-linux-arm64" : "${commandName}-linux-x64";
}

try {
  execFileSync(path.join(__dirname, bin), process.argv.slice(2), { stdio: "inherit" });
} catch (e) {
  process.exit(e.status || 1);
}
`;
}

const isMain = process.argv[1] === fileURLToPath(import.meta.url);
if (isMain) {
  const [artifacts, out, packageName, commandName, binaryPrefix, version, description, keywords = ''] = process.argv.slice(2);
  if (!artifacts || !out || !packageName || !commandName || !binaryPrefix || !version || !description) {
    console.error('usage: package-npm.mjs <artifacts> <out> <packageName> <commandName> <binaryPrefix> <version> <description> [commaKeywords]');
    process.exit(2);
  }
  await preparePackage({ artifacts, out, packageName, commandName, binaryPrefix, version, description, keywords: keywords.split(',').filter(Boolean) });
}

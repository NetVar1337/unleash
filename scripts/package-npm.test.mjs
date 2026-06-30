import { mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import test from 'node:test';
import assert from 'node:assert/strict';
import { preparePackage } from './package-npm.mjs';

test('preparePackage writes package metadata, launcher, and platform binaries', async () => {
  const root = await mkdir(join(tmpdir(), `unleash-package-${process.pid}-`), { recursive: true });
  const artifacts = join(root, 'artifacts');
  const out = join(root, 'pkg');
  await mkdir(artifacts, { recursive: true });
  for (const name of [
    'tool-linux-amd64',
    'tool-linux-arm64',
    'tool-darwin-amd64',
    'tool-darwin-arm64',
    'tool-windows-amd64.exe',
    'tool-windows-arm64.exe',
  ]) {
    await writeFile(join(artifacts, name), name);
  }

  await preparePackage({ artifacts, out, packageName: 'tool-pkg', commandName: 'tool', binaryPrefix: 'tool', version: '1.2.3', description: 'desc', keywords: ['one'] });

  const pkg = JSON.parse(await readFile(join(out, 'package.json'), 'utf8'));
  assert.equal(pkg.name, 'tool-pkg');
  assert.equal(pkg.version, '1.2.3');
  assert.equal(pkg.bin.tool, 'bin/run.js');
  assert.match(await readFile(join(out, 'bin', 'run.js'), 'utf8'), /execFileSync/);
  assert.equal(await readFile(join(out, 'bin', 'tool-linux-x64'), 'utf8'), 'tool-linux-amd64');

  await rm(root, { recursive: true, force: true });
});

#!/usr/bin/env node
import { cp, mkdir, readdir, readFile, rm } from 'node:fs/promises';
import { join, relative } from 'node:path';

const pairs = [
  ['patches', 'go/embed/patches'],
  ['contrib', 'go/embed/contrib'],
];

async function listFiles(dir, base = dir) {
  const entries = await readdir(dir, { withFileTypes: true });
  const out = [];
  for (const entry of entries) {
    const path = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...await listFiles(path, base));
    else out.push(relative(base, path));
  }
  return out.sort();
}

async function equalDirs(a, b) {
  const [af, bf] = await Promise.all([listFiles(a), listFiles(b)]);
  if (af.join('\n') !== bf.join('\n')) return false;
  for (const file of af) {
    const [left, right] = await Promise.all([readFile(join(a, file)), readFile(join(b, file))]);
    if (!left.equals(right)) return false;
  }
  return true;
}

if (process.argv.includes('--check')) {
  for (const [src, dst] of pairs) {
    if (!await equalDirs(src, dst)) {
      console.error(`${dst} is out of sync with ${src}`);
      process.exit(1);
    }
  }
  process.exit(0);
}

for (const [src, dst] of pairs) {
  await rm(dst, { recursive: true, force: true });
  await mkdir(dst, { recursive: true });
  await cp(src, dst, { recursive: true });
}

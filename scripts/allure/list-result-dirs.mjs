#!/usr/bin/env node

import { readdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");

/**
 * Find directories that directly contain Allure `*-result.json` files.
 * Allure 3 does not scan nested result trees recursively.
 */
export async function findAllureResultDirs(rootDir = path.join(repoRoot, "allure-results")) {
  const leafDirs = [];

  async function walk(dir) {
    let entries;
    try {
      entries = await readdir(dir, { withFileTypes: true });
    } catch {
      return;
    }

    const hasResultJson = entries.some(
      (entry) => entry.isFile() && entry.name.endsWith("-result.json"),
    );

    if (hasResultJson) {
      leafDirs.push(dir);
      return;
    }

    await Promise.all(
      entries
        .filter((entry) => entry.isDirectory())
        .map((entry) => walk(path.join(dir, entry.name))),
    );
  }

  await walk(rootDir);
  return leafDirs.sort();
}

/**
 * Count `*-result.json` files in the given result directories.
 */
export async function countResultJsonFiles(resultDirs) {
  let count = 0;

  for (const resultDir of resultDirs) {
    let entries;
    try {
      entries = await readdir(resultDir, { withFileTypes: true });
    } catch {
      continue;
    }

    count += entries.filter(
      (entry) => entry.isFile() && entry.name.endsWith("-result.json"),
    ).length;
  }

  return count;
}

async function main() {
  const rootDir = process.argv[2]
    ? path.resolve(process.cwd(), process.argv[2])
    : path.join(repoRoot, "allure-results");
  const resultDirs = await findAllureResultDirs(rootDir);

  for (const resultDir of resultDirs) {
    console.log(path.relative(repoRoot, resultDir) || ".");
  }
}

const isMain = process.argv[1] && fileURLToPath(import.meta.url) === path.resolve(process.argv[1]);

if (isMain) {
  await main();
}

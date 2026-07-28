#!/usr/bin/env node

import { cp, mkdir, readdir, rm, stat } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const mergedDir = path.join(repoRoot, "allure-results");
const workspaceRoots = ["apps", "packages", "services"];

async function exists(filePath) {
  try {
    await stat(filePath);
    return true;
  } catch {
    return false;
  }
}

async function findPackageResultDirs() {
  const resultDirs = [];

  for (const workspaceRoot of workspaceRoots) {
    const workspacePath = path.join(repoRoot, workspaceRoot);
    if (!(await exists(workspacePath))) {
      continue;
    }

    const entries = await readdir(workspacePath, { withFileTypes: true });
    for (const entry of entries) {
      if (!entry.isDirectory()) {
        continue;
      }

      const candidate = path.join(workspacePath, entry.name, "allure-results");
      if (await exists(candidate)) {
        resultDirs.push(candidate);
      }
    }
  }

  return resultDirs.sort();
}

async function copyResultFiles(sourceDir, targetDir) {
  const entries = await readdir(sourceDir, { withFileTypes: true });

  for (const entry of entries) {
    const sourcePath = path.join(sourceDir, entry.name);
    const targetPath = path.join(targetDir, entry.name);

    if (entry.isDirectory()) {
      await mkdir(targetPath, { recursive: true });
      await copyResultFiles(sourcePath, targetPath);
      continue;
    }

    await cp(sourcePath, targetPath, { force: true });
  }
}

async function main() {
  const packageResultDirs = await findPackageResultDirs();

  await rm(mergedDir, { recursive: true, force: true });
  await mkdir(mergedDir, { recursive: true });

  for (const sourceDir of packageResultDirs) {
    await copyResultFiles(sourceDir, mergedDir);
  }

  const relativeMergedDir = path.relative(repoRoot, mergedDir);
  if (packageResultDirs.length === 0) {
    console.log(`No per-package allure-results directories found; ${relativeMergedDir}/ is empty.`);
    return;
  }

  console.log(
    `Merged Allure results from ${packageResultDirs.length} package director${packageResultDirs.length === 1 ? "y" : "ies"} into ${relativeMergedDir}/.`,
  );
  for (const sourceDir of packageResultDirs) {
    console.log(`  - ${path.relative(repoRoot, sourceDir)}`);
  }
}

await main();

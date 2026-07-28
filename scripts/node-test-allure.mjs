#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import path from "node:path";

/**
 * Run node:test with Allure reporter instrumentation.
 *
 * Reporter-only (Node 22): --test-reporter allure-node-test/reporter
 * Full Runtime API (Node 26.1+): also --import allure-node-test/setup
 */
function nodeSupportsAllureSetup() {
  const [major, minor] = process.versions.node.split(".").map(Number);
  return major > 26 || (major === 26 && minor >= 1);
}

function parseArgs(argv) {
  let packageName = process.env.ALLURE_PACKAGE ?? null;
  const testArgs = [];

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    if (arg === "--package" || arg === "-p") {
      packageName = argv[++i];
      continue;
    }
    testArgs.push(arg);
  }

  if (!packageName) {
    console.error("node-test-allure: --package <name> is required (e.g. web, ui)");
    process.exit(2);
  }

  if (testArgs.length === 0) {
    console.error("node-test-allure: no test files or globs provided");
    process.exit(2);
  }

  return { packageName, testArgs };
}

const { packageName, testArgs } = parseArgs(process.argv.slice(2));
const resultsDir = path.join("allure-results", "js", packageName);

const nodeArgs = ["--test", "--test-reporter", "allure-node-test/reporter"];
if (nodeSupportsAllureSetup()) {
  nodeArgs.push("--import", "allure-node-test/setup");
}
nodeArgs.push(...testArgs);

const result = spawnSync(process.execPath, nodeArgs, {
  cwd: process.cwd(),
  env: { ...process.env, ALLURE_RESULTS_DIR: resultsDir },
  stdio: "inherit",
});

process.exit(result.status ?? 1);

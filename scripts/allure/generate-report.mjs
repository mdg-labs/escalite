#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { readFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { countResultJsonFiles, findAllureResultDirs } from "./list-result-dirs.mjs";

const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "../..");
const resultsRoot = path.join(repoRoot, "allure-results");
const reportDir = path.join(repoRoot, "allure-report");

async function main() {
  const resultDirs = await findAllureResultDirs(resultsRoot);
  const resultFileCount = await countResultJsonFiles(resultDirs);

  const generateArgs = ["exec", "allure", "generate", "--config", "./allurerc.mjs"];

  if (resultDirs.length === 0) {
    console.log("No Allure result directories found; generating empty report.");
    generateArgs.push(path.relative(repoRoot, resultsRoot));
  } else {
    console.log(
      `Generating Allure report from ${resultDirs.length} result director${resultDirs.length === 1 ? "y" : "ies"} (${resultFileCount} *-result.json file${resultFileCount === 1 ? "" : "s"}).`,
    );
    for (const resultDir of resultDirs) {
      console.log(`  - ${path.relative(repoRoot, resultDir)}`);
      generateArgs.push(path.relative(repoRoot, resultDir));
    }
  }

  const generate = spawnSync("pnpm", generateArgs, {
    cwd: repoRoot,
    env: process.env,
    stdio: "inherit",
  });

  if (generate.status !== 0) {
    process.exit(generate.status ?? 1);
  }

  if (resultFileCount === 0) {
    return;
  }

  const summaryPath = path.join(reportDir, "summary.json");
  let summary;
  try {
    summary = JSON.parse(await readFile(summaryPath, "utf8"));
  } catch (error) {
    console.error(`Failed to read Allure summary at ${path.relative(repoRoot, summaryPath)}:`, error);
    process.exit(1);
  }

  const total = summary.stats?.total ?? 0;
  if (total === 0) {
    console.error(
      `Allure report has stats.total=0 but ${resultFileCount} *-result.json file(s) were present.`,
    );
    console.error("Result directories:");
    for (const resultDir of resultDirs) {
      console.error(`  - ${path.relative(repoRoot, resultDir)}`);
    }
    process.exit(1);
  }

  console.log(`Allure report generated with ${total} test(s).`);
}

await main();

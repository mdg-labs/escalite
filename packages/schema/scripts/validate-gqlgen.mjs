import { spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import path from "node:path";

const packageRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");

const result = spawnSync(
  "go",
  ["run", "github.com/99designs/gqlgen", "generate", "--config", "gqlgen.yml"],
  {
    cwd: packageRoot,
    stdio: "inherit",
    env: { ...process.env, GOWORK: "off" },
  },
);

if (result.status !== 0) {
  process.exit(result.status ?? 1);
}

console.log("@escalite/schema: gqlgen SDL validation passed");

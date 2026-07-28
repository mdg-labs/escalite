import { defineConfig } from "allure";

const historyPath =
  process.env.ALLURE_HISTORY_PATH ?? "./.allure/history.jsonl";

export default defineConfig({
  name: "Escalite Test Report",
  output: "./allure-report",
  historyPath,
  plugins: {
    awesome: {
      options: {
        singleFile: true,
        reportLanguage: "en",
      },
    },
  },
});

import { defineConfig } from "allure";

export default defineConfig({
  name: "Escalite Test Report",
  output: "./allure-report",
  historyPath: "./.allure/history.jsonl",
  plugins: {
    awesome: {
      options: {
        singleFile: true,
        reportLanguage: "en",
      },
    },
  },
});

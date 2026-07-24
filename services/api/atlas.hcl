// Atlas project config for Escalite API (doc 08 / ADR 0001).
// Run from services/api: atlas migrate diff <name> --env local

variable "database_url" {
  type    = string
  default = getenv("ESCALITE_DATABASE_URL")
}

env "local" {
  url = var.database_url
  dev = "docker://postgres/16/dev?search_path=public"

  schema {
    src = "file://schema"
  }

  migration {
    dir = "file://migrations"
  }
}

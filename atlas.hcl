data "external_schema" "gorm" {
  program = [
    "go", "run", "-mod=mod",
    "./loader",
  ]
}

env "gorm" {
  src = data.external_schema.gorm.url
  dev = "docker://postgres/14/dev"
  migration {
    dir = "file://internal/database/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

env "ci" {
  src = data.external_schema.gorm.url
  dev = getenv("ATLAS_DEV_URL")
  migration {
    dir = "file://internal/database/migrations"
  }
  format {
    migrate {
      diff = "{{ sql . \"  \" }}"
    }
  }
}

# Used only for `atlas migrate down` in local development (docker-compose).
# No `src` intentionally: this env cannot be used with `migrate diff`.
env "local" {
  url = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
  dev = "docker://postgres/14/dev"
  migration {
    dir = "file://internal/database/migrations"
  }
}

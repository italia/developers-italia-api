CREATE TABLE "publishers" (
  "id" text,
  "email" text,
  "description" text NOT NULL,
  "active" boolean NOT NULL DEFAULT true,
  "alternative_id" text,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_publishers_created_at" ON "publishers" ("created_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_publishers_alternative_id" ON "publishers" ("alternative_id");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_publishers_description" ON "publishers" ("description");

CREATE TABLE "publishers_code_hosting" (
  "id" text,
  "url" text NOT NULL,
  "group" boolean NOT NULL DEFAULT true,
  "publisher_id" text,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_publishers_code_hosting_created_at" ON "publishers_code_hosting" ("created_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_publishers_code_hosting_url" ON "publishers_code_hosting" ("url");

CREATE TABLE "software_urls" (
  "id" text,
  "url" text,
  "software_id" text NOT NULL,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_software_urls_created_at" ON "software_urls" ("created_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_software_urls_url" ON "software_urls" ("url");

CREATE TABLE "software" (
  "id" text,
  "software_url_id" text NOT NULL,
  "publiccode_yml" text,
  "active" boolean NOT NULL DEFAULT true,
  "vitality" text,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_software_created_at" ON "software" ("created_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_software_software_url_id" ON "software" ("software_url_id");

CREATE TABLE "webhooks" (
  "id" text,
  "url" text,
  "secret" text,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  "entity_id" text,
  "entity_type" text,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_webhooks_created_at" ON "webhooks" ("created_at");
CREATE UNIQUE INDEX IF NOT EXISTS "idx_webhook_url" ON "webhooks" ("url", "entity_id", "entity_type");

CREATE TABLE "events" (
  "id" text,
  "type" text,
  "entity_type" text,
  "entity_id" text,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  "deleted_at" timestamptz,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_events_deleted_at" ON "events" ("deleted_at");

CREATE TABLE "logs" (
  "id" text,
  "message" text NOT NULL,
  "created_at" timestamptz,
  "updated_at" timestamptz,
  "deleted_at" timestamptz,
  "entity_id" text,
  "entity_type" text,
  "entity" text GENERATED ALWAYS AS (
    CASE WHEN entity_id IS NULL THEN NULL
    ELSE ('/' || entity_type || '/' || entity_id)
    END
  ) STORED,
  PRIMARY KEY ("id")
);
CREATE INDEX IF NOT EXISTS "idx_logs_deleted_at" ON "logs" ("deleted_at");
CREATE INDEX IF NOT EXISTS "idx_logs_created_at" ON "logs" ("created_at");

ALTER TABLE "publishers_code_hosting"
  ADD CONSTRAINT "fk_publishers_code_hosting"
  FOREIGN KEY ("publisher_id") REFERENCES "publishers"("id")
  ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE "software_urls"
  ADD CONSTRAINT "fk_software_aliases"
  FOREIGN KEY ("software_id") REFERENCES "software"("id");

ALTER TABLE "software_urls"
  ADD CONSTRAINT "fk_software_url"
  FOREIGN KEY ("software_id") REFERENCES "software"("id");

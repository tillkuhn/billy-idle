-- init database
CREATE TABLE IF NOT EXISTS track (
 "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
 "busy_start" DATETIME NOT NULL DEFAULT (datetime(CURRENT_TIMESTAMP, 'localtime')),
 "busy_end" DATETIME,
 "message" TEXT,
 "task" TEXT,
 "client" TEXT );

CREATE TABLE IF NOT EXISTS punch (
 "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
 "day" DATE NOT NULL DEFAULT (current_date),
 "busy_secs" INTEGER NOT NULL DEFAULT 0,
 "created" DATETIME NOT NULL DEFAULT (datetime(CURRENT_TIMESTAMP, 'localtime')),
 "client" TEXT );
-- SQLite does not support add column if not exists https://stackoverflow.com/q/3604310/4292075
-- ALTER TABLE track ADD COLUMN IF NOT EXISTS task TEXT;
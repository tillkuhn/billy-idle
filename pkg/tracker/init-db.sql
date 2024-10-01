-- init database
CREATE TABLE IF NOT EXISTS track (
 "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
 "busy_start" DATETIME NOT NULL DEFAULT (datetime(CURRENT_TIMESTAMP, 'localtime')),
 "busy_end" DATETIME,
 "message" TEXT,
 "task" TEXT,
 "client" TEXT );
-- SQLite does not support add column if not exists https://stackoverflow.com/q/3604310/4292075
-- ALTER TABLE track ADD COLUMN IF NOT EXISTS task TEXT;
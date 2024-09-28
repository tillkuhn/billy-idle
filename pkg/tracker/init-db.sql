-- init database
CREATE TABLE IF NOT EXISTS track (
 "id" integer NOT NULL PRIMARY KEY AUTOINCREMENT,
 "busy_start" DATETIME NOT NULL DEFAULT (datetime(CURRENT_TIMESTAMP, 'localtime')),
 "busy_end" DATETIME,
 "client" TEXT,
 "message" TEXT);

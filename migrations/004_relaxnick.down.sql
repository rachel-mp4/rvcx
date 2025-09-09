UPDATE messages SET nick = 'wanderer' WHERE nick IS NULL;
ALTER TABLE messages ALTER COLUMN nick SET DEFAULT 'wanderer';
ALTER TABLE messages ALTER COLUMN nick SET NOT NULL;

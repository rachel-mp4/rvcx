CREATE TABLE reports (
  id SERIAL primary key,
  uri TEXT NOT NULL,
  reason TEXT NOT NULL,
  did TEXT,
  addr TEXT,
  posted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
);

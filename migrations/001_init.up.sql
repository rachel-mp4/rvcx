CREATE TABLE profiles (
	did TEXT PRIMARY KEY,
	display_name TEXT,
	default_nick TEXT,
	status TEXT,
	avatar_cid TEXT,
	avatar_mime TEXT,
	color INTEGER CHECK (color BETWEEN 0 and 16777215),
	uri TEXT NOT NULL UNIQUE,
	cid TEXT NOT NULL,
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE did_handle (
	handle TEXT PRIMARY KEY,
	did TEXT NOT NULL UNIQUE,
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE channels (
	uri TEXT PRIMARY KEY,
	cid TEXT NOT NULL,
	did TEXT NOT NULL,
	host TEXT NOT NULL,
	title TEXT NOT NULL,
	topic TEXT,
	created_at TIMESTAMPTZ NOT NULL,
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE signets (
	uri TEXT PRIMARY KEY,
	did TEXT NOT NULL,
	channel_uri TEXT NOT NULL,
	FOREIGN KEY (channel_uri) REFERENCES channels(uri) ON DELETE CASCADE,
	message_id INTEGER CHECK (message_id BETWEEN 0 AND 4294967295),
	cid TEXT NOT NULL,
	started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON signets (channel_uri, message_id DESC);

CREATE TABLE messages (
	uri TEXT PRIMARY KEY,
	did TEXT NOT NULL,
	signet_uri TEXT NOT NULL,
	FOREIGN KEY (signet_uri) REFERENCES signets(uri) ON DELETE CASCADE,
	body TEXT,
	nick TEXT NOT NULL DEFAULT 'wanderer',
	color INTEGER CHECK (color BETWEEN 0 AND 16777215),
	cid TEXT NOT NULL,
	posted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON messages (signet_uri);

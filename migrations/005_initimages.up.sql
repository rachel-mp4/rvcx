CREATE TABLE medias (
	uri TEXT PRIMARY KEY,
	did TEXT NOT NULL,
	signet_uri TEXT NOT NULL,
	FOREIGN KEY (signet_uri) REFERENCES signets(uri) ON DELETE CASCADE,
	media_cid TEXT,
	media_mime TEXT,
	alt TEXT,
	nick TEXT NOT NULL,
	color INTEGER CHECK (color BETWEEN 0 AND 16777215),
	cid TEXT NOT NULL,
	posted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
	indexed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX ON medias (signet_uri);

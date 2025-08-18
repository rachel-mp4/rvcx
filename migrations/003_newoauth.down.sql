DROP TABLE IF EXISTS sessions;
DROP TABLE IF EXISTS requests;

CREATE TABLE oauthrequests (
	id SERIAL PRIMARY KEY,
	authserver_iss TEXT,
	state TEXT,
	did TEXT,
	pds_url TEXT,
	pkce_verifier TEXT,
	dpop_auth_server_nonce TEXT,
	dpop_private_jwk TEXT
);

CREATE TABLE oauthsessions (
	id SERIAL PRIMARY KEY,
	authserver_iss TEXT,
	state TEXT,
	did TEXT,
	pds_url TEXT,
	pkce_verifier TEXT,
	dpop_auth_server_nonce TEXT,
	dpop_private_jwk TEXT,
	dpop_pds_nonce TEXT,	
	access_token TEXT,
	refresh_token TEXT,
	expiration TIMESTAMPTZ
);

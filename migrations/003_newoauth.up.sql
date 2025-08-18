DROP TABLE IF EXISTS oauthsessions;
DROP TABLE IF EXISTS oauthrequests;

CREATE TABLE requests (
  state TEXT NOT NULL PRIMARY KEY,
  authserver_url TEXT NOT NULL,
  account_did TEXT,
  scopes TEXT NOT NULL,
  request_uri TEXT NOT NULL,
  authserver_token_endpoint TEXT NOT NULL,
  pkce_verifier TEXT NOT NULL,
  dpop_authserver_nonce TEXT NOT NULL,
  dpop_privatekey_multibase TEXT NOT NULL
);

CREATE TABLE sessions (
  session_id TEXT NOT NULL PRIMARY KEY,
  account_did TEXT NOT NULL,
  host_url TEXT NOT NULL,
  authserver_url TEXT NOT NULL,
  authserver_token_endpoint TEXT NOT NULL,
  scopes TEXT NOT NULL,
  access_token TEXT NOT NULL,
  refresh_token TEXT NOT NULL,
  dpop_authserver_nonce TEXT NOT NULL,
  dpop_host_nonce TEXT NOT NULL,
  dpop_privatekey_multibase TEXT NOT NULL
);

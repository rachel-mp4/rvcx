INSERT INTO profiles (did, display_name, default_nick, status, avatar_cid, avatar_mime, color, uri, cid)
VALUES
('did:example:alice', 'Alice Example', 'alice', 'Chilling', 'bafybeib6...', 'image/png', 16711680, 'at://did:example:alice/app.bsky.actor.profile/self', 'cid1'),
('did:example:bob', 'Bob Example', 'bobby', 'Working hard', 'bafybeib7...', 'image/jpeg', 65280, 'at://did:example:bob/app.bsky.actor.profile/self', 'cid2');

INSERT INTO did_handles (handle, did)
VALUES
('alice.com', 'did:example:alice'),
('bob.net', 'did:example:bob');

INSERT INTO channels (uri, cid, did, host, title, topic, created_at)
VALUES
('at://did:example:alice/org.xcvr.feed.channel/general', 'chanCid1', 'did:example:alice', 'xcvr.org', 'General Chat', 'All-purpose chatter', now() - interval '2 days'),
('at://did:example:bob/org.xcvr.feed.channel/help', 'chanCid2', 'did:example:bob', 'xcvr.org', 'Help Channel', 'Support and help', now() - interval '1 day');

INSERT INTO signets (uri, did, channel_uri, message_id, cid)
VALUES
('at://did:example:xcvr/org.xcvr.lrc.signet/signet1', 'did:example:alice', 'at://did:example:alice/org.xcvr.feed.channel/general', 1, 'signetCid1'),
('at://did:example:xcvr/org.xcvr.lrc.signet/signet2', 'did:example:bob', 'at://did:example:bob/org.xcvr.feed.channel/help', 2, 'signetCid2');

INSERT INTO messages (uri, did, signet_uri, body, nick, color, cid)
VALUES
('at://did:example:alice/org.xcvr.lrc.message/msg1', 'did:example:alice', 'at://did:example:xcvr/org.xcvr.lrc.signet/signet1', 'Hey, welcome to the general chat!', 'alice', 16711680, 'msgCid1'),
('at://did:example:bob/org.xcvr.lrc.message/msg2', 'did:example:bob', 'at://did:example:xcvr/org.xcvr.lrc.signet/signet2', 'How can I help you today?', 'bobby', 65280, 'msgCid2');

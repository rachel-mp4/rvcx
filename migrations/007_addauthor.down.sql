UPDATE signets SET author_handle = '' WHERE author_handle IS NULL;
ALTER TABLE signets ALTER COLUMN author_handle SET NOT NULL;
ALTER TABLE signets DROP COLUMN author;

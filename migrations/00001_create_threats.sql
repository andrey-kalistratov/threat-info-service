-- +goose Up
CREATE TABLE threats (
    md5 text PRIMARY KEY,
    name text NOT NULL,
    description text NOT NULL,
    malware_types text[] NOT NULL,
    confidence integer,

    CONSTRAINT threats_md5_format
        CHECK (md5 COLLATE "C" ~ '^[0-9a-f]{32}$'),
    CONSTRAINT threats_description_length
        CHECK (char_length(description) <= 4096),
    CONSTRAINT threats_confidence_range
        CHECK (confidence BETWEEN 0 AND 100),
    CONSTRAINT threats_malware_types_dimensions
        CHECK (cardinality(malware_types) = 0 OR array_ndims(malware_types) = 1),
    CONSTRAINT threats_malware_types_values
        CHECK (malware_types <@ ARRAY[
            'backdoor', 'ransomware', 'spyware',
            'trojan', 'virus', 'worm'
        ]::text[])
);

-- +goose Down
DROP TABLE threats;

CREATE TABLE denylist (
  network cidr NOT NULL UNIQUE
);

CREATE INDEX denylist_network_gist
ON denylist
USING gist (network inet_ops);
CREATE TABLE screenshots (
  id bigint PRIMARY KEY
);

CREATE TABLE blocks (
  id            serial PRIMARY KEY,
  screenshot_id bigint NOT NULL REFERENCES screenshots (id),
  body          text NOT NULL
);

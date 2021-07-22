CREATE TABLE connectors (
    id                   SERIAL PRIMARY KEY,
    name                 TEXT NOT NULL,
    type                 TEXT NOT NULL,
    docker_image_name    TEXT NOT NULL,
    docker_image_tag     TEXT NOT NULL,
    destination_type     TEXT NOT NULL,
    spec                 TEXT NOT NULL,
    created_at           TEXT DEFAULT TO_CHAR(CURRENT_TIMESTAMP, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
    updated_at           TEXT DEFAULT TO_CHAR(CURRENT_TIMESTAMP, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),

    UNIQUE(name, type),
    UNIQUE(docker_image_name, docker_image_tag)
);

CREATE INDEX connectors_type_idx ON connectors (type);

CREATE TABLE endpoints (
    id                   SERIAL PRIMARY KEY,
    name                 TEXT NOT NULL,
    type                 TEXT NOT NULL,
    connector_id         INT NOT NULL REFERENCES connectors (id) ON DELETE CASCADE,
    config               TEXT NOT NULL,
    catalog              TEXT NOT NULL,
    last_discovered      TEXT NOT NULL,
    created_at           TEXT NOT NULL,
    updated_at           TEXT NOT NULL,

    UNIQUE(name, type)
);

CREATE INDEX endpoints_type_idx ON endpoints (type);

CREATE TABLE syncs (
    id                        SERIAL PRIMARY KEY,
    name                      TEXT NOT NULL,
    source_endpoint_id        INT NOT NULL REFERENCES endpoints (id) ON DELETE CASCADE,
    destination_endpoint_id   INT NOT NULL REFERENCES endpoints (id) ON DELETE CASCADE,
    schedule_interval         INT NOT NULL,
    enabled                   BOOLEAN NOT NULL,
    basic_normalization       BOOLEAN NOT NULL,
    state                     TEXT NOT NULL,
    config                    TEXT NOT NULL,
    configured_catalog        TEXT NOT NULL,
    created_at                TEXT NOT NULL,
    updated_at                TEXT NOT NULL,

    UNIQUE(name)
);

CREATE TABLE runs (
    id                      SERIAL PRIMARY KEY,
    sync_id                 INT NOT NULL REFERENCES syncs (id) ON DELETE CASCADE,
    execution_date          TEXT NOT NULL,
    status                  TEXT NOT NULL,
    stats                   TEXT NOT NULL,
    options                 TEXT NOT NULL,
    temporal_workflow_id    TEXT NOT NULL,
    temporal_run_id         TEXT NOT NULL,

    UNIQUE(sync_id, execution_date)
);

CREATE INDEX runs_status_idx ON runs (status);
CREATE INDEX runs_sync_id_idx ON runs (sync_id);
CREATE UNIQUE INDEX runs_sync_id_execution_date_idx ON runs (sync_id, execution_date);



INSERT INTO connectors(name, type, docker_image_name, docker_image_tag, destination_type, spec) VALUES
('Local JSON', 'destination', 'airbyte/destination-local-json', '0.2.8', 'other', '{"type": "SPEC"}'),
('Local CSV', 'destination', 'airbyte/destination-csv', '0.2.8', 'other', '{"type": "SPEC"}'),
('Postgres', 'destination', 'airbyte/destination-postgres', '0.3.8', 'postgres', '{"type": "SPEC"}'),
('BigQuery', 'destination', 'airbyte/destination-bigquery', '0.3.8', 'bigquery', '{"type": "SPEC"}'),
('BigQuery (denormalized typed struct)', 'destination', 'airbyte/destination-bigquery-denormalized', '0.1.1', 'other', '{"type": "SPEC"}'),
('Google Cloud Storage (GCS)', 'destination', 'airbyte/destination-gcs', '0.1.0', 'other', '{"type": "SPEC"}'),
('Google PubSub', 'destination', 'airbyte/destination-pubsub', '0.1.0', 'other', '{"type": "SPEC"}'),
('Snowflake', 'destination', 'airbyte/destination-snowflake', '0.3.11', 'snowflake', '{"type": "SPEC"}'),
('S3', 'destination', 'airbyte/destination-s3', '0.1.9', 'other', '{"type": "SPEC"}'),
('Redshift', 'destination', 'airbyte/destination-redshift', '0.3.12', 'redshift', '{"type": "SPEC"}'),
('MeiliSearch', 'destination', 'airbyte/destination-meilisearch', '0.2.8', 'other', '{"type": "SPEC"}'),
('MySQL', 'destination', 'airbyte/destination-mysql', '0.1.9', 'mysql', '{"type": "SPEC"}'),
('MS SQL Server', 'destination', 'airbyte/destination-mssql', '0.1.6', 'other', '{"type": "SPEC"}'),
('Oracle (Alpha)', 'destination', 'airbyte/destination-oracle', '0.1.3', 'other', '{"type": "SPEC"}'),
('Kafka', 'destination', 'airbyte/destination-kafka', '0.1.0', 'other', '{"type": "SPEC"}')
;

INSERT INTO connectors(name, type, docker_image_name, docker_image_tag, destination_type, spec) VALUES
('Amazon Seller Partner', 'source', 'airbyte/source-amazon-seller-partner', '0.1.3', '', '{"type": "SPEC"}'),
('Asana', 'source', 'airbyte/source-asana', '0.1.1', '', '{"type": "SPEC"}'),
('Exchange Rates Api', 'source', 'airbyte/source-exchange-rates', '0.2.3', '', '{"type": "SPEC"}'),
('File', 'source', 'airbyte/source-file', '0.2.4', '', '{"type": "SPEC"}'),
('Google Ads', 'source', 'airbyte/source-google-ads', '0.1.2', '', '{"type": "SPEC"}'),
('Google Adwords (Deprecated)', 'source', 'airbyte/source-google-adwords-singer', '0.2.6', '', '{"type": "SPEC"}'),
('GitHub', 'source', 'airbyte/source-github', '0.1.2', '', '{"type": "SPEC"}'),
('Microsoft SQL Server (MSSQL)', 'source', 'airbyte/source-mssql', '0.3.3', '', '{"type": "SPEC"}'),
('Pipedrive', 'source', 'airbyte/source-pipedrive', '0.1.0', '', '{"type": "SPEC"}'),
('Postgres', 'source', 'airbyte/source-postgres', '0.3.7', '', '{"type": "SPEC"}'),
('Cockroachdb', 'source', 'airbyte/source-cockroachdb', '0.1.1', '', '{"type": "SPEC"}'),
('PostHog', 'source', 'airbyte/source-posthog', '0.1.2', '', '{"type": "SPEC"}'),
('Recurly', 'source', 'airbyte/source-recurly', '0.2.4', '', '{"type": "SPEC"}'),
('Sendgrid', 'source', 'airbyte/source-sendgrid', '0.2.6', '', '{"type": "SPEC"}'),
('Marketo', 'source', 'airbyte/source-marketo-singer', '0.2.3', '', '{"type": "SPEC"}'),
('Google Sheets', 'source', 'airbyte/source-google-sheets', '0.2.3', '', '{"type": "SPEC"}'),
('MySQL', 'source', 'airbyte/source-mysql', '0.4.0', '', '{"type": "SPEC"}'),
('Salesforce', 'source', 'airbyte/source-salesforce-singer', '0.2.4', '', '{"type": "SPEC"}'),
('Stripe', 'source', 'airbyte/source-stripe', '0.1.14', '', '{"type": "SPEC"}'),
('Mailchimp', 'source', 'airbyte/source-mailchimp', '0.2.5', '', '{"type": "SPEC"}'),
('Google Analytics', 'source', 'airbyte/source-googleanalytics-singer', '0.2.6', '', '{"type": "SPEC"}'),
('Facebook Marketing', 'source', 'airbyte/source-facebook-marketing', '0.2.14', '', '{"type": "SPEC"}'),
('Hubspot', 'source', 'airbyte/source-hubspot', '0.1.5', '', '{"type": "SPEC"}'),
('Klaviyo', 'source', 'airbyte/source-klaviyo', '0.1.1', '', '{"type": "SPEC"}'),
('Shopify', 'source', 'airbyte/source-shopify', '0.1.10', '', '{"type": "SPEC"}'),
('HTTP Request', 'source', 'airbyte/source-http-request', '0.2.4', '', '{"type": "SPEC"}'),
('Redshift', 'source', 'airbyte/source-redshift', '0.3.1', '', '{"type": "SPEC"}'),
('Twilio', 'source', 'airbyte/source-twilio', '0.1.0', '', '{"type": "SPEC"}'),
('Freshdesk', 'source', 'airbyte/source-freshdesk', '0.2.5', '', '{"type": "SPEC"}'),
('Braintree', 'source', 'airbyte/source-braintree-singer', '0.2.3', '', '{"type": "SPEC"}'),
('Greenhouse', 'source', 'airbyte/source-greenhouse', '0.2.3', '', '{"type": "SPEC"}'),
('Zendesk Chat', 'source', 'airbyte/source-zendesk-chat', '0.1.1', '', '{"type": "SPEC"}'),
('Zendesk Support', 'source', 'airbyte/source-zendesk-support-singer', '0.2.3', '', '{"type": "SPEC"}'),
('Intercom', 'source', 'airbyte/source-intercom', '0.1.0', '', '{"type": "SPEC"}'),
('Jira', 'source', 'airbyte/source-jira', '0.2.7', '', '{"type": "SPEC"}'),
('Mixpanel', 'source', 'airbyte/source-mixpanel', '0.1.0', '', '{"type": "SPEC"}'),
('Mixpanel Singer', 'source', 'airbyte/source-mixpanel-singer', '0.2.4', '', '{"type": "SPEC"}'),
('Zoom', 'source', 'airbyte/source-zoom-singer', '0.2.4', '', '{"type": "SPEC"}'),
('Microsoft teams', 'source', 'airbyte/source-microsoft-teams', '0.2.2', '', '{"type": "SPEC"}'),
('Drift', 'source', 'airbyte/source-drift', '0.2.2', '', '{"type": "SPEC"}'),
('Looker', 'source', 'airbyte/source-looker', '0.2.4', '', '{"type": "SPEC"}'),
('Plaid', 'source', 'airbyte/source-plaid', '0.2.1', '', '{"type": "SPEC"}'),
('Appstore', 'source', 'airbyte/source-appstore-singer', '0.2.4', '', '{"type": "SPEC"}'),
('Mongo DB', 'source', 'airbyte/source-mongodb', '0.3.3', '', '{"type": "SPEC"}'),
('Google Directory', 'source', 'airbyte/source-google-directory', '0.1.3', '', '{"type": "SPEC"}'),
('Instagram', 'source', 'airbyte/source-instagram', '0.1.7', '', '{"type": "SPEC"}'),
('Gitlab', 'source', 'airbyte/source-gitlab', '0.1.0', '', '{"type": "SPEC"}'),
('Google Workspace Admin Reports', 'source', 'airbyte/source-google-workspace-admin-reports', '0.1.4', '', '{"type": "SPEC"}'),
('Tempo', 'source', 'airbyte/source-tempo', '0.2.3', '', '{"type": "SPEC"}'),
('Smartsheets', 'source', 'airbyte/source-smartsheets', '0.1.5', '', '{"type": "SPEC"}'),
('Oracle DB', 'source', 'airbyte/source-oracle', '0.3.1', '', '{"type": "SPEC"}'),
('Zendesk Talk', 'source', 'airbyte/source-zendesk-talk', '0.1.2', '', '{"type": "SPEC"}'),
('Quickbooks', 'source', 'airbyte/source-quickbooks-singer', '0.1.2', '', '{"type": "SPEC"}'),
('Iterable', 'source', 'airbyte/source-iterable', '0.1.6', '', '{"type": "SPEC"}'),
('PokeAPI', 'source', 'airbyte/source-pokeapi', '0.1.1', '', '{"type": "SPEC"}'),
('Google Search Console', 'source', 'airbyte/source-google-search-console-singer', '0.1.3', '', '{"type": "SPEC"}'),
('ClickHouse', 'source', 'airbyte/source-clickhouse', '0.1.1', '', '{"type": "SPEC"}'),
('Recharge', 'source', 'airbyte/source-recharge', '0.1.1', '', '{"type": "SPEC"}'),
('Harvest', 'source', 'airbyte/source-harvest', '0.1.3', '', '{"type": "SPEC"}'),
('Amplitude', 'source', 'airbyte/source-amplitude', '0.1.1', '', '{"type": "SPEC"}'),
('Snowflake', 'source', 'airbyte/source-snowflake', '0.1.0', '', '{"type": "SPEC"}'),
('IBM Db2', 'source', 'airbyte/source-db2', '0.1.0', '', '{"type": "SPEC"}'),
('Slack', 'source', 'airbyte/source-slack', '0.1.8', '', '{"type": "SPEC"}'),
('AWS CloudTrail', 'source', 'airbyte/source-aws-cloudtrail', '0.1.1', '', '{"type": "SPEC"}'),
('US Census', 'source', 'airbyte/source-us-census', '0.1.0', '', '{"type": "SPEC"}'),
('Okta', 'source', 'airbyte/source-okta', '0.1.2', '', '{"type": "SPEC"}'),
('Survey Monkey', 'source', 'airbyte/source-surveymonkey', '0.1.0', '', '{"type": "SPEC"}'),
('Square', 'source', 'airbyte/source-square', '0.1.1', '', '{"type": "SPEC"}'),
('Zendesk Sunshine', 'source', 'airbyte/source-zendesk-sunshine', '0.1.0', '', '{"type": "SPEC"}'),
('Paypal Transaction', 'source', 'airbyte/source-paypal-transaction', '0.1.0', '', '{"type": "SPEC"}'),
('Dixa', 'source', 'airbyte/source-dixa', '0.1.0', '', '{"type": "SPEC"}'),
('Typeform', 'source', 'airbyte/source-typeform', '0.1.0', '', '{"type": "SPEC"}')
;

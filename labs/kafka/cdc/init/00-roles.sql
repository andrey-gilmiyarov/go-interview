CREATE ROLE cdc_app
    LOGIN
    PASSWORD 'cdc_app';

CREATE ROLE cdc_replication
    LOGIN
    REPLICATION
    PASSWORD 'cdc_replication';

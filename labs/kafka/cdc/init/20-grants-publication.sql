REVOKE CREATE ON SCHEMA public FROM PUBLIC;

GRANT CONNECT ON DATABASE cdc_lab TO cdc_app, cdc_replication;
GRANT USAGE ON SCHEMA public TO cdc_app, cdc_replication;

GRANT SELECT, INSERT, UPDATE, DELETE
    ON TABLE public.cdc_orders, public.cdc_commands, public.cdc_inbox, public.cdc_projections
    TO cdc_app;
GRANT SELECT, INSERT ON TABLE public.cdc_outbox TO cdc_app;
GRANT SELECT ON TABLE public.cdc_outbox TO cdc_replication;

CREATE PUBLICATION cdc_orders_pub
    FOR TABLE public.cdc_outbox
    WITH (publish = 'insert');

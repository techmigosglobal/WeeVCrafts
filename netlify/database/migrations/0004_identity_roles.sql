INSERT INTO roles (slug, name) VALUES
    ('customer', 'Customer'),
    ('seller_owner', 'Seller owner'),
    ('seller_staff', 'Seller staff'),
    ('marketplace_admin', 'Marketplace admin'),
    ('super_admin', 'Super admin'),
    ('support_agent', 'Support agent'),
    ('finance_operator', 'Finance operator'),
    ('operations_returns', 'Operations and returns')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO schema_migrations (version) VALUES (0004) ON CONFLICT (version) DO NOTHING;

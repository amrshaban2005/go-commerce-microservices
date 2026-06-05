-- Create service users
DO
$$
BEGIN
   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'catalog_write_user') THEN
      CREATE ROLE catalog_write_user LOGIN PASSWORD 'catalog_write_pass';
   END IF;

   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'inventory_user') THEN
      CREATE ROLE inventory_user LOGIN PASSWORD 'inventory_pass';
   END IF;

   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'order_user') THEN
      CREATE ROLE order_user LOGIN PASSWORD 'order_pass';
   END IF;

   IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'notification_user') THEN
      CREATE ROLE notification_user LOGIN PASSWORD 'notification_pass';
   END IF;
END
$$;

-- Create databases if they do not exist
SELECT 'CREATE DATABASE catalog_write_db OWNER catalog_write_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'catalog_write_db')\gexec

SELECT 'CREATE DATABASE inventory_db OWNER inventory_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'inventory_db')\gexec

SELECT 'CREATE DATABASE order_db OWNER order_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'order_db')\gexec

SELECT 'CREATE DATABASE notification_db OWNER notification_user'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'notification_db')\gexec


-- Catalog Write DB setup
\connect catalog_write_db

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

GRANT ALL PRIVILEGES ON DATABASE catalog_write_db TO catalog_write_user;
GRANT ALL ON SCHEMA public TO catalog_write_user;


-- Inventory DB setup
\connect inventory_db

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

GRANT ALL PRIVILEGES ON DATABASE inventory_db TO inventory_user;
GRANT ALL ON SCHEMA public TO inventory_user;


-- Order DB setup
\connect order_db

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

GRANT ALL PRIVILEGES ON DATABASE order_db TO order_user;
GRANT ALL ON SCHEMA public TO order_user;


-- Notification DB setup
\connect notification_db

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

GRANT ALL PRIVILEGES ON DATABASE notification_db TO notification_user;
GRANT ALL ON SCHEMA public TO notification_user;
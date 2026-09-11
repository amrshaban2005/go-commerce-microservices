INSERT INTO inventory (product_id, available_quantity, reserved_quantity)
VALUES
    ('1c47247b-5f3e-41ae-bd3e-c3191ee63b99', 10, 0),
    ('a6500dba-cb86-42a8-86d2-033091952b15', 10, 0)
ON CONFLICT (product_id) DO UPDATE SET
    available_quantity = EXCLUDED.available_quantity,
    reserved_quantity = EXCLUDED.reserved_quantity,
    updated_at = CURRENT_TIMESTAMP;

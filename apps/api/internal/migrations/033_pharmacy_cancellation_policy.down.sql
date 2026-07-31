UPDATE cancellation_rules
SET full_refund = TRUE, merchant_comp_pct = 0, rider_comp_pct_of_delivery = 0
WHERE vertical = 'pharmacy'
  AND order_status_at_cancel IN ('preparing', 'ready_for_pickup', 'driver_assigned', 'in_transit');

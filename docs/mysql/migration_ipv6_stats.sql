-- Migration script to add is_ipv6 to the UNIQUE constraint
-- This is required for proper IPv6 dual-stack statistics tracking

-- Step 1: Drop the old unique constraint
ALTER TABLE `requests` DROP INDEX `uniq_traffic_dedupe`;

-- Step 2: Add the new unique constraint that includes is_ipv6
ALTER TABLE `requests` ADD UNIQUE KEY `uniq_traffic_dedupe` (
    `date`,
    `domain_name`,
    `member_name`,
    `network_asn`,
    `network_name`,
    `country_code`,
    `country_name`,
    `is_ipv6`
);

-- Note: After applying this migration, IPv4 and IPv6 traffic will be tracked separately
-- All existing records will have is_ipv6 = NULL or 0 (IPv4)

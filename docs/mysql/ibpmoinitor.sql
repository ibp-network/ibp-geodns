USE ibpmonitor;

DROP TABLE `ibpmonitor_members`;
CREATE TABLE `ibpmonitor_members` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `member_index` VARCHAR(48) NOT NULL,
  `member_name` VARCHAR(48) NOT NULL,
  `member_website` VARCHAR(128) NOT NULL,
  `member_logo` VARCHAR(128) NOT NULL,
  `membership_level` TINYINT NOT NULL,
  `membership_joined_ts` DATETIME NOT NULL,
  `membership_promotion_ts` DATETIME NOT NULL,
  `service_active` TINYINT NOT NULL,
  `service_ipv4` VARCHAR(32) NOT NULL,
  `service_ipv6` VARCHAR(128) NOT NULL,
  `service_monitorUrl` VARCHAR(256) NOT NULL,
  `location_region` TINYINT NOT NULL,
  `location_latitude` DECIMAL(9,6)  NOT NULL,
  `location_longitude`  DECIMAL(9,6)  NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`member_index`, `member_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE `ibpmonitor_services`;
CREATE TABLE `ibpmonitor_services` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `service_index` VARCHAR(48) NOT NULL,
  `configuration_name` VARCHAR(48) NOT NULL,
  `configuration_type` TINYINT NOT NULL,
  `configuration_active` TINYINT NOT NULL,
  `configuration_memberLevelReq` TINYINT NOT NULL,
  `configuration_networkname` VARCHAR(48) NOT NULL,
  `configuration_stateroothash` VARCHAR(384) NOT NULL,
  `provisioned_nodes` SMALLINT NOT NULL,
  `provisioned_cores` SMALLINT NOT NULL,
  `provisioned_memory` SMALLINT NOT NULL,
  `provisioned_disk` SMALLINT NOT NULL,
  `provisioned_bandwidth` SMALLINT NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`service_index`, `configuration_name`, `configuration_networkname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE `ibpmonitor_service_assignment`;
CREATE TABLE `ibpmonitor_service_assignment` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `service_id` BIGINT NOT NULL,
  `member_id` BIGINT NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`service_id`, `member_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE `ibpmonitor_service_provider`;
CREATE TABLE `ibpmonitor_service_provider` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `service_id` INT UNSIGNED NOT NULL,
  `provider_index` VARCHAR(48) DEFAULT NULL,
  `provider_rpcUrl1` VARCHAR(128) DEFAULT NULL,
  `provider_rpcUrl2` VARCHAR(128) DEFAULT NULL,
  `provider_rpcUrl3` VARCHAR(128) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`service_id`, `provider_index`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE `ibpmonitor_usage`;
CREATE TABLE `ibpmonitor_usage` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `date` DATE NOT NULL,
  `node_id` VARCHAR(32) NOT NULL,
  `domain_name` VARCHAR(128) NOT NULL,
  `member_name` VARCHAR(64) DEFAULT NULL,
  `network_asn` VARCHAR(32) DEFAULT NULL,
  `network_name` VARCHAR(96) DEFAULT NULL,
  `country_code` VARCHAR(2) DEFAULT NULL,
  `country_name` VARCHAR(64) DEFAULT NULL,
  `is_ipv6` TINYINT NOT NULL,
  `hits` INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`date`,`domain_name`,`member_name`,`network_asn`,`network_name`,`country_code`,`country_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE `ibpmonitor_netStatus`;
CREATE TABLE `ibpmonitor_netStatus` (
  `id` INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `check_type` TINYINT NOT NULL,
  `check_name` VARCHAR(32) NOT NULL,
  `check_url` VARCHAR(128) NOT NULL,
  `member_name` VARCHAR(48) NOT NULL,
  `domain_name` VARCHAR(96) NOT NULL,
  `status` TINYINT NOT NULL,
  `is_ipv6` TINYINT NOT NULL,
  `start_time`  DATETIME NOT NULL DEFAULT(NOW()),
  `end_time` DATETIME DEFAULT NULL,
  `error` TEXT,
  `vote_data` JSON NOT NULL,
  `additional_data` JSON DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY (`check_type`,`check_name`,`check_url`,`member_name`,`domain_name`,`is_ipv6`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

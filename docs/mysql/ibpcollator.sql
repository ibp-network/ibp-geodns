CREATE TABLE `ibpcollator_usage` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `date` date NOT NULL,
  `node_id` varchar(24) COLLATE utf8mb4_general_ci NOT NULL,
  `domain_name` varchar(96) COLLATE utf8mb4_general_ci NOT NULL,
  `member_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `network_asn` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `network_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `country_code` char(2) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `country_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `is_ipv6` enum('ipv4','ipv6') NOT NULL,
  `hits` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`date`,`domain_name`(48),`member_name`(24),`network_asn`(13),`network_name`(24),`country_code`(2),`country_name`(24)),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci

CREATE TABLE `ibpcollator_netStatus` (
  `check_type` TINYINT NOT NULL,
  `check_name` VARCHAR(32) NOT NULL,
  `check_url` VARCHAR(128) NOT NULL,
  `domain_name` VARCHAR(96) NOT NULL,
  `member_name` VARCHAR(48) NOT NULL,
  `status` TINYINT NOT NULL,
  `is_ipv6` TINYINT NOT NULL,
  `start_time`  DATETIME NOT NULL DEFAULT(NOW()),
  `end_time` DATETIME DEFAULT NULL,
  `error` TEXT,
  `vote_data` JSON NOT NULL,
  `additional_data` JSON DEFAULT NULL,
  PRIMARY KEY (`check_type`(2),`check_name`(16),`check_url`(64),`member_name`(24),`domain_name`(48),`is_ipv6`(1)),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci

CREATE TABLE `ibpcollator_members` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci

CREATE TABLE `ibpcollator_services` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci

CREATE TABLE `ibpcollator_service_assignment` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `service_id` BIGINT NOT NULL,
  `member_id` BIGINT NOT NULL,
  PRIMARY KEY (`id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci

CREATE TABLE `iibpcollator_service_provider` (
  `id` BIGINT NOT NULL AUTO_INCREMENT,
  `service_id` BIGINT NOT NULL,
  `provider_index` VARCHAR(48) DEFAULT NULL,
  `provider_rpcUrl1` VARCHAR(128) DEFAULT NULL,
  `provider_rpcUrl2` VARCHAR(128) DEFAULT NULL,
  `provider_rpcUrl3` VARCHAR(128) DEFAULT NULL,
  PRIMARY KEY (`id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci


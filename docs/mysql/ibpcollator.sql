CREATE TABLE `ibpcollator_usage` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `date` date NOT NULL,
  `domain_name` varchar(96) COLLATE utf8mb4_general_ci NOT NULL,
  `member_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `network_asn` varchar(16) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `network_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `country_code` char(2) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `country_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `is_ipv6` enum('ipv4','ipv6') NOT NULL,
  `hits` int unsigned NOT NULL DEFAULT '0',
  PRIMARY KEY (`date`,`domain_name`(48),`member_name`(24),`network_asn`(13),`network_name`(24),`country_code`(2),`country_name`(24),`is_ipv6`(1)),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci

CREATE TABLE `ibpcollator_netStatus` (
  `check_type` enum('site','domain','endpoint') NOT NULL,
  `check_name` varchar(32) NOT NULL,
  `check_url` varchar(128) DEFAULT NULL,
  `domain_name` varchar(96) COLLATE utf8mb4_general_ci NOT NULL,
  `member_name` varchar(48) COLLATE utf8mb4_general_ci DEFAULT NULL,
  `status` enum('online','offline') NOT NULL,
  `is_ipv6` enum('ipv4','ipv6') NOT NULL,
  `start_time` datetime NOT NULL,
  `end_time` datetime DEFAULT NULL,
  `error` text,
  `additional_data` json DEFAULT NULL,
PRIMARY KEY (`check_type`(2),`check_name`(16),`check_url`(64),`member_name`(24),`domain_name`(48),`is_ipv6`(1)),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci
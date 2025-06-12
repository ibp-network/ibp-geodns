CREATE TABLE `ibpdns_usage` (
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
  PRIMARY KEY (`date`,`domain_name`(48),`member_name`(24),`network_asn`(13),`network_name`(24),`country_code`(2),`country_name`(24)),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci


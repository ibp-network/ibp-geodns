SET FOREIGN_KEY_CHECKS = 0;

DROP TABLE IF EXISTS `members`;
CREATE TABLE `members` (
  `id`                       INT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `member_index`             VARCHAR(48)   NOT NULL,
  `member_name`              VARCHAR(48)   NOT NULL,
  `member_website`           VARCHAR(128)  NOT NULL,
  `member_logo`              VARCHAR(128)  NOT NULL,
  `membership_level`         TINYINT       NOT NULL,
  `membership_joined_ts`     DATETIME      NOT NULL,
  `membership_promotion_ts`  DATETIME      NOT NULL,
  `service_active`           TINYINT       NOT NULL,
  `service_ipv4`             VARCHAR(32)   NOT NULL,
  `service_ipv6`             VARCHAR(128)  NOT NULL,
  `service_monitorUrl`       VARCHAR(256)  NOT NULL,
  `location_region`          TINYINT       NOT NULL,
  `location_latitude`        DECIMAL(9,6)  NOT NULL,
  `location_longitude`       DECIMAL(9,6)  NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_member_idx_name` (`member_index`, `member_name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `member_events`;
CREATE TABLE `member_events` (
  `id` int unsigned NOT NULL AUTO_INCREMENT,
  `check_type` varchar(24) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `check_name` varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `endpoint` varchar(512) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `endpoint_hash` binary(32) GENERATED ALWAYS AS (unhex(sha2(`endpoint`,256))) STORED,
  `member_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `domain_name` varchar(128) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin DEFAULT NULL,
  `status` tinyint DEFAULT NULL,
  `is_ipv6` tinyint DEFAULT NULL,
  `start_time` datetime DEFAULT CURRENT_TIMESTAMP,
  `end_time` datetime DEFAULT NULL,
  `error` text COLLATE utf8mb4_general_ci,
  `vote_data` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
  `additional_data` longtext CHARACTER SET utf8mb4 COLLATE utf8mb4_bin,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_member_event` (`check_type`,`check_name`,`endpoint_hash`,`member_name`,`domain_name`,`is_ipv6`),
  CONSTRAINT `member_events_chk_1` CHECK (json_valid(`vote_data`)),
  CONSTRAINT `member_events_chk_2` CHECK (json_valid(`additional_data`))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `services`;
CREATE TABLE `services` (
  `id`                           INT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `service_index`                VARCHAR(48)   NOT NULL,
  `configuration_name`           VARCHAR(48)   NOT NULL,
  `configuration_type`           TINYINT       NOT NULL,
  `configuration_active`         TINYINT       NOT NULL,
  `configuration_memberLevelReq` TINYINT       NOT NULL,
  `configuration_networkname`    VARCHAR(48)   NOT NULL,
  `configuration_stateroothash`  VARCHAR(384)  NOT NULL,
  `provisioned_nodes`            SMALLINT      NOT NULL,
  `provisioned_cores`            SMALLINT      NOT NULL,
  `provisioned_memory`           SMALLINT      NOT NULL,
  `provisioned_disk`             SMALLINT      NOT NULL,
  `provisioned_bandwidth`        SMALLINT      NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_service_cfg` (
        `service_index`, `configuration_name`, `configuration_networkname`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `service_provider`;
CREATE TABLE `service_provider` (
  `id`              INT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `service_id`      INT UNSIGNED  NOT NULL,
  `provider_index`  VARCHAR(48)   DEFAULT NULL,
  `provider_rpcUrl1` VARCHAR(128) DEFAULT NULL,
  `provider_rpcUrl2` VARCHAR(128) DEFAULT NULL,
  `provider_rpcUrl3` VARCHAR(128) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_service_provider_idx` (`service_id`, `provider_index`),
  CONSTRAINT `fk_provider_service`
    FOREIGN KEY (`service_id`) REFERENCES `services` (`id`)
      ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `service_assignment`;
CREATE TABLE `service_assignment` (
  `id`         INT UNSIGNED NOT NULL AUTO_INCREMENT,
  `service_id` INT UNSIGNED NOT NULL,
  `member_id`  INT UNSIGNED NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_service_member` (`service_id`, `member_id`),
  CONSTRAINT `fk_assignment_service`
    FOREIGN KEY (`service_id`) REFERENCES `services` (`id`)
      ON UPDATE CASCADE ON DELETE CASCADE,
  CONSTRAINT `fk_assignment_member`
    FOREIGN KEY (`member_id`)  REFERENCES `members` (`id`)
      ON UPDATE CASCADE ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

DROP TABLE IF EXISTS `requests`;
CREATE TABLE `requests` (
  `id`            INT UNSIGNED  NOT NULL AUTO_INCREMENT,
  `date`          DATE          DEFAULT NULL,
  `node_id`       VARCHAR(32)   DEFAULT NULL,
  `domain_name`   VARCHAR(128)  DEFAULT NULL,
  `member_name`   VARCHAR(64)   DEFAULT NULL,
  `network_asn`   VARCHAR(32)   DEFAULT NULL,
  `network_name`  VARCHAR(96)   DEFAULT NULL,
  `country_code`  VARCHAR(2)    DEFAULT NULL,
  `country_name`  VARCHAR(64)   DEFAULT NULL,
  `is_ipv6`       TINYINT       DEFAULT NULL,
  `hits`          INT UNSIGNED  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uniq_traffic_dedupe` (
        `date`,`domain_name`,`member_name`,
        `network_asn`,`network_name`,
        `country_code`,`country_name`
  )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;

SET FOREIGN_KEY_CHECKS = 1;

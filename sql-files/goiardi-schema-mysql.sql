-- MySQL dump 10.13  Distrib 5.6.16, for osx10.9 (x86_64)
--
-- Host: localhost    Database: goiardi_test
-- ------------------------------------------------------
-- Server version	5.6.16-log

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Table structure for table `clients`
--

DROP TABLE IF EXISTS `clients`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `clients` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(2048) NOT NULL,
  `nodename` varchar(2048) DEFAULT NULL,
  `validator` tinyint(4) DEFAULT '0',
  `admin` tinyint(4) DEFAULT '0',
  `organization_id` int(11) NOT NULL DEFAULT '1',
  `public_key` text,
  `certificate` text,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_id` (`organization_id`,`name`(250))
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `clients`
--

LOCK TABLES `clients` WRITE;
/*!40000 ALTER TABLE `clients` DISABLE KEYS */;
/*!40000 ALTER TABLE `clients` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cookbook_versions`
--

DROP TABLE IF EXISTS `cookbook_versions`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `cookbook_versions` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `cookbook_id` int(11) NOT NULL,
  `major_ver` bigint(20) NOT NULL,
  `minor_ver` bigint(20) NOT NULL,
  `patch_ver` bigint(20) NOT NULL DEFAULT '0',
  `frozen` tinyint(4) DEFAULT '0',
  `metadata` mediumtext,
  `definitions` mediumtext,
  `libraries` mediumtext,
  `attributes` mediumtext,
  `recipes` mediumtext,
  `providers` mediumtext,
  `resources` mediumtext,
  `templates` mediumtext,
  `root_files` mediumtext,
  `files` mediumtext,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `cookbook_id` (`cookbook_id`,`major_ver`,`minor_ver`,`patch_ver`),
  KEY `frozen` (`frozen`),
  CONSTRAINT `cookbook_versions_ibfk_1` FOREIGN KEY (`cookbook_id`) REFERENCES `cookbooks` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cookbook_versions`
--

LOCK TABLES `cookbook_versions` WRITE;
/*!40000 ALTER TABLE `cookbook_versions` DISABLE KEYS */;
/*!40000 ALTER TABLE `cookbook_versions` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `cookbooks`
--

DROP TABLE IF EXISTS `cookbooks`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `cookbooks` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_name` (`organization_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `cookbooks`
--

LOCK TABLES `cookbooks` WRITE;
/*!40000 ALTER TABLE `cookbooks` DISABLE KEYS */;
/*!40000 ALTER TABLE `cookbooks` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `data_bag_items`
--

DROP TABLE IF EXISTS `data_bag_items`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `data_bag_items` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `orig_name` varchar(255) NOT NULL,
  `data_bag_id` int(11) NOT NULL,
  `raw_data` mediumtext,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `data_bag_id` (`data_bag_id`,`name`),
  UNIQUE KEY `data_bag_id_2` (`data_bag_id`,`orig_name`),
  CONSTRAINT `data_bag_items_ibfk_1` FOREIGN KEY (`data_bag_id`) REFERENCES `data_bags` (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `data_bag_items`
--

LOCK TABLES `data_bag_items` WRITE;
/*!40000 ALTER TABLE `data_bag_items` DISABLE KEYS */;
/*!40000 ALTER TABLE `data_bag_items` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `data_bags`
--

DROP TABLE IF EXISTS `data_bags`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `data_bags` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_name` (`organization_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `data_bags`
--

LOCK TABLES `data_bags` WRITE;
/*!40000 ALTER TABLE `data_bags` DISABLE KEYS */;
/*!40000 ALTER TABLE `data_bags` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `environments`
--

DROP TABLE IF EXISTS `environments`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `environments` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `description` text,
  `default_attr` mediumtext,
  `override_attr` mediumtext,
  `cookbook_vers` mediumtext,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_name` (`organization_id`,`name`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `environments`
--

LOCK TABLES `environments` WRITE;
/*!40000 ALTER TABLE `environments` DISABLE KEYS */;
INSERT INTO `environments` VALUES (1,'_default','The default Chef environment',NULL,NULL,NULL,'2014-07-20 23:15:08','2014-07-20 23:15:08',1);
/*!40000 ALTER TABLE `environments` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `file_checksums`
--

DROP TABLE IF EXISTS `file_checksums`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `file_checksums` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  `checksum` varchar(32) DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `org_id` (`organization_id`,`checksum`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `file_checksums`
--

LOCK TABLES `file_checksums` WRITE;
/*!40000 ALTER TABLE `file_checksums` DISABLE KEYS */;
/*!40000 ALTER TABLE `file_checksums` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Temporary table structure for view `joined_cookbook_version`
--

DROP TABLE IF EXISTS `joined_cookbook_version`;
/*!50001 DROP VIEW IF EXISTS `joined_cookbook_version`*/;
SET @saved_cs_client     = @@character_set_client;
SET character_set_client = utf8;
/*!50001 CREATE TABLE `joined_cookbook_version` (
  `major_ver` tinyint NOT NULL,
  `minor_ver` tinyint NOT NULL,
  `patch_ver` tinyint NOT NULL,
  `version` tinyint NOT NULL,
  `id` tinyint NOT NULL,
  `metadata` tinyint NOT NULL,
  `recipes` tinyint NOT NULL,
  `organization_id` tinyint NOT NULL,
  `name` tinyint NOT NULL
) ENGINE=MyISAM */;
SET character_set_client = @saved_cs_client;

--
-- Table structure for table `log_infos`
--

DROP TABLE IF EXISTS `log_infos`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `log_infos` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `actor_id` int(11) NOT NULL DEFAULT '0',
  `actor_info` text,
  `actor_type` enum('user','client') NOT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  `time` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `action` enum('create','delete','modify') NOT NULL,
  `object_type` varchar(100) NOT NULL,
  `object_name` varchar(255) NOT NULL,
  `extended_info` text,
  PRIMARY KEY (`id`),
  KEY `actor_id` (`actor_id`),
  KEY `action` (`action`),
  KEY `object_type` (`object_type`,`object_name`),
  KEY `time` (`time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `log_infos`
--

LOCK TABLES `log_infos` WRITE;
/*!40000 ALTER TABLE `log_infos` DISABLE KEYS */;
/*!40000 ALTER TABLE `log_infos` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `nodes`
--

DROP TABLE IF EXISTS `nodes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `nodes` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `chef_environment` varchar(255) NOT NULL DEFAULT '_default',
  `run_list` mediumtext,
  `automatic_attr` mediumtext,
  `normal_attr` mediumtext,
  `default_attr` mediumtext,
  `override_attr` mediumtext,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_name` (`organization_id`,`name`),
  KEY `chef_environment` (`chef_environment`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `nodes`
--

LOCK TABLES `nodes` WRITE;
/*!40000 ALTER TABLE `nodes` DISABLE KEYS */;
/*!40000 ALTER TABLE `nodes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `organizations`
--

DROP TABLE IF EXISTS `organizations`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `organizations` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `description` text,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB AUTO_INCREMENT=2 DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `organizations`
--

LOCK TABLES `organizations` WRITE;
/*!40000 ALTER TABLE `organizations` DISABLE KEYS */;
INSERT INTO `organizations` VALUES (1,'default',NULL,'0000-00-00 00:00:00','0000-00-00 00:00:00');
/*!40000 ALTER TABLE `organizations` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `reports`
--

DROP TABLE IF EXISTS `reports`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `reports` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `run_id` varchar(36) NOT NULL,
  `node_name` varchar(255) DEFAULT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  `start_time` datetime DEFAULT NULL,
  `end_time` datetime DEFAULT NULL,
  `total_res_count` int(11) DEFAULT '0',
  `status` enum('started','success','failure') DEFAULT NULL,
  `run_list` text,
  `resources` mediumtext,
  `data` mediumtext,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `run_id` (`run_id`),
  KEY `organization_id` (`organization_id`),
  KEY `node_name` (`node_name`,`organization_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `reports`
--

LOCK TABLES `reports` WRITE;
/*!40000 ALTER TABLE `reports` DISABLE KEYS */;
/*!40000 ALTER TABLE `reports` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `roles`
--

DROP TABLE IF EXISTS `roles`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `roles` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `description` text,
  `run_list` mediumtext,
  `env_run_lists` mediumtext,
  `default_attr` mediumtext,
  `override_attr` mediumtext,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  `organization_id` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_name` (`organization_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8 ROW_FORMAT=COMPRESSED;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `roles`
--

LOCK TABLES `roles` WRITE;
/*!40000 ALTER TABLE `roles` DISABLE KEYS */;
/*!40000 ALTER TABLE `roles` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sandboxes`
--

DROP TABLE IF EXISTS `sandboxes`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `sandboxes` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `sbox_id` varchar(32) NOT NULL,
  `creation_time` datetime NOT NULL,
  `checksums` mediumtext,
  `completed` tinyint(4) DEFAULT '0',
  `organization_id` int(11) NOT NULL DEFAULT '1',
  PRIMARY KEY (`id`),
  UNIQUE KEY `organization_sbox` (`organization_id`,`sbox_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sandboxes`
--

LOCK TABLES `sandboxes` WRITE;
/*!40000 ALTER TABLE `sandboxes` DISABLE KEYS */;
/*!40000 ALTER TABLE `sandboxes` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `users`
--

DROP TABLE IF EXISTS `users`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
CREATE TABLE `users` (
  `id` int(11) NOT NULL AUTO_INCREMENT,
  `name` varchar(255) NOT NULL,
  `displayname` varchar(1024) DEFAULT NULL,
  `email` varchar(255) DEFAULT NULL,
  `admin` tinyint(4) DEFAULT '0',
  `public_key` text,
  `passwd` varchar(128) DEFAULT NULL,
  `salt` varbinary(64) DEFAULT NULL,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  UNIQUE KEY `email` (`email`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `users`
--

LOCK TABLES `users` WRITE;
/*!40000 ALTER TABLE `users` DISABLE KEYS */;
/*!40000 ALTER TABLE `users` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Final view structure for view `joined_cookbook_version`
--

/*!50001 DROP TABLE IF EXISTS `joined_cookbook_version`*/;
/*!50001 DROP VIEW IF EXISTS `joined_cookbook_version`*/;
/*!50001 SET @saved_cs_client          = @@character_set_client */;
/*!50001 SET @saved_cs_results         = @@character_set_results */;
/*!50001 SET @saved_col_connection     = @@collation_connection */;
/*!50001 SET character_set_client      = utf8 */;
/*!50001 SET character_set_results     = utf8 */;
/*!50001 SET collation_connection      = utf8_general_ci */;
/*!50001 CREATE ALGORITHM=UNDEFINED */
/*!50013 DEFINER=`root`@`localhost` SQL SECURITY DEFINER */
/*!50001 VIEW `joined_cookbook_version` AS select `v`.`major_ver` AS `major_ver`,`v`.`minor_ver` AS `minor_ver`,`v`.`patch_ver` AS `patch_ver`,concat(`v`.`major_ver`,'.',`v`.`minor_ver`,'.',`v`.`patch_ver`) AS `version`,`v`.`id` AS `id`,`v`.`metadata` AS `metadata`,`v`.`recipes` AS `recipes`,`c`.`organization_id` AS `organization_id`,`c`.`name` AS `name` from (`cookbooks` `c` join `cookbook_versions` `v` on((`c`.`id` = `v`.`cookbook_id`))) */;
/*!50001 SET character_set_client      = @saved_cs_client */;
/*!50001 SET character_set_results     = @saved_cs_results */;
/*!50001 SET collation_connection      = @saved_col_connection */;
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2014-07-20 23:16:52

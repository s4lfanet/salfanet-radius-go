-- Migration: create troubleshooting and invoice_templates tables
-- Run: mysql -u salfanet_user -pSalfaDB2026! salfanet_radius < scripts/migrate-missing-tables.sql

CREATE TABLE IF NOT EXISTS `troubleshooting_checklists` (
  `id` varchar(191) NOT NULL,
  `title` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `category` varchar(50) NOT NULL DEFAULT 'OTHER',
  `steps` text NOT NULL,
  `isActive` tinyint(1) NOT NULL DEFAULT 1,
  `createdAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `troubleshooting_jobs` (
  `id` varchar(191) NOT NULL,
  `title` varchar(255) NOT NULL,
  `description` text DEFAULT NULL,
  `checklistId` varchar(191) DEFAULT NULL,
  `assignedToId` varchar(191) DEFAULT NULL,
  `status` varchar(50) NOT NULL DEFAULT 'OPEN',
  `priority` varchar(50) NOT NULL DEFAULT 'MEDIUM',
  `createdAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `idx_troubleshooting_jobs_checklistId` (`checklistId`),
  KEY `idx_troubleshooting_jobs_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `troubleshooting_materials` (
  `id` varchar(191) NOT NULL,
  `jobId` varchar(191) NOT NULL,
  `name` varchar(255) NOT NULL,
  `qty` int NOT NULL DEFAULT 0,
  `unit` varchar(50) NOT NULL DEFAULT '',
  `notes` varchar(500) NOT NULL DEFAULT '',
  PRIMARY KEY (`id`),
  KEY `idx_troubleshooting_materials_jobId` (`jobId`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `invoice_templates` (
  `id` varchar(191) NOT NULL,
  `name` varchar(255) NOT NULL,
  `subject` varchar(500) NOT NULL DEFAULT '',
  `htmlBody` longtext NOT NULL,
  `isDefault` tinyint(1) NOT NULL DEFAULT 0,
  `templateType` varchar(50) NOT NULL DEFAULT 'INVOICE',
  `createdAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` datetime(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

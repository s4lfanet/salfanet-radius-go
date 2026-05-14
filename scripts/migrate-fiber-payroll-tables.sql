-- migrate-fiber-payroll-tables.sql
-- Creates missing tables: payroll_templates, fiber_cables, fiber_tubes, fiber_cores,
-- splice_points, cable_segments, core_assignment_history
-- Safe to run multiple times (uses IF NOT EXISTS)

CREATE TABLE IF NOT EXISTS `payroll_templates` (
  `id`        VARCHAR(191) NOT NULL,
  `name`      VARCHAR(255) NOT NULL,
  `baseWage`  INT          NOT NULL DEFAULT 0,
  `allowance` INT          NOT NULL DEFAULT 0,
  `deduction` INT          NOT NULL DEFAULT 0,
  `notes`     TEXT,
  `isDefault` TINYINT(1)   NOT NULL DEFAULT 0,
  `createdAt` DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt` DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `fiber_cables` (
  `id`            VARCHAR(191) NOT NULL,
  `code`          VARCHAR(100) NOT NULL,
  `name`          VARCHAR(255) NOT NULL,
  `cableType`     ENUM('SM_G652','SM_G657','MM_OM1','MM_OM2','MM_OM3','MM_OM4','MM_OM5') NOT NULL DEFAULT 'SM_G652',
  `tubeCount`     INT          NOT NULL,
  `coresPerTube`  INT          NOT NULL,
  `totalCores`    INT,
  `outerDiameter` DECIMAL(5,2),
  `manufacturer`  VARCHAR(100),
  `partNumber`    VARCHAR(100),
  `status`        ENUM('ACTIVE','RETIRED','RESERVED') NOT NULL DEFAULT 'ACTIVE',
  `notes`         TEXT,
  `createdAt`     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`     DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `fiber_cables_code_key` (`code`),
  KEY `fiber_cables_cableType_idx` (`cableType`),
  KEY `fiber_cables_status_idx` (`status`)
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `fiber_tubes` (
  `id`             VARCHAR(191) NOT NULL,
  `cableId`        VARCHAR(191) NOT NULL,
  `tubeNumber`     INT          NOT NULL,
  `colorCode`      VARCHAR(20)  NOT NULL,
  `colorHex`       VARCHAR(7)   NOT NULL DEFAULT '#000000',
  `coreCount`      INT          NOT NULL,
  `usedCores`      INT          NOT NULL DEFAULT 0,
  `availableCores` INT,
  `status`         ENUM('ACTIVE','RETIRED') NOT NULL DEFAULT 'ACTIVE',
  `notes`          TEXT,
  `createdAt`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `fiber_tubes_cableId_tubeNumber_key` (`cableId`, `tubeNumber`),
  KEY `fiber_tubes_cableId_idx` (`cableId`),
  KEY `fiber_tubes_status_idx` (`status`),
  CONSTRAINT `fiber_tubes_cableId_fkey` FOREIGN KEY (`cableId`) REFERENCES `fiber_cables` (`id`) ON DELETE CASCADE
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `fiber_cores` (
  `id`               VARCHAR(191) NOT NULL,
  `tubeId`           VARCHAR(191) NOT NULL,
  `coreNumber`       INT          NOT NULL,
  `colorCode`        VARCHAR(20)  NOT NULL,
  `colorHex`         VARCHAR(7)   NOT NULL DEFAULT '#000000',
  `status`           ENUM('AVAILABLE','USED','RESERVED','FAULTY') NOT NULL DEFAULT 'AVAILABLE',
  `assignedToType`   VARCHAR(50),
  `assignedToId`     VARCHAR(191),
  `attenuation`      DECIMAL(6,3),
  `notes`            TEXT,
  `createdAt`        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`        DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  UNIQUE KEY `fiber_cores_tubeId_coreNumber_key` (`tubeId`, `coreNumber`),
  KEY `fiber_cores_assignedToId_idx` (`assignedToId`),
  KEY `fiber_cores_assignedToType_idx` (`assignedToType`),
  KEY `fiber_cores_status_idx` (`status`),
  KEY `fiber_cores_tubeId_idx` (`tubeId`),
  CONSTRAINT `fiber_cores_tubeId_fkey` FOREIGN KEY (`tubeId`) REFERENCES `fiber_tubes` (`id`) ON DELETE CASCADE
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `splice_points` (
  `id`             VARCHAR(191) NOT NULL,
  `deviceType`     ENUM('ODP','ODC','CLOSURE','SPLITTER','OLT','ONT') NOT NULL,
  `deviceId`       VARCHAR(191) NOT NULL,
  `trayNumber`     INT          NOT NULL DEFAULT 1,
  `incomingCoreId` VARCHAR(191) NOT NULL,
  `outgoingCoreId` VARCHAR(191) NOT NULL,
  `spliceType`     ENUM('FUSION','MECHANICAL') NOT NULL DEFAULT 'FUSION',
  `insertionLoss`  DECIMAL(5,3),
  `reflectance`    DECIMAL(5,3),
  `spliceDate`     DATETIME(3),
  `splicedBy`      VARCHAR(191),
  `status`         ENUM('ACTIVE','RETIRED') NOT NULL DEFAULT 'ACTIVE',
  `notes`          TEXT,
  `createdAt`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`      DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `splice_points_deviceType_deviceId_idx` (`deviceType`, `deviceId`),
  KEY `splice_points_incomingCoreId_idx` (`incomingCoreId`),
  KEY `splice_points_outgoingCoreId_idx` (`outgoingCoreId`),
  KEY `splice_points_status_idx` (`status`),
  CONSTRAINT `splice_points_incomingCoreId_fkey` FOREIGN KEY (`incomingCoreId`) REFERENCES `fiber_cores` (`id`) ON DELETE CASCADE,
  CONSTRAINT `splice_points_outgoingCoreId_fkey` FOREIGN KEY (`outgoingCoreId`) REFERENCES `fiber_cores` (`id`) ON DELETE CASCADE
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `cable_segments` (
  `id`                    VARCHAR(191) NOT NULL,
  `cableId`               VARCHAR(191) NOT NULL,
  `fromDeviceType`        VARCHAR(50)  NOT NULL,
  `fromDeviceId`          VARCHAR(191) NOT NULL,
  `fromPort`              INT,
  `toDeviceType`          VARCHAR(50)  NOT NULL,
  `toDeviceId`            VARCHAR(191) NOT NULL,
  `toPort`                INT,
  `lengthMeters`          DECIMAL(10,2) NOT NULL DEFAULT 0.00,
  `attenuationPerKm`      DECIMAL(5,3)  NOT NULL DEFAULT 0.350,
  `calculatedAttenuation` DECIMAL(6,3),
  `status`                ENUM('ACTIVE','RETIRED','PLANNED') NOT NULL DEFAULT 'ACTIVE',
  `installDate`           DATE,
  `notes`                 TEXT,
  `createdAt`             DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  `updatedAt`             DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `cable_segments_cableId_idx` (`cableId`),
  KEY `cable_segments_fromDeviceType_fromDeviceId_idx` (`fromDeviceType`, `fromDeviceId`),
  KEY `cable_segments_status_idx` (`status`),
  KEY `cable_segments_toDeviceType_toDeviceId_idx` (`toDeviceType`, `toDeviceId`),
  CONSTRAINT `cable_segments_cableId_fkey` FOREIGN KEY (`cableId`) REFERENCES `fiber_cables` (`id`) ON DELETE CASCADE
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `core_assignment_history` (
  `id`              VARCHAR(191) NOT NULL,
  `coreId`          VARCHAR(191) NOT NULL,
  `action`          ENUM('ASSIGNED','RELEASED','RESERVED','FAULTED') NOT NULL,
  `previousStatus`  VARCHAR(50),
  `newStatus`       VARCHAR(50)  NOT NULL,
  `assignedToType`  VARCHAR(50),
  `assignedToId`    VARCHAR(191),
  `performedBy`     VARCHAR(191),
  `notes`           TEXT,
  `createdAt`       DATETIME(3)  NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (`id`),
  KEY `core_assignment_history_coreId_idx` (`coreId`),
  CONSTRAINT `core_assignment_history_coreId_fkey` FOREIGN KEY (`coreId`) REFERENCES `fiber_cores` (`id`) ON DELETE CASCADE
) DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci ENGINE=InnoDB;

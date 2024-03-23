CREATE TABLE IF NOT EXISTS `product_spus` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '商品SPU自增ID',
    `sn` varchar(255) NOT NULL COMMENT '商品SPU序列号',
    `name` varchar(255) NOT NULL COMMENT '商品名称',
    `description` longtext NOT NULL COMMENT '商品描述',
    `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '状态 1=下架 2=上架',
    `ctime` bigint DEFAULT NULL,
    `utime` bigint DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_product_spu_sn` (`sn`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;

INSERT INTO `product_spus` (sn, name, description, status, ctime, utime)
VALUES ('SPU001', '会员服务', '提供不同期限的会员服务', 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
       ('SPU002', '面试项目', '提供不同规模的面试项目', 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());

CREATE TABLE IF NOT EXISTS  `product_skus` (
    `id` bigint NOT NULL AUTO_INCREMENT COMMENT '商品SKU自增ID',
    `sn` varchar(255) NOT NULL COMMENT '商品SKU序列号',
    `product_spu_id` bigint NOT NULL COMMENT '商品SPU自增ID',
    `name` varchar(255) NOT NULL COMMENT 'SKU名称',
    `description` longtext NOT NULL COMMENT '商品描述',
    `price` bigint NOT NULL COMMENT '商品单价',
    `stock` bigint NOT NULL COMMENT '库存数量',
    `stock_limit` bigint NOT NULL COMMENT '库存限制',
    `sale_type` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '销售类型: 1=无限期 2=限时促销 3=预售',
    `status` tinyint unsigned NOT NULL DEFAULT '1' COMMENT '状态 1=下架 2=上架',
    `ctime` bigint DEFAULT NULL,
    `utime` bigint DEFAULT NULL,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_product_sku_sn` (`sn`),
    KEY `idx_product_spu_id` (`product_spu_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;


INSERT INTO `product_skus` (sn, product_spu_id, name, description, price, stock, stock_limit, sale_type, status, ctime, utime)
VALUES
    ('SKU001', 1, '星期会员', '提供一周的会员服务', 799, 1000, 100000000, 1, 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
    ('SKU002', 1, '月会员', '提供一个月的会员服务', 990, 1000, 100000000, 1, 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
    ('SKU003', 1, '季度会员', '提供一个季度的会员服务', 2970, 1000, 100000000, 1, 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
    ('SKU004', 1, '年会员', '提供一年的会员服务', 11880, 1000, 100000000, 1, 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
    ('SKU005', 2, '用户项目', '中小型面试项目', 9999, 1000, 100000000, 1, 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP()),
    ('SKU006', 2, '权限项目', '中大型面试项目', 19999, 1000, 100000000, 1, 2, UNIX_TIMESTAMP(), UNIX_TIMESTAMP());

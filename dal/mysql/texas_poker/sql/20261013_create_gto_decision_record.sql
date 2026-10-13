CREATE TABLE `gto_decision_record` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增主键',
  `uid` VARCHAR(128) NOT NULL COMMENT '客户端用户标识',
  `decision_time` DATETIME(3) NOT NULL COMMENT '决策完成时间',
  `hero_cards` JSON NOT NULL COMMENT 'Hero手牌',
  `position` VARCHAR(32) NOT NULL COMMENT 'Hero位置',
  `stage` VARCHAR(16) NOT NULL COMMENT '牌局阶段',
  `decision_request` JSON NOT NULL COMMENT '发送给GTO服务的完整请求',
  `decision_response` JSON NOT NULL COMMENT 'GTO服务返回的原始响应',
  `final_response` JSON NOT NULL COMMENT '校验并规范化后的最终响应',
  PRIMARY KEY (`id`),
  KEY `idx_gto_decision_uid` (`uid`),
  KEY `idx_gto_decision_time` (`decision_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='GTO决策审计记录';

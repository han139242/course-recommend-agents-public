-- 修复因终端编码导致的 target 乱码（按需执行，非 init 自动跑）
SET NAMES utf8mb4;
UPDATE users
SET target = '掌握人工智能基础'
WHERE email = 'maintainer@example.org';

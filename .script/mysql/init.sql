-- create the databases
CREATE DATABASE IF NOT EXISTS `webook`;

-- create the users for each database
CREATE USER 'webook'@'%' IDENTIFIED BY 'webook';
GRANT CREATE, ALTER, INDEX, LOCK TABLES, REFERENCES, UPDATE, DELETE, DROP, SELECT, INSERT ON `webook`.* TO 'webook'@'%';

FLUSH PRIVILEGES;
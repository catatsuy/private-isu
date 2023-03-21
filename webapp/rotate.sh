#!/bin/sh

mv ./logs/nginx/access.log ./logs/nginx/access_`date +%Y%m%d-%H%M%S`.log
mv ./logs/nginx/error.log ./logs/nginx/error_`date +%Y%m%d-%H%M%S`.log
mv ./logs/mysql/mysqlslow.log ./logs/mysql/mysqlslow_`date +%Y%m%d-%H%M%S`.log
echo 'restart nginx'
echo 'restart mysql'

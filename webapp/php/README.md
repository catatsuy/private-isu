# Iscogram PHP implementation

# Require
* PHP > 5.6

# Run in built-in server

```
cd src/github.com/catatsuy/private-isu/webapp/php
php -S localhost:8080 -t ../public app.php
```

# Run in php-fpm

```
php-fpm -f config/php-fpm.conf
nginx -c config/nginx.conf
```

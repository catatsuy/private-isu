# Iscogram PHP implementation

# Require
* PHP > 5.6

# Run in built-in server

```
cd src/github.com/catatsuy/private-isu/webapp/php
php -S localhost:8080 -t ../public app.php
```

# Run in php-fpm
```bash
$ sudo systemctl start php7.4-fpm
$ sudo cp /etc/nginx/sites-available/isucon-php.conf /etc/nginx/sites-enabled/isucon.conf 
$ sudo systemctl restart nginx
```

FROM php:8.2-fpm-bullseye

RUN apt update && apt install -y \
  unzip libmemcached-dev zlib1g-dev

RUN docker-php-ext-install pdo pdo_mysql

RUN pecl install memcached \
  && docker-php-ext-enable memcached

RUN curl -sS https://getcomposer.org/installer | php \
  && mv composer.phar /usr/local/bin/composer

COPY ./composer.json /var/www/html
COPY ./composer.lock /var/www/html
WORKDIR /var/www/html

RUN composer install --no-dev

COPY . /var/www/html

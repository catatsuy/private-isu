# 初期データ

http://twilog.org/catatsuy をスクレイピングしてラーメン画像を取得してきて、ImageMagickでフィルターをかけて1万枚ぐらいの画像を用意する

# フィルター

```bash
convert $file -blur 20 blur-$file
convert $file -median 10 median-$file
convert $file -sepia-tone '70%' sepia2-$file
convert $file -sepia-tone '90%' sepia-$file
convert $file -modulate 100,90,90 red-$file
convert $file -emboss 10 emboss-$file
convert $file -modulate 100,90,90 red-$file
convert $file -type GrayScale gray-$file
./toycamera -i 50 -o 150 -s 100 -O 3 -I 3 $file toycamera-$file # http://fmwconcepts.com/imagemagick/toycamera/index.php
convert $file -flop flop-$file
```

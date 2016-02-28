# 初期データ

http://twilog.org/catatsuy をスクレイピングしてラーメン画像を取得してきて、ImageMagickでフィルターをかけて1万枚ぐらいの画像を用意する

## フィルター

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

## git lfs

`git lfs` を使っているので、

```
brew install git-lfs
```

しておく必要がある。

単純に `git pull` しただけだと、画像ファイルは単にテキストファイルになっている。

その状態で `git lfs pull origin master` みたいにやる。

http://qiita.com/kiida/items/0d51c43ac73f14f09f5a

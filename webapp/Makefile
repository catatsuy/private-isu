.PHONY: rotate alp slow

rotate:
	sh ./rotate.sh

alp:
	# logファイルはローテートしてるはずなので、ファイル名は適宜変更
	alp json --sort sum -r -m "/posts/[0-9]+,/@\w+,/image/\d+" -o count,method,uri,min,avg,max,sum < logs/nginx/access.log

slow:
	# logファイルはローテートしてるはずなので、ファイル名は適宜変更
	mysqldumpslow logs/mysql/mysqlslow_20230321-173552.log

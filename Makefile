csvsplit: main.go
	go build

demo: csvsplit example.csv
	split -l 1 example.csv split-example-
	./csvsplit --prefix=csvsplit-example- --line-bytes=10 < example.csv

	echo "Notice that split generates 3 files of one line each..."
	cat -n split-example-*

	echo "...while csvsplit generates 2 files, one with 2 lines."
	cat -n csvsplit-example-*
 
clean:
	rm -f csvsplit
	rm -f split-example-*
	rm -f csvsplit-example-*


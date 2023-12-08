build: hex2raw hex2raw-old

hex2raw: hex2raw.go
	go build hex2raw.go

hex2raw-old: hex2raw.c
	gcc hex2raw.c -o hex2raw-old

clean:
	rm -f hex2raw hex2raw-old

GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build -tags lambda.norpc -o bootstrap main.go

zip cgo.zip bootstrap libadd.so
GOOS=linux GOARCH=amd64 go build -tags lambda.norpc -o bootstrap main.go

zip distributor.zip bootstrap
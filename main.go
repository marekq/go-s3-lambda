package main

import (
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"io"
	"math/rand"
	"os"
	"strconv"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/epsagon/epsagon-go/epsagon"
)

func hash_file_crc32(filePath string, polynomial uint32) (string, error) {
	var returnCRC32String string
	file, err := os.Open(filePath)
	if err != nil {
		return returnCRC32String, err
	}
	defer file.Close()
	tablePolynomial := crc32.MakeTable(polynomial)
	hash := crc32.New(tablePolynomial)
	if _, err := io.Copy(hash, file); err != nil {
		return returnCRC32String, err
	}
	hashInBytes := hash.Sum(nil)[:]
	returnCRC32String = hex.EncodeToString(hashInBytes)
	return returnCRC32String, nil
}

func forloop() {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String(os.Getenv("awsregion"))})

	downloader := s3manager.NewDownloader(sess)

	i := 1
	for i <= 10 {
		i = i + 1

		rand.Seed(time.Now().UnixNano())
		bucket := os.Getenv("awsbucket")
		num := rand.Intn(99) + 1
		item := "1000/" + strconv.Itoa(num) + ".file"
		fname := "/tmp/" + strconv.Itoa(num) + ".file"

		file, err := os.Create(fname)
		if err != nil {
			exitErrorf("Unable to open file %q, %v", err)
		}
		defer file.Close()

		numBytes, err := downloader.Download(file,
			&s3.GetObjectInput{
				Bucket: aws.String(bucket),
				Key:    aws.String(item),
			})
		if err != nil {
			exitErrorf("Unable to download item %q, %v", item, err)
		}

		hash, err := hash_file_crc32(fname, 0xedb88320)
		if err == nil {
			fmt.Println(hash, numBytes, item)
		}

		os.Remove(fname)
	}
}

func exitErrorf(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, msg+"\n", args...)
	os.Exit(1)
}

func main() {
	lambda.Start(epsagon.WrapLambdaHandler(
		&epsagon.Config{ApplicationName: os.Getenv("appname")},
		forloop))
}

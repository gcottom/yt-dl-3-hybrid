package s3

import (
	"io"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func DeleteFromS3(id, bucket string) error {
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	svc := s3.New(sess)
	_, err := svc.DeleteObject(&s3.DeleteObjectInput{Bucket: aws.String(bucket), Key: aws.String(id)})
	return err
}

func DownloadFromS3Buf(id, bucket string) (*aws.WriteAtBuffer, error) {
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	buf := aws.NewWriteAtBuffer([]byte{})
	downloader := s3manager.NewDownloader(sess)
	if _, err := downloader.Download(buf, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &id,
	}); err != nil {
		return nil, err
	}
	return buf, nil
}

func DownloadFromS3File(id, tempext, bucket string) (*os.File, error) {
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	file, err := os.Create("/tmp/" + id + tempext)
	if err != nil {
		return nil, err
	}
	downloader := s3manager.NewDownloader(sess)
	if _, err = downloader.Download(file, &s3.GetObjectInput{
		Bucket: &bucket,
		Key:    &id,
	}); err != nil {
		return nil, err
	}
	if _, err = file.Seek(0, 0); err != nil {
		return nil, err
	}
	return file, nil
}

func UploadToS3(input io.Reader, id, bucket string) error {
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	uploader := s3manager.NewUploader(sess)
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &id,
		Body:   input,
	})
	return err
}

func UploadLargeFileToS3(input io.Reader, id, bucket string) error {
	var partMiBs int64 = 10
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	uploader := s3manager.NewUploader(sess, func(u *s3manager.Uploader) {
		u.PartSize = partMiBs * 1024 * 1024
	}, func(u *s3manager.Uploader) {
		u.Concurrency = 20
	})
	_, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: &bucket,
		Key:    &id,
		Body:   input,
	})
	return err
}

func S3GetFileExists(id, bucket string) bool {
	conf := aws.Config{Region: region}
	sess := session.Must(session.NewSession(&conf))
	s3svc := s3.New(sess)
	_, err := s3svc.HeadObject(&s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &id,
	})
	return err == nil
}

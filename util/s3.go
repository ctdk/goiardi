/*
 * Copyright (c) 2013-2016, Jeremy Bingham (<jbingham@gmail.com>)
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package util

import (
	"encoding/base64"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/ctdk/goiardi/config"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type s3client struct {
	bucket     string
	filePeriod time.Duration
	s3         *s3.S3
}

var s3cli *s3client

// InitS3 sets up the session and whatnot for using goiardi with S3.
func InitS3(conf *config.Conf) error {
	// TODO: configure parameters
	sess := session.New(&aws.Config{Region: aws.String(conf.AWSRegion), DisableSSL: aws.Bool(conf.AWSDisableSSL), Endpoint: aws.String(conf.S3Endpoint), S3ForcePathStyle: aws.Bool(false)})

	s3cli = new(s3client)
	s3cli.bucket = conf.S3Bucket
	s3cli.filePeriod = time.Duration(conf.S3FilePeriod) * time.Minute
	s3cli.s3 = s3.New(sess)
	return nil
}

func S3GetURL(orgname string, checksum string) (string, error) {
	key := makeBukkitKey(orgname, checksum)
	req, _ := s3cli.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: aws.String(s3cli.bucket),
		Key:    aws.String(key),
	})
	urlStr, err := req.Presign(s3cli.filePeriod)
	return urlStr, err
}

func S3PutURL(orgname string, checksum string) (string, error) {
	key := makeBukkitKey(orgname, checksum)
	req, _ := s3cli.s3.PutObjectRequest(&s3.PutObjectInput{
		Bucket: aws.String(s3cli.bucket),
		Key:    aws.String(key),
	})

	// there may be an easier way
	re := regexp.MustCompile(`[0-9A-Fa-f]{2}`)
	chopped := re.FindAllString(checksum, -1)
	b := make([]byte, len(chopped))
	for i, v := range chopped {
		m, err := strconv.ParseUint(v, 16, 8)
		if err != nil {
			return "", err
		}
		b[i] = byte(m)
	}
	contentmd5 := base64.StdEncoding.EncodeToString(b)
	req.HTTPRequest.Header.Set("Content-MD5", contentmd5)

	urlStr, err := req.Presign(s3cli.filePeriod)
	return urlStr, err
}

func CheckForObject(orgname string, checksum string) (bool, error) {
	key := makeBukkitKey(orgname, checksum)
	output, err := s3cli.s3.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(s3cli.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		// hmm?
		return false, err
	}
	if output != nil {
		return true, nil
	}
	return false, nil
}

func makeBukkitKey(orgname, checksum string) string {
	dir := fmt.Sprintf("%c%c", checksum[0], checksum[1])
	key := strings.Join([]string{orgname, "file_store", dir, checksum}, "/")
	return key
}

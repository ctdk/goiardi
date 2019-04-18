/*
 * Copyright (c) 2013-2019, Jeremy Bingham (<jeremy@goiardi.gl>)
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
	"github.com/tideland/golib/logger"
	"net"
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
	req.HTTPRequest.URL.Host = s3cli.makeHostPort(req.HTTPRequest.URL.Host)
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
	req.HTTPRequest.URL.Host = s3cli.makeHostPort(req.HTTPRequest.URL.Host)

	urlStr, err := req.Presign(s3cli.filePeriod)
	logger.Debugf("presign: %s %s", urlStr, contentmd5)
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

func S3DeleteHashes(orgName string, fileHashes []string) {
	// only do this if there are actually file hashes to delete.
	if len(fileHashes) == 0 {
		return
	}
	// break this up in case we have more than 1000 hashes to delete.
	objs := make([]*s3.ObjectIdentifier, len(fileHashes))
	for i, k := range fileHashes {
		objs[i] = &s3.ObjectIdentifier{Key: aws.String(makeBukkitKey(orgName, k))}
	}
	params := &s3.DeleteObjectsInput{
		Bucket: aws.String(s3cli.bucket),
		Delete: &s3.Delete{
			Objects: objs,
			Quiet:   aws.Bool(true),
		},
	}
	logger.Debugf("delete hash s3 params: %v", params)

	r, err := s3cli.s3.DeleteObjects(params)
	if err != nil {
		logger.Errorf(err.Error())
	} else {
		logger.Debugf("%v", r)
	}
}

func S3CheckFile(orgname, checksum string) bool {
	params := &s3.HeadObjectInput{
		Bucket: aws.String(s3cli.bucket),
		Key:    aws.String(makeBukkitKey(orgname, checksum)),
	}
	_, err := s3cli.s3.HeadObject(params)
	var ret bool
	if err == nil {
		ret = true
	}
	return ret
}

func makeBukkitKey(orgname, checksum string) string {
	dir := fmt.Sprintf("%c%c", checksum[0], checksum[1])
	key := strings.Join([]string{orgname, "file_store", dir, checksum}, "/")
	return key
}

// chef insists on putting the port number in the Host: header it sends to
// amazon, even when using normal ports. ?!?!
func (s3c *s3client) makeHostPort(host string) string {
	z, _ := regexp.MatchString(`:\d+$`, host)
	q := net.ParseIP(host)
	var rethost string
	if !z && q == nil {
		var port string
		if *s3c.s3.Config.DisableSSL {
			port = "80"
		} else {
			port = "443"
		}
		rethost = net.JoinHostPort(host, port)
	} else {
		rethost = host
	}
	return rethost
}

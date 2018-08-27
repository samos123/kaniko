/*
Copyright 2018 Google LLC

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package util

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
	"strconv"
	"syscall"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// SetLogLevel sets the logrus logging level
func SetLogLevel(logLevel string) error {
	lvl, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return errors.Wrap(err, "parsing log level")
	}
	logrus.SetLevel(lvl)
	return nil
}

// Hasher returns a hash function, used in snapshotting to determine if a file has changed
func Hasher() func(string) (string, error) {
	hasher := func(p string) (string, error) {
		h := md5.New()
		fi, err := os.Lstat(p)
		if err != nil {
			return "", err
		}
		h.Write([]byte(fi.Mode().String()))
		h.Write([]byte(fi.ModTime().String()))

		h.Write([]byte(strconv.FormatUint(uint64(fi.Sys().(*syscall.Stat_t).Uid), 36)))
		h.Write([]byte(","))
		h.Write([]byte(strconv.FormatUint(uint64(fi.Sys().(*syscall.Stat_t).Gid), 36)))

		if fi.Mode().IsRegular() {
			f, err := os.Open(p)
			if err != nil {
				return "", err
			}
			defer f.Close()
			if _, err := io.Copy(h, f); err != nil {
				return "", err
			}
		}

		return hex.EncodeToString(h.Sum(nil)), nil
	}
	return hasher
}

// MtimeHasher returns a hash function, which only looks at mtime to determine if a file has changed.
// Note that the mtime can lag, so it's possible that a file will have changed but the mtime may look the same.
func MtimeHasher() func(string) (string, error) {
	hasher := func(p string) (string, error) {
		h := md5.New()
		fi, err := os.Lstat(p)
		if err != nil {
			return "", err
		}
		h.Write([]byte(fi.ModTime().String()))
		return hex.EncodeToString(h.Sum(nil)), nil
	}
	return hasher
}

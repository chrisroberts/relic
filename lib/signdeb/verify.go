/*
 * Copyright (c) SAS Institute Inc.
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

package signdeb

import (
	"bufio"
	"bytes"
	"crypto"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/pgptools"
	"github.com/qur/ar"
	"golang.org/x/crypto/openpgp"
)

func Verify(r io.Reader, keyring openpgp.EntityList, skipDigest bool) (map[string]*pgptools.PgpSignature, error) {
	reader := ar.NewReader(r)
	digests := make(map[string]string)
	sigs := make(map[string][]byte)
	for {
		hdr, err := reader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}
		if strings.HasPrefix(hdr.Name, "_gpg") {
			role := hdr.Name[4:]
			sigs[role], err = ioutil.ReadAll(reader)
			if err != nil {
				return nil, err
			}
		} else if !skipDigest {
			md5 := crypto.MD5.New()
			sha1 := crypto.SHA1.New()
			if _, err := io.Copy(io.MultiWriter(md5, sha1), reader); err != nil {
				return nil, err
			}
			digests[hdr.Name] = fmt.Sprintf("%x %x", md5.Sum(nil), sha1.Sum(nil))
		}
	}
	ret := make(map[string]*pgptools.PgpSignature, len(sigs))
	for role, sig := range sigs {
		info, body, err := pgptools.VerifyClearSign(bytes.NewReader(sig), keyring)
		if err != nil {
			return nil, err
		}
		if !skipDigest {
			if err := checkSig(role, body, digests); err != nil {
				return nil, err
			}
		}
		ret[role] = info
	}
	return ret, nil
}

func checkSig(role string, body []byte, digests map[string]string) error {
	sawFiles := false
	scanner := bufio.NewScanner(bytes.NewReader(body))
	// read header
	for scanner.Scan() {
		line := scanner.Text()
		if line == "Files:" {
			sawFiles = true
			break
		}
	}
	if !sawFiles {
		return errors.New("malformed signature")
	}
	// read digests
	checked := make(map[string]bool, len(digests))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break
		} else if line[0] != '\t' || len(line) < 76 {
			return errors.New("malformed signature")
		}
		parts := strings.SplitN(line[1:], " ", 4)
		sums := parts[0] + " " + parts[1]
		name := parts[3]
		calculated := digests[name]
		if calculated == "" {
			return fmt.Errorf("signature references unknown file %s", name)
		} else if calculated != sums {
			return fmt.Errorf("signature mismatch on file %s: (%s) != (%s)", name, calculated, sums)
		}
		checked[name] = true
	}
	// make sure everything was checked
	for name := range digests {
		if checked[name] {
			continue
		}
		return fmt.Errorf("signature does not cover file: %s", name)
	}
	return nil
}

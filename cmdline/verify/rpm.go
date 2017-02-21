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

package verify

import (
	"errors"
	"fmt"
	"os"

	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/pgptools"

	"github.com/sassoftware/go-rpmutils"
)

func verifyRpm(f *os.File) error {
	_, sigs, err := rpmutils.Verify(f, trustedPgp)
	if err != nil {
		return err
	}
	if len(sigs) == 0 {
		return errors.New("RPM is not signed")
	}
	seen := make(map[uint64]bool)
	for _, sig := range sigs {
		status := "OK"
		if seen[sig.KeyId] {
			continue
		} else if sig.Signer == nil {
			if argNoChain {
				status = "UNKNOWN"
			} else {
				return fmt.Errorf("unknown keyId %x; use --cert to specify trusted keys", sig.KeyId)
			}
		}
		seen[sig.KeyId] = true
		fmt.Printf("%s: %s - %s(%x) [%s]\n", f.Name(), status, pgptools.EntityName(sig.Signer), sig.KeyId, sig.CreationTime)
	}
	return nil
}

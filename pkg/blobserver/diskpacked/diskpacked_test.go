/*
Copyright 2013 The Camlistore Authors

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

package diskpacked

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"testing"

	"camlistore.org/pkg/blob"
	"camlistore.org/pkg/blobserver"
	"camlistore.org/pkg/blobserver/storagetest"
	"camlistore.org/pkg/jsonconfig"
	"camlistore.org/pkg/test"
)

func newTempDiskpacked(t *testing.T) (sto blobserver.Storage, cleanup func()) {
	return newTempDiskpackedWithIndex(t, jsonconfig.Obj{})
}

func newTempDiskpackedMemory(t *testing.T) (sto blobserver.Storage, cleanup func()) {
	return newTempDiskpackedWithIndex(t, jsonconfig.Obj{
		"type": "memory",
	})
}

func newTempDiskpackedWithIndex(t *testing.T, indexConf jsonconfig.Obj) (sto blobserver.Storage, cleanup func()) {
	dir, err := ioutil.TempDir("", "diskpacked-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("diskpacked test dir is %q", dir)
	s, err := newStorage(dir, 1<<20, indexConf)
	if err != nil {
		t.Fatalf("newStorage: %v", err)
	}
	return s, func() {
		s.Close()

		if camliDebug {
			t.Logf("CAMLI_DEBUG set, skipping cleanup of dir %q", dir)
		} else {
			os.RemoveAll(dir)
		}
	}
}

func TestDiskpacked(t *testing.T) {
	storagetest.Test(t, newTempDiskpacked)
}

func TestDiskpackedAltIndex(t *testing.T) {
	storagetest.Test(t, newTempDiskpackedMemory)
}

func TestDoubleReceive(t *testing.T) {
	sto, cleanup := newTempDiskpacked(t)
	defer cleanup()

	size := func(n int) int64 {
		path := sto.(*storage).filename(n)
		fi, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		return fi.Size()
	}

	const blobSize = 5 << 10
	b := &test.Blob{Contents: strings.Repeat("a", blobSize)}
	br := b.BlobRef()

	_, err := blobserver.Receive(sto, br, b.Reader())
	if err != nil {
		t.Fatal(err)
	}
	if size(0) < blobSize {
		t.Fatalf("size = %d; want at least %d", size(0), blobSize)
	}
	sto.(*storage).nextPack()

	_, err = blobserver.Receive(sto, br, b.Reader())
	if err != nil {
		t.Fatal(err)
	}
	sizePostDup := size(1)
	if sizePostDup >= blobSize {
		t.Fatalf("size(pack1) = %d; appeared to double-write.", sizePostDup)
	}

	os.Remove(sto.(*storage).filename(0))
	_, err = blobserver.Receive(sto, br, b.Reader())
	if err != nil {
		t.Fatal(err)
	}
	sizePostDelete := size(1)
	if sizePostDelete < blobSize {
		t.Fatalf("after packfile delete + reupload, not big enough. want size of a blob")
	}
}

func TestDelete(t *testing.T) {
	sto, cleanup := newTempDiskpacked(t)
	defer cleanup()

	var (
		A = &test.Blob{Contents: "some small blob"}
		B = &test.Blob{Contents: strings.Repeat("some middle blob", 100)}
		C = &test.Blob{Contents: strings.Repeat("A 8192 bytes length largish blob", 8192/32)}
	)

	type step func() error

	stepAdd := func(tb *test.Blob) step { // add the blob
		return func() error {
			sb, err := sto.ReceiveBlob(tb.BlobRef(), tb.Reader())
			if err != nil {
				return fmt.Errorf("ReceiveBlob of %s: %v", sb, err)
			}
			if sb != tb.SizedRef() {
				return fmt.Errorf("Received %v; want %v", sb, tb.SizedRef())
			}
			return nil
		}
	}

	stepCheck := func(want ...*test.Blob) step { // check the blob
		wantRefs := make([]blob.SizedRef, len(want))
		for i, tb := range want {
			wantRefs[i] = tb.SizedRef()
		}
		return func() error {
			if err := storagetest.CheckEnumerate(sto, wantRefs); err != nil {
				return err
			}
			return nil
		}
	}

	stepDelete := func(tb *test.Blob) step {
		return func() error {
			if err := sto.RemoveBlobs([]blob.Ref{tb.BlobRef()}); err != nil {
				return fmt.Errorf("RemoveBlob(%s): %v", tb.BlobRef(), err)
			}
			return nil
		}
	}

	var deleteTests = [][]step{
		[]step{
			stepAdd(A),
			stepDelete(A),
			stepCheck(),
			stepAdd(B),
			stepCheck(B),
			stepDelete(B),
			stepCheck(),
			stepAdd(C),
			stepCheck(C),
			stepAdd(A),
			stepCheck(A, C),
			stepDelete(A),
			stepDelete(C),
			stepCheck(),
		},
		[]step{
			stepAdd(A),
			stepAdd(B),
			stepAdd(C),
			stepCheck(A, B, C),
			stepDelete(C),
			stepCheck(A, B),
		},
	}
	for i, steps := range deleteTests {
		for j, s := range steps {
			if err := s(); err != nil {
				t.Errorf("error at test %d, step %d: %v", i+1, j+1, err)
			}
		}
	}
}

package internals

import (
	"bytes"
	"encoding/hex"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestAllHashAlgosDefined checks that *distinctive* hash algorithms
// are available via HashAlgo(i) for i=0..CountHashAlgos
func TestAllHashAlgosDefined(t *testing.T) {
	names := make([]string, 0, 16)
	for i := 0; i < CountHashAlgos; i++ {
		h := HashAlgo(i)
		if !Contains(names, h.Instance().Name()) {
			names = append(names, h.Instance().Name())
		}
	}
	if len(names) != CountHashAlgos {
		t.Errorf(`Expected %d distinctive names, got %v`, CountHashAlgos, names)
	}
}

// TestRequiredHashAlgos checks that all required hash algorithms are supported
func TestRequiredHashAlgos(t *testing.T) {
	required := []string{`crc64`, `fnv-1a-32`, `fnv-1a-128`, `sha-256`, `sha-512`, `sha-3-512`}

	supported := make([]string, 0, 16)
	for i := 0; i < CountHashAlgos; i++ {
		h := HashAlgo(i)
		supported = append(supported, h.Instance().Name())
	}

	for _, req := range required {
		if !Contains(supported, req) {
			t.Errorf(`Hash algorithm '%s' unsupported, but support is required`, req)
		}
	}
}

func TestMD5sumCompatibility(t *testing.T) {
	// create temporary file
	fd, err := ioutil.TempFile("", "dupfiles-compat-test")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Remove(fd.Name()); err != nil {
			t.Fatal(err)
		}
	}()

	_, err = fd.Write([]byte{
		0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
		0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
		0x0a,
	})
	if err != nil {
		t.Fatal(err)
	}

	// determine digest with md5sum
	executable := os.Getenv("MD5SUM_EXEC")
	if executable == "" {
		executable = "md5sum"
	}
	cmd := exec.Command(executable, fd.Name())
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		t.Fatal("problems running the md5sum executable: " + err.Error() + ". Maybe set env var MD5SUM_EXEC to find the executable?")
	}
	sumDigest := strings.TrimSpace(string(out.Bytes()))
	i := strings.Index(sumDigest, "  ")
	sumDigest = sumDigest[0:i]

	// determine dupfiles hash
	h := HashMD5.Instance()
	err = h.ReadFile(fd.Name())
	if err != nil {
		t.Fatal(err)
	}

	dupDigest := h.Hash().Digest()
	if sumDigest != dupDigest {
		t.Errorf(`digests of md5sum (%s) and dupfiles (%s) differ`, sumDigest, dupDigest)
	}
}

// TestExampleContentModeFileHashes uses the internal hash algorithms
// in content mode to compute the digest of example.txt. These digests
// should match the static digests stored here
func TestExampleContentModeFileHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:        `6365a94a3d11ef39`,
		HashCRC32:        `68e17d95`,
		HashFNV1_32:      `e6d4bcab`,
		HashFNV1_64:      `23f7f66c310d910b`,
		HashFNV1_128:     `c1a46322af2cc1806384306316df2333`,
		HashFNV1A32:      `067b6d97`,
		HashFNV1A64:      `3deefb5fdfa573f7`,
		HashFNV1A128:     `32bdefadfe82776b2e5c6c1f97d949ef`,
		HashADLER32:      `ea3c0e4d`,
		HashMD5:          `eddc51f98f9367bffe0dec96da83648c`,
		HashSHA1:         `3af73983ad876cc108ef4cf7b045450a20b35780`,
		HashSHA256:       `2f837632f54939e1824950eeaf5924e8c275a1b8443fc8bf1eab11902d185c4c`,
		HashSHA512:       `295e43d93006798b3608170e92ac883a84eb0635be6041226ca9eda6dab7d1ab7319a59cce44187216e1fb17f94a8ec24ca6df64532765be0da0fef27a88c3f4`,
		HashSHA3_512:     `b9f07ba425705709308518d521f0b93a22844fd181769b49afef188b72ae2e9a4d106fe91c0e4bf008687b2305fa73eb05493f373e50404036dd853de7e37805`,
		HashSHAKE256_128: `11797e0d409ed892bda314a0ada2b9da`,
	}
	for hash, refDigest := range data {
		h := hash.Instance()
		err := h.ReadBytes([]byte{
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}

		actual := h.Hash().Digest()
		if actual != refDigest {
			t.Errorf(`digest for example.txt in content mode incorrect (%s): expected %s, got %s`, hash.Instance().Name(), refDigest, actual)
		}
	}
}

// TestExampleContentModeFileHashes uses the internal hash algorithms
// in three mode to compute the digest of example.txt. These digests
// should match the static digests stored here
func TestExampleThreeModeFileHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:        `247d38d8a97d4ada`,
		HashCRC32:        `4dc37780`,
		HashFNV1_32:      `a1ef7fa5`,
		HashFNV1_64:      `f1baddc2785b7f65`,
		HashFNV1_128:     `03b802ae9459705510a53505a1877d55`,
		HashFNV1A32:      `ef782d35`,
		HashFNV1A64:      `af85927b79193735`,
		HashFNV1A128:     `24320f225ed7fba3caadb050b89a2a25`,
		HashADLER32:      `a540130d`,
		HashMD5:          `65f3bff06d0089f9f95e5058c5c0d025`,
		HashSHA1:         `628a6a795243194e54cd328e6e7e81b6104e930b`,
		HashSHA256:       `8aa21f44bf222246e0769d84c7a1ebaee036d163d191e8295cac3f397c6a92a6`,
		HashSHA512:       `61cf29883376d8e833671cef8466156288097cd4e11c04cc22d94bd4a1c5fe3ec9150bbcc30a456a741a802742fb29b0f6143dd629d5027368a2c2a94cf23b98`,
		HashSHA3_512:     `25126efd5a36422360fd660125d861ea10f8d1f6124bdca1291f4e2df7e0253f1cae2d23493f5e5fa492ad7a8ad9ecf49faca2ea83b272a4a73b380d3f54f1a9`,
		HashSHAKE256_128: `14413027270deb7f07e1f70c190b3cfd`,
	}
	for hash, refDigest := range data {
		h := hash.Instance()
		err := h.ReadBytes([]byte{
			'F', 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 't', 'x', 't',
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}

		actual := h.Hash().Digest()
		if actual != refDigest {
			t.Errorf(`digest for example.txt in three mode incorrect (%s): expected %s, got %s`, hash.Instance().Name(), refDigest, actual)
		}
	}
}

// TestExampleContentModeFolderHashes uses the internal hash algorithms
// in content mode to compute the digest of folder "example-folder"
// containing files {"example.txt", "empty.txt" an empty file}. These digests
// should match the static digests stored here
func TestExampleContentModeFolderHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:        `6365a94a3d11ef39`,
		HashCRC32:        `68e17d95`,
		HashFNV1_32:      `67c8216e`,
		HashFNV1_64:      `e8056a88b52fb22e`,
		HashFNV1_128:     `adc6440ca897c0c2013c1116744ae6be`,
		HashFNV1A32:      `8767f052`,
		HashFNV1A64:      `f61c67bb5b8750d2`,
		HashFNV1A128:     `5edfc883f93976294ce44d6af54c8c62`,
		HashADLER32:      `ea3c0e4c`,
		HashMD5:          `39c1dd200093d5bb178de50e367b26f2`,
		HashSHA1:         `e0ce9a6df3ec27cc3abaf31825255d9a8f6b5089`,
		HashSHA256:       `cc33b2706db525f518b2a42636369dcce5dbe05c20a45bf3ba3e888b554ae419`,
		HashSHA512:       `e6dda2ec4ee9c136c75c3f5e44c1083d52cbe230b53754feef5d448709db386534c974a093c1eac2e962e3c57e3462ed2f1feed914661f3fa898cc8883af19ca`,
		HashSHA3_512:     `1f6f0868874acdccf8307f0939aacc54b54dcdc7ce94c3104f3ec44a35f2ae3c58a27dd3edfbb2bc198b926329c0b6b3f04926aaabe693a337a800bbcffeb523`,
		HashSHAKE256_128: `57c0a3264b3655819e982b4bd99c52fe`,
	}
	for hash, refDigest := range data {
		// ./example-folder/example.txt
		f1 := hash.Instance()
		err := f1.ReadBytes([]byte{
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}
		hashA := f1.Hash()

		// ./example-folder/empty.txt
		f2 := hash.Instance()
		err = f2.ReadBytes([]byte{})
		if err != nil {
			t.Fatal(err)
		}
		hashB := f2.Hash()

		// digest(example-folder) = digest(example.txt) ⊕ digest(empty.txt)
		XORByteSlices(hashA, hashB)

		actual := hex.EncodeToString(hashA)
		if actual != refDigest {
			t.Errorf(`digest for example folder incorrect (%s): expected %s, got %s`, hash.Instance().Name(), refDigest, actual)
		}
	}
}

// TestExampleThreeModeFolderHashes uses the internal hash algorithms
// in three mode to compute the digest of folder "example-folder"
// containing files {"example.txt", "example2.txt"}. These digests
// should match the static digests stored here
func TestExampleThreeModeFolderHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:        `f29a99561520b220`,
		HashCRC32:        `3b005b41`,
		HashFNV1_32:      `a050e59b`,
		HashFNV1_64:      `4693305ad95b499b`,
		HashFNV1_128:     `6356edc7f4d77ad498e1345a246c957b`,
		HashFNV1A32:      `53f2c5a3`,
		HashFNV1A64:      `e81d4f1642db74e3`,
		HashFNV1A128:     `f7885490097b3d7a3b95bf5566d0d893`,
		HashADLER32:      `9e0212d3`,
		HashMD5:          `ecc97458a95d0d5e9daccd973474d327`,
		HashSHA1:         `dc397b0496964d98d1053b994c4e18e7569cb449`,
		HashSHA256:       `dfafdd6c79be55fe0c3cb1b37a299224aec6bcb274bb8fae93ad23d3043d398e`,
		HashSHA512:       `76d61c80db7384ea6b9dba1c9f3d8ea9430962b50943d60bbd909a605232d03ad43289b10a81ccb38b69649b786f200ded78a487973960e9561f4162be64843c`,
		HashSHA3_512:     `47993a5a8ad7f1220e0231bf83d8f408f54231879fcd1777736f08b30b99fdc9da76857a0a21a13f204352916d4e005a810fae6d41c9a5bffac19a680c9b8f1c`,
		HashSHAKE256_128: `1796399a183b5bbbd7d12c6da04a3460`,
	}
	for hash, refDigest := range data {
		// ./example-folder/example.txt
		// digest(example.txt) = hex(H('F' ‖ 'example.txt' ‖ file.content))
		f1 := hash.Instance()
		err := f1.ReadBytes([]byte{
			'F', 'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 't', 'x', 't',
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}
		hashA := f1.Hash()

		// ./example-folder/empty.txt
		// digest(empty.txt) = hex(H('F' ‖ 'empty.txt'))
		f2 := hash.Instance()
		err = f2.ReadBytes([]byte{'F', 'e', 'm', 'p', 't', 'y', '.', 't', 'x', 't'})
		if err != nil {
			t.Fatal(err)
		}
		hashB := f2.Hash()

		// ./example-folder
		// F := hex(H('D' ‖ 'example-folder'))
		f := hash.Instance()
		err = f.ReadBytes([]byte("Dexample-folder"))
		if err != nil {
			t.Fatal(err)
		}
		hashF := f.Hash()

		// digest(example-folder) = F ⊕ digest(example.txt) ⊕ digest(empty.txt)
		XORByteSlices(hashF, hashA)
		XORByteSlices(hashF, hashB)

		actual := hex.EncodeToString(hashF)
		if actual != refDigest {
			t.Errorf(`digest for example folder incorrect (%s): expected %s, got %s`, hash.Instance().Name(), refDigest, actual)
		}
	}
}

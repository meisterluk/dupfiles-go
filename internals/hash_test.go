package internals

import (
	"encoding/hex"
	"testing"
)

func TestSupportedHashAlgorithms(t *testing.T) {
	algos := SupportedHashAlgorithms()
	required := []string{`crc64`, `fnv-1a-32`, `fnv-1a-128`, `sha-256`, `sha-512`, `sha-3-512`}
	for _, requiredAlgo := range required {
		if !Contains(algos, requiredAlgo) {
			t.Errorf(`Expected required hash algo %s, but is not supported`, requiredAlgo)
		}
	}
}

func TestExampleBasenameModeFileHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:       `9e786676af805611`,
		HashCRC32:       `2d908674`,
		HashFNV1_32:     `5c9c3b7e`,
		HashFNV1_64:     `691dc3641785db1e`,
		HashFNV1_128:    `7ad9dbc8df3730b5210420d3fe1f4f66`,
		HashFNV1A32:     `80b426f0`,
		HashFNV1A64:     `20993d48e00ad930`,
		HashFNV1A128:    `e3de215ef6a0c1733b03f77bbb7eb3a0`,
		HashADLER32:     `a18a12e6`,
		HashMD5:         `f49398788c779271464e2ea7c9683710`,
		HashSHA1:        `dc4287e5c9a59af8929b49cff30f759ca9b32181`,
		HashSHA256:      `6dacf7a2ba7a269c846445f1373233b00385eabadd386b4d1bf87d472b656793`,
		HashSHA512:      `830a56639167b60387a4227f7487700d9008583c2db23283e64d112e3e01b59fbc771d795c837080855b829c83ed9e977180d9e94ce6f5b707e296c0c056db47`,
		HashSHA3_512:    `cc7f48110964b6af587e82d2ea5c1e2abbde5e6c047d3408f5f30db1ac5c50357f063846eb5a5caaa78d025d64b2d36b34a0a858e898544cef80601edccc9bdc`,
		HashSHAKE256_64: `0cbbb0b3b0f718ea5ae2b590cd3f27b6253e8cb375dcd04fb9542b7698ef22184f4787feccf0031e499ee6f85da8a0930e7b48d26804b24b51f78b84c0ff3d2bf9498faadd9032d8caacfad3470f4b8d8306025e4bd32bec39d9f06cd4ea8fa351f1f47f89110b496bf58771a179b6cdc71f5f5c50534d0ad46f67ecdd7a1768`,
	}
	for hash, digest := range data {
		h := hash.Algorithm()
		err := h.ReadBytes([]byte{
			'e', 'x', 'a', 'm', 'p', 'l', 'e', '.', 't', 'x', 't', 0x1f,
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}

		actual := h.HexDigest()
		if digest != actual {
			t.Errorf(`digest for example.txt in basename mode incorrect (%s): expected %s, got %s`, hash, digest, actual)
		}
	}
}

func TestExampleEmptyModeFileHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:       `6365a94a3d11ef39`,
		HashCRC32:       `68e17d95`,
		HashFNV1_32:     `e6d4bcab`,
		HashFNV1_64:     `23f7f66c310d910b`,
		HashFNV1_128:    `c1a46322af2cc1806384306316df2333`,
		HashFNV1A32:     `067b6d97`,
		HashFNV1A64:     `3deefb5fdfa573f7`,
		HashFNV1A128:    `32bdefadfe82776b2e5c6c1f97d949ef`,
		HashADLER32:     `ea3c0e4d`,
		HashMD5:         `eddc51f98f9367bffe0dec96da83648c`,
		HashSHA1:        `3af73983ad876cc108ef4cf7b045450a20b35780`,
		HashSHA256:      `2f837632f54939e1824950eeaf5924e8c275a1b8443fc8bf1eab11902d185c4c`,
		HashSHA512:      `295e43d93006798b3608170e92ac883a84eb0635be6041226ca9eda6dab7d1ab7319a59cce44187216e1fb17f94a8ec24ca6df64532765be0da0fef27a88c3f4`,
		HashSHA3_512:    `b9f07ba425705709308518d521f0b93a22844fd181769b49afef188b72ae2e9a4d106fe91c0e4bf008687b2305fa73eb05493f373e50404036dd853de7e37805`,
		HashSHAKE256_64: `11797e0d409ed892bda314a0ada2b9dad31b95f4f77c126a0f4de480bd45b98ade12a00b53c3755cfe7251d35ee88677b13632f7555a3bcc398e9d90b11f37fe9bef7cf75ec2e97dafe9a70acf625fdcaa4f92891346f783e25f026423e687e8905c36174fc5af2628a84bbf4c975024970b48789790c8dd054c930d519500c7`,
	}
	for hash, digest := range data {
		h := hash.Algorithm()
		err := h.ReadBytes([]byte{
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}

		actual := h.HexDigest()
		if digest != actual {
			t.Errorf(`digest for example.txt in empty mode incorrect (%s): expected %s, got %s`, hash, digest, actual)
		}
	}
}

func TestExampleBasenameModeFolderHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:       `12fdd36eec3032fd`,
		HashCRC32:       `3f391536`,
		HashFNV1_32:     `4e9d9cd6`,
		HashFNV1_64:     `9af25620401ca756`,
		HashFNV1_128:    `d6ef1d185bf1907dc12234ae33177b06`,
		HashFNV1A32:     `df684008`,
		HashFNV1A64:     `f7144cd19e46c2e8`,
		HashFNV1A128:    `d94a7ad6d3279812f9337691bf9478a0`,
		HashADLER32:     `0896027a`,
		HashMD5:         `30a69d798b3f08d0d9cf2158a0e5b38b`,
		HashSHA1:        `103f55c4986da499b73feaf6db017ff7ee47a18a`,
		HashSHA256:      `74f89ae535f5985b646b42de2638624ce71df2311018c2469ca44189ee5cf278`,
		HashSHA512:      `4cf6df22f8489be675674236238db9481054d9b66d2ba6ffefb21dcceca0d1aa3da2047fb9c6a65359ed639045a80dfb14610faf81e39022114de9f5b404619e`,
		HashSHA3_512:    `72076ebd716fe9ea90f70556e12c40e6f7ddcdceb4419af61561f01748e9dffa225dd67b31de1304f36491b0e80cbfc82aa647eec0b816f57f1ee71147699fb2`,
		HashSHAKE256_64: `2aabf233527d02d986ca8bf2ed526c0c93dc631c5e84513250e2b8a6f700b152387c579d6e3c4ada99b6cfb4695dcb14cb017fb2b7652b653482f7eb010af609f9efd1e38683aabbf0921dff58ecc4aaf1842a68dd9a2987dc14f4a60579216f0484ba3a67e8572bb5c4a5bbb24cbbf909acbf84b012c3d6e5fbd2b37899ba6c`,
	}
	for hash, digest := range data {
		// Determine digest of the following folder structure:
		// ./folder/a.txt
		// ./folder/b.txt

		f1 := hash.Algorithm()
		err := f1.ReadBytes([]byte{
			'a', '.', 't', 'x', 't', 0x1f,
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}
		hashA := f1.Digest()

		f2 := hash.Algorithm()
		err = f2.ReadBytes([]byte{
			'b', '.', 't', 'x', 't', 0x1f,
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}
		hashB := f2.Digest()

		f := hash.Algorithm()
		err = f.ReadBytes([]byte("folder"))
		if err != nil {
			t.Fatal(err)
		}
		hashF := f.Digest()

		XORByteSlices(hashF, hashA)
		XORByteSlices(hashF, hashB)

		actual := hex.EncodeToString(hashF)
		if actual != digest {
			t.Errorf(`digest for example folder incorrect (%s): expected %s, got %s`, hash, digest, actual)
		}
	}
}

func TestExampleEmptyModeFolderHashes(t *testing.T) {
	data := map[HashAlgo]string{
		HashCRC64:       `0000000000000000`,
		HashCRC32:       `00000000`,
		HashFNV1_32:     `00000000`,
		HashFNV1_64:     `0000000000000000`,
		HashFNV1_128:    `00000000000000000000000000000000`,
		HashFNV1A32:     `00000000`,
		HashFNV1A64:     `0000000000000000`,
		HashFNV1A128:    `00000000000000000000000000000000`,
		HashADLER32:     `00000000`,
		HashMD5:         `00000000000000000000000000000000`,
		HashSHA1:        `0000000000000000000000000000000000000000`,
		HashSHA256:      `0000000000000000000000000000000000000000000000000000000000000000`,
		HashSHA512:      `00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000`,
		HashSHA3_512:    `00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000`,
		HashSHAKE256_64: `0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000`,
	}
	for hash, digest := range data {
		// Determine digest of the following folder structure:
		// ./folder/a.txt
		// ./folder/b.txt

		f1 := hash.Algorithm()
		err := f1.ReadBytes([]byte{
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}
		hashA := f1.Digest()

		f2 := hash.Algorithm()
		err = f2.ReadBytes([]byte{
			0x64, 0x75, 0x70, 0x66, 0x69, 0x6c, 0x65, 0x73, 0x20, 0x67, 0x65, 0x6e, 0x65, 0x72, 0x61, 0x74,
			0x65, 0x73, 0x20, 0x72, 0xce, 0xb5, 0x70, 0x6f, 0x72, 0x74, 0x73, 0x0a, 0xf0, 0x9f, 0x98, 0x8a,
			0x0a,
		})
		if err != nil {
			t.Fatal(err)
		}
		hashB := f2.Digest()

		XORByteSlices(hashA, hashB)

		actual := hex.EncodeToString(hashA)
		if actual != digest {
			t.Errorf(`digest for example folder incorrect (%s): expected %s, got %s`, hash, digest, actual)
		}
	}
}

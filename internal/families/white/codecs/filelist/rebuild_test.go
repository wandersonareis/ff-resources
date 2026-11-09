package filelist

import (
	"bytes"
	"encoding/binary"
	"os"
	"testing"
)

// TestRebuildStructuralEquality avalia a exatidão do "re-crypt"/rebuild:
//
//   - o cabeçalho, a tabela de entradas e o conteúdo DESCOMPRIMIDO dos chunks
//     devem ser idênticos ao original;
//   - os bytes COMPRIMIDOS podem divergir (compressores diferentes dificilmente
//     geram o mesmo fluxo), bem como o corpo cifrado (que depende deles) e o
//     checksum final.
//
// Isso confirma que o rebuild é semanticamente idêntico e criptograficamente
// válido, mesmo quando o arquivo não é byte a byte igual.
func TestRebuildStructuralEquality(t *testing.T) {
	cases := []struct {
		name string
		path string
		game GameCode
	}{
		{"ff13-1", `D:\Steam\steamapps\common\FINAL FANTASY XIII\white_data\sys\filelistu.win32.bin`, GameFF131},
		{"ff13-2", `D:\Steam\steamapps\common\FINAL FANTASY XIII-2\alba_data\sys\filelistu.win32.bin`, GameFF132},
		{"ff13-lr", `D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\filelista.win32.bin`, GameFF132},
		// PS3 (FF13-2, do disco).
		{"ff13-2-ps3", `H:\PS3_GAME\USRDIR\alba_data\sys\filelistu.ps3.bin`, GameFF132},
		{"ff13-2-ps3-scr", `H:\PS3_GAME\USRDIR\alba_data\sys\filelist_scru.ps3.bin`, GameFF132},
		// Xbox 360 (FF13-2, RGH).
		{"ff13-2-x360", `D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\fileliste.x360.bin`, GameFF132},
		{"ff13-2-x360-scr", `D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\filelist_scre.x360.bin`, GameFF132},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			orig, err := os.ReadFile(tc.path)
			if err != nil {
				t.Skipf("filelist Steam não encontrado: %v", err)
			}

			coll, err := Parse(orig, tc.game)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			rebuilt, err := coll.Build()
			if err != nil {
				t.Fatalf("Build: %v", err)
			}

			plainOrig, err := Decrypted(orig, tc.game)
			if err != nil {
				t.Fatalf("Decrypted(orig): %v", err)
			}
			plainReb, err := Decrypted(rebuilt, tc.game)
			if err != nil {
				t.Fatalf("Decrypted(rebuilt): %v", err)
			}

			base := 0
			if coll.Crypt.Encrypted {
				base = 32
			}

			// Cabeçalho lógico.
			hdrO := plainOrig[base : base+12]
			hdrR := plainReb[base : base+12]
			if !bytes.Equal(hdrO, hdrR) {
				t.Fatalf("cabeçalho difere: % x vs % x", hdrO, hdrR)
			}

			chunkInfoOff := int(binary.LittleEndian.Uint32(hdrO[0:4]))
			chunkDataOff := int(binary.LittleEndian.Uint32(hdrO[4:8]))
			fileCount := int(binary.LittleEndian.Uint32(hdrO[8:12]))
			chunkCount := coll.ChunkCount()

			// Tabela de entradas.
			entO := plainOrig[base+12 : base+12+fileCount*8]
			entR := plainReb[base+12 : base+12+fileCount*8]
			if !bytes.Equal(entO, entR) {
				t.Fatalf("tabela de entradas difere")
			}

			var uncmpDiff, cmpDiff int
			var cmpOrigTotal, cmpRebTotal int
			var decompDiff int

			for i := 0; i < chunkCount; i++ {
				p := base + chunkInfoOff + i*12
				uncmpO := binary.LittleEndian.Uint32(plainOrig[p:])
				cmpO := binary.LittleEndian.Uint32(plainOrig[p+4:])
				posO := binary.LittleEndian.Uint32(plainOrig[p+8:])
				uncmpR := binary.LittleEndian.Uint32(plainReb[p:])
				cmpR := binary.LittleEndian.Uint32(plainReb[p+4:])
				posR := binary.LittleEndian.Uint32(plainReb[p+8:])

				if uncmpO != uncmpR {
					uncmpDiff++
				}
				if cmpO != cmpR {
					cmpDiff++
				}
				cmpOrigTotal += int(cmpO)
				cmpRebTotal += int(cmpR)

				dO, err := zlibDecompress(plainOrig[base+chunkDataOff+int(posO) : base+chunkDataOff+int(posO)+int(cmpO)])
				if err != nil {
					t.Fatalf("decompress chunk %d (orig): %v", i, err)
				}
				dR, err := zlibDecompress(plainReb[base+chunkDataOff+int(posR) : base+chunkDataOff+int(posR)+int(cmpR)])
				if err != nil {
					t.Fatalf("decompress chunk %d (rebuilt): %v", i, err)
				}
				if !bytes.Equal(dO, dR) {
					decompDiff++
				}
			}

			t.Logf("files=%d chunks=%d encrypted=%v", fileCount, chunkCount, coll.Crypt.Encrypted)
			t.Logf("chunks com tamanho descomprimido diferente: %d", uncmpDiff)
			t.Logf("chunks com tamanho comprimido diferente:    %d (esperado: quase todos)", cmpDiff)
			t.Logf("chunks com conteúdo descomprimido diferente:%d (esperado: 0)", decompDiff)
			t.Logf("bytes comprimidos: orig=%d rebuilt=%d (delta=%d)", cmpOrigTotal, cmpRebTotal, cmpRebTotal-cmpOrigTotal)
			t.Logf("tamanho do arquivo: orig=%d rebuilt=%d", len(orig), len(rebuilt))

			if !coll.Crypt.Encrypted {
				// Sem criptografia, só os bytes comprimidos podem diferir. O
				// resto do arquivo deve ser idêntico.
				diff := countDiff(orig, rebuilt)
				t.Logf("bytes diferentes no arquivo (plaintext): %d de %d", diff, len(orig))
			} else {
				cipherDiff := countDiff(orig, rebuilt)
				t.Logf("bytes diferentes no arquivo (cifrado):   %d de %d", cipherDiff, len(orig))
				// Confirma que o checksum do rebuild é internamente válido.
				if _, err := Parse(rebuilt, tc.game); err != nil {
					t.Fatalf("rebuild não valida: %v", err)
				}
			}

			if uncmpDiff != 0 || cmpDiff == 0 || decompDiff != 0 {
				t.Fatalf("avaliação inesperada: uncmpDiff=%d cmpDiff=%d decompDiff=%d", uncmpDiff, cmpDiff, decompDiff)
			}
		})
	}
}

// TestCipherIsExactInverse prova que a criptografia em si é byte-exata:
// decifrar e cifrar de volta o MESMO conteúdo reproduz o arquivo original
// exatamente. A não-exatidão do rebuild vem apenas da recompressão dos chunks.
func TestCipherIsExactInverse(t *testing.T) {
	paths := []string{
		`D:\Steam\steamapps\common\FINAL FANTASY XIII-2\alba_data\sys\filelistu.win32.bin`,
		`D:\Steam\steamapps\common\LIGHTNING RETURNS FINAL FANTASY XIII\weiss_data\sys\filelista.win32.bin`,
		`H:\PS3_GAME\USRDIR\alba_data\sys\filelistu.ps3.bin`,
		`D:\Mestre\Compressed\Final Fantasy XIII-2 Xbox 360 Rgh\Final Fantasy XIII-2\alba_data\sys\fileliste.x360.bin`,
	}
	for _, path := range paths {
		path := path
		t.Run(path, func(t *testing.T) {
			data, err := os.ReadFile(path)
			if err != nil {
				t.Skipf("filelist não encontrado: %v", err)
			}
			if len(data) < 32 {
				t.Skip("arquivo curto")
			}
			var md5 [16]byte
			copy(md5[:], data[0:16])
			bodySize := binary.BigEndian.Uint32(data[16:20])
			cryptBodySize := bodySize + 8
			xt := generateXORTable(seedFromMD5(md5))

			work := append([]byte(nil), data...)
			cryptBody(xt, work[32:], cryptBodySize/8, false) // decrypt
			cryptBody(xt, work[32:], cryptBodySize/8, true)  // encrypt de volta
			if !bytes.Equal(work, data) {
				t.Fatalf("encrypt(decrypt(x)) != x (cifra não é inversa exata)")
			}
		})
	}
}

func countDiff(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	diff := abs(len(a) - len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			diff++
		}
	}
	return diff
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

package filelist

import "encoding/binary"

// Porte de WhiteBinTools.Crypto (Generator, CryptoBase, CryptoFunctions,
// Decryption, Encryption, CryptFilelist). O esquema é um "stream cifra" por
// blocos de 8 bytes, com tabela XOR derivada de uma semente e uma chave
// especial que muda a cada bloco.
//
// É o mesmo esquema em PC, PS3 e Xbox 360: a semente vem dos 16 primeiros
// bytes do próprio arquivo, então o algoritmo não tem nenhum parâmetro por
// plataforma. Validado contra os três (decriptação byte-idêntica ao
// ffxiiicrypt).

// integers é a tabela de permutação usada por LoopAByte/LoopAByteReverse.
// Equivale a IntegersArray.Integers, ou seja (i + 120) mod 256.
var integers = func() [256]byte {
	var a [256]byte
	for i := range a {
		a[i] = byte((i + 120) & 0xFF)
	}
	return a
}()

// integersInv é a inversa de integers.
var integersInv = func() [256]byte {
	var a [256]byte
	for i := 0; i < 256; i++ {
		a[(i+120)&0xFF] = byte(i)
	}
	return a
}()

// generateXORTable reproduz Generator.GenerateXORtable a partir de uma
// semente de 8 bytes (seedArray). A tabela tem 264 bytes (33 blocos de 8).
func generateXORTable(seedArray [8]byte) []byte {
	xorTable := make([]byte, 264)

	// Array.Reverse(seedArray)
	rev := [8]byte{seedArray[7], seedArray[6], seedArray[5], seedArray[4], seedArray[3], seedArray[2], seedArray[1], seedArray[0]}

	seedHalfA := binary.LittleEndian.Uint32(rev[0:4])
	seedHalfB := binary.LittleEndian.Uint32(rev[4:8])

	seedHalfA = (seedHalfA << 0x08) | (seedHalfA >> 0x18)
	seedHalfB = (seedHalfB >> 0x10) | (seedHalfB << 0x10)

	xorBlock := make([]byte, 8)
	binary.LittleEndian.PutUint32(xorBlock[0:4], seedHalfB)
	binary.LittleEndian.PutUint32(xorBlock[4:8], seedHalfA)
	xorBlock[0] += 0x45

	// Loop 1
	for i := 1; i < 8; i++ {
		tmp := int(xorBlock[i]) + 0xD4 + int(xorBlock[i-1])
		tmp ^= int(xorBlock[i-1]) << 2
		tmp ^= 0x45
		xorBlock[i] = byte(tmp)
	}

	copy(xorTable[0:8], xorBlock)

	// Loop 2
	previousXORBlock := binary.LittleEndian.Uint64(xorBlock)
	copyIndex := 8
	for i := 1; i < 0x21; i++ {
		blockHalfA := uint32(previousXORBlock & 0xFFFFFFFF)
		blockHalfB := uint32(previousXORBlock >> 32)

		a := 5 * previousXORBlock
		a ^= uint64(blockHalfB) << 32

		tmpBlockHalfA := uint32(blockHalfA ^ uint32(a))
		tmpBlockHalfB := uint32(a >> 32)

		a = uint64(blockHalfA) | (a & 0xFFFFFFFF00000000)

		b := uint64(tmpBlockHalfA)

		xorBlockHalfA := uint32(a ^ b)
		tmpBlockHalfB ^= blockHalfB
		xorBlockHalfB := uint32(tmpBlockHalfB)

		binary.LittleEndian.PutUint32(xorBlock[0:4], xorBlockHalfA)
		binary.LittleEndian.PutUint32(xorBlock[4:8], xorBlockHalfB)

		copy(xorTable[copyIndex:copyIndex+8], xorBlock)

		previousXORBlock = binary.LittleEndian.Uint64(xorBlock)
		copyIndex += 8
	}

	return xorTable
}

// seedFromMD5 extrai a semente de 8 bytes dos 16 bytes de cabeçalho, do mesmo
// modo que CryptFilelist.ProcessFilelist.
func seedFromMD5(baseSeed [16]byte) [8]byte {
	// O WhiteBinTools calcula isso em aritmética de int32 e depois converte
	// para ulong. Quando o byte alto (baseSeed[9]) tem o bit 0x80 ligado, o
	// int32 fica negativo e o cast para ulong faz extensão de sinal
	// (0xFFFFFFFF...). Replicamos exatamente esse comportamento.
	raw := int32(baseSeed[9])<<24 | int32(baseSeed[12])<<16 | int32(baseSeed[2])<<8 | int32(baseSeed[0])
	var out [8]byte
	putLE64(out[:], uint64(int64(raw)))
	return out
}

// blockCounterSetup reproduz CryptoBase.BlockCounterSetup. Retorna o offset na
// tabela XOR e os valores "eval"/"fval" usados na chave especial.
func blockCounterSetup(blockCounter uint32) (tableOffset, eval, fval uint32) {
	bcLow := uint16(blockCounter & 0xFFFF)
	bcLow >>= 3
	bcLow <<= 3
	tableOffset = uint32(bcLow) & 0xF8

	abShift := int64(blockCounter) << 10
	cdShift := int64(blockCounter) << 20
	efShift := int64(blockCounter) << 30

	a := uint32(uint64(abShift) & 0xFFFFFFFF)
	bb := uint32(uint64(abShift) >> 32)
	c := uint32(uint64(cdShift) & 0xFFFFFFFF)
	d := uint32(uint64(cdShift) >> 32)

	c |= a
	d |= bb

	e := uint32(uint64(efShift) & 0xFFFFFFFF)
	f := uint32(uint64(efShift) >> 32)

	e |= blockCounter
	f |= 0
	e |= c
	f |= d

	return tableOffset, e, f
}

// loopAByte é CryptoFunctions.LoopAByte: 8 rodadas de b = (integers[b]-x) mod 256.
func loopAByte(v uint32, xorTable []byte, tableOffset uint32) uint32 {
	for i := 0; i < 8; i++ {
		computed := int(integers[byte(v)]) - int(xorTable[tableOffset+uint32(i)])
		v = uint32(computed) & 0xFF
	}
	return v
}

// loopAByteReverse é CryptoFunctions.LoopAByteReverse. O if/else do original
// é equivalente a usar (x + b) mod 256 e buscar na tabela inversa.
func loopAByteReverse(b byte, xorTable []byte, tableOffset uint32) uint32 {
	for i := 7; i >= 0; i-- {
		v := int(xorTable[tableOffset+uint32(i)]) + int(b)
		b = integersInv[byte(v)]
	}
	return uint32(b)
}

// specialKeys reproduz CryptoBase.SpecialKeySetup.
func specialKeys(eval, fval uint32) (specialKey1, specialKey2 uint64) {
	var carry uint32
	if eval > ^uint32(0xA1652347) { // > 0x5E9ADCB8
		carry = 1
	}
	return uint64(eval) + 0xA1652347, uint64(fval) + uint64(carry)
}

// arrayToFFNum replica CryptoFunctions.ArrayToFFNum: "FFFFFFFF"+BE(bytes).
func arrayToFFNum(a [4]byte) uint64 {
	return 0xFFFFFFFF00000000 | uint64(a[0])<<24 | uint64(a[1])<<16 | uint64(a[2])<<8 | uint64(a[3])
}

// computeCheckSum é CryptoFunctions.ComputeCheckSum: soma dos bytes em
// data[readPos], data[readPos+4], ... totalizando "blocks" leituras.
func computeCheckSum(data []byte, blocks, readPos uint32) uint32 {
	var chkSum uint32
	pos := readPos
	for i := uint32(0); i < blocks; i++ {
		if int(pos) >= len(data) {
			break
		}
		chkSum += uint32(data[pos])
		pos += 4
	}
	return chkSum
}

// cryptBody cifra ou decifra, in-place, os "blockCount" blocos de 8 bytes em
// body (que corresponde ao corpo a partir do offset 32 do arquivo).
func cryptBody(xorTable, body []byte, blockCount uint32, encrypt bool) {
	var blockCounter uint32
	for i := uint32(0); i < blockCount; i++ {
		readPos := int(i) * 8
		if readPos+8 > len(body) {
			break
		}
		b := body[readPos : readPos+8]
		currentBlockId := blockCounter >> 3
		tableOffset, eval, fval := blockCounterSetup(blockCounter)
		xorLower := binary.LittleEndian.Uint32(xorTable[tableOffset:])
		xorHigher := binary.LittleEndian.Uint32(xorTable[tableOffset+4:])
		sk1, sk2 := specialKeys(eval, fval)

		if encrypt {
			lowerArr := [4]byte{b[7], b[6], b[5], b[4]}
			higherArr := [4]byte{b[3], b[2], b[1], b[0]}
			lower := arrayToFFNum(lowerArr)
			higher := arrayToFFNum(higherArr)

			lower ^= uint64(xorLower)
			higher ^= uint64(xorHigher)
			lower ^= sk1
			higher ^= sk2
			lower += uint64(xorLower)
			higher += uint64(xorHigher)

			var borrow uint64
			if lower&0xFFFFFFFF < uint64(xorLower) {
				borrow = 1
			}
			higher += borrow

			var computed [8]byte
			putLE32(computed[0:4], uint32(lower))
			putLE32(computed[4:8], uint32(higher))

			prev := uint32((currentBlockId ^ 69) & 0xFF)
			for k := 0; k < 8; k++ {
				prev ^= loopAByteReverse(computed[k], xorTable, tableOffset)
				b[k] = byte(prev)
			}
			blockCounter += 8
			continue
		}

		// Decifrar
		d1 := loopAByte(((currentBlockId^69)&0xFF)^uint32(b[0]), xorTable, tableOffset)
		d2 := loopAByte(uint32(b[0])^uint32(b[1]), xorTable, tableOffset)
		d3 := loopAByte(uint32(b[1])^uint32(b[2]), xorTable, tableOffset)
		d4 := loopAByte(uint32(b[2])^uint32(b[3]), xorTable, tableOffset)
		d5 := loopAByte(uint32(b[3])^uint32(b[4]), xorTable, tableOffset)
		d6 := loopAByte(uint32(b[4])^uint32(b[5]), xorTable, tableOffset)
		d7 := loopAByte(uint32(b[5])^uint32(b[6]), xorTable, tableOffset)
		d8 := loopAByte(uint32(b[6])^uint32(b[7]), xorTable, tableOffset)

		higher := uint64(binary.LittleEndian.Uint32([]byte{byte(d5), byte(d6), byte(d7), byte(d8)}))
		lower := uint64(binary.LittleEndian.Uint32([]byte{byte(d1), byte(d2), byte(d3), byte(d4)}))

		var borrow uint64
		if lower < uint64(xorLower) {
			borrow = 1
		}
		lower -= uint64(xorLower)
		higher -= uint64(xorHigher)
		higher -= borrow

		lower ^= sk1
		higher ^= sk2
		lower ^= uint64(xorLower)
		higher ^= uint64(xorHigher)

		putLE32(b[0:4], uint32(higher))
		putLE32(b[4:8], uint32(lower))

		blockCounter += 8
	}
}

func putLE32(b []byte, v uint32) { binary.LittleEndian.PutUint32(b, v) }
func putLE64(b []byte, v uint64) { binary.LittleEndian.PutUint64(b, v) }

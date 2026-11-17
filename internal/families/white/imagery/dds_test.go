package imagery

import (
	"encoding/binary"
	"testing"
)

func testDDSHeader(fourCC, pfFlags, alphaMask, width, height, mips, caps2 uint32) []byte {
	buf := make([]byte, ddsHeaderLen)
	copy(buf, "DDS ")
	binary.LittleEndian.PutUint32(buf[4:8], 124)
	binary.LittleEndian.PutUint32(buf[12:16], height)
	binary.LittleEndian.PutUint32(buf[16:20], width)
	binary.LittleEndian.PutUint32(buf[28:32], mips)
	binary.LittleEndian.PutUint32(buf[80:84], pfFlags)
	binary.LittleEndian.PutUint32(buf[84:88], fourCC)
	binary.LittleEndian.PutUint32(buf[104:108], alphaMask)
	binary.LittleEndian.PutUint32(buf[112:116], caps2)
	return buf
}

func TestParseDDSHeaderFormats(t *testing.T) {
	cases := []struct {
		name    string
		dds     []byte
		want    DDSFormat
		wantErr bool
	}{
		{"dxt1", testDDSHeader(fccDXT1, ddpfFourCC, 0, 64, 32, 3, 0), DDSFormatDXT1, false},
		{"dxt3", testDDSHeader(fccDXT3, ddpfFourCC, 0, 64, 32, 1, 0), DDSFormatDXT3, false},
		{"dxt5", testDDSHeader(fccDXT5, ddpfFourCC, 0, 64, 32, 1, 0), DDSFormatDXT5, false},
		{"bc5", testDDSHeader(fccBC5U, ddpfFourCC, 0, 64, 32, 1, 0), DDSFormatBC5, false},
		{"rgb", testDDSHeader(0, ddpfRGB, 0, 64, 32, 1, 0), DDSFormatRGB, false},
		{"rgba", testDDSHeader(0, ddpfRGB|ddpfAlphaPixels, 0xFF000000, 64, 32, 1, 0), DDSFormatRGBA, false},
		{"dx10", testDDSHeader(ddsFourCCDX10, ddpfFourCC, 0, 64, 32, 1, 0), DDSFormatUnknown, true},
		{"magic", []byte("NOPE"), DDSFormatUnknown, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := ParseDDSHeader(tc.dds)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("esperava erro, veio %+v", info)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseDDSHeader: %v", err)
			}
			if info.Format != tc.want {
				t.Fatalf("formato %v, quer %v", info.Format, tc.want)
			}
			if info.Width != 64 || info.Height != 32 {
				t.Fatalf("dimensões %dx%d, quer 64x32", info.Width, info.Height)
			}
			if info.MipMapCount == 0 {
				t.Fatal("mipmaps zerado")
			}
		})
	}
}

func TestParseDDSHeaderCubemap(t *testing.T) {
	info, err := ParseDDSHeader(testDDSHeader(fccDXT1, ddpfFourCC, 0, 8, 8, 2, 0xFE00))
	if err != nil {
		t.Fatal(err)
	}
	if info.Caps2&0x200 == 0 {
		t.Fatal("cubemap não detectado pelo caps2")
	}
}

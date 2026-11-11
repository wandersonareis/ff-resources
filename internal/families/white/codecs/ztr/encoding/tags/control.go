// Origem: ztrgo/backend/core/encoding/key_tables.go, reorganizado por grupo
// de tag. Os bytes são a identidade; nomes e swatches
// são apenas display.

package tags

// controlTags: controles de byte único (0x00..0x05).
var controlTags = []ByteTag{
	{Key: 0x00, Value: "{END}"},
	{Key: 0x01, Value: "{ESC}"},
	{Key: 0x02, Value: "{TEXT_ITALIC}"},
	{Key: 0x03, Value: "{LINE}"},
	{Key: 0x04, Value: "{ART}"},
	{Key: 0x05, Value: "{ART2}"},
}

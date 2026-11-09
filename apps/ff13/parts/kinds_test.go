package parts

import "testing"

// TestTextKind confere a regra: o kind do TEXTO sai do DIRETORIO (a pasta
// abaixo de txtres), nunca da extensao do arquivo — o FF13 tem um formato de
// texto so (o .ztr), entao a pasta e o que separa os tipos.
func TestTextKind(t *testing.T) {
	cases := []struct {
		part string
		path string
		want string
	}{
		// txtres GLOBAL: a pasta abaixo de txtres e o kind.
		{"part1", "txtres/event/ev_comn_000/txtres_us.ztr", KindEvent},
		{"part1", "txtres/resident/book/txtres_us.ztr", KindResident},
		{"part1", "txtres/resident/system/txtres_us.ztr", KindResident},
		{"part1", "txtres/zone/z000/txtres_us.ztr", KindZone},
		// txtres DE ZONA (part1): o mesmo `ac` vive dentro das zonas.
		{"part1", "zone/z004/txtres/ac/ac_bodm_230/txtres_us.ztr", KindAC},
		{"part1", "zone/z004/txtres/event/ev_bodm_200/txtres_us.ztr", KindEvent},
		{"part1", "zone/z004/txtres/zone/z004/txtres_us.ztr", KindZone},
		// part2: ac/resident/zone/event globais.
		{"part2", "txtres/ac/ac_aaea_007/txtres_us.ztr", KindAC},
		{"part2", "txtres/resident/mission/txtres_us.ztr", KindResident},
		{"part2", "txtres/zone/z0030/txtres_us.ztr", KindZone},
		{"part2", "txtres/event/ev_aaea_010/txtres_us.ztr", KindEvent},
		// part3: resident/zone/event + scenario (db/ai/npc/pack).
		{"part3", "txtres/event/ev_comn_020/txtres_us.ztr", KindEvent},
		{"part3", "txtres/resident/book/txtres_us.ztr", KindResident},
		{"part3", "txtres/zone/z0001/txtres_us.ztr", KindZone},
		{"part3", "db/ai/npc/pack/auto_c01_us.bin", KindScenario},
		{"part3", "db/ai/npc/pack/quest1008_us.bin", KindScenario},
		// Fora do txtres e fora do scenario: fallback.
		{"part1", "gui/resident/tutorial/x_us.ztr", KindText},
		{"part3", "sys/script/btl/x.bin", KindText},
		// Cache do SO nao importa.
		{"part2", "TXTRES/AC/AC_AAEA_007/TXTRES_US.ZTR", KindAC},
	}
	for _, c := range cases {
		if got := Get(c.part).TextKind(c.path); got != c.want {
			t.Errorf("%s: TextKind(%q) = %q, quer %q", c.part, c.path, got, c.want)
		}
	}
}

// TestKindsPerPart documenta quais kinds cada part SERVE: o escopo de
// export/import so oferece os que existem. O `ac` do part1 so aparece na
// zona (mas o kind existe); o `scenario` e so do part3; o part3 nao tem `ac`.
func TestKindsPerPart(t *testing.T) {
	cases := map[string][]string{
		"part1": {KindAC, KindResident, KindZone, KindEvent},
		"part2": {KindAC, KindResident, KindZone, KindEvent},
		"part3": {KindResident, KindZone, KindEvent, KindScenario},
	}
	for part, want := range cases {
		got := Kinds(part)
		if len(got) != len(want) {
			t.Errorf("%s: kinds = %v, quer %v", part, got, want)
			continue
		}
		for _, k := range want {
			if !Get(part).HasTextKind(k) {
				t.Errorf("%s: deveria servir o kind %q", part, k)
			}
		}
	}
	// O part3 nao tem `ac`; o part1/2 nao tem `scenario`.
	if Get("part3").HasTextKind(KindAC) {
		t.Error("part3 nao deveria servir ac")
	}
	if Get("part1").HasTextKind(KindScenario) || Get("part2").HasTextKind(KindScenario) {
		t.Error("part1/part2 nao deveriam servir scenario")
	}
}

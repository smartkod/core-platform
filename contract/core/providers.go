package core

// IdentityProvider is implemented by any module that can authenticate requests.
//
// The bootstrap fills Deps.Identity from whichever module satisfies this, so no
// module has to import another. Files, notifications and application modules all
// depend on the interface and never on the identity package.
type IdentityProvider interface {
	Identity() Identity
}

// FileProvider is implemented by the files module.
//
// It exists so other modules can request the storage service through Deps
// without importing mod/files, and so a replacement implementation stays
// possible.
type FileProvider interface {
	Files() Files
}

// StorageProvider is implemented by whichever module owns object storage.
type StorageProvider interface {
	Storage() Storage
}

// EconomyProvider is implemented by the economy module (L2).
//
// The bootstrap fills Deps.Economy from whichever module satisfies this, so
// an application calls Wallet()/Ads() through contract/economy and never
// imports the module package.
type EconomyProvider interface {
	Economy() Economy
}

package core

// IdentityProvider is implemented by any module that can authenticate requests.
//
// The bootstrap fills Deps.Identity from whichever module satisfies this, so no
// module has to import another. Files, notifications and application modules all
// depend on the interface and never on the identity package.
type IdentityProvider interface {
	Identity() Identity
}

// UserProvider is implemented by the identity module.
//
// Deps.Users is the read-only account projection engines use to display a
// person's name; it cannot change an account.
type UserProvider interface {
	Users() UserStore
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

// AuthorizerProvider is implemented by the authorization module.
//
// The bootstrap fills Deps.Authorizer from it, so organization, catalog and
// application modules check permissions without importing mod/authz.
type AuthorizerProvider interface {
	Authorizer() Authorizer
}

// OrganizationProvider is implemented by the organization module.
type OrganizationProvider interface {
	Organizations() Organizations
}

// CatalogProvider is implemented by the catalog module.
type CatalogProvider interface {
	Catalog() Catalog
}
